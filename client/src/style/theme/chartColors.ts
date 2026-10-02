import { computed } from "vue";
import { useThemeStore } from "../../services/stores/theme_store.ts";
import {
  accentTokens,
  tokenColor,
  SEMANTIC_TOKENS,
  neutrals,
} from "./tokens.ts";

export function categoryPalette(): string[] {
  return accentTokens(useThemeStore().accent).category.map((token) =>
    tokenColor(token),
  );
}

export function useChartColors() {
  const themeStore = useThemeStore();
  const isDark = computed(() => themeStore.isDark);

  const colors = computed(() => {
    const n = neutrals(isDark.value);
    const flow = accentTokens(themeStore.accent).flow;
    return {
      // Common scaffolding (theme-aware neutrals)
      axisText: n.axisText,
      axisBorder: n.axisBorder,
      sliceBorder: n.sliceBorder,
      guide: n.guide,

      // Tooltip
      ttipBg: n.ttipBg,
      ttipText: n.ttipText,
      ttipTitle: n.ttipTitle,
      ttipBorder: n.ttipBorder,

      // Data semantics
      pos: tokenColor(SEMANTIC_TOKENS.positive),
      neg: tokenColor(SEMANTIC_TOKENS.negative),

      // Cash-flow (sankey) targets
      flow: {
        savings: tokenColor(flow.savings),
        investments: tokenColor(flow.investments),
        debt: tokenColor(flow.debt),
        unallocated: n.unallocated,
      },

      dim: n.dim,
    };
  });

  return { colors };
}
