// Single source of truth for data-viz and semantic colors, built on PrimeVue
export function tokenColor(token: string, fallback = "#6b7280"): string {
  if (typeof document === "undefined") return fallback;
  const value = getComputedStyle(document.documentElement)
    .getPropertyValue(token)
    .trim();
  if (!value) {
    if (import.meta.env.DEV) {
      console.warn(`tokenColor: "${token}" resolved empty; using fallback`);
    }
    return fallback;
  }
  return value;
}

// Categorical chart palette. Blurple-forward to match the app accent.
const CATEGORY_TOKENS = [
  "--p-indigo-500",
  "--p-blue-500",
  "--p-violet-500",
  "--p-cyan-500",
  "--p-purple-500",
  "--p-pink-500",
  "--p-teal-500",
  "--p-sky-500",
  "--p-fuchsia-500",
  "--p-emerald-500",
  "--p-rose-500",
  "--p-amber-500",
];

export function categoryPalette(): string[] {
  return CATEGORY_TOKENS.map((token) => tokenColor(token));
}

// Account type -> base hue. Assets read cool/green, liabilities warm.
export const ACCOUNT_TOKENS: Record<string, string> = {
  cash: "--p-indigo-500",
  investment: "--p-blue-500",
  crypto: "--p-sky-500",
  property: "--p-teal-500",
  vehicle: "--p-emerald-500",
  other_asset: "--p-green-500",
  credit_card: "--p-red-500",
  loan: "--p-orange-500",
  other_liability: "--p-amber-500",
};

export const SEMANTIC_TOKENS = {
  positive: "--p-green-500",
  negative: "--p-red-500",
};

export const FLOW_TOKENS = {
  savings: "--p-blue-500",
  investments: "--p-violet-500",
  debt: "--p-orange-500",
};

// Theme-aware neutrals for chart scaffolding (axes, tooltips, guides).
export function neutrals(dark: boolean) {
  const s = (step: number) => tokenColor(`--p-surface-${step}`);
  return {
    axisText: dark ? s(400) : s(600),
    axisBorder: dark ? s(700) : s(300),
    guide: dark ? s(600) : s(400),
    ttipBg: dark ? s(800) : s(0),
    ttipText: dark ? s(0) : s(900),
    ttipTitle: dark ? s(300) : s(600),
    ttipBorder: dark ? s(700) : s(200),
    dim: dark ? s(500) : s(400),
    unallocated: s(500),
  };
}
