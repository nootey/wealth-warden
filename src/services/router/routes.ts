import { useAuthStore } from '../stores/authStore.ts';

import DashboardIndex from "../../components/Dashboard/DashboardIndex.vue";
import InflowsIndex from "../../components/Inflows/InflowsIndex.vue";
import Login from "../../components/Auth/Login.vue";
import InvestmentsIndex from "../../components/Investments/InvestmentsIndex.vue";
import OutflowsIndex from "../../components/Outflows/OutflowsIndex.vue";
import CashIndex from "../../components/Cash/CashIndex.vue";
import SavingsIndex from "../../components/Savings/SavingsIndex.vue";
import DebtIndex from "../../components/Debt/DebtIndex.vue";
import ChartingIndex from "../../components/Charting/ChartingIndex.vue";
import LoggingHub from "../../components/Logging/LoggingHub.vue";

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
        path: '/Outflows',
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

function requiresActiveBudget() {
    const authStore = useAuthStore();
    if (authStore.isAuthenticated && authStore.user?.secrets?.budget_initialized) {
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