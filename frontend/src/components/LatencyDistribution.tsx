interface Props {
  label: string;
  p50?: number;
  p90?: number;
  p95?: number;
  p99?: number;
  unit?: string;
}

// Compact horizontal percentile distribution for a single latency metric.
// Each percentile is drawn as a bar whose length is proportional to its value
// relative to p99 (or p95 if p99 absent). Designed to live in a 4-up row
// alongside siblings for TTFT / E2E / ITL / TPOT, so the scale is local — a
// reader compares percentile spread within each metric, not across metrics.
export default function LatencyDistribution({
  label,
  p50,
  p90,
  p95,
  p99,
  unit = "ms",
}: Props) {
  const values: Array<{ key: string; value: number | undefined; opacity: number }> = [
    { key: "p50", value: p50, opacity: 1 },
    { key: "p90", value: p90, opacity: 0.72 },
    { key: "p95", value: p95, opacity: 0.5 },
    { key: "p99", value: p99, opacity: 0.3 },
  ];

  const present = values.filter((v) => v.value !== undefined) as Array<{
    key: string;
    value: number;
    opacity: number;
  }>;
  const scale = present.length > 0 ? Math.max(...present.map((v) => v.value)) : 0;

  return (
    <div className="panel p-3">
      <div className="flex items-baseline justify-between mb-2">
        <span className="eyebrow">{label}</span>
        <span className="caption">{unit}</span>
      </div>
      {scale === 0 ? (
        <div className="font-mono text-[12px] text-ink-2 py-2">—</div>
      ) : (
        <div className="flex flex-col gap-1">
          {values.map(({ key, value, opacity }) => {
            const pct = value === undefined ? 0 : (value / scale) * 100;
            return (
              <div key={key} className="flex items-center gap-2">
                <span className="eyebrow w-8 shrink-0">{key}</span>
                <div className="relative flex-1 h-4 bg-surface-2">
                  <div
                    className="absolute inset-y-0 left-0 bg-signal"
                    style={{ width: `${pct}%`, opacity }}
                  />
                </div>
                <span className="font-mono text-[11.5px] text-ink-0 tabular w-12 text-right shrink-0">
                  {value !== undefined ? value.toFixed(0) : "—"}
                </span>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
