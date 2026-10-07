import { useEffect, useRef, useCallback } from "react";
import { WebRTCManager } from "../lib/webrtc";
import type { ICEServer } from "../types/protocol";

/**
 * useWebRTC — manages WebRTC PeerConnection lifecycle within React.
 */
export function useWebRTC() {
  const managerRef = useRef<WebRTCManager | null>(null);

  useEffect(() => {
    managerRef.current = new WebRTCManager();
    return () => {
      managerRef.current?.close();
      managerRef.current = null;
    };
  }, []);

  const init = useCallback(async (iceServers: ICEServer[]) => {
    await managerRef.current?.init(iceServers);
  }, []);

  const startAudio = useCallback(async () => {
    return managerRef.current?.startLocalAudio();
  }, []);

  const createOffer = useCallback(async () => {
    return managerRef.current?.createOffer();
  }, []);

  const createAnswer = useCallback(async (offerSdp: string) => {
    return managerRef.current?.createAnswer(offerSdp);
  }, []);

  const setRemoteAnswer = useCallback(async (answerSdp: string) => {
    await managerRef.current?.setRemoteAnswer(answerSdp);
  }, []);

  const addIceCandidate = useCallback(async (candidate: string) => {
    await managerRef.current?.addIceCandidate(candidate);
  }, []);

  const setMuted = useCallback((muted: boolean) => {
    managerRef.current?.setMuted(muted);
  }, []);

  return {
    init,
    startAudio,
    createOffer,
    createAnswer,
    setRemoteAnswer,
    addIceCandidate,
    setMuted,
  };
}
