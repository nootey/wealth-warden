import { computed } from "vue";
import { useThemeStore } from "../../services/stores/theme_store.ts";
import {
  categoryPalette,
  tokenColor,
  SEMANTIC_TOKENS,
  FLOW_TOKENS,
  neutrals,
} from "./tokens.ts";

export { categoryPalette };

export function useChartColors() {
  const themeStore = useThemeStore();
  const isDark = computed(() => themeStore.isDark);

  const colors = computed(() => {
    const n = neutrals(isDark.value);
    return {
      // Common scaffolding (theme-aware neutrals)
      axisText: n.axisText,
      axisBorder: n.axisBorder,
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
        savings: tokenColor(FLOW_TOKENS.savings),
        investments: tokenColor(FLOW_TOKENS.investments),
        debt: tokenColor(FLOW_TOKENS.debt),
        unallocated: n.unallocated,
      },

      dim: n.dim,
    };
  });

  return { colors };
}
