import type { ICEServer } from "../types/protocol";

/**
 * WebRTCManager manages PeerConnection creation and audio track handling.
 * Decoupled from React — used by useWebRTC hook.
 */
export class WebRTCManager {
  private peerConnection: RTCPeerConnection | null = null;
  private localStream: MediaStream | null = null;
  private onRemoteTrack: ((userId: string, track: MediaStreamTrack) => void) | null = null;

  /** Initialize with ICE servers from the signaling server. */
  async init(iceServers: ICEServer[]): Promise<void> {
    this.peerConnection = new RTCPeerConnection({
      iceServers: iceServers.map((s) => ({
        urls: s.urls,
        username: s.username,
        credential: s.credential,
      })),
    });

    this.peerConnection.ontrack = (event) => {
      if (this.onRemoteTrack) {
        this.onRemoteTrack("remote", event.track);
      }
    };
  }

  /** Capture local audio from microphone. */
  async startLocalAudio(): Promise<MediaStream> {
    this.localStream = await navigator.mediaDevices.getUserMedia({
      audio: {
        echoCancellation: true,
        noiseSuppression: true,
        autoGainControl: true,
      },
      video: false,
    });

    // Add local tracks to the peer connection
    if (this.peerConnection) {
      this.localStream.getAudioTracks().forEach((track) => {
        this.peerConnection!.addTrack(track, this.localStream!);
      });
    }

    return this.localStream;
  }

  /** Create an SDP offer. */
  async createOffer(): Promise<string> {
    if (!this.peerConnection) throw new Error("PeerConnection not initialized");
    const offer = await this.peerConnection.createOffer();
    await this.peerConnection.setLocalDescription(offer);
    return offer.sdp ?? "";
  }

  /** Create an SDP answer in response to an offer. */
  async createAnswer(offerSdp: string): Promise<string> {
    if (!this.peerConnection) throw new Error("PeerConnection not initialized");
    await this.peerConnection.setRemoteDescription({
      type: "offer",
      sdp: offerSdp,
    });
    const answer = await this.peerConnection.createAnswer();
    await this.peerConnection.setLocalDescription(answer);
    return answer.sdp ?? "";
  }

  /** Set remote answer SDP. */
  async setRemoteAnswer(answerSdp: string): Promise<void> {
    if (!this.peerConnection) return;
    await this.peerConnection.setRemoteDescription({
      type: "answer",
      sdp: answerSdp,
    });
  }

  /** Add a remote ICE candidate. */
  async addIceCandidate(candidate: string): Promise<void> {
    if (!this.peerConnection) return;
    await this.peerConnection.addIceCandidate(
      new RTCIceCandidate({ candidate }),
    );
  }

  /** Mute or unmute local audio. */
  setMuted(muted: boolean): void {
    this.localStream?.getAudioTracks().forEach((track) => {
      track.enabled = !muted;
    });
  }

  /** Set handler for remote tracks. */
  setOnRemoteTrack(handler: (userId: string, track: MediaStreamTrack) => void): void {
    this.onRemoteTrack = handler;
  }

  /** Clean up all resources. */
  close(): void {
    this.localStream?.getTracks().forEach((t) => t.stop());
    this.localStream = null;
    this.peerConnection?.close();
    this.peerConnection = null;
    this.onRemoteTrack = null;
  }
}
