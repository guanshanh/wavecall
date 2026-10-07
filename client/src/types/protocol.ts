// Signaling protocol types — must stay in sync with:
//   - docs/protocol.md (source of truth)
//   - server/pkg/proto/messages.go (Go structs)

// --- Message types ---

export type MessageType =
  // Client → Server
  | "join"
  | "leave"
  | "offer"
  | "answer"
  | "candidate"
  | "mute"
  | "unmute"
  // Server → Client
  | "joined"
  | "peerJoined"
  | "peerLeft"
  | "peerMuted"
  | "speaking"
  | "error";

// --- Client → Server ---

export interface JoinMessage {
  type: "join";
  roomId: string;
  userName: string;
  password?: string;
}

export interface LeaveMessage {
  type: "leave";
}

export interface OfferMessage {
  type: "offer";
  targetId: string;
  sdp: string;
}

export interface AnswerMessage {
  type: "answer";
  targetId: string;
  sdp: string;
}

export interface CandidateMessage {
  type: "candidate";
  targetId: string;
  candidate: string;
}

export interface MuteMessage {
  type: "mute" | "unmute";
}

// --- Server → Client ---

export interface PeerInfo {
  userId: string;
  userName: string;
  muted: boolean;
}

export interface ICEServer {
  urls: string[];
  username?: string;
  credential?: string;
}

export interface JoinedMessage {
  type: "joined";
  userId: string;
  roomId: string;
  peers: PeerInfo[];
  iceServers: ICEServer[];
}

export interface PeerJoinedMessage {
  type: "peerJoined";
  userId: string;
  userName: string;
}

export interface PeerLeftMessage {
  type: "peerLeft";
  userId: string;
}

export interface PeerMutedMessage {
  type: "peerMuted";
  userId: string;
  muted: boolean;
}

export interface SpeakingMessage {
  type: "speaking";
  userId: string;
  speaking: boolean;
}

export interface ErrorMessage {
  type: "error";
  code: string;
  message: string;
}

// --- Union types ---

export type ClientMessage =
  | JoinMessage
  | LeaveMessage
  | OfferMessage
  | AnswerMessage
  | CandidateMessage
  | MuteMessage;

export type ServerMessage =
  | JoinedMessage
  | PeerJoinedMessage
  | PeerLeftMessage
  | PeerMutedMessage
  | SpeakingMessage
  | ErrorMessage;

export type SignalingMessage = ClientMessage | ServerMessage;
