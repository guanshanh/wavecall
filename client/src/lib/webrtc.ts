import type { ICEServer } from "../types/protocol";

/**
 * WebRTCManager manages the single PeerConnection to the SFU server.
 *
 * Negotiation is server-initiated: the SFU sends offers (the initial offer
 * already includes every track published in the room, and each new sender
 * triggers a renegotiation offer). The manager only answers and trickles
 * ICE candidates.
 */
export interface WebRTCCallbacks {
  /** Called when the manager has an answer for a server offer. */
  onAnswer: (sdp: string) => void;
  /** Called for each local ICE candidate to trickle to the server. */
  onCandidate: (init: RTCIceCandidateInit) => void;
  /** Called when a forwarded track from a room peer arrives. */
  onRemoteTrack: (userId: string, track: MediaStreamTrack) => void;
}

export class WebRTCManager {
  private peerConnection: RTCPeerConnection | null = null;
  private localStream: MediaStream | null = null;
  private micPromise: Promise<MediaStream> | null = null;
  private localTrackAdded = false;
  private callbacks: WebRTCCallbacks | null = null;
  /** Desired mute when tracks do not exist yet (e.g. PTT join before getUserMedia). */
  private desiredMuted = false;
  /** Trickle ICE may arrive before the server offer is applied. */
  private pendingRemoteCandidates: RTCIceCandidateInit[] = [];

  /** Create the PeerConnection and register callbacks. */
  init(iceServers: ICEServer[], callbacks: WebRTCCallbacks): void {
    this.callbacks = callbacks;
    this.peerConnection = new RTCPeerConnection({
      iceServers: iceServers.map((s) => ({
        urls: s.urls,
        username: s.username,
        credential: s.credential,
      })),
    });

    this.peerConnection.onicecandidate = (event) => {
      if (event.candidate && this.callbacks) {
        // Pass the full init (sdpMid / sdpMLineIndex); browsers reject both-null.
        this.callbacks.onCandidate(event.candidate.toJSON());
      }
    };

    this.peerConnection.ontrack = (event) => {
      if (!this.callbacks) return;
      // The SFU names each forwarded MediaStream after the sender's userId.
      const senderId = event.streams[0]?.id ?? "unknown";
      this.callbacks.onRemoteTrack(senderId, event.track);
    };
  }

  /** Capture local audio from microphone. Safe to call multiple times. */
  async enableMic(): Promise<MediaStream> {
    this.micPromise ??= navigator.mediaDevices
      .getUserMedia({
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        },
        video: false,
      })
      .then((stream) => {
        this.localStream = stream;
        this.applyDesiredMuted();
        return stream;
      });
    return this.micPromise;
  }

  /**
   * Respond to a server-initiated offer: attach the mic track (waiting for
   * mic permission if needed) BEFORE setRemoteDescription so the track's
   * transceiver pairs with the SFU's recvonly m-line, then send the answer.
   */
  async handleServerOffer(sdp: string): Promise<void> {
    const pc = this.peerConnection;
    if (!pc) throw new Error("PeerConnection not initialized");

    if (!this.localTrackAdded) {
      try {
        await this.enableMic();
      } catch (err) {
        console.error("Microphone unavailable, joining without audio:", err);
      }
      if (this.localStream) {
        this.localStream.getAudioTracks().forEach((track) => {
          pc.addTrack(track, this.localStream!);
        });
        this.localTrackAdded = true;
      }
    }

    await pc.setRemoteDescription({ type: "offer", sdp });
    await this.flushPendingRemoteCandidates();

    const answer = await pc.createAnswer();
    await pc.setLocalDescription(answer);
    this.callbacks?.onAnswer(answer.sdp ?? "");
  }

  /**
   * Add a remote ICE candidate from the server. Candidates that arrive before
   * setRemoteDescription are queued (same pattern as the SFU peer).
   */
  async addIceCandidate(init: RTCIceCandidateInit): Promise<void> {
    const pc = this.peerConnection;
    if (!pc) return;
    if (!pc.remoteDescription) {
      this.pendingRemoteCandidates.push(init);
      return;
    }
    await pc.addIceCandidate(init);
  }

  private async flushPendingRemoteCandidates(): Promise<void> {
    const pc = this.peerConnection;
    if (!pc || this.pendingRemoteCandidates.length === 0) return;
    const pending = this.pendingRemoteCandidates;
    this.pendingRemoteCandidates = [];
    for (const init of pending) {
      try {
        await pc.addIceCandidate(init);
      } catch (err) {
        console.error("Failed to add buffered ICE candidate:", err);
      }
    }
  }

  /** Mute or unmute local audio. Remembers desired state if mic is not ready. */
  setMuted(muted: boolean): void {
    this.desiredMuted = muted;
    this.applyDesiredMuted();
  }

  private applyDesiredMuted(): void {
    this.localStream?.getAudioTracks().forEach((track) => {
      track.enabled = !this.desiredMuted;
    });
  }

  /** Clean up all resources. */
  close(): void {
    this.localStream?.getTracks().forEach((t) => t.stop());
    this.localStream = null;
    this.micPromise = null;
    this.localTrackAdded = false;
    this.desiredMuted = false;
    this.pendingRemoteCandidates = [];
    this.peerConnection?.close();
    this.peerConnection = null;
    this.callbacks = null;
  }
}
