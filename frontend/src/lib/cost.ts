import type { PricingRow, PricingTier } from "../types";

// Returns the hourly rate for an instance at the given pricing tier, or null
// when the row isn't found or the tier doesn't have a price (e.g., no 3yr
// reserved commitment).
export function hourlyRate(
  pricing: PricingRow | undefined,
  tier: PricingTier
): number | null {
  if (!pricing) return null;
  switch (tier) {
    case "on_demand":
      return pricing.on_demand_hourly_usd;
    case "reserved_1yr":
      return pricing.reserved_1yr_hourly_usd ?? null;
    case "reserved_3yr":
      return pricing.reserved_3yr_hourly_usd ?? null;
  }
}

export function hourlyRateFromMap(
  pricingMap: Map<string, PricingRow>,
  instanceTypeName: string,
  tier: PricingTier
): number | null {
  return hourlyRate(pricingMap.get(instanceTypeName), tier);
}

export function costPerRequest(
  hourly: number | null,
  rps: number | undefined
): number | null {
  if (hourly == null || !rps || rps <= 0) return null;
  return hourly / rps / 3600;
}

export function costPer1MTokens(
  hourly: number | null,
  aggregateTps: number | undefined
): number | null {
  if (hourly == null || !aggregateTps || aggregateTps <= 0) return null;
  return (hourly / aggregateTps / 3600) * 1_000_000;
}

// Total spent for a run: hourly rate × duration (hours). Returns null when
// either the price or the duration is unknown.
export function totalSpent(
  hourly: number | null,
  durationSeconds: number | undefined
): number | null {
  if (hourly == null || !durationSeconds || durationSeconds <= 0) return null;
  return hourly * (durationSeconds / 3600);
}

// Convenience: build a pricing map keyed by instance_type_name.
export function toPricingMap(rows: PricingRow[]): Map<string, PricingRow> {
  const m = new Map<string, PricingRow>();
  for (const r of rows) m.set(r.instance_type_name, r);
  return m;
}
