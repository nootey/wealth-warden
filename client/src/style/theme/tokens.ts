import type { Accent } from "../../models/settings_models.ts";

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

type AccentTokens = {
  primary: string;
  // Light-mode button text; white fails contrast on amber
  lightContrast: string;
  category: string[];
  // Account type -> base hue. Liabilities stay red-leaning in every accent.
  account: Record<string, string>;
  flow: { savings: string; investments: string; debt: string };
};

const ACCENT_TOKENS: Record<Accent, AccentTokens> = {
  blurple: {
    primary: "indigo",
    lightContrast: "#ffffff",
    category: [
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
    ],
    account: {
      cash: "--p-indigo-500",
      investment: "--p-blue-500",
      crypto: "--p-sky-500",
      property: "--p-teal-500",
      vehicle: "--p-emerald-500",
      other_asset: "--p-green-500",
      credit_card: "--p-red-500",
      loan: "--p-orange-500",
      other_liability: "--p-amber-500",
    },
    flow: {
      savings: "--p-blue-500",
      investments: "--p-violet-500",
      debt: "--p-orange-500",
    },
  },
  // Neighbours alternate warm/cool so adjacent slices stay distinguishable.
  amber: {
    primary: "amber",
    lightContrast: "{surface.950}",
    category: [
      "--p-amber-400",
      "--p-orange-500",
      "--p-rose-400",
      "--p-lime-500",
      "--p-teal-400",
      "--p-pink-500",
      "--p-amber-600",
      "--p-fuchsia-400",
      "--p-orange-300",
      "--p-emerald-500",
      "--p-sky-400",
      "--p-yellow-600",
    ],
    account: {
      cash: "--p-amber-400",
      investment: "--p-orange-400",
      crypto: "--p-yellow-500",
      property: "--p-lime-500",
      vehicle: "--p-teal-500",
      other_asset: "--p-emerald-500",
      credit_card: "--p-red-500",
      loan: "--p-rose-500",
      other_liability: "--p-pink-500",
    },
    flow: {
      savings: "--p-amber-400",
      investments: "--p-orange-500",
      debt: "--p-rose-500",
    },
  },
  green: {
    primary: "green",
    lightContrast: "{surface.950}",
    category: [
      "--p-green-400",
      "--p-teal-500",
      "--p-lime-400",
      "--p-sky-500",
      "--p-emerald-600",
      "--p-cyan-400",
      "--p-lime-600",
      "--p-indigo-400",
      "--p-teal-300",
      "--p-amber-400",
      "--p-green-600",
      "--p-violet-400",
    ],
    account: {
      cash: "--p-green-400",
      investment: "--p-emerald-500",
      crypto: "--p-lime-400",
      property: "--p-teal-500",
      vehicle: "--p-cyan-500",
      other_asset: "--p-sky-500",
      credit_card: "--p-red-500",
      loan: "--p-orange-500",
      other_liability: "--p-amber-500",
    },
    flow: {
      savings: "--p-lime-400",
      investments: "--p-teal-500",
      debt: "--p-orange-500",
    },
  },
};

export function accentTokens(accent: string): AccentTokens {
  return ACCENT_TOKENS[accent as Accent] ?? ACCENT_TOKENS.blurple;
}

const STEPS = [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950];

// Semantic overrides for updatePreset. Every accent sets the same keys so switching back resets them.
export function accentPreset(accent: string) {
  const t = accentTokens(accent);
  return {
    semantic: {
      primary: Object.fromEntries(
        STEPS.map((step) => [step, `{${t.primary}.${step}}`]),
      ),
      colorScheme: {
        light: {
          primary: { contrastColor: t.lightContrast },
        },
      },
    },
  };
}

export const SEMANTIC_TOKENS = {
  positive: "--positive",
  negative: "--negative",
};

// Theme-aware neutrals for chart scaffolding (axes, tooltips, guides).
export function neutrals(dark: boolean) {
  const s = (step: number) => tokenColor(`--p-surface-${step}`);
  return {
    axisText: dark ? s(400) : s(600),
    axisBorder: dark ? s(700) : s(300),
    sliceBorder: dark ? s(300) : s(600),
    guide: dark ? s(600) : s(400),
    ttipBg: dark ? s(800) : s(0),
    ttipText: dark ? s(0) : s(900),
    ttipTitle: dark ? s(300) : s(600),
    ttipBorder: dark ? s(700) : s(200),
    dim: dark ? s(500) : s(400),
    unallocated: s(500),
  };
}
