import type {RouteRecordRaw} from 'vue-router';
import DashboardPage from "../../_vue/pages/DashboardPage.vue";
import Login from "../../_vue/features/auth/Login.vue";
import SignUp from "../../_vue/features/auth/SignUp.vue";
import ConfirmEmail from "../../_vue/features/auth/ConfirmEmail.vue";
import ForgotPassword from '../../_vue/features/auth/ForgotPassword.vue';
import ActivityLogsPage from "../../_vue/pages/ActivityLogsPage.vue";
import TransactionsPage from "../../_vue/pages/TransactionsPage.vue";
import AccountsPage from "../../_vue/pages/AccountsPage.vue";
import SettingsPage from "../../_vue/pages/SettingsPage.vue";
import ResetPassword from "../../_vue/features/auth/ResetPassword.vue";
import UsersPage from "../../_vue/pages/UsersPage.vue";
import NotFound from "../../_vue/components/base/NotFound.vue";
import GeneralSettings from "../../_vue/pages/Settings/GeneralSettings.vue";
import ProfileSettings from "../../_vue/pages/Settings/ProfileSettings.vue";
import PreferencesSettings from "../../_vue/pages/Settings/PreferencesSettings.vue";
import AccountsSettings from "../../_vue/pages/Settings/AccountsSettings.vue";
import CategoriesSettings from "../../_vue/pages/Settings/CategoriesSettings.vue";

declare module 'vue-router' {
    interface RouteMeta {
        requiresAuth?: boolean;
        requiresAdmin?: boolean;
        requiresSuperAdmin?: boolean;
        guestOnly?: boolean;
        emailConfirmed?: boolean;
    }
}

const routes: RouteRecordRaw[] = [
    {
        path: '/',
        name: 'dashboard',
        meta: { title: 'Dash', requiresAuth: true },
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
        meta: { title: 'Login', guestOnly: true },
        component: Login,
    },
    {
        path: '/signup',
        name: 'sign.up',
        meta: { title: 'Sign up', guestOnly: true },
        component: SignUp,
    },
    {
        path: '/forgot-password',
        name: 'forgot.password',
        meta: { title: 'Forgot password', guestOnly: true },
        component: ForgotPassword,
    },
    {
        path: '/reset-password/:token',
        name: 'reset.password',
        meta: { title: 'Reset password', guestOnly: true },
        component: ResetPassword,
    },
    {
        path: '/accounts',
        name: 'accounts',
        meta: { title: 'Accounts', requiresAuth: true},
        component: AccountsPage,
    },
    {
        path: '/transactions',
        name: 'transactions',
        meta: { title: 'Transactions', requiresAuth: true },
        component: TransactionsPage,
    },
    {
        path: '/users',
        name: 'users',
        meta: { title: 'Users', requiresAuth: true, requiresAdmin: true },
        component: UsersPage,
    },
    {
        path: '/logs',
        name: 'logs',
        meta: { title: 'Audit', requiresAuth: true, requiresAdmin: true },
        component: ActivityLogsPage,
    },
    {
        path: '/settings',
        component: SettingsPage,
        meta: { title: 'Settings', requiresAuth: true },
        children: [
            { path: '', redirect: { name: 'settings.profile' } },
            { path: 'general',     name: 'settings.general',     component: GeneralSettings,     meta: { title: 'General',     requiresAdmin: true } },
            { path: 'profile',     name: 'settings.profile',     component: ProfileSettings,     meta: { title: 'Profile' } },
            { path: 'preferences', name: 'settings.preferences', component: PreferencesSettings, meta: { title: 'Preferences' } },
            { path: 'accounts',    name: 'settings.accounts',    component: AccountsSettings,    meta: { title: 'Accounts' } },
            { path: 'categories',  name: 'settings.categories',  component: CategoriesSettings,  meta: { title: 'Categories' } },
        ],
    },
    {
        path: '/:pathMatch(.*)*',
        name: 'NotFound',
        component: NotFound,
        meta: {title: '404'}}
];

export default routes