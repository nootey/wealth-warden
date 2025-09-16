<script setup lang="ts">
import { computed } from 'vue';
import {useRoute, RouterLink, useRouter} from 'vue-router';
import vueHelper from "../../utils/vue_helper.ts";
import {useAuthStore} from "../../services/stores/auth_store.ts";

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();

type SettingsMenuItem = {
    name: string;
    label: string;
    icon: string;
    adminOnly?: boolean;
    roleBlock?: boolean;
};

const items: SettingsMenuItem[] = [
    { name: 'settings.general',     label: 'General',     icon: 'pi-cog' , roleBlock: !authStore.isAdmin },
    { name: 'settings.profile',     label: 'Profile',     icon: 'pi-user' },
    { name: 'settings.preferences', label: 'Preferences', icon: 'pi-cog' },
    { name: 'settings.accounts',    label: 'Accounts',    icon: 'pi-book' },
    { name: 'settings.categories',  label: 'Categories',  icon: 'pi-box' },
];

const visibleItems = computed(() =>
    items.filter(item => !item.roleBlock)
);

const pageTitle = computed(() => {
    if (!route.name) return 'Settings';
    const parts = String(route.name).split('.');
    const last = parts[parts.length - 1];
    return vueHelper.capitalize(last);
});

const isActive = (name: SettingsMenuItem['name']) => route.name === name;

function goBack() {
    const hasBack = !!(router.options.history.state && router.options.history.state.back);
    if (hasBack) router.back();
    else router.push({ name: 'dashboard' });
}

</script>

<template>
    <div class="settings flex p-2 w-full">
        <aside class="w-16rem text-white h-full flex flex-column gap-2 p-3">
            <div class="flex flex-row gap-2 p-2 mb-2 align-items-center cursor-pointer font-bold hoverable"
                 style="color: var(--text-primary)">
                <i class="pi pi-angle-left"></i>
                <span @click="goBack">Back</span>
            </div>

            <h6 class="text-xs font-bold uppercase mb-2" style="color: var(--text-primary);">General</h6>

            <RouterLink v-for="item in visibleItems" :key="item.name" :to="{ name: item.name }"
                        class="flex align-items-center text-center gap-2 p-2 cursor-pointer"
                        :class="{ active: isActive(item.name) }"
                        style="text-decoration: none; transition: all 0.2s ease; color: var(--text-primary);"
            >

                <i class="pi text-sm" :class="item.icon" style="color: var(--text-secondary)"></i>
                <span>{{ item.label }}</span>
            </RouterLink>
        </aside>

        <main class="w-full flex-1 pt-3" style="max-width: 850px; margin: 0 auto;">
            <div class="flex flex-row gap-2 mb-2 align-items-center">
                <span class="text-sm" style="color: var(--text-secondary)">Home</span>
                <i class="pi pi-angle-right"></i>
                <span style="color: var(--text-primary)">{{ pageTitle }}</span>
            </div>

            <router-view />
        </main>
    </div>
</template>

<style scoped>
.settings { display: grid; grid-template-columns: 220px 1fr; gap: 1rem; }
.active,
.hoverable:hover {
    font-weight: bold;
    background-color: var(--background-secondary);
    border-radius: 8px;
}
</style>
