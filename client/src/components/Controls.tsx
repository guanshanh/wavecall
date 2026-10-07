interface ControlsProps {
  isMuted: boolean;
  onToggleMute: () => void;
  onLeave: () => void;
}

export default function Controls({ isMuted, onToggleMute, onLeave }: ControlsProps) {
  return (
    <footer className="bg-gray-800 px-6 py-4">
      <div className="flex items-center justify-center gap-4 max-w-md mx-auto">
        {/* Mute toggle */}
        <button
          onClick={onToggleMute}
          className={`px-6 py-3 rounded-lg font-medium transition-colors cursor-pointer ${
            isMuted
              ? "bg-red-600 hover:bg-red-500 text-white"
              : "bg-gray-600 hover:bg-gray-500 text-white"
          }`}
        >
          {isMuted ? "🔇 取消静音" : "🎤 静音"}
        </button>

        {/* Leave */}
        <button
          onClick={onLeave}
          className="px-6 py-3 bg-red-700 hover:bg-red-600 text-white rounded-lg font-medium transition-colors cursor-pointer"
        >
          📞 挂断
        </button>
      </div>
    </footer>
  );
}
