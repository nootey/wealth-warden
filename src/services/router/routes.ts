import type {RouteRecordRaw} from 'vue-router';
import DashboardPage from "../../_vue/pages/DashboardPage.vue";
import Login from "../../_vue/features/auth/Login.vue";
import SignUp from "../../_vue/features/auth/SignUp.vue";
import ConfirmEmail from "../../_vue/features/auth/ConfirmEmail.vue";
import ActivityLogsPage from "../../_vue/pages/ActivityLogsPage.vue";
import TransactionsPage from "../../_vue/pages/TransactionsPage.vue";
import AccountsPage from "../../_vue/pages/AccountsPage.vue";
import SettingsPage from "../../_vue/pages/SettingsPage.vue";

declare module 'vue-router' {
    interface RouteMeta {
        requiresAuth?: boolean;
        guestOnly?: boolean;
        emailConfirmed?: boolean;
    }
}

const routes: RouteRecordRaw[] = [
    {
        path: '/',
        name: 'Dashboard',
        meta: {title: 'Dash', requiresAuth: true},
        component: DashboardPage,
    },
    {
        path: '/confirm-email',
        name: 'Confirm email',
        meta: { title: 'Confirm email', requiresAuth: true, emailConfirmed: false },
        component: ConfirmEmail,
    },
    {
        path: '/login',
        name: 'Login',
        meta: {title: 'Login', guestOnly: true},
        component: Login,
    },
    {
        path: '/signup',
        name: 'Sign up',
        meta: {title: 'Sign up', guestOnly: true},
        component: SignUp,
    },
    {
        path: '/accounts',
        name: 'Accounts',
        meta: {title: 'Accounts', requiresAuth: true},
        component: AccountsPage,
    },
    {
        path: '/transactions',
        name: 'Transactions',
        meta: {title: 'Transactions', requiresAuth: true},
        component: TransactionsPage,
    },
    {
        path: '/logs',
        name: 'Logs',
        meta: {title: 'Audit', requiresAuth: true},
        component: ActivityLogsPage,
    },
    { path: '/settings', redirect: '/settings/profile' },
    // one component, different URLs
    {
        path: '/settings/:section(general|profile|preferences|accounts|categories)',
        name: 'SettingsSection',
        meta: {title: 'Settings', requiresAuth: true},
        component: SettingsPage,
        props: true,
    },
];

export default routes