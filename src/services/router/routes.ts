import { useAuthStore } from '../stores/authStore.ts';

import DashboardIndex from "../../_vue/views/DashboardIndex.vue";
import InflowsIndex from "../../_vue/views/InflowsIndex.vue";
import Login from "../../_vue/features/auth/Login.vue";
import InvestmentsIndex from "../../_vue/views/InvestmentsIndex.vue";
import OutflowsIndex from "../../_vue/views/OutflowsIndex.vue";
import CashIndex from "../../_vue/views/CashIndex.vue";
import SavingsIndex from "../../_vue/views/SavingsIndex.vue";
import DebtIndex from "../../_vue/views/DebtIndex.vue";
import ChartingIndex from "../../_vue/views/ChartingIndex.vue";
import LoggingHub from "../../_vue/views/LoggingHub.vue";

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
        beforeEnter: [requiresGuest],
    },
    {
        path: '/inflows',
        name: 'Inflows',
        component: InflowsIndex,
        beforeEnter: [requiresAuth, requiresActiveBudget],
    },
    {
        path: '/outflows',
        name: 'Outflows',
        component: OutflowsIndex,
        beforeEnter: [requiresAuth, requiresActiveBudget],
    },
    {
        path: '/investments',
        name: 'Investments',
        component: InvestmentsIndex,
        beforeEnter: [requiresAuth, requiresActiveBudget],
    },
    {
        path: '/savings',
        name: 'Savings',
        component: SavingsIndex,
        beforeEnter: [requiresAuth, requiresActiveBudget],
    },
    {
        path: '/debt',
        name: 'Debt',
        component: DebtIndex,
        beforeEnter: [requiresAuth, requiresActiveBudget],
    },
    {
        path: '/cash',
        name: 'Cash',
        component: CashIndex,
        beforeEnter: [requiresAuth, requiresActiveBudget],
    },
    {
        path: '/charts',
        name: 'Charts',
        component: ChartingIndex,
        beforeEnter: [requiresAuth, requiresActiveBudget],
    },
    {
        path: '/logs',
        name: 'Logs',
        component: LoggingHub,
        beforeEnter: [requiresAuth, requiresActiveBudget],
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

async function requiresActiveBudget() {
    const authStore = useAuthStore();
    await authStore.waitForUser();

    if (authStore.isAuthenticated && authStore.hasUserInitializedBudget) {
        return true;
    } else {
        return { path: '/' };
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