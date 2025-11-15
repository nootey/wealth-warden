<script setup lang="ts">
import {computed, onMounted, ref} from 'vue';
import { useAuthStore } from './services/stores/auth_store.ts';
import { useThemeStore } from './services/stores/theme_store.ts';
import AppNavBar from "./AppNavBar.vue";
import {storeToRefs} from "pinia";
import {useRoute} from 'vue-router';
import AppSideBar from "./AppSideBar.vue";
import vueHelper from "./utils/vue_helper.ts";
import router from "./services/router/main.ts";

const authStore = useAuthStore();
const themeStore = useThemeStore();
const route = useRoute();

themeStore.initializeTheme();

const {isAuthenticated, isInitialized} = storeToRefs(authStore);

const requiresAuthView = computed<boolean>(() => route.matched.some(r => r.meta.requiresAuth));
const isGuestOnlyView = computed<boolean>(() => route.matched.some(r => r.meta.guestOnly));

const sidebarRef = ref<InstanceType<typeof AppSideBar> | null>(null);

onMounted(async () => {
  if (isAuthenticated.value) {
    await authStore.init();
  }
});

const pageTitle = computed<string[]>(() => {
    const path = route.path ?? '';
    const name = typeof route.name === 'string' ? route.name : '';

    const raw = path || name;
    if (!raw) return ['Home'];

    const delimiter = raw.startsWith('/') ? '/' : '.';
    const parts = raw
        .split(delimiter)
        .filter(Boolean)
        .map(p => vueHelper.capitalize(p.replace(/[-_]/g, ' ')));

    if (path === '/' || parts.length === 0) {
        return ['Home', 'Dashboard'];
    }

    return ['Home', ...parts];
});

const goHome = () => router.push('/');

const isSettingsView = computed(() => route.path.startsWith('/settings'));

</script>

<template>
    <Toast position="top-center" group="bc" />
    <Toast position="bottom-right" group="br" />
    <ConfirmDialog unstyled>
        <template #container="{ message, acceptCallback, rejectCallback }">
            <div class="flex justify-content-center align-items-center p-overlay-mask p-overlay-mask-enter">
                <div class="flex flex-column p-5 gap-4 border-round-lg"
                     style="background-color: var(--background-secondary);">
                    <div class="font-bold text-xl" style="color: var(--text-primary);">{{ message.header }}</div>
                    <div style="color: var(--text-primary);" v-html="message.message.replace(/\n/g, '<br>')"></div>
                    <div class="flex justify-content-end gap-2"  >
                        <Button class="p-2 border-round-lg" :label="message.rejectProps?.label || 'Cancel'" variant="outlined" style="color: var(--text-primary); border-color: var(--text-primary)" @click="rejectCallback" />
                        <Button class="p-2 border-round-lg" :label="message.acceptProps?.label || 'Confirm'" :severity="message.acceptProps?.severity" style="color: var(--text-primary);" @click="acceptCallback" />
                    </div>
                </div>
            </div>
        </template>
    </ConfirmDialog>

    <div id="app">
        <AppNavBar v-if="isAuthenticated && isInitialized && !isGuestOnlyView" />

        <div class="flex-1 app-content" :style="{ 'margin-left': (isAuthenticated && isInitialized && !isGuestOnlyView) ? '80px' : '0px' }">
            <div v-if="requiresAuthView && !isInitialized" class="w-full h-full flex items-center justify-center">
                <i class="pi pi-spin pi-spinner text-2xl"></i>
            </div>
            <div v-else>
                <div v-if="!isSettingsView && isAuthenticated && isInitialized" id="breadcrumb" class="flex flex-row gap-2 mb-2 align-items-center justify-content-between"
                     style="max-width: 1000px; margin: 0 auto;padding: 1rem 0.5rem 0 0;">

                    <div class="flex gap-1 text-center align-items-center">
                        <i class="pi pi-ellipsis-v mobile-only text-xs hover-icon" />
                        <template v-for="(part, index) in pageTitle" :key="index">
                        <span class="text-sm"
                              :style="{
                        color: index === pageTitle.length - 1 ? 'var(--text-primary)' : 'var(--text-secondary)',
                        cursor: part === 'Home' ? 'pointer' : 'default'}"
                              @click="part === 'Home' && goHome()">
                      {{ part }}
                    </span>
                            <i v-if="index < pageTitle.length - 1" class="pi pi-angle-right" />
                        </template>
                    </div>

                    <i class="pi pi-book hover-icon" style="margin-left: 0;"
                       @click="sidebarRef?.toggle && sidebarRef.toggle()"/>
                </div>
                <router-view />
            </div>
        </div>

        <AppSideBar ref="sidebarRef" v-if="isAuthenticated && isInitialized && !isGuestOnlyView"/>
    </div>
</template>

<style scoped lang="scss">

main {
  @media (max-width: 768px) {
    padding-left: 0;
  }
}

@media (max-width: 768px) {
  .app-content {
    margin-left: 0 !important;
    padding-bottom: 72px;
  }
  .mobile-only { display: inline-block; }
  .settings { padding: 0 1rem 0 1rem !important; }
  .no-mobile { display: none !important; }
  #breadcrumb {
     padding: 1rem 0.7rem 0 0.7rem !important;
  }
}
@media (max-width: 1111px) {
  #breadcrumb {
    padding: 1rem 0.5rem 0 0.5rem !important;
  }
}
.mobile-only { display: none; }

</style>
