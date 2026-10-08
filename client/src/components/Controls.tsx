import type { VoiceMode } from "../lib/voiceMode";

export interface ControlsProps {
  isMuted: boolean;
  voiceMode: VoiceMode;
  pttKeyLabel: string;
  isCapturingKey: boolean;
  isPttHeld: boolean;
  isTauri: boolean;
  hotkeyError: string | null;
  onToggleMute: () => void;
  onLeave: () => void;
  onVoiceModeChange: (mode: VoiceMode) => void;
  onStartCaptureKey: () => void;
  onCancelCaptureKey: () => void;
}

export default function Controls({
  isMuted,
  voiceMode,
  pttKeyLabel,
  isCapturingKey,
  isPttHeld,
  isTauri,
  hotkeyError,
  onToggleMute,
  onLeave,
  onVoiceModeChange,
  onStartCaptureKey,
}: ControlsProps) {
  const talking = voiceMode === "ptt" && isPttHeld && !isMuted;

  let muteLabel: string;
  if (talking) {
    muteLabel = "🎙 说话中";
  } else if (isMuted) {
    muteLabel = "🔇 取消静音";
  } else {
    muteLabel = "🎤 静音";
  }

  let hintText = "";
  let hintWarn = false;
  if (hotkeyError) {
    hintText = hotkeyError;
    hintWarn = true;
  } else if (isCapturingKey) {
    hintText = "按下任意单键以绑定；Esc 取消";
  } else if (!isTauri) {
    hintText = "桌面版支持后台按键";
  } else if (voiceMode === "ptt") {
    hintText = "按住说话键开麦，松开静音";
  }

  const rebindDisabled = voiceMode !== "ptt" || isCapturingKey;

  return (
    <footer className="bg-gray-800 border-t border-gray-700 px-5 py-4 pb-5">
      <div className="max-w-[640px] mx-auto flex flex-col items-center gap-3">
        <div className="flex flex-wrap items-center justify-center gap-2.5">
          <div
            className="inline-flex bg-gray-900 rounded-[10px] p-[3px] gap-0.5"
            role="group"
            aria-label="语音模式"
          >
            <button
              type="button"
              onClick={() => onVoiceModeChange("free")}
              className={`border-none rounded-lg px-3.5 py-2 text-[13px] font-medium cursor-pointer transition-colors ${
                voiceMode === "free"
                  ? "bg-blue-600 text-white"
                  : "bg-transparent text-gray-400 hover:text-gray-300"
              }`}
            >
              自由聊天
            </button>
            <button
              type="button"
              onClick={() => onVoiceModeChange("ptt")}
              className={`border-none rounded-lg px-3.5 py-2 text-[13px] font-medium cursor-pointer transition-colors ${
                voiceMode === "ptt"
                  ? "bg-blue-600 text-white"
                  : "bg-transparent text-gray-400 hover:text-gray-300"
              }`}
            >
              按键说话
            </button>
          </div>

          <div
            className={`inline-flex items-center gap-2 bg-gray-900 rounded-[10px] py-1.5 pl-3 pr-2.5 text-[13px] text-gray-400 ${
              isCapturingKey ? "outline outline-1 outline-blue-600 text-blue-300" : ""
            }`}
          >
            <span>{isCapturingKey ? "请按一个键…" : "说话键："}</span>
            <kbd
              className="inline-block min-w-[1.6em] text-center px-2 py-0.5 rounded-md bg-gray-700 text-gray-100 font-mono text-[13px] border border-gray-600"
            >
              {isCapturingKey ? "?" : pttKeyLabel}
            </kbd>
            <button
              type="button"
              disabled={rebindDisabled}
              onClick={onStartCaptureKey}
              className="border-none bg-gray-700 text-gray-200 rounded-lg px-2.5 py-1.5 text-xs cursor-pointer hover:bg-gray-600 disabled:opacity-40 disabled:cursor-not-allowed"
            >
              改键
            </button>
          </div>
        </div>

        <div className="flex flex-wrap items-center justify-center gap-2.5">
          <button
            type="button"
            onClick={onToggleMute}
            className={`border-none rounded-[10px] px-[22px] py-3 text-sm font-medium text-white cursor-pointer transition-colors ${
              talking
                ? "bg-green-700 hover:bg-green-600"
                : isMuted
                  ? "bg-red-600 hover:bg-red-500"
                  : "bg-gray-600 hover:bg-gray-500"
            }`}
          >
            {muteLabel}
          </button>
          <button
            type="button"
            onClick={onLeave}
            className="border-none rounded-[10px] px-[22px] py-3 text-sm font-medium text-white bg-red-700 hover:bg-red-600 cursor-pointer transition-colors"
          >
            📞 挂断
          </button>
        </div>

        <p
          className={`text-[11px] text-center min-h-[1.2em] ${
            hintWarn ? "text-amber-400" : "text-gray-500"
          }`}
        >
          {hintText}
        </p>
      </div>
    </footer>
  );
}
