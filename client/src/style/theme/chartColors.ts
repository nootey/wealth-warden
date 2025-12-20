import { computed } from "vue";
import { useThemeStore } from "../../services/stores/theme_store.ts";

export function useChartColors() {
  const themeStore = useThemeStore();
  const isDark = computed(() => themeStore.isDark);

  const colors = computed(() => ({
    // Common
    axisText: isDark.value ? "#9ca3af" : "#6b7280",
    axisBorder: isDark.value ? "rgba(255,255,255,0.12)" : "rgba(0,0,0,0.35)",
    guide: isDark.value ? "rgba(255,255,255,0.6)" : "rgba(0,0,0,0.4)",

    // Tooltip
    ttipBg: isDark.value ? "rgba(31,31,35,0.95)" : "rgba(255,255,255,0.95)",
    ttipText: isDark.value ? "#e5e7eb" : "#111827",
    ttipTitle: isDark.value ? "#9ca3af" : "#374151",
    ttipBorder: isDark.value ? "#2a2a2e" : "#e5e7eb",

    // Data semantics
    pos: "#22c55e", // green up
    neg: "#ef4444", // red down

    dim: isDark.value ? "rgba(156,163,175,0.55)" : "rgba(107,114,128,0.6)",
  }));

  return { colors };
}
