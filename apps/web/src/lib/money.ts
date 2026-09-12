/**
 * Money formatting — BDT integer taka, always rendered with tnum
 * (apply the `.tnum` class alongside). Locale-aware grouping.
 */
export function formatTaka(
  amount: number | null | undefined,
  locale: "en" | "bn" = "en",
): string {
  if (amount === null || amount === undefined) return "৳0";
  const formatter = new Intl.NumberFormat(locale === "bn" ? "bn-BD" : "en-BD", {
    maximumFractionDigits: 0,
  });
  return `৳${formatter.format(amount)}`;
}

/** Compact form for dashboard tiles: ৳12.5k */
export function formatTakaCompact(
  amount: number | null | undefined,
  locale: "en" | "bn" = "en",
): string {
  if (amount === null || amount === undefined) return "৳0";
  const suffixes: Record<string, [number, string]> = {
    en: [1, "k"],
    bn: [1, "হাজার"],
  };
  const [divisor, suffix] = suffixes[locale] ?? suffixes.en;
  if (amount < 1000) return formatTaka(amount, locale);
  const value = amount / (1000 * divisor);
  const rounded = value >= 100 ? Math.round(value) : Math.round(value * 10) / 10;
  return `৳${rounded}${suffix}`;
}
