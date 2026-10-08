import { describe, expect, it } from "vitest";
import {
  DEFAULT_PTT_KEY,
  DEFAULT_VOICE_MODE,
  PTT_KEY_STORAGE_KEY,
  VOICE_MODE_STORAGE_KEY,
  codeToGlobalShortcut,
  formatPttKeyLabel,
  loadPttKey,
  loadVoiceMode,
  savePttKey,
  saveVoiceMode,
} from "./voiceMode";

function memoryStorage(initial: Record<string, string> = {}) {
  const store = { ...initial };
  return {
    store,
    storage: {
      getItem: (k: string) => store[k] ?? null,
      setItem: (k: string, v: string) => {
        store[k] = v;
      },
    },
  };
}

describe("loadVoiceMode / saveVoiceMode", () => {
  it("defaults to free when missing or invalid", () => {
    const { storage } = memoryStorage();
    expect(loadVoiceMode(storage)).toBe(DEFAULT_VOICE_MODE);
    expect(loadVoiceMode(memoryStorage({ [VOICE_MODE_STORAGE_KEY]: "nope" }).storage)).toBe(
      "free",
    );
  });

  it("round-trips ptt", () => {
    const { storage, store } = memoryStorage();
    saveVoiceMode("ptt", storage);
    expect(store[VOICE_MODE_STORAGE_KEY]).toBe("ptt");
    expect(loadVoiceMode(storage)).toBe("ptt");
  });
});

describe("loadPttKey / savePttKey", () => {
  it("defaults to Backquote when missing or empty", () => {
    expect(loadPttKey(memoryStorage().storage)).toBe(DEFAULT_PTT_KEY);
    expect(loadPttKey(memoryStorage({ [PTT_KEY_STORAGE_KEY]: "  " }).storage)).toBe(
      DEFAULT_PTT_KEY,
    );
  });

  it("round-trips KeyV", () => {
    const { storage, store } = memoryStorage();
    savePttKey("KeyV", storage);
    expect(store[PTT_KEY_STORAGE_KEY]).toBe("KeyV");
    expect(loadPttKey(storage)).toBe("KeyV");
  });
});

describe("formatPttKeyLabel", () => {
  it("maps common codes", () => {
    expect(formatPttKeyLabel("Backquote")).toBe("`");
    expect(formatPttKeyLabel("KeyV")).toBe("V");
    expect(formatPttKeyLabel("Space")).toBe("Space");
    expect(formatPttKeyLabel("F13")).toBe("F13");
  });
});

describe("codeToGlobalShortcut", () => {
  it("maps codes to tauri-plugin-global-shortcut strings", () => {
    expect(codeToGlobalShortcut("KeyV")).toBe("V");
    expect(codeToGlobalShortcut("Backquote")).toBe("Backquote");
    expect(codeToGlobalShortcut("Space")).toBe("Space");
    expect(codeToGlobalShortcut("Digit1")).toBe("1");
  });
});
