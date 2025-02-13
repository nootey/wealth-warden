import { defineStore } from 'pinia';

export const useThemeStore = defineStore('theme', {
    state: () => ({
        darkModeActive: false
    }),
    actions: {
        toggleDarkMode() {
            this.darkModeActive = !this.darkModeActive;
            const rootEl = document.documentElement;
            if (this.darkModeActive) {
                rootEl.classList.add('my-app-dark');
            } else {
                rootEl.classList.remove('my-app-dark');
            }
        }
    }
});