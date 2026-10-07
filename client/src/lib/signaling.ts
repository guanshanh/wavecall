import { isDispatchConfigError } from "./dispatch";
import type { ClientMessage, ServerMessage } from "../types/protocol";

export type ConnectionState =
  | "connecting"
  | "connected"
  | "reconnecting"
  | "disconnected"
  | "error";
export type MessageHandler = (message: ServerMessage) => void;

const MAX_RETRY_DELAY_MS = 15_000;

/**
 * SignalingClient manages the WebSocket connection to the signaling server.
 * Handles message serialization, queued sending before connect, and state
 * notifications. Unexpected disconnects trigger automatic reconnection with
 * exponential backoff; only an explicit disconnect() suppresses it.
 */
export class SignalingClient {
  private ws: WebSocket | null = null;
  private getUrl: () => Promise<string>;
  private onMessage: MessageHandler;
  private onStateChange: (state: ConnectionState) => void;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private retryCount = 0;
  private manualClose = false;
  private _state: ConnectionState = "disconnected";
  private pendingQueue: ClientMessage[] = [];

  constructor(
    getUrl: () => Promise<string>,
    onMessage: MessageHandler,
    onStateChange: (state: ConnectionState) => void,
  ) {
    this.getUrl = getUrl;
    this.onMessage = onMessage;
    this.onStateChange = onStateChange;
  }

  /** Connect to the signaling server. Also used for each reconnect attempt. */
  async connect(): Promise<void> {
    if (this.ws) return;
    this.manualClose = false;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    this.setState(this.retryCount === 0 ? "connecting" : "reconnecting");

    let url: string;
    try {
      url = await this.getUrl();
    } catch (err) {
      console.error("dispatch/url resolve failed:", err);
      if (isDispatchConfigError(err)) {
        this.setState("error");
        return;
      }
      if (!this.manualClose) {
        this.scheduleReconnect();
      }
      return;
    }

    if (this.manualClose || this.ws) return;

    try {
      this.ws = new WebSocket(url);
    } catch {
      this.scheduleReconnect();
      return;
    }

    this.ws.onopen = () => {
      this.retryCount = 0;
      this.setState("connected");
      // Flush pending messages
      while (this.pendingQueue.length > 0) {
        const msg = this.pendingQueue.shift();
        if (msg) {
          this.ws?.send(JSON.stringify(msg));
        }
      }
    };

    // onclose always follows onerror, so it alone drives the state machine
    this.ws.onclose = () => {
      this.ws = null;
      if (this.manualClose) {
        this.setState("disconnected");
      } else {
        this.scheduleReconnect();
      }
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

  /** Disconnect intentionally — no reconnection will be attempted. */
  disconnect(): void {
    this.manualClose = true;
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

  /** Exponential backoff (1s, 2s, 4s … capped at 15s) with jitter. */
  private scheduleReconnect(): void {
    if (this.manualClose) return;
    this.retryCount += 1;
    this.setState("reconnecting");

    const backoff = Math.min(1000 * 2 ** (this.retryCount - 1), MAX_RETRY_DELAY_MS);
    const delay = backoff + Math.random() * 500; // jitter avoids reconnect storms
    this.reconnectTimer = setTimeout(() => void this.connect(), delay);
  }

  private setState(state: ConnectionState): void {
    this._state = state;
    this.onStateChange(state);
  }
}
