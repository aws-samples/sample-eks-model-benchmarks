import type { CatalogEntry } from "../types";

// Return the index of the winning entry for a metric, or null when the
// values are tied within `tolerance` (relative) or when fewer than two
// entries have values.
export function winnerIndex(
  values: Array<number | undefined | null>,
  direction: "min" | "max",
  tolerance = 0.01
): number | null {
  const present: Array<{ idx: number; v: number }> = [];
  for (let i = 0; i < values.length; i++) {
    const v = values[i];
    if (v !== null && v !== undefined && Number.isFinite(v)) {
      present.push({ idx: i, v });
    }
  }
  if (present.length < 2) return null;

  const best = present.reduce((a, b) =>
    direction === "min" ? (a.v <= b.v ? a : b) : (a.v >= b.v ? a : b)
  );

  // Tie check: if any other value is within tolerance of best, no winner.
  for (const p of present) {
    if (p.idx === best.idx) continue;
    const diff = Math.abs(p.v - best.v);
    const rel = best.v === 0 ? 0 : diff / Math.abs(best.v);
    if (rel <= tolerance) return null;
  }
  return best.idx;
}

// Fields we try to diff on, in the order they should appear.
const CONFIG_FIELDS: Array<{
  label: string;
  get: (e: CatalogEntry) => string;
}> = [
  { label: "Model", get: (e) => e.model_hf_id },
  { label: "Instance", get: (e) => e.instance_type_name },
  { label: "TP Degree", get: (e) => String(e.tensor_parallel_degree) },
  { label: "Quantization", get: (e) => e.quantization ?? "default" },
  { label: "Concurrency", get: (e) => String(e.concurrency) },
  { label: "Framework", get: (e) => `${e.framework} ${e.framework_version}` },
  { label: "Input Seq", get: (e) => String(e.input_sequence_length) },
  { label: "Output Seq", get: (e) => String(e.output_sequence_length) },
];

export interface ConfigDiffRow {
  label: string;
  values: string[];
}

// Returns only the config fields whose values differ across the entries.
export function configDiff(entries: CatalogEntry[]): ConfigDiffRow[] {
  if (entries.length < 2) return [];
  const rows: ConfigDiffRow[] = [];
  for (const f of CONFIG_FIELDS) {
    const vals = entries.map(f.get);
    const allSame = vals.every((v) => v === vals[0]);
    if (!allSame) rows.push({ label: f.label, values: vals });
  }
  return rows;
}

// Short summary sentences synthesizing the most notable winners. At most 5.
export function compareSummary(entries: CatalogEntry[]): string[] {
  if (entries.length < 2) return [];
  const labels = entries.map((e) => e.model_hf_id.split("/").pop() ?? e.run_id.slice(0, 8));
  const sentences: string[] = [];

  const tryCompare = (
    metric: string,
    direction: "min" | "max",
    values: Array<number | undefined>,
    unit: string,
    format: (v: number) => string
  ) => {
    const idx = winnerIndex(values, direction);
    if (idx === null) return;
    const best = values[idx]!;
    // Find next-best for the delta
    let runner = -1;
    for (let i = 0; i < values.length; i++) {
      if (i === idx) continue;
      const v = values[i];
      if (v === undefined || v === null) continue;
      if (runner === -1) {
        runner = i;
      } else {
        const rv = values[runner]!;
        if (direction === "min" ? v < rv : v > rv) runner = i;
      }
    }
    if (runner === -1) return;
    const runnerV = values[runner]!;
    const deltaPct = runnerV === 0 ? 0 : ((best - runnerV) / runnerV) * 100;
    const sign = direction === "min" ? "-" : "+";
    sentences.push(
      `${labels[idx]} leads on ${metric} (${format(best)} ${unit} vs ${format(runnerV)} ${unit}, ${sign}${Math.abs(
        deltaPct
      ).toFixed(0)}%).`
    );
  };

  tryCompare(
    "TTFT p99",
    "min",
    entries.map((e) => e.ttft_p99_ms),
    "ms",
    (v) => v.toFixed(0)
  );
  tryCompare(
    "Throughput",
    "max",
    entries.map((e) => e.throughput_aggregate_tps),
    "tok/s",
    (v) => v.toFixed(0)
  );
  tryCompare(
    "Success Rate",
    "max",
    entries.map((e) => {
      const ok = e.successful_requests ?? 0;
      const total = ok + (e.failed_requests ?? 0);
      return total > 0 ? (ok / total) * 100 : undefined;
    }),
    "%",
    (v) => v.toFixed(1)
  );

  // Success-rate floor note: if all are ≥ 99%, mention it instead of highlighting.
  const rates = entries.map((e) => {
    const ok = e.successful_requests ?? 0;
    const total = ok + (e.failed_requests ?? 0);
    return total > 0 ? (ok / total) * 100 : undefined;
  });
  if (rates.every((r) => r !== undefined && r >= 99)) {
    // Remove an earlier Success Rate sentence if present, since the floor note is more useful.
    const idx = sentences.findIndex((s) => s.includes("Success Rate"));
    if (idx !== -1) sentences.splice(idx, 1);
    sentences.push("All runs exceed 99% success rate.");
  }

  return sentences.slice(0, 5);
}
