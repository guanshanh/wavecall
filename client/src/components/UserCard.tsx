import VolumeIndicator from "./VolumeIndicator";

interface UserCardProps {
  userName: string;
  isMuted: boolean;
  isSelf: boolean;
  speaking: boolean;
}

export default function UserCard({ userName, isMuted, isSelf, speaking }: UserCardProps) {
  return (
    <div
      className={`relative bg-gray-800 rounded-xl p-4 flex flex-col items-center gap-3 transition-all ${
        speaking ? "ring-2 ring-green-400 shadow-lg shadow-green-400/20" : ""
      }`}
    >
      {/* Avatar */}
      <div className="w-16 h-16 rounded-full bg-gray-600 flex items-center justify-center text-2xl">
        {userName.charAt(0).toUpperCase()}
      </div>

      {/* Name */}
      <span className="text-white text-sm font-medium truncate max-w-full">
        {userName}
        {isSelf && <span className="text-gray-400 ml-1">(我)</span>}
      </span>

      {/* Mute indicator */}
      {isMuted && (
        <div className="absolute top-2 right-2 text-red-400 text-xs" title="已静音">
          🔇
        </div>
      )}

      {/* Volume indicator */}
      {!isMuted && <VolumeIndicator level={speaking ? 0.7 : 0} />}
    </div>
  );
}
