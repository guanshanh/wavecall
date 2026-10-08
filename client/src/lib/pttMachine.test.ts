import { describe, expect, it } from "vitest";
import { initialPttState, reducePtt } from "./pttMachine";

describe("pttMachine", () => {
  it("free: mute button toggles", () => {
    let s = initialPttState("free");
    expect(s.muted).toBe(false);
    s = reducePtt(s, { type: "muteButton" });
    expect(s.muted).toBe(true);
    s = reducePtt(s, { type: "muteButton" });
    expect(s.muted).toBe(false);
  });

  it("switch to ptt mutes; hold talks; release mutes", () => {
    let s = initialPttState("free");
    s = reducePtt(s, { type: "setMode", mode: "ptt" });
    expect(s.muted).toBe(true);
    s = reducePtt(s, { type: "pttDown" });
    expect(s.muted).toBe(false);
    expect(s.isHeld).toBe(true);
    s = reducePtt(s, { type: "pttUp" });
    expect(s.muted).toBe(true);
    expect(s.isHeld).toBe(false);
  });

  it("force mute then needs full press-release before next talk", () => {
    let s = initialPttState("ptt");
    s = reducePtt(s, { type: "pttDown" });
    s = reducePtt(s, { type: "muteButton" });
    expect(s.muted).toBe(true);
    expect(s.forceMuted).toBe(true);
    s = reducePtt(s, { type: "pttUp" });
    expect(s.muted).toBe(true);
    expect(s.forceMuted).toBe(true);
    s = reducePtt(s, { type: "pttDown" });
    expect(s.muted).toBe(true);
    s = reducePtt(s, { type: "pttUp" });
    expect(s.forceMuted).toBe(false);
    s = reducePtt(s, { type: "pttDown" });
    expect(s.muted).toBe(false);
  });

  it("ignores ptt while capturing key", () => {
    let s = initialPttState("ptt");
    s = reducePtt(s, { type: "startCapture" });
    s = reducePtt(s, { type: "pttDown" });
    expect(s.isHeld).toBe(false);
    expect(s.muted).toBe(true);
  });

  it("startCapture drops an active hold", () => {
    let s = initialPttState("ptt");
    s = reducePtt(s, { type: "pttDown" });
    expect(s.muted).toBe(false);
    s = reducePtt(s, { type: "startCapture" });
    expect(s.isHeld).toBe(false);
    expect(s.muted).toBe(true);
    expect(s.isCapturingKey).toBe(true);
  });

  it("ptt mute button is no-op when already idle muted", () => {
    let s = initialPttState("ptt");
    s = reducePtt(s, { type: "muteButton" });
    expect(s.forceMuted).toBe(false);
    expect(s.muted).toBe(true);
  });

  it("switch to free unmutes", () => {
    let s = initialPttState("ptt");
    s = reducePtt(s, { type: "setMode", mode: "free" });
    expect(s.mode).toBe("free");
    expect(s.muted).toBe(false);
  });
});
