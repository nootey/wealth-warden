import { defineStore } from 'pinia';

export const useThemeStore = defineStore('theme', {
    state: () => ({
        darkModeActive: true
    }),
    actions: {
        initializeTheme() {
            const savedTheme = localStorage.getItem('darkModeActive');
            if (savedTheme !== null) {
                this.darkModeActive = savedTheme === 'true';
            }

            const rootEl = document.documentElement;
            if (this.darkModeActive) {
                rootEl.classList.add('my-app-dark');
            } else {
                rootEl.classList.remove('my-app-dark');
            }
        },
        toggleDarkMode() {
            this.darkModeActive = !this.darkModeActive;
            localStorage.setItem('darkModeActive', this.darkModeActive.toString());

            const rootEl = document.documentElement;
            if (this.darkModeActive) {
                rootEl.classList.add('my-app-dark');
            } else {
                rootEl.classList.remove('my-app-dark');
            }
        }
    }
});