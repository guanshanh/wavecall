import { useEffect, useRef, useCallback } from "react";
import { SignalingClient, type ConnectionState } from "../lib/signaling";
import type { ClientMessage, ServerMessage } from "../types/protocol";

/**
 * useSignaling — manages WebSocket connection lifecycle within React.
 */
export function useSignaling(
  serverUrl: string,
  onMessage: (msg: ServerMessage) => void,
  onStateChange?: (state: ConnectionState) => void,
) {
  const clientRef = useRef<SignalingClient | null>(null);
  const onMessageRef = useRef(onMessage);
  const onStateChangeRef = useRef(onStateChange);

  onMessageRef.current = onMessage;
  onStateChangeRef.current = onStateChange;

  useEffect(() => {
    const client = new SignalingClient(
      serverUrl,
      (msg) => onMessageRef.current?.(msg),
      (state) => onStateChangeRef.current?.(state),
    );
    clientRef.current = client;
    client.connect();

    return () => {
      client.disconnect();
      clientRef.current = null;
    };
  }, [serverUrl]);

  const send = useCallback((message: ClientMessage) => {
    clientRef.current?.send(message);
  }, []);

  const disconnect = useCallback(() => {
    clientRef.current?.disconnect();
  }, []);

  return { send, disconnect };
}
