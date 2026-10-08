// Package e2e contains end-to-end tests that drive the real signaling
// server with two pion-based fake browsers and verify SFU audio forwarding.
package e2e

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"

	"github.com/guanshanh/wavecall/internal/auth"
	"github.com/guanshanh/wavecall/internal/config"
	"github.com/guanshanh/wavecall/internal/room"
	"github.com/guanshanh/wavecall/internal/signaling"
	"github.com/guanshanh/wavecall/pkg/proto"
)

// payloadMagic marks packets we generate so the receiver can verify
// the forwarded RTP actually carries the sender's bytes.
var payloadMagic = uint64(0x5741564543414C4C) // "WAVECALL"

// fakeBrowser is a pion PeerConnection driven through the real WebSocket
// signaling protocol, standing in for a browser client.
type fakeBrowser struct {
	t    *testing.T
	name string
	token string

	conn   *websocket.Conn
	userID string

	pc              *webrtc.PeerConnection
	remoteSet       bool
	pendingCands    []webrtc.ICECandidateInit
	offerCh         chan string
	joinedCh        chan proto.JoinedMessage
	remoteTrackCh   chan *webrtc.TrackRemote
	signalingClosed chan struct{}
}

func newFakeBrowser(t *testing.T, name string, serverURL string) *fakeBrowser {
	t.Helper()

	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		t.Fatalf("%s: dial signaling server: %v", name, err)
	}

	b := &fakeBrowser{
		t:               t,
		name:            name,
		conn:            conn,
		offerCh:         make(chan string, 4),
		joinedCh:        make(chan proto.JoinedMessage, 1),
		remoteTrackCh:   make(chan *webrtc.TrackRemote, 4),
		signalingClosed: make(chan struct{}),
	}
	go b.readLoop()
	return b
}

func (b *fakeBrowser) readLoop() {
	defer close(b.signalingClosed)
	for {
		_, raw, err := b.conn.ReadMessage()
		if err != nil {
			return
		}

		var env proto.Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			b.t.Logf("%s: ignoring unparsable message: %s", b.name, raw)
			continue
		}

		switch env.Type {
		case proto.TypeJoined:
			var msg proto.JoinedMessage
			mustUnmarshal(b.t, raw, &msg)
			b.userID = msg.UserID
			b.joinedCh <- msg

		case proto.TypeOffer:
			var msg proto.OfferMessage
			mustUnmarshal(b.t, raw, &msg)
			b.offerCh <- msg.SDP

		case proto.TypeCandidate:
			var msg proto.CandidateMessage
			mustUnmarshal(b.t, raw, &msg)
			init := candidateInitFromMsg(msg)
			if b.remoteSet {
				b.addCandidate(init)
			} else {
				b.pendingCands = append(b.pendingCands, init)
			}
		}
	}
}

func (b *fakeBrowser) join(roomID string) proto.JoinedMessage {
	b.t.Helper()
	mustSend(b.t, b.conn, proto.JoinMessage{
		Type:     proto.TypeJoin,
		RoomID:   roomID,
		UserName: b.name,
		Token:    b.token,
	})
	select {
	case msg := <-b.joinedCh:
		return msg
	case <-time.After(5 * time.Second):
		b.t.Fatalf("%s: timed out waiting for joined", b.name)
		return proto.JoinedMessage{}
	}
}

