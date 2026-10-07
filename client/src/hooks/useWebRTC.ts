import { useRef, useCallback, useEffect } from "react";
import { WebRTCManager, type WebRTCCallbacks } from "../lib/webrtc";
import type { ICEServer } from "../types/protocol";

/**
 * useWebRTC — manages the SFU PeerConnection lifecycle within React.
 * Callbacks are kept in a ref so the manager always invokes the latest ones.
 */
export function useWebRTC() {
  const managerRef = useRef<WebRTCManager | null>(null);
  const callbacksRef = useRef<WebRTCCallbacks | null>(null);

  useEffect(() => {
    return () => {
      managerRef.current?.close();
      managerRef.current = null;
    };
  }, []);

  const init = useCallback((iceServers: ICEServer[]) => {
    if (managerRef.current) return;
    const manager = new WebRTCManager();
    manager.init(iceServers, {
      onAnswer: (sdp) => callbacksRef.current?.onAnswer(sdp),
      onCandidate: (init) => callbacksRef.current?.onCandidate(init),
      onRemoteTrack: (userId, track) =>
        callbacksRef.current?.onRemoteTrack(userId, track),
    });
    managerRef.current = manager;
  }, []);

  const enableMic = useCallback(async () => {
    return await managerRef.current?.enableMic();
  }, []);

  const handleServerOffer = useCallback(async (sdp: string) => {
    await managerRef.current?.handleServerOffer(sdp);
  }, []);

  const addIceCandidate = useCallback(async (init: RTCIceCandidateInit) => {
    await managerRef.current?.addIceCandidate(init);
  }, []);

  const setMuted = useCallback((muted: boolean) => {
    managerRef.current?.setMuted(muted);
  }, []);

  /** Register the latest signaling callbacks (called on every render). */
  const setCallbacks = useCallback((callbacks: WebRTCCallbacks) => {
    callbacksRef.current = callbacks;
  }, []);

  /** Tear down the current manager so the next init() creates a fresh one. */
  const reset = useCallback(() => {
    managerRef.current?.close();
    managerRef.current = null;
  }, []);

  return { init, enableMic, handleServerOffer, addIceCandidate, setMuted, setCallbacks, reset };
}
