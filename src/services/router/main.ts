import { createRouter, createWebHistory } from 'vue-router';
import routes from './routes';
import {useAuthStore} from "../stores/auth_store.ts";
import {usePermissions} from "../../utils/use_permissions.ts";

const router = createRouter({
    history: createWebHistory(),
    routes
});

router.beforeEach(async (to) => {
    const auth = useAuthStore();
    const { hasPermission } = usePermissions()

    const requiresAuth = to.matched.some(r => r.meta.requiresAuth);
    const guestOnly = to.matched.some(r => r.meta.guestOnly);
    const emailValidated = to.matched.some(r => r.meta.emailValidated);
    const permsAny                = to.matched.flatMap(r => r.meta.permsAny ?? [])
    const permsAll                = to.matched.flatMap(r => r.meta.permsAll ?? [])

    // Initialize user if we need auth info
    if (requiresAuth && !auth.isInitialized) {
        await auth.init()
    }

    // Auth gate
    if (requiresAuth && !auth.isAuthenticated) {
        return {name: 'login', query: {redirect: to.fullPath}};
    }

    // Logged in but NOT verified
    if (requiresAuth && auth.isAuthenticated && !auth.isValidated && !emailValidated) {
        if (to.name !== 'confirm.email') {
            return { name: 'confirm.email', query: { redirect: to.fullPath } };
        }
    }

    // Guest-only pages
    if (guestOnly && auth.isAuthenticated) {
        return { name: 'dashboard' }
    }

    // Permission gates (only when authenticated)
    if (auth.isAuthenticated) {
        const has = (p: string | string[]) => hasPermission ? hasPermission(p) : false

        const anyOk = permsAny.length === 0 || permsAny.some(p => has(p))
        const allOk = permsAll.length === 0 || permsAll.every(p => has(p))

        if (!anyOk || !allOk) {
            return { name: 'dashboard' }
        }
    }

});

router.afterEach((to) => {
    const baseTitle = 'Wealth Warden';
    const pageTitle = to.meta.title as string | undefined;

    document.title = pageTitle ? `${baseTitle} | ${pageTitle}` : baseTitle;
});


export default router;