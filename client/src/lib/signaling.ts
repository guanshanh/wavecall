import type { ClientMessage, ServerMessage } from "../types/protocol";

export type ConnectionState = "connecting" | "connected" | "disconnected" | "error";
export type MessageHandler = (message: ServerMessage) => void;

/**
 * SignalingClient manages the WebSocket connection to the signaling server.
 * Handles message serialization, queued sending before connect, and state notifications.
 */
export class SignalingClient {
  private ws: WebSocket | null = null;
  private url: string;
  private onMessage: MessageHandler;
  private onStateChange: (state: ConnectionState) => void;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private _state: ConnectionState = "disconnected";
  private pendingQueue: ClientMessage[] = [];

  constructor(
    url: string,
    onMessage: MessageHandler,
    onStateChange: (state: ConnectionState) => void,
  ) {
    this.url = url;
    this.onMessage = onMessage;
    this.onStateChange = onStateChange;
  }

  /** Connect to the signaling server. */
  connect(): void {
    if (this.ws) return;

    this.setState("connecting");
    try {
      this.ws = new WebSocket(this.url);
    } catch {
      this.setState("error");
      return;
    }

    this.ws.onopen = () => {
      this.setState("connected");
      // Flush pending messages
      while (this.pendingQueue.length > 0) {
        const msg = this.pendingQueue.shift();
        if (msg) {
          this.ws?.send(JSON.stringify(msg));
        }
      }
    };

    this.ws.onclose = () => {
      this.ws = null;
      this.setState("disconnected");
    };

    this.ws.onerror = () => {
      this.setState("error");
    };

    this.ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data as string) as ServerMessage;
        this.onMessage(message);
      } catch (err) {
        console.error("Failed to parse signaling message:", err);
      }
    };
  }

  /** Send a message to the server (queues if still connecting). */
  send(message: ClientMessage): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      this.pendingQueue.push(message);
    }
  }

  /** Disconnect from the server. */
  disconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.pendingQueue = [];
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.setState("disconnected");
  }

  get state(): ConnectionState {
    return this._state;
  }

  private setState(state: ConnectionState): void {
    this._state = state;
    this.onStateChange(state);
  }
}
