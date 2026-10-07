package signaling

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/ice/v2"
	"github.com/pion/webrtc/v4"

	"github.com/guanshanh/wavecall/internal/config"
	"github.com/guanshanh/wavecall/internal/room"
	"github.com/guanshanh/wavecall/internal/sfu"
	"github.com/guanshanh/wavecall/pkg/proto"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 开发环境允许所有跨域
	},
}

// Handler manages WebSocket connections and routes signaling messages.
type Handler struct {
	manager *room.Manager
	config  *config.Config
	mu      sync.Mutex

	// api 非 nil 表示已启用单端口媒体复用：SettingEngine 携带 ICE UDPMux，
	// 并在配置了 PublicIP 时把 host 候选直接改写为公网地址（无需 STUN）。
	api    *webrtc.API
	udpMux ice.UDPMux

	// roomID -> (userID -> *Client)
	roomClients map[string]map[string]*Client

	// client -> clientMeta
	clientMeta map[*Client]clientMeta
}

type clientMeta struct {
	roomID   string
	userID   string
	userName string
}

// NewHandler creates a new signaling handler. When cfg.UDPPort > 0 all peer
// connections share one UDP socket (UDPMux); with cfg.PublicIP set, host
// candidates are rewritten to the public IP so no STUN server is needed.
func NewHandler(manager *room.Manager, cfg *config.Config) (*Handler, error) {
	h := &Handler{
		manager:     manager,
		config:      cfg,
		roomClients: make(map[string]map[string]*Client),
		clientMeta:  make(map[*Client]clientMeta),
	}

	if cfg.UDPPort > 0 {
		se := webrtc.SettingEngine{}
		udpMux, err := ice.NewMultiUDPMuxFromPort(cfg.UDPPort)
		if err != nil {
			return nil, fmt.Errorf("create udp mux on port %d: %w", cfg.UDPPort, err)
		}
		h.udpMux = udpMux
		se.SetICEUDPMux(h.udpMux)
		if cfg.PublicIP != "" {
			// Rewrite host candidates to the configured address, and ignore
			// other interfaces (docker / VPN TUN like 198.18.0.1) so clients
			// don't try unreachable ICE paths.
			pubIP := net.ParseIP(cfg.PublicIP)
			if pubIP == nil {
				return nil, fmt.Errorf("invalid public-ip %q", cfg.PublicIP)
			}
			se.SetNAT1To1IPs([]string{cfg.PublicIP}, webrtc.ICECandidateTypeHost)
			se.SetIPFilter(func(ip net.IP) bool {
				return ip.Equal(pubIP)
			})
		}
		h.api = webrtc.NewAPI(webrtc.WithSettingEngine(se))
		slog.Info("udp mux enabled", "port", cfg.UDPPort, "publicIP", cfg.PublicIP)
	}

	return h, nil
}

// Close releases shared resources (the UDP mux socket).
func (h *Handler) Close() {
	if h.udpMux != nil {
		if err := h.udpMux.Close(); err != nil {
			slog.Warn("failed to close udp mux", "err", err)
		}
	}
}

// HandleWebSocket upgrades HTTP to WebSocket and starts message processing.
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "err", err)
		return
	}

	slog.Info("new websocket connection", "remote", r.RemoteAddr)

	client := &Client{
		conn: conn,
		send: make(chan []byte, 64),
	}

	// Start write pump
	go client.writePump()

	// Read loop (blocks until disconnect)
	h.readPump(client)
}

// readPump reads messages from the WebSocket and routes them.
func (h *Handler) readPump(client *Client) {
	defer func() {
		h.handleDisconnect(client)
		client.conn.Close()
	}()

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Error("websocket read error", "err", err)
			}
			return
		}
		h.routeMessage(client, message)
	}
}

// routeMessage parses the message type and dispatches to the appropriate handler.
func (h *Handler) routeMessage(client *Client, raw []byte) {
	var env proto.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		slog.Warn("invalid message", "err", err)
		return
	}

	switch env.Type {
	case proto.TypeJoin:
		var msg proto.JoinMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			slog.Warn("invalid join message", "err", err)
			return
		}
		h.handleJoin(client, &msg)

	case proto.TypeLeave:
		h.handleLeave(client)

	case proto.TypeOffer:
		// SFU 模式下协商由服务端发起，客户端不应发送 offer
		slog.Warn("unexpected offer from client (SFU initiates negotiation)")

	case proto.TypeAnswer:
		var msg proto.AnswerMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			return
		}
		h.handleAnswer(client, &msg)

	case proto.TypeCandidate:
		var msg proto.CandidateMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			return
		}
		h.handleCandidate(client, &msg)

	case proto.TypeMute, proto.TypeUnmute:
		h.handleMuteToggle(client, env.Type)

	default:
		slog.Warn("unknown message type", "type", env.Type)
	}
}

