import { create } from "zustand";
import {
  loadPttKey,
  loadVoiceMode,
  savePttKey,
  saveVoiceMode,
} from "../lib/voiceMode";
import {
  initialPttState,
  reducePtt,
  type PttEvent,
  type PttMachineState,
} from "../lib/pttMachine";

type VoiceState = PttMachineState & {
  pttKey: string;
  hotkeyError: string | null;
  dispatch: (event: PttEvent) => void;
  setPttKey: (code: string) => void;
  setHotkeyError: (msg: string | null) => void;
};

export const useVoiceStore = create<VoiceState>((set, get) => ({
  ...initialPttState(loadVoiceMode()),
  pttKey: loadPttKey(),
  hotkeyError: null,

  dispatch: (event) => {
    const prev = get();
    const next = reducePtt(
      {
        mode: prev.mode,
        muted: prev.muted,
        forceMuted: prev.forceMuted,
        isHeld: prev.isHeld,
        clearArmed: prev.clearArmed,
        isCapturingKey: prev.isCapturingKey,
      },
      event,
    );
    if (next.mode !== prev.mode) saveVoiceMode(next.mode);
    set(next);
  },

  setPttKey: (code) => {
    savePttKey(code);
    set({ pttKey: code, isCapturingKey: false, hotkeyError: null });
  },

  setHotkeyError: (msg) => set({ hotkeyError: msg }),
}));
