import { useAuthStore } from '../stores/auth.ts';

import DashboardIndex from "../../components/Dashboard/DashboardIndex.vue";
import InflowsIndex from "../../components/Inflows/InflowsIndex.vue";
import Login from "../../components/Auth/Login.vue";

const routes = [
    {
        path: '/',
        name: 'Dashboard',
        component: DashboardIndex,
        beforeEnter: [requiresAuth],
    },
    {
        path: '/login',
        name: 'Login',
        component: Login,
    },
    {
        path: '/inflows',
        name: 'Inflows',
        component: InflowsIndex,
        beforeEnter: [requiresAuth],
    }
];

function requiresAuth() {
    const authStore = useAuthStore();
    if (authStore.isAuthenticated) {
        return true;
    } else {
        return { path: '/login' };
    }
}

export default routes