// handleJoin processes a join request.
func (h *Handler) handleJoin(client *Client, msg *proto.JoinMessage) {
	if msg.RoomID == "" || msg.UserName == "" {
		_ = client.Send(proto.ErrorMessage{
			Type:    proto.TypeError,
			Code:    "INVALID_PARAM",
			Message: "房间号和昵称不能为空",
		})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	r := h.manager.GetOrCreate(msg.RoomID, msg.Password)
	if !r.CheckPassword(msg.Password) {
		_ = client.Send(proto.ErrorMessage{
			Type:    proto.TypeError,
			Code:    "WRONG_PASSWORD",
			Message: "房间密码错误",
		})
		return
	}

	if r.PeerCount() >= 5 {
		_ = client.Send(proto.ErrorMessage{
			Type:    proto.TypeError,
			Code:    "ROOM_FULL",
			Message: "房间已满（最多5人）",
		})
		return
	}

	// 生成唯一 userID
	userID := fmt.Sprintf("u_%d_%03d", time.Now().UnixMilli()%1000000, rand.Intn(1000))
	client.userID = userID
	client.roomID = msg.RoomID

	// 获取已在房间中的 peer 列表
	existingPeers := r.Peers()
	peerInfos := make([]proto.PeerInfo, 0, len(existingPeers))
	for _, p := range existingPeers {
		peerInfos = append(peerInfos, proto.PeerInfo{
			UserID:   p.UserID,
			UserName: p.UserName,
			Muted:    p.Muted,
		})
	}

	// 记录到房间
	r.AddPeer(userID, msg.UserName)

	// 记录到 Handler 活跃客户端表
	if _, ok := h.roomClients[msg.RoomID]; !ok {
		h.roomClients[msg.RoomID] = make(map[string]*Client)
	}
	h.roomClients[msg.RoomID][userID] = client
	h.clientMeta[client] = clientMeta{
		roomID:   msg.RoomID,
		userID:   userID,
		userName: msg.UserName,
	}

	// 构建 ICE 服务器列表
	iceServers := make([]proto.ICEServer, 0)
	for _, s := range h.config.ICEServers() {
		iceServers = append(iceServers, proto.ICEServer{
			URLs:       s.URLs,
			Username:   s.Username,
			Credential: s.Credential,
		})
	}

	// 1. 发送 joined 确认给加入的客户端
	_ = client.Send(proto.JoinedMessage{
		Type:       proto.TypeJoined,
		UserID:     userID,
		RoomID:     msg.RoomID,
		Peers:      peerInfos,
		ICEServers: iceServers,
	})

	// 2. 广播 peerJoined 给房间其他成员
	peerJoinedMsg := proto.PeerJoinedMessage{
		Type:     proto.TypePeerJoined,
		UserID:   userID,
		UserName: msg.UserName,
	}
	for uid, otherClient := range h.roomClients[msg.RoomID] {
		if uid != userID {
			_ = otherClient.Send(peerJoinedMsg)
		}
	}

	// 3. 创建 SFU Peer 并发起初始协商（joined 先行发出，客户端需先拿到 userId 才能回复 answer）
	if err := h.setupPeer(client, r, userID); err != nil {
		_ = client.Send(proto.ErrorMessage{
			Type:    proto.TypeError,
			Code:    "PEER_SETUP_FAILED",
			Message: "媒体连接建立失败，请重新加入",
		})
		slog.Error("failed to setup sfu peer", "room", msg.RoomID, "userId", userID, "err", err)
		return
	}

	slog.Info("user joined room", "room", msg.RoomID, "user", msg.UserName, "userId", userID, "totalPeers", r.PeerCount())
}

// setupPeer creates the SFU Peer for a freshly joined user, attaches every
// track already published in the room, and kicks off the initial offer.
func (h *Handler) setupPeer(client *Client, r *room.Room, userID string) error {
	iceServers := make([]webrtc.ICEServer, 0)
	for _, s := range h.config.ICEServers() {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs:       s.URLs,
			Username:   s.Username,
			Credential: s.Credential,
		})
	}

	peer, err := sfu.NewPeer(userID, iceServers, h.api,
		func(sdp string) {
			_ = client.Send(proto.OfferMessage{
				Type:     proto.TypeOffer,
				TargetID: userID,
				SDP:      sdp,
			})
		},
		func(c webrtc.ICECandidateInit) {
			msg := proto.CandidateMessage{
				Type:      proto.TypeCandidate,
				TargetID:  userID,
				Candidate: c.Candidate,
			}
			if c.SDPMid != nil {
				msg.SDPMid = *c.SDPMid
			}
			msg.SDPMLineIndex = c.SDPMLineIndex
			slog.Debug("send ice candidate", "userId", userID, "candidate", c.Candidate, "sdpMid", msg.SDPMid, "sdpMLineIndex", msg.SDPMLineIndex)
			_ = client.Send(msg)
		},
	)
	if err != nil {
		return fmt.Errorf("create peer: %w", err)
	}

	router := r.Router()
	router.AddPeer(peer)
	router.AttachPublishedTracks(peer)

	if err := peer.Start(); err != nil {
		router.RemovePeer(userID)
		return fmt.Errorf("start negotiation: %w", err)
	}
	return nil
}

