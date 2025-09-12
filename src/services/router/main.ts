import { createRouter, createWebHistory } from 'vue-router';
import routes from './routes';
import {useAuthStore} from "../stores/auth_store.ts";

const router = createRouter({
    history: createWebHistory(),
    routes
});

router.beforeEach(async (to) => {
    const auth = useAuthStore();
    const requiresAuth = to.matched.some(r => r.meta.requiresAuth);
    const guestOnly = to.matched.some(r => r.meta.guestOnly);

    if (requiresAuth && !auth.isInitialized) {
        await auth.init();
    }
    if (requiresAuth && !auth.isAuthenticated) {
        return {name: 'Login', query: {redirect: to.fullPath}};
    }
    if (guestOnly && auth.isAuthenticated) {
        return {name: 'Dashboard'};
    }
});

router.afterEach((to) => {
    const baseTitle = 'Wealth Warden';
    const pageTitle = to.meta.title as string | undefined;

    document.title = pageTitle ? `${baseTitle} | ${pageTitle}` : baseTitle;
});


export default router;