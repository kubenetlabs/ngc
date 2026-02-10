import { KVCacheRing } from "./KVCacheRing";
import { GPUUtilizationBar } from "./GPUUtilizationBar";

interface PodCardProps {
  podName: string;
  gpuUtilPct: number;
  kvCacheUtilPct: number;
  queueDepth: number;
  requestsInFlight: number;
  highlighted?: boolean;
}

export function PodCard({
  podName,
  gpuUtilPct,
  kvCacheUtilPct,
  queueDepth,
  requestsInFlight,
  highlighted = false,
}: PodCardProps) {
  return (
    <div
      className={`rounded-lg border p-3 transition-all duration-300 ${
        highlighted
          ? "border-blue-500 bg-blue-500/10 shadow-lg shadow-blue-500/20"
          : "border-border bg-card"
      }`}
    >
      <div className="mb-2 flex items-center justify-between">
        <span className="text-xs font-medium text-foreground truncate">{podName}</span>
        <span className="text-[10px] text-muted-foreground">{requestsInFlight} in-flight</span>
      </div>

      <div className="flex items-center gap-3">
        <KVCacheRing percentage={kvCacheUtilPct} size={44} />
        <div className="flex-1 space-y-1.5">
          <div>
            <span className="text-[10px] text-muted-foreground">GPU</span>
            <GPUUtilizationBar value={gpuUtilPct} />
          </div>
          <div className="flex items-center gap-1">
            <span className="text-[10px] text-muted-foreground">Queue:</span>
            <div className="flex gap-0.5">
              {Array.from({ length: Math.min(queueDepth, 10) }).map((_, i) => (
                <div
                  key={i}
                  className={`h-2 w-1.5 rounded-sm ${
                    queueDepth > 8 ? "bg-red-500" : queueDepth > 4 ? "bg-yellow-500" : "bg-emerald-500"
                  }`}
                />
              ))}
              {queueDepth > 10 && (
                <span className="text-[9px] text-muted-foreground">+{queueDepth - 10}</span>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
