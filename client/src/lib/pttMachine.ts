import type { VoiceMode } from "./voiceMode";

export type PttMachineState = {
  mode: VoiceMode;
  muted: boolean;
  forceMuted: boolean;
  isHeld: boolean;
  clearArmed: boolean;
  isCapturingKey: boolean;
};

export type PttEvent =
  | { type: "setMode"; mode: VoiceMode }
  | { type: "pttDown" }
  | { type: "pttUp" }
  | { type: "muteButton" }
  | { type: "startCapture" }
  | { type: "cancelCapture" }
  | { type: "bindKey" };

export function initialPttState(mode: VoiceMode): PttMachineState {
  return {
    mode,
    muted: mode === "ptt",
    forceMuted: false,
    isHeld: false,
    clearArmed: false,
    isCapturingKey: false,
  };
}

export function reducePtt(state: PttMachineState, event: PttEvent): PttMachineState {
  switch (event.type) {
    case "setMode":
      if (event.mode === "ptt") {
        return {
          ...state,
          mode: "ptt",
          muted: true,
          forceMuted: false,
          isHeld: false,
          clearArmed: false,
          isCapturingKey: false,
        };
      }
      return {
        ...state,
        mode: "free",
        muted: false,
        forceMuted: false,
        isHeld: false,
        clearArmed: false,
        isCapturingKey: false,
      };
    case "startCapture":
      // Drop any active hold so release during capture cannot leave the mic open.
      return {
        ...state,
        isCapturingKey: true,
        isHeld: false,
        clearArmed: false,
        muted: state.mode === "ptt" ? true : state.muted,
      };
    case "cancelCapture":
    case "bindKey":
      return { ...state, isCapturingKey: false };
    case "muteButton": {
      if (state.isCapturingKey) return state;
      if (state.mode === "free") {
        return { ...state, muted: !state.muted };
      }
      // PTT: force-mute only while talking / held; idle muted is a no-op.
      if (state.muted && !state.isHeld) {
        return state;
      }
      return {
        ...state,
        muted: true,
        forceMuted: true,
        clearArmed: false,
        isHeld: false,
      };
    }
    case "pttDown": {
      if (state.isCapturingKey || state.mode !== "ptt") return state;
      if (state.forceMuted) {
        return { ...state, isHeld: true, clearArmed: true };
      }
      return { ...state, isHeld: true, muted: false };
    }
    case "pttUp": {
      if (state.mode !== "ptt" || state.isCapturingKey) return state;
      const next: PttMachineState = {
        ...state,
        isHeld: false,
        muted: true,
      };
      if (state.forceMuted && state.clearArmed) {
        next.forceMuted = false;
        next.clearArmed = false;
      }
      return next;
    }
  }
}