// answerOffer applies the server's initial offer, publishes an Opus track
// that streams marked RTP packets, and sends the answer back.
func (b *fakeBrowser) answerOffer(sdp string) *webrtc.TrackLocalStaticRTP {
	b.t.Helper()

	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		b.t.Fatalf("%s: new peer connection: %v", b.name, err)
	}
	b.pc = pc

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		init := c.ToJSON()
		msg := proto.CandidateMessage{
			Type:      proto.TypeCandidate,
			TargetID:  b.userID,
			Candidate: init.Candidate,
		}
		if init.SDPMid != nil {
			msg.SDPMid = *init.SDPMid
		}
		msg.SDPMLineIndex = init.SDPMLineIndex
		mustSend(b.t, b.conn, msg)
	})

	pc.OnTrack(func(remote *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		b.remoteTrackCh <- remote
	})

	// 加 track 必须在 setRemoteDescription 之前：先建的 sendrecv transceiver
	// 会与 offer 中第一个 m-line（服务端留给麦克风的 recvonly）正确关联，
	// 反过来加则会落到无法进入 answer 的悬空 transceiver 上。
	track, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		"audio", b.name,
	)
	if err != nil {
		b.t.Fatalf("%s: create track: %v", b.name, err)
	}
	if _, err := pc.AddTrack(track); err != nil {
		b.t.Fatalf("%s: add track: %v", b.name, err)
	}

	if err := pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	}); err != nil {
		b.t.Fatalf("%s: set remote offer: %v", b.name, err)
	}
	b.setRemote()

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		b.t.Fatalf("%s: create answer: %v", b.name, err)
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		b.t.Fatalf("%s: set local answer: %v", b.name, err)
	}
	mustSend(b.t, b.conn, proto.AnswerMessage{
		Type:     proto.TypeAnswer,
		TargetID: b.userID,
		SDP:      answer.SDP,
	})

	go b.streamPackets(track)
	return track
}

// answerRenegotiation answers a follow-up server offer on the existing
// PeerConnection (no new track, no new transport).
func (b *fakeBrowser) answerRenegotiation(sdp string) {
	b.t.Helper()

	if b.pc == nil {
		b.t.Fatalf("%s: renegotiation before initial offer", b.name)
	}
	if err := b.pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	}); err != nil {
		b.t.Fatalf("%s: set remote renegotiation offer: %v", b.name, err)
	}
	b.setRemote()

	answer, err := b.pc.CreateAnswer(nil)
	if err != nil {
		b.t.Fatalf("%s: create renegotiation answer: %v", b.name, err)
	}
	if err := b.pc.SetLocalDescription(answer); err != nil {
		b.t.Fatalf("%s: set local renegotiation answer: %v", b.name, err)
	}
	mustSend(b.t, b.conn, proto.AnswerMessage{
		Type:     proto.TypeAnswer,
		TargetID: b.userID,
		SDP:      answer.SDP,
	})
}

func (b *fakeBrowser) setRemote() {
	b.remoteSet = true
	for _, c := range b.pendingCands {
		b.addCandidate(c)
	}
	b.pendingCands = nil
}

func candidateInitFromMsg(msg proto.CandidateMessage) webrtc.ICECandidateInit {
	init := webrtc.ICECandidateInit{Candidate: msg.Candidate}
	if msg.SDPMid != "" {
		mid := msg.SDPMid
		init.SDPMid = &mid
	}
	init.SDPMLineIndex = msg.SDPMLineIndex
	return init
}

func (b *fakeBrowser) addCandidate(init webrtc.ICECandidateInit) {
	if err := b.pc.AddICECandidate(init); err != nil {
		b.t.Logf("%s: add ICE candidate: %v", b.name, err)
	}
}

func (b *fakeBrowser) waitAnswerApplied() {
	b.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if b.pc.ConnectionState() == webrtc.PeerConnectionStateConnected {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	b.t.Fatalf("%s: peer connection never reached connected (state=%s)", b.name, b.pc.ConnectionState())
}

// streamPackets writes marked RTP packets at 50 pps, as if from a live mic.
func (b *fakeBrowser) streamPackets(track *webrtc.TrackLocalStaticRTP) {
	seq := uint16(0)
	ts := uint32(0)
	payload := make([]byte, 64)
	binary.BigEndian.PutUint64(payload, payloadMagic)

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		seq++
		ts += 960
		pkt := rtp.Packet{
			Header: rtp.Header{
				Version:        2,
				PayloadType:    111,
				SequenceNumber: seq,
				Timestamp:      ts,
				SSRC:           uint32(len(b.name))<<16 | uint32(seq),
			},
			Payload: payload,
		}
		if err := track.WriteRTP(&pkt); err != nil {
			b.t.Logf("%s: write RTP failed: %v", b.name, err)
			return
		}
	}
}

