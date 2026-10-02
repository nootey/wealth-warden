import { defineStore } from "pinia";
import { updatePreset } from "@primeuix/themes";
import { accentPreset } from "../../style/theme/tokens.ts";

export const useThemeStore = defineStore("theme", {
  state: () => ({
    theme: "system" as "system" | "dark" | "light",
    accent: "blurple" as string,
    systemDark: window.matchMedia("(prefers-color-scheme: dark)").matches,
  }),
  getters: {
    isDark(): boolean {
      if (this.theme === "system") {
        return this.systemDark;
      }
      return this.theme === "dark";
    },
  },
  actions: {
    initializeTheme() {
      // Set initial theme (system default)
      this.applyTheme();

      // Listen for system theme changes
      window
        .matchMedia("(prefers-color-scheme: dark)")
        .addEventListener("change", (e) => {
          this.systemDark = e.matches;
          if (this.theme === "system") {
            this.applyTheme();
          }
        });
    },

    setTheme(theme: "system" | "dark" | "light", accent?: string) {
      this.theme = theme;
      if (accent) this.accent = accent;
      this.applyTheme();
    },

    applyTheme() {
      const rootEl = document.documentElement;
      if (this.isDark) {
        rootEl.classList.add("my-app-dark");
      } else {
        rootEl.classList.remove("my-app-dark");
      }

      updatePreset(accentPreset(this.accent));
      rootEl.dataset.accent = this.accent;
    },
  },
});
