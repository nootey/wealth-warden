import { defineStore } from 'pinia';

export const useThemeStore = defineStore('theme', {
    state: () => ({
        theme: 'system' as 'system' | 'dark' | 'light',
        accent: 'blurple' as string
    }),
    getters: {
        isDark(): boolean {
            if (this.theme === 'system') {
                return window.matchMedia('(prefers-color-scheme: dark)').matches;
            }
            return this.theme === 'dark';
        }
    },
    actions: {
        initializeTheme() {
            // Set initial theme (system default)
            this.applyTheme();

            // Listen for system theme changes
            window.matchMedia('(prefers-color-scheme: dark)')
                .addEventListener('change', () => {
                    if (this.theme === 'system') {
                        this.applyTheme();
                    }
                });
        },

        setTheme(theme: 'system' | 'dark' | 'light', accent?: string) {
            this.theme = theme;
            if (accent) this.accent = accent;
            this.applyTheme();
        },

        applyTheme() {
            const rootEl = document.documentElement;
            if (this.isDark) {
                rootEl.classList.add('my-app-dark');
            } else {
                rootEl.classList.remove('my-app-dark');
            }

            // Apply accent color
            // rootEl.style.setProperty('--accent-color', this.accent);
        }
    }
});