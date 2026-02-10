interface KVCacheRingProps {
  percentage: number;
  size?: number;
}

export function KVCacheRing({ percentage, size = 48 }: KVCacheRingProps) {
  const clamped = Math.min(100, Math.max(0, percentage));
  const radius = (size - 6) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (clamped / 100) * circumference;

  const color = clamped >= 90 ? "#ef4444" : clamped >= 70 ? "#eab308" : "#10b981";

  return (
    <svg width={size} height={size} className="shrink-0">
      <circle
        cx={size / 2}
        cy={size / 2}
        r={radius}
        fill="none"
        stroke="hsl(var(--border))"
        strokeWidth={3}
      />
      <circle
        cx={size / 2}
        cy={size / 2}
        r={radius}
        fill="none"
        stroke={color}
        strokeWidth={3}
        strokeDasharray={circumference}
        strokeDashoffset={offset}
        strokeLinecap="round"
        transform={`rotate(-90 ${size / 2} ${size / 2})`}
        className="transition-all duration-500"
      />
      <text
        x={size / 2}
        y={size / 2}
        textAnchor="middle"
        dominantBaseline="central"
        className="fill-foreground text-[9px] font-medium"
      >
        {clamped.toFixed(0)}%
      </text>
    </svg>
  );
}
