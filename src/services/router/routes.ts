import { useAuthStore } from '../stores/auth_store.ts';

import DashboardPage from "../../_vue/pages/DashboardPage.vue";
import Login from "../../_vue/features/auth/Login.vue";
import LoggingHub from "../../_vue/pages/LoggingHub.vue";

const routes = [
    {
        path: '/',
        name: 'Dashboard',
        component: DashboardPage,
        beforeEnter: [requiresAuth],
    },
    {
        path: '/login',
        name: 'Login',
        component: Login,
        beforeEnter: [requiresGuest],
    },
    {
        path: '/logs',
        name: 'Logs',
        component: LoggingHub,
        beforeEnter: [requiresAuth],
    },
];



function requiresAuth() {
    const authStore = useAuthStore();
    if (authStore.isAuthenticated) {
        return true;
    } else {
        return { path: '/login' };
    }
}

function requiresGuest() {
    const authStore = useAuthStore();
    if (!authStore.isAuthenticated) {
        return true;
    } else {
        return { path: '/' };
    }
}

export default routes