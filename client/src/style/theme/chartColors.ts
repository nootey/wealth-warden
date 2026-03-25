import { computed } from "vue";
import { useThemeStore } from "../../services/stores/theme_store.ts";

export const CATEGORY_PALETTE = [
  "#6366f1", // indigo-500 (primary accent)
  "#3b82f6", // blue-500
  "#8b5cf6", // violet-500
  "#0ea5e9", // sky-500
  "#a855f7", // purple-500
  "#60a5fa", // blue-400
  "#ec4899", // pink-500
  "#818cf8", // indigo-400
  "#06b6d4", // cyan-500
  "#c084fc", // purple-400
  "#4f46e5", // indigo-600
  "#f472b6", // pink-400
  "#7c3aed", // violet-600
  "#38bdf8", // sky-400
  "#db2777", // pink-600
];

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
