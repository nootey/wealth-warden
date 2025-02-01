import DashboardIndex from "../../components/Dashboard/DashboardIndex.vue";
import InflowsIndex from "../../components/Inflows/InflowsIndex.vue";

const routes = [
    {
        path: '/',
        name: 'Dashboard',
        component: DashboardIndex
    },
    {
        path: '/inflows',
        name: 'Inflows',
        component: InflowsIndex
    }
];

export default routes