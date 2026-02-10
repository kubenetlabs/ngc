import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, Area, AreaChart } from "recharts";
import type { TimeseriesPoint } from "@/types/inference";

interface TimeseriesChartProps {
  title: string;
  data: TimeseriesPoint[];
  color?: string;
  unit?: string;
  variant?: "line" | "area";
}

export function TimeseriesChart({
  title,
  data,
  color = "hsl(217, 91%, 60%)",
  unit = "",
  variant = "line",
}: TimeseriesChartProps) {
  const chartData = data.map((p) => ({
    time: new Date(p.timestamp).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
    value: p.value,
  }));

  const Chart = variant === "area" ? AreaChart : LineChart;

  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <h3 className="mb-3 text-sm font-medium text-muted-foreground">{title}</h3>
      <ResponsiveContainer width="100%" height={200}>
        <Chart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
          <XAxis
            dataKey="time"
            tick={{ fontSize: 10, fill: "hsl(var(--muted-foreground))" }}
            interval="preserveStartEnd"
          />
          <YAxis tick={{ fontSize: 11, fill: "hsl(var(--muted-foreground))" }} />
          <Tooltip
            contentStyle={{
              backgroundColor: "hsl(var(--card))",
              border: "1px solid hsl(var(--border))",
              borderRadius: "6px",
              fontSize: "12px",
            }}
            formatter={(value: number | undefined) => [`${(value ?? 0).toFixed(1)}${unit}`, title]}
          />
          {variant === "area" ? (
            <Area
              type="monotone"
              dataKey="value"
              stroke={color}
              fill={color}
              fillOpacity={0.1}
              strokeWidth={2}
              dot={false}
            />
          ) : (
            <Line type="monotone" dataKey="value" stroke={color} strokeWidth={2} dot={false} />
          )}
        </Chart>
      </ResponsiveContainer>
    </div>
  );
}
