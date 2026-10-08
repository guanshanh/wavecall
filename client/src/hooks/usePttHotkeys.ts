import { useEffect } from "react";
import { codeToGlobalShortcut } from "../lib/voiceMode";
import { isTauri } from "../lib/tauriEnv";
import { useVoiceStore } from "../stores/voiceStore";

function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  const tag = target.tagName;
  if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return true;
  return target.isContentEditable;
}

export function usePttHotkeys(): void {
  const pttKey = useVoiceStore((s) => s.pttKey);
  const voiceMode = useVoiceStore((s) => s.mode);

  useEffect(() => {
    const onKeyDownCapture = (e: KeyboardEvent) => {
      const { isCapturingKey, dispatch, setPttKey } = useVoiceStore.getState();
      if (!isCapturingKey) return;
      if (e.code === "Escape") {
        dispatch({ type: "cancelCapture" });
        e.preventDefault();
        return;
      }
      if (!e.repeat) {
        setPttKey(e.code);
        e.preventDefault();
      }
    };
    window.addEventListener("keydown", onKeyDownCapture, true);
    return () => window.removeEventListener("keydown", onKeyDownCapture, true);
  }, []);

  const hotkeyError = useVoiceStore((s) => s.hotkeyError);

  // Pure Web: always listen. Tauri: only when global register failed (avoid double fire while focused).
  useEffect(() => {
    if (isTauri() && !hotkeyError) return;

    const onKeyDown = (e: KeyboardEvent) => {
      const { isCapturingKey, pttKey: key, dispatch } = useVoiceStore.getState();
      if (isCapturingKey) return;
      if (e.code !== key) return;
      if (isEditableTarget(e.target)) return;
      if (e.repeat) return;
      dispatch({ type: "pttDown" });
    };
    const onKeyUp = (e: KeyboardEvent) => {
      const { isCapturingKey, pttKey: key, dispatch } = useVoiceStore.getState();
      if (isCapturingKey) return;
      if (e.code !== key) return;
      if (isEditableTarget(e.target)) return;
      dispatch({ type: "pttUp" });
    };
    window.addEventListener("keydown", onKeyDown);
    window.addEventListener("keyup", onKeyUp);
    return () => {
      window.removeEventListener("keydown", onKeyDown);
      window.removeEventListener("keyup", onKeyUp);
    };
  }, [hotkeyError]);

  useEffect(() => {
    if (!isTauri() || voiceMode !== "ptt") {
      useVoiceStore.getState().setHotkeyError(null);
      return;
    }

    const shortcut = codeToGlobalShortcut(pttKey);
    let active = true;
    let generation = 0;

    void (async () => {
      const gen = ++generation;
      const { register, unregister } = await import(
        "@tauri-apps/plugin-global-shortcut"
      );
      try {
        await unregister(shortcut);
      } catch {
        // not registered yet
      }
      if (!active || gen !== generation) return;

      const handler = (event: { state: string }) => {
        const { isCapturingKey, dispatch } = useVoiceStore.getState();
        if (isCapturingKey) return;
        if (event.state === "Pressed") {
          dispatch({ type: "pttDown" });
        } else if (event.state === "Released") {
          dispatch({ type: "pttUp" });
        }
      };

      try {
        await register(shortcut, handler);
        if (!active || gen !== generation) {
          await unregister(shortcut);
          return;
        }
        useVoiceStore.getState().setHotkeyError(null);
      } catch {
        if (active && gen === generation) {
          useVoiceStore.getState().setHotkeyError("快捷键注册失败，请换一个键");
        }
      }
    })();

    return () => {
      active = false;
      generation += 1;
      void (async () => {
        const { unregister } = await import("@tauri-apps/plugin-global-shortcut");
        try {
          await unregister(shortcut);
        } catch {
          // ignore cleanup errors
        }
      })();
    };
  }, [pttKey, voiceMode]);
}
