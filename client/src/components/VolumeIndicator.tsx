interface VolumeIndicatorProps {
  /** Volume level from 0 to 1 */
  level: number;
}

export default function VolumeIndicator({ level }: VolumeIndicatorProps) {
  const bars = 5;
  const activeBars = Math.round(level * bars);

  return (
    <div className="flex items-end gap-0.5 h-4">
      {Array.from({ length: bars }, (_, i) => (
        <div
          key={i}
          className={`w-1 rounded-full transition-all ${
            i < activeBars ? "bg-green-400" : "bg-gray-600"
          }`}
          style={{ height: `${((i + 1) / bars) * 100}%` }}
        />
      ))}
    </div>
  );
}
