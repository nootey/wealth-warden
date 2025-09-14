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
        name: 'dashboard',
        meta: {title: 'Dash', requiresAuth: true},
        component: DashboardPage,
    },
    {
        path: '/confirm-email',
        name: 'confirm.email',
        meta: { title: 'Confirm email', requiresAuth: true, emailConfirmed: false },
        component: ConfirmEmail,
    },
    {
        path: '/login',
        name: 'login',
        meta: {title: 'Login', guestOnly: true},
        component: Login,
    },
    {
        path: '/signup',
        name: 'sign.up',
        meta: {title: 'Sign up', guestOnly: true},
        component: SignUp,
    },
    {
        path: '/accounts',
        name: 'accounts',
        meta: {title: 'Accounts', requiresAuth: true},
        component: AccountsPage,
    },
    {
        path: '/transactions',
        name: 'transactions',
        meta: {title: 'Transactions', requiresAuth: true},
        component: TransactionsPage,
    },
    {
        path: '/logs',
        name: 'logs',
        meta: {title: 'Audit', requiresAuth: true},
        component: ActivityLogsPage,
    },
    { path: '/settings', redirect: '/settings/profile' },
    // one component, different URLs
    {
        path: '/settings/:section(general|profile|preferences|accounts|categories)',
        name: 'settings.section',
        meta: {title: 'Settings', requiresAuth: true},
        component: SettingsPage,
        props: true,
    },
];

export default routes