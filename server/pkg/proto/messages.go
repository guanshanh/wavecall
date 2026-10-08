package proto

// MessageType identifies the kind of signaling message.
type MessageType string

const (
	// Client → Server
	TypeJoin      MessageType = "join"
	TypeLeave     MessageType = "leave"
	TypeAnswer    MessageType = "answer"
	TypeCandidate MessageType = "candidate"
	TypeMute      MessageType = "mute"
	TypeUnmute    MessageType = "unmute"

	// Server → Client
	TypeJoined     MessageType = "joined"
	TypePeerJoined MessageType = "peerJoined"
	TypePeerLeft   MessageType = "peerLeft"
	TypePeerMuted  MessageType = "peerMuted"
	TypeOffer      MessageType = "offer"
	TypeSpeaking   MessageType = "speaking"
	TypeError      MessageType = "error"
)

// Envelope wraps every message with a type discriminator.
type Envelope struct {
	Type MessageType `json:"type"`
}

// --- Client → Server ---

// JoinMessage is sent when a client wants to join a room.
type JoinMessage struct {
	Type     MessageType `json:"type"`
	RoomID   string      `json:"roomId"`
	UserName string      `json:"userName"`
	Password string      `json:"password,omitempty"`
	Token    string      `json:"token,omitempty"`
}

// LeaveMessage is sent when a client leaves the room.
type LeaveMessage struct {
	Type MessageType `json:"type"`
}

// OfferMessage carries an SDP offer. In SFU mode the server initiates all
// negotiation, so offers flow Server → Client only.
type OfferMessage struct {
	Type     MessageType `json:"type"`
	TargetID string      `json:"targetId"`
	SDP      string      `json:"sdp"`
}

// AnswerMessage carries an SDP answer from a client, replying to the
// server-initiated offer.
type AnswerMessage struct {
	Type     MessageType `json:"type"`
	TargetID string      `json:"targetId"`
	SDP      string      `json:"sdp"`
}

// CandidateMessage carries an ICE candidate (Trickle ICE).
// Browsers require sdpMid and/or sdpMLineIndex when calling addIceCandidate;
// SDPMLineIndex is a pointer so 0 is preserved in JSON.
type CandidateMessage struct {
	Type          MessageType `json:"type"`
	TargetID      string      `json:"targetId"`
	Candidate     string      `json:"candidate"`
	SDPMid        string      `json:"sdpMid,omitempty"`
	SDPMLineIndex *uint16     `json:"sdpMLineIndex,omitempty"`
}

// MuteMessage is sent when a client mutes/unmutes.
type MuteMessage struct {
	Type MessageType `json:"type"`
}

// --- Server → Client ---

// PeerInfo describes a peer already in the room.
type PeerInfo struct {
	UserID   string `json:"userId"`
	UserName string `json:"userName"`
	Muted    bool   `json:"muted"`
}

// JoinedMessage confirms the client has joined a room.
type JoinedMessage struct {
	Type       MessageType `json:"type"`
	UserID     string      `json:"userId"`
	RoomID     string      `json:"roomId"`
	Peers      []PeerInfo  `json:"peers"`
	ICEServers []ICEServer `json:"iceServers"`
}

// ICEServer is a copy for embedding in signaling messages.
type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// PeerJoinedMessage notifies that a new peer joined the room.
type PeerJoinedMessage struct {
	Type     MessageType `json:"type"`
	UserID   string      `json:"userId"`
	UserName string      `json:"userName"`
}

// PeerLeftMessage notifies that a peer left the room.
type PeerLeftMessage struct {
	Type   MessageType `json:"type"`
	UserID string      `json:"userId"`
}

// PeerMutedMessage notifies a peer's mute state change.
type PeerMutedMessage struct {
	Type   MessageType `json:"type"`
	UserID string      `json:"userId"`
	Muted  bool        `json:"muted"`
}

// SpeakingMessage notifies which peer is currently speaking.
type SpeakingMessage struct {
	Type     MessageType `json:"type"`
	UserID   string      `json:"userId"`
	Speaking bool        `json:"speaking"`
}

// ErrorMessage reports an error to the client.
type ErrorMessage struct {
	Type    MessageType `json:"type"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
}
