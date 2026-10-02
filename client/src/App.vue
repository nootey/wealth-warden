<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useAuthStore } from "./services/stores/auth_store.ts";
import { useThemeStore } from "./services/stores/theme_store.ts";
import AppNavBar from "./AppNavBar.vue";
import { storeToRefs } from "pinia";
import { useRoute } from "vue-router";
import AppSideBar from "./_vue/features/AppSideBar.vue";
import AccountSideBar from "./AccountSideBar.vue";
import NotificationSideBar from "./_vue/features/NotificationSideBar.vue";
import { useNotificationStore } from "./services/stores/notification_store.ts";
import { NAV_WIDTH } from "./models/shared_models.ts";

const authStore = useAuthStore();
const notificationStore = useNotificationStore();
const themeStore = useThemeStore();
const route = useRoute();

themeStore.initializeTheme();

const { isAuthenticated, isInitialized } = storeToRefs(authStore);

const requiresAuthView = computed<boolean>(() =>
  route.matched.some((r) => r.meta.requiresAuth),
);
const hideNavigation = computed<boolean>(() =>
  route.matched.some((r) => r.meta.hideNavigation),
);
const showShell = computed(
  () => isAuthenticated.value && isInitialized.value && !hideNavigation.value,
);
const navWidth = computed(() =>
  navCollapsed.value ? NAV_WIDTH.collapsed : NAV_WIDTH.expanded,
);

const appSidebarRef = ref<InstanceType<typeof AppSideBar> | null>(null);
const accSidebarRef = ref<InstanceType<typeof AccountSideBar> | null>(null);
const notifSidebarRef = ref<InstanceType<typeof NotificationSideBar> | null>(
  null,
);

const NAV_KEY = "ww_nav_collapsed";
function readCollapsed(): boolean {
  try {
    return localStorage.getItem(NAV_KEY) === "true";
  } catch {
    return false;
  }
}
const navCollapsed = ref(readCollapsed());
watch(navCollapsed, (v) => {
  try {
    localStorage.setItem(NAV_KEY, String(v));
  } catch {
    // Storage can be unavailable; the toggle still works for this session.
  }
});

onMounted(async () => {
  if (isAuthenticated.value) {
    await authStore.init();
  }
});
</script>

<template>
  <Toast position="top-center" group="bc" />
  <Toast position="bottom-right" group="br" />
  <ConfirmDialog unstyled>
    <template #container="{ message, acceptCallback, rejectCallback }">
      <div
        class="flex justify-center items-center p-overlay-mask p-overlay-mask-enter"
      >
        <div
          class="flex flex-col p-6 gap-4 rounded-2xl border border-line bg-card shadow-2xl w-[min(420px,calc(100vw-2rem))]"
        >
          <div class="text-xl font-medium tracking-tight text-ink">
            {{ message.header }}
          </div>
          <div
            class="text-sm leading-relaxed text-muted"
            v-html="message.message.replace(/\n/g, '<br>')"
          />
          <div class="flex justify-end gap-2 pt-2">
            <Button
              class="outline-button"
              :label="message.rejectProps?.label || 'Cancel'"
              @click="rejectCallback"
            />
            <Button
              :class="
                message.acceptProps?.severity === 'danger'
                  ? 'delete-button'
                  : 'main-button'
              "
              :label="message.acceptProps?.label || 'Confirm'"
              @click="acceptCallback"
            />
          </div>
        </div>
      </div>
    </template>
  </ConfirmDialog>

  <div id="app">
    <AppNavBar
      v-if="showShell"
      v-model:collapsed="navCollapsed"
      :has-unread="notificationStore.hasUnread"
      @open-accounts="accSidebarRef?.toggle()"
      @open-stats="appSidebarRef?.toggle()"
      @open-notifications="notifSidebarRef?.toggle()"
    />

    <AccountSideBar v-if="showShell" ref="accSidebarRef" />

    <div
      :id="showShell ? 'app-content' : 'app-bare'"
      class="flex-1 min-w-0 transition-[padding] duration-200 ease-out"
      :style="{
        paddingLeft: showShell ? navWidth : '0px',
      }"
    >
      <div
        v-if="requiresAuthView && !isInitialized"
        class="w-full min-h-screen flex items-center justify-center"
      >
        <i class="pi pi-spin pi-spinner text-2xl text-muted" />
      </div>
      <div v-else :class="showShell ? 'px-4 pt-8 pb-4' : ''">
        <router-view />
      </div>
    </div>

    <AppSideBar v-if="showShell" ref="appSidebarRef" />

    <NotificationSideBar v-if="showShell" ref="notifSidebarRef" />
  </div>
</template>

<style scoped lang="scss">
#app {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

@media (max-width: 768px) {
  #app-content {
    padding-left: 0 !important;
    padding-top: 2.5rem;
    padding-bottom: 4.5rem;
  }
}
</style>
