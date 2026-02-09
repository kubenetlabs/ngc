import type { Condition } from "@/types/gateway";

interface StatusBadgeProps {
  condition: Condition;
}

const statusColors: Record<string, string> = {
  True: "bg-emerald-500/15 text-emerald-400 border-emerald-500/30",
  False: "bg-red-500/15 text-red-400 border-red-500/30",
  Unknown: "bg-zinc-500/15 text-zinc-400 border-zinc-500/30",
};

export function StatusBadge({ condition }: StatusBadgeProps) {
  const color = statusColors[condition.status] ?? statusColors.Unknown;

  return (
    <span
      className={`inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium ${color}`}
      title={condition.message}
    >
      {condition.type}: {condition.reason}
    </span>
  );
}

interface StatusDotProps {
  status: "True" | "False" | "Unknown";
  label?: string;
}

const dotColors: Record<string, string> = {
  True: "bg-emerald-400",
  False: "bg-red-400",
  Unknown: "bg-zinc-400",
};

export function StatusDot({ status, label }: StatusDotProps) {
  return (
    <span className="inline-flex items-center gap-1.5 text-xs">
      <span className={`h-2 w-2 rounded-full ${dotColors[status] ?? dotColors.Unknown}`} />
      {label && <span className="text-muted-foreground">{label}</span>}
    </span>
  );
}
