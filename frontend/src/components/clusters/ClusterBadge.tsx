import { Server } from "lucide-react";

interface Props {
  name: string;
  region?: string;
}

export function ClusterBadge({ name, region }: Props) {
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
      <Server className="h-3 w-3" />
      {name}
      {region && <span className="opacity-60">({region})</span>}
    </span>
  );
}
