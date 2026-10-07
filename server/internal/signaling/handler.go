package signaling

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/guanshanh/wavecall/internal/config"
	"github.com/guanshanh/wavecall/internal/room"
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

// NewHandler creates a new signaling handler.
func NewHandler(manager *room.Manager, cfg *config.Config) *Handler {
	return &Handler{
		manager:     manager,
		config:      cfg,
		roomClients: make(map[string]map[string]*Client),
		clientMeta:  make(map[*Client]clientMeta),
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
		var msg proto.OfferMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			return
		}
		h.handleOffer(client, &msg)

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

	slog.Info("user joined room", "room", msg.RoomID, "user", msg.UserName, "userId", userID, "totalPeers", r.PeerCount())
}

// handleLeave processes an explicit leave request.
func (h *Handler) handleLeave(client *Client) {
	h.handleDisconnect(client)
}

// handleOffer forwards an SDP offer for WebRTC negotiation.
func (h *Handler) handleOffer(client *Client, msg *proto.OfferMessage) {
	slog.Debug("offer received", "target", msg.TargetID)
}

// handleAnswer forwards an SDP answer for WebRTC negotiation.
func (h *Handler) handleAnswer(client *Client, msg *proto.AnswerMessage) {
	slog.Debug("answer received", "target", msg.TargetID)
}

// handleCandidate forwards an ICE candidate.
func (h *Handler) handleCandidate(client *Client, msg *proto.CandidateMessage) {
	slog.Debug("candidate received", "target", msg.TargetID)
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
