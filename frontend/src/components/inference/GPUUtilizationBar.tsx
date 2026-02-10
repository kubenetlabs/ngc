interface GPUUtilizationBarProps {
  value: number;
  showLabel?: boolean;
}

function getBarColor(value: number): string {
  if (value >= 90) return "bg-red-500";
  if (value >= 75) return "bg-yellow-500";
  return "bg-emerald-500";
}

export function GPUUtilizationBar({ value, showLabel = true }: GPUUtilizationBarProps) {
  const clamped = Math.min(100, Math.max(0, value));
  return (
    <div className="flex items-center gap-2">
      <div className="h-2 w-24 rounded-full bg-muted">
        <div
          className={`h-2 rounded-full transition-all ${getBarColor(clamped)}`}
          style={{ width: `${clamped}%` }}
        />
      </div>
      {showLabel && (
        <span className="text-xs text-muted-foreground">{clamped.toFixed(0)}%</span>
      )}
    </div>
  );
}
