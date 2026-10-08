export type VoiceMode = "free" | "ptt";

export const VOICE_MODE_STORAGE_KEY = "wavecall.voiceMode";
export const PTT_KEY_STORAGE_KEY = "wavecall.pttKey";
export const DEFAULT_VOICE_MODE: VoiceMode = "free";
export const DEFAULT_PTT_KEY = "Backquote";

export type VoiceStorage = {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
};

function defaultStorage(): VoiceStorage | undefined {
  try {
    if (typeof localStorage === "undefined") return undefined;
    return localStorage;
  } catch {
    return undefined;
  }
}

export function loadVoiceMode(storage?: VoiceStorage): VoiceMode {
  const s = storage ?? defaultStorage();
  if (!s) return DEFAULT_VOICE_MODE;
  const raw = s.getItem(VOICE_MODE_STORAGE_KEY);
  return raw === "free" || raw === "ptt" ? raw : DEFAULT_VOICE_MODE;
}

export function saveVoiceMode(mode: VoiceMode, storage?: VoiceStorage): void {
  const s = storage ?? defaultStorage();
  if (!s) return;
  s.setItem(VOICE_MODE_STORAGE_KEY, mode);
}

export function loadPttKey(storage?: VoiceStorage): string {
  const s = storage ?? defaultStorage();
  if (!s) return DEFAULT_PTT_KEY;
  const raw = (s.getItem(PTT_KEY_STORAGE_KEY) ?? "").trim();
  return raw.length > 0 ? raw : DEFAULT_PTT_KEY;
}

export function savePttKey(code: string, storage?: VoiceStorage): void {
  const s = storage ?? defaultStorage();
  if (!s) return;
  const trimmed = code.trim();
  s.setItem(PTT_KEY_STORAGE_KEY, trimmed.length > 0 ? trimmed : DEFAULT_PTT_KEY);
}

export function formatPttKeyLabel(code: string): string {
  if (code === "Backquote") return "`";
  if (code === "Space") return "Space";
  if (code.startsWith("Key") && code.length === 4) return code.slice(3);
  if (code.startsWith("Digit") && code.length === 6) return code.slice(5);
  return code;
}

/** Single-key shortcut string for @tauri-apps/plugin-global-shortcut. */
export function codeToGlobalShortcut(code: string): string {
  if (code.startsWith("Key") && code.length === 4) return code.slice(3);
  if (code.startsWith("Digit") && code.length === 6) return code.slice(5);
  return code;
}
