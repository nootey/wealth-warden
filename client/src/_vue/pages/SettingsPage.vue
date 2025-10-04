<script setup lang="ts">
import {computed, ref} from 'vue';
import {useRoute, RouterLink, useRouter} from 'vue-router';
import vueHelper from "../../utils/vue_helper.ts";
import {usePermissions} from "../../utils/use_permissions.ts";

const router = useRouter();
const route = useRoute();
const { hasPermission } = usePermissions();

type SettingsMenuItem = {
    name: string;
    label: string;
    icon?: string;
    block?: boolean;
    separator?: boolean;
};

const items: SettingsMenuItem[] = [
    { name: 'settings.general',     label: 'General',     icon: 'pi-cog' , block: !hasPermission("root_access") },
    { name: 'settings.profile',     label: 'Profile',     icon: 'pi-user' },
    { name: 'settings.preferences', label: 'Preferences', icon: 'pi-cog' },
    { name: '', label: 'Transactions', separator: true, block: !hasPermission("manage_data") },
    { name: 'settings.accounts',    label: 'Accounts',    icon: 'pi-book', block: !hasPermission("manage_data") },
    { name: 'settings.categories',  label: 'Categories',  icon: 'pi-box', block: !hasPermission("manage_data") },
    { name: '', label: 'Roles', separator: true, block: !hasPermission("manage_roles") },
    { name: 'settings.roles',    label: 'Roles',    icon: 'pi-unlock', block: !hasPermission("manage_roles") },
];

const visibleItems = computed(() =>
    items.filter(item => !item.block)
);

const pageTitle = computed(() => {
    if (!route.name) return 'Settings';
    const parts = String(route.name).split('.');
    const last = parts[parts.length - 1];
    return vueHelper.capitalize(last);
});

const settingsMenuRef = ref<any>(null);

const isActive = (name: SettingsMenuItem['name']) => route.name === name;

function goBack() {
    const hasBack = !!(router.options.history.state && router.options.history.state.back);
    if (hasBack) router.back();
    else router.push({ name: 'dashboard' });
}

function toggleOverlay(event: any) {
    if(window.innerWidth > 768) return;
    settingsMenuRef.value.toggle(event);
}

</script>

<template>
    <div class="settings flex p-2 w-full">
        <aside class="no-mobile text-white h-full flex flex-column gap-2 p-3 w-12rem">

            <div class="flex flex-row gap-2 p-2 mb-2 align-items-center cursor-pointer font-bold hoverable"
                 style="color: var(--text-primary)">
                <i class="pi pi-angle-left"></i>
                <span @click="goBack">Back</span>
            </div>

            <h6 class="text-xs font-bold uppercase mb-2" style="color: var(--text-primary);">General</h6>

            <template v-for="item in visibleItems" :key="item.name ?? item.label">

                <h6 v-if="item.separator"
                    class="text-xs font-bold uppercase mb-2 mt-3"
                    style="color: var(--text-primary);">
                    {{ item.label }}
                </h6>

                <RouterLink v-else :to="{ name: item.name }"
                        class="flex align-items-center text-center gap-2 p-2 cursor-pointer"
                        :class="{ active: isActive(item.name!) }"
                        style="text-decoration: none; transition: all 0.2s ease; color: var(--text-primary);">

                    <i class="pi text-sm" :class="item.icon" style="color: var(--text-secondary)"></i>
                    <span class="no-mobile">{{ item.label }}</span>
                </RouterLink>
            </template>
        </aside>

        <main class="w-full flex-1 pt-3" style="max-width: 850px; margin: 0 auto;">
            <div class="flex flex-row gap-2 mb-2 align-items-center text-center">
                <i class="pi pi-ellipsis-v mobile-only text-xs" @click="toggleOverlay" style="cursor:pointer;" />
                <span @click="toggleOverlay" class="text-sm" style="color: var(--text-secondary)">Home</span>
                <i class="pi pi-angle-right" />
                <span style="color: var(--text-primary)">{{ pageTitle }}</span>
            </div>

            <router-view />
        </main>

        <Popover ref="settingsMenuRef" class="rounded-popover" :style="{width: '200px'}" :breakpoints="{'226px': '90vw'}">

            <div class="flex flex-row gap-2 p-2 mb-2 align-items-center cursor-pointer font-bold hoverable"
                 style="color: var(--text-primary)">
                <i class="pi pi-angle-left"></i>
                <span @click="goBack">Back</span>
            </div>

            <h6 class="text-xs font-bold uppercase mb-2" style="color: var(--text-primary);">General</h6>

            <template v-for="item in visibleItems" :key="item.name ?? item.label">

                <h6 v-if="item.separator"
                    class="text-xs font-bold uppercase mb-2 mt-3"
                    style="color: var(--text-primary);">
                    {{ item.label }}
                </h6>

                <RouterLink v-else :to="{ name: item.name }"
                            class="flex align-items-center text-center gap-2 p-2 cursor-pointer"
                            :class="{ active: isActive(item.name!) }"
                            style="text-decoration: none; transition: all 0.2s ease; color: var(--text-primary);"
                            @click="toggleOverlay">

                    <i class="pi text-sm" :class="item.icon" style="color: var(--text-secondary)"></i>
                    <span>{{ item.label }}</span>
                </RouterLink>
            </template>
        </Popover>
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

.mobile-only { display: none; }

@media (max-width: 875px) {
    .mobile-only { display: inline-block; }
    .settings { padding: 0 1rem 0 1rem !important; }
    .no-mobile { display: none !important; }
}
</style>
