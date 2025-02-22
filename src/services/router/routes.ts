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
    },
    {
        path: '/inflows',
        name: 'Inflows',
        component: InflowsIndex,
        beforeEnter: [requiresAuth],
    },
    {
        path: '/Outflows',
        name: 'Outflows',
        component: OutflowsIndex,
        beforeEnter: [requiresAuth],
    },
    {
        path: '/investments',
        name: 'Investments',
        component: InvestmentsIndex,
        beforeEnter: [requiresAuth],
    },
    {
        path: '/savings',
        name: 'Savings',
        component: SavingsIndex,
        beforeEnter: [requiresAuth],
    },
    {
        path: '/debt',
        name: 'Debt',
        component: DebtIndex,
        beforeEnter: [requiresAuth],
    },
    {
        path: '/cash',
        name: 'Cash',
        component: CashIndex,
        beforeEnter: [requiresAuth],
    },
    {
        path: '/charts',
        name: 'Charts',
        component: ChartingIndex,
        beforeEnter: [requiresAuth],
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

export default routes