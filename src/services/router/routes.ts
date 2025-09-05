import { useAuthStore } from '../stores/auth_store.ts';

import DashboardPage from "../../_vue/pages/DashboardPage.vue";
import Login from "../../_vue/features/auth/Login.vue";
import ActivityLogsPage from "../../_vue/pages/ActivityLogsPage.vue";
import TransactionsPage from "../../_vue/pages/TransactionsPage.vue";
import AccountsPage from "../../_vue/pages/AccountsPage.vue";
import SettingsPage from "../../_vue/pages/SettingsPage.vue";

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
        path: '/accounts',
        name: 'Accounts',
        component: AccountsPage,
        beforeEnter: [requiresAuth],
    },
    {
        path: '/transactions',
        name: 'Transactions',
        component: TransactionsPage,
        beforeEnter: [requiresAuth],
    },
    {
        path: '/logs',
        name: 'Logs',
        component: ActivityLogsPage,
        beforeEnter: [requiresAuth],
    },
    { path: '/settings', redirect: '/settings/profile' },
    // one component, different URLs
    {
        path: '/settings/:section(profile|preferences|accounts|categories)',
        name: 'SettingsSection',
        component: SettingsPage,
        props: true,
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