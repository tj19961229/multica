// Compact token-count formatter. Shared by usage-related UI surfaces.
//   < 1k   → "999"
//   < 1M   → "1.2k"
//   ≥ 1M   → "1.5M"
// Always one decimal in the k/M branches; no trailing zero stripping —
// "1.0k" reads more honestly than "1k" once you've seen "1.2k" next to it.
export function formatTokenCount(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`;
  return String(n);
}
