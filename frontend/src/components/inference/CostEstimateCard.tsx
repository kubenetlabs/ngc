import type { CostEstimate } from "@/types/inference";

interface CostEstimateCardProps {
  cost: CostEstimate;
}

function formatCurrency(val: number): string {
  return `$${val.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

export function CostEstimateCard({ cost }: CostEstimateCardProps) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <h3 className="mb-3 text-sm font-medium text-muted-foreground">Cost Estimate</h3>
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">GPU Type</span>
          <span className="text-sm font-medium text-foreground">{cost.gpuType}</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Replicas</span>
          <span className="text-sm font-medium text-foreground">{cost.replicaCount}</span>
        </div>
        <div className="border-t border-border pt-3">
          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">Hourly</span>
            <span className="text-sm font-medium text-foreground">{formatCurrency(cost.hourlyRate)}</span>
          </div>
          <div className="mt-1 flex items-center justify-between">
            <span className="text-sm text-muted-foreground">Daily</span>
            <span className="text-sm font-medium text-foreground">{formatCurrency(cost.dailyCost)}</span>
          </div>
          <div className="mt-1 flex items-center justify-between">
            <span className="text-sm text-muted-foreground">Monthly</span>
            <span className="text-lg font-bold text-foreground">{formatCurrency(cost.monthlyCost)}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