// handleLeave processes an explicit leave request.
func (h *Handler) handleLeave(client *Client) {
	h.handleDisconnect(client)
}

// handleAnswer applies the client's SDP answer to its SFU PeerConnection.
func (h *Handler) handleAnswer(client *Client, msg *proto.AnswerMessage) {
	meta, ok := h.clientMetaOf(client)
	if !ok {
		return
	}

	r, err := h.manager.Get(meta.roomID)
	if err != nil {
		return
	}
	peer := r.Router().Peer(meta.userID)
	if peer == nil {
		return
	}

	if err := peer.HandleRemoteAnswer(msg.SDP); err != nil {
		slog.Error("failed to handle answer", "room", meta.roomID, "userId", meta.userID, "err", err)
	}
}

// handleCandidate forwards an ICE candidate to the client's SFU PeerConnection.
func (h *Handler) handleCandidate(client *Client, msg *proto.CandidateMessage) {
	if msg.Candidate == "" {
		return
	}

	meta, ok := h.clientMetaOf(client)
	if !ok {
		return
	}

	r, err := h.manager.Get(meta.roomID)
	if err != nil {
		return
	}
	peer := r.Router().Peer(meta.userID)
	if peer == nil {
		return
	}

	init := webrtc.ICECandidateInit{Candidate: msg.Candidate}
	if msg.SDPMid != "" {
		mid := msg.SDPMid
		init.SDPMid = &mid
	}
	init.SDPMLineIndex = msg.SDPMLineIndex
	if err := peer.HandleRemoteCandidate(init); err != nil {
		slog.Error("failed to add ICE candidate", "room", meta.roomID, "userId", meta.userID, "err", err)
	}
}

// clientMetaOf returns the room/user metadata registered for a client.
func (h *Handler) clientMetaOf(client *Client) (clientMeta, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	meta, ok := h.clientMeta[client]
	return meta, ok
}

// handleMuteToggle processes mute/unmute state changes.
func (h *Handler) handleMuteToggle(client *Client, msgType proto.MessageType) {
	h.mu.Lock()
	defer h.mu.Unlock()

	meta, ok := h.clientMeta[client]
	if !ok {
		return
	}

	muted := (msgType == proto.TypeMute)
	if r, err := h.manager.Get(meta.roomID); err == nil {
		r.SetMuted(meta.userID, muted)
	}

	peerMutedMsg := proto.PeerMutedMessage{
		Type:   proto.TypePeerMuted,
		UserID: meta.userID,
		Muted:  muted,
	}
	if clients, ok := h.roomClients[meta.roomID]; ok {
		for _, otherClient := range clients {
			_ = otherClient.Send(peerMutedMsg)
		}
	}
	slog.Info("mute state changed", "room", meta.roomID, "userId", meta.userID, "muted", muted)
}

// handleDisconnect cleans up when a client disconnects.
func (h *Handler) handleDisconnect(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	meta, ok := h.clientMeta[client]
	if !ok {
		return
	}
	delete(h.clientMeta, client)

	roomID, userID := meta.roomID, meta.userID
	if clients, ok := h.roomClients[roomID]; ok {
		delete(clients, userID)
		if len(clients) == 0 {
			delete(h.roomClients, roomID)
		}
	}

	if r, err := h.manager.Get(roomID); err == nil {
		r.RemovePeer(userID)
		if r.IsEmpty() {
			h.manager.Remove(roomID)
		}
	}

	// 广播 peerLeft 给其他人
	peerLeftMsg := proto.PeerLeftMessage{
		Type:   proto.TypePeerLeft,
		UserID: userID,
	}
	if clients, ok := h.roomClients[roomID]; ok {
		for _, otherClient := range clients {
			_ = otherClient.Send(peerLeftMsg)
		}
	}

	slog.Info("user left room", "room", roomID, "userId", userID)
}
