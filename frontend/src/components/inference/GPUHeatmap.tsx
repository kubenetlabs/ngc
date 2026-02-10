import type { PodGPUMetrics } from "@/types/inference";

interface GPUHeatmapProps {
  pods: PodGPUMetrics[];
}

function getCellColor(util: number): string {
  if (util >= 90) return "bg-red-500";
  if (util >= 80) return "bg-orange-500";
  if (util >= 60) return "bg-yellow-500";
  if (util >= 40) return "bg-emerald-500";
  return "bg-emerald-700";
}

export function GPUHeatmap({ pods }: GPUHeatmapProps) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <h3 className="mb-3 text-sm font-medium text-muted-foreground">GPU Utilization Heatmap</h3>
      <div className="grid grid-cols-3 gap-2 sm:grid-cols-4 md:grid-cols-6">
        {pods.map((pod) => (
          <div
            key={pod.podName}
            className={`group relative rounded-md p-3 ${getCellColor(pod.gpuUtilPct)} cursor-default transition-transform hover:scale-105`}
            title={`${pod.podName}: ${pod.gpuUtilPct.toFixed(0)}% GPU, ${pod.kvCacheUtilPct.toFixed(0)}% KV Cache`}
          >
            <div className="text-center text-xs font-medium text-white">
              <div className="truncate">{pod.podName.split("-").pop()}</div>
              <div className="text-lg font-bold">{pod.gpuUtilPct.toFixed(0)}%</div>
              <div className="opacity-75">Q:{pod.queueDepth} KV:{pod.kvCacheUtilPct.toFixed(0)}%</div>
            </div>
          </div>
        ))}
      </div>
      <div className="mt-3 flex items-center gap-2 text-xs text-muted-foreground">
        <span>Low</span>
        <div className="flex gap-0.5">
          <div className="h-2 w-6 rounded bg-emerald-700" />
          <div className="h-2 w-6 rounded bg-emerald-500" />
          <div className="h-2 w-6 rounded bg-yellow-500" />
          <div className="h-2 w-6 rounded bg-orange-500" />
          <div className="h-2 w-6 rounded bg-red-500" />
        </div>
        <span>High</span>
      </div>
    </div>
  );
}
