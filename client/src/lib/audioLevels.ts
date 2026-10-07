/**
 * AudioLevelMonitor computes real-time RMS levels for multiple audio streams
 * (local mic + remote streams) through a shared AudioContext.
 * Pure JS — no React dependency, used by Room via a ref.
 */
export class AudioLevelMonitor {
  private ctx: AudioContext | null = null;
  private sources = new Map<
    string,
    { source: MediaStreamAudioSourceNode; analyser: AnalyserNode; data: Float32Array<ArrayBuffer> }
  >();
  private timer: ReturnType<typeof setInterval> | null = null;
  private onLevels: ((levels: Map<string, number>) => void) | null = null;

  /** Begin polling levels every 100ms. Safe to call repeatedly. */
  start(onLevels: (levels: Map<string, number>) => void): void {
    this.onLevels = onLevels;
    if (this.timer) return;
    // Lazy-create: needs a user gesture (mic permission) to run unmuted
    this.ctx ??= new AudioContext();
    void this.ctx.resume();
    this.timer = setInterval(() => this.poll(), 100);
  }

  /** Attach a stream for level analysis (local mic or a remote stream). */
  addStream(id: string, stream: MediaStream): void {
    if (!this.ctx || this.sources.has(id)) return;
    const source = this.ctx.createMediaStreamSource(stream);
    const analyser = this.ctx.createAnalyser();
    analyser.fftSize = 512;
    source.connect(analyser);
    this.sources.set(id, {
      source,
      analyser,
      data: new Float32Array(analyser.fftSize),
    });
  }

  removeStream(id: string): void {
    const entry = this.sources.get(id);
    if (!entry) return;
    entry.source.disconnect();
    this.sources.delete(id);
  }

  /** Drop all streams (e.g. before rejoining), keeping the context alive. */
  clear(): void {
    for (const entry of this.sources.values()) {
      entry.source.disconnect();
    }
    this.sources.clear();
  }

  stop(): void {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
    this.clear();
    void this.ctx?.close();
    this.ctx = null;
    this.onLevels = null;
  }

  private poll(): void {
    if (!this.onLevels) return;
    const levels = new Map<string, number>();
    for (const [id, entry] of this.sources) {
      entry.analyser.getFloatTimeDomainData(entry.data);
      let sum = 0;
      for (const sample of entry.data) {
        sum += sample * sample;
      }
      const rms = Math.sqrt(sum / entry.data.length);
      // Speech RMS typically lands around 0.05–0.3; scale into 0..1
      levels.set(id, Math.min(rms * 4, 1));
    }
    this.onLevels(levels);
  }
}