// awaitRemoteTrack verifies the browser receives a forwarded track from the
// expected sender and that its payload survived the round trip.
func (b *fakeBrowser) awaitRemoteTrack(senderID string) {
	b.t.Helper()
	select {
	case remote := <-b.remoteTrackCh:
		if remote.StreamID() != senderID {
			b.t.Fatalf("%s: got track from %q, want %q", b.name, remote.StreamID(), senderID)
		}
		pkt, _, err := remote.ReadRTP()
		if err != nil {
			b.t.Fatalf("%s: read forwarded RTP: %v", b.name, err)
		}
		if binary.BigEndian.Uint64(pkt.Payload) != payloadMagic {
			b.t.Fatalf("%s: forwarded payload corrupted: %x", b.name, pkt.Payload)
		}
	case <-time.After(15 * time.Second):
		b.t.Fatalf("%s: timed out waiting for track from %s", b.name, senderID)
	}
}

func mustUnmarshal(t *testing.T, raw []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
}

func mustSend(t *testing.T, conn *websocket.Conn, msg any) {
	t.Helper()
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("send %T: %v", msg, err)
	}
}

// TestSFUAudioForwarding 无 STUN、无公网 IP 的本机回环场景：
// 验证默认部署形态（host 候选直连）。
func TestSFUAudioForwarding(t *testing.T) {
	runForwardingScenario(t, &config.Config{Port: 0})
}

// TestSFUAudioForwardingSinglePort 单端口复用（UDPMux）：
// 验证 -udp-port 部署形态下所有媒体走同一 UDP 端口。
func TestSFUAudioForwardingSinglePort(t *testing.T) {
	// 挑一个空闲 UDP 端口给 mux 用
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("find free udp port: %v", err)
	}
	udpPort := conn.LocalAddr().(*net.UDPAddr).Port
	if err := conn.Close(); err != nil {
		t.Fatalf("close probe socket: %v", err)
	}

	runForwardingScenario(t, &config.Config{Port: 0, UDPPort: udpPort})
}

func runForwardingScenario(t *testing.T, cfg *config.Config) {
	users, err := auth.NewDirectory("e2e-secret", []auth.User{{Account: "e2e", Password: "e2e-pass"}})
	if err != nil {
		t.Fatal(err)
	}
	token, err := users.Issue("e2e", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	manager := room.NewManager()
	handler, err := signaling.NewHandler(manager, cfg, users)
	if err != nil {
		t.Fatalf("new signaling handler: %v", err)
	}
	defer handler.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.HandleWebSocket)
	server := httptest.NewServer(mux)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	const roomID = "e2e-room"

	// A 加入并发布音频
	a := newFakeBrowser(t, "alice", wsURL)
	defer a.conn.Close()
	a.token = token
	a.join(roomID)
	select {
	case sdp := <-a.offerCh:
		a.answerOffer(sdp)
	case <-time.After(5 * time.Second):
		t.Fatal("alice: timed out waiting for initial offer")
	}
	a.waitAnswerApplied()

	// B 加入：初始 offer 应已包含 A 的音频轨（无需重协商）
	bob := newFakeBrowser(t, "bob", wsURL)
	defer bob.conn.Close()
	bob.token = token
	bob.join(roomID)
	select {
	case sdp := <-bob.offerCh:
		bob.answerOffer(sdp)
	case <-time.After(5 * time.Second):
		t.Fatal("bob: timed out waiting for initial offer")
	}
	bob.waitAnswerApplied()
	bob.awaitRemoteTrack(a.userID)

	// A 也应能通过重协商收到 B 的音频轨（B 发布后触发 offer → answer）
	select {
	case sdp := <-a.offerCh:
		a.answerRenegotiation(sdp)
	case <-time.After(15 * time.Second):
		t.Fatal("alice: timed out waiting for renegotiation offer")
	}
	a.waitAnswerApplied()
	a.awaitRemoteTrack(bob.userID)
}
