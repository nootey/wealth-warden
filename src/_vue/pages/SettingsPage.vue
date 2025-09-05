<script setup lang="ts">
import { computed } from 'vue';
import { useRoute, RouterLink } from 'vue-router';

import PreferencesSettings from '../features/Settings/PreferencesSettings.vue';
import ProfileSettings from "../features/Settings/ProfileSettings.vue";
import AccountsSettings from "../features/Settings/AccountsSettings.vue";
import CategoriesSettings from "../features/Settings/CategoriesSettings.vue";
import vueHelper from "../../utils/vue_helper.ts";

const route = useRoute();

type Section = 'profile' | 'preferences' | 'categories' | 'accounts';
const allowed: Record<Section, any> = {
    profile: ProfileSettings,
    preferences: PreferencesSettings,
    categories: CategoriesSettings,
    accounts: AccountsSettings,
};

const section = computed<Section>(() => {
    const s = (route.params.section as string) || 'account';
    return (Object.keys(allowed) as Section[]).includes(s as Section) ? (s as Section) : 'account';
});

const CurrentView = computed(() => allowed[section.value]);
</script>

<template>
    <div class="settings flex p-2">
        <aside class="w-16rem text-white h-full flex flex-column gap-2 p-3">

            <div class="flex flex-row gap-2 p-2 mb-2 align-items-center cursor-pointer font-bold hoverable" style="color: var(--text-primary)">
                <i class="pi pi-angle-left"></i>
                <span>Back</span>
            </div>

            <h6 class="text-xs font-bold uppercase mb-2" style="color: var(--text-primary);">General</h6>

            <RouterLink
                    :to="{ name: 'SettingsSection', params: { section: 'profile' } }"
                    class="flex align-items-center text-center gap-2 p-2 cursor-pointer"
                    style="text-decoration: none; transition: all 0.2s ease; color: var(--text-primary);"
                    :class="{ 'active': section === 'profile' }">
                <i class="pi pi-user text-sm" style="color: var(--text-secondary)"></i>
                <span>Profile</span>
            </RouterLink>

            <RouterLink
                    :to="{ name: 'SettingsSection', params: { section: 'preferences' } }"
                    class="flex align-items-center text-center gap-2 p-2 cursor-pointer"
                    style="text-decoration: none; transition: all 0.2s ease; color: var(--text-primary);"
                    :class="{ 'active': section === 'preferences' }">
                <i class="pi pi-cog text-sm" style="color: var(--text-secondary)"></i>
                <span>Preferences</span>
            </RouterLink>

            <h6 class="text-xs font-bold uppercase mb-2" style="color: var(--text-primary);">Transactions</h6>

            <RouterLink
                    :to="{ name: 'SettingsSection', params: { section: 'accounts' } }"
                    class="flex align-items-center text-center gap-2 p-2 cursor-pointer"
                    style="text-decoration: none; transition: all 0.2s ease; color: var(--text-primary);"
                    :class="{ 'active': section === 'accounts' }">
                <i class="pi pi-book text-sm" style="color: var(--text-secondary)"></i>
                <span>Accounts</span>
            </RouterLink>

            <RouterLink
                    :to="{ name: 'SettingsSection', params: { section: 'categories' } }"
                    class="flex align-items-center text-center gap-2 p-2 cursor-pointer"
                    style="text-decoration: none; transition: all 0.2s ease; color: var(--text-primary);"
                    :class="{ 'active': section === 'categories' }">
                <i class="pi pi-box text-sm" style="color: var(--text-secondary)"></i>
                <span>Categories</span>
            </RouterLink>

        </aside>

        <main class="flex-1 pt-3">
            <div class="flex flex-row gap-2 mb-2 align-items-center">
                <span class="text-sm" style="color: var(--text-secondary)">Home</span>
                <i class="pi pi-angle-right"></i>
                <span style="color: var(--text-primary)">{{ vueHelper.capitalize(section) }}</span>
            </div>
            <component :is="CurrentView" />
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
