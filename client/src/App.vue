<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useAuthStore } from "./services/stores/auth_store.ts";
import { useThemeStore } from "./services/stores/theme_store.ts";
import AppNavBar from "./AppNavBar.vue";
import { storeToRefs } from "pinia";
import { useRoute } from "vue-router";
import AppSideBar from "./AppSideBar.vue";
import vueHelper from "./utils/vue_helper.ts";
import router from "./services/router/main.ts";
import AppFooter from "./AppFooter.vue";
import AppNotesBar from "./AppNotesBar.vue";

const authStore = useAuthStore();
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

const sidebarRef = ref<InstanceType<typeof AppSideBar> | null>(null);
const notesRef = ref<InstanceType<typeof AppNotesBar> | null>(null);

onMounted(async () => {
  if (isAuthenticated.value) {
    await authStore.init();
  }
});

const pageTitle = computed<string[]>(() => {
  const path = route.path ?? "";
  const name = typeof route.name === "string" ? route.name : "";

  const raw = path || name;
  if (!raw) return ["Home"];

  const delimiter = raw.startsWith("/") ? "/" : ".";
  const parts = raw
    .split(delimiter)
    .filter(Boolean)
    .map((p) => vueHelper.capitalize(p.replace(/[-_]/g, " ")));

  if (path === "/" || parts.length === 0) {
    return ["Home", "Dashboard"];
  }

  return ["Home", ...parts];
});

const goHome = () => router.push("/");

const isSettingsView = computed(() => route.path.startsWith("/settings"));
</script>

<template>
  <Toast position="top-center" group="bc" />
  <Toast position="bottom-right" group="br" />
  <ConfirmDialog unstyled>
    <template #container="{ message, acceptCallback, rejectCallback }">
      <div
        class="flex justify-content-center align-items-center p-overlay-mask p-overlay-mask-enter"
      >
        <div
          class="flex flex-column p-5 gap-4 border-round-lg"
          style="background-color: var(--background-secondary)"
        >
          <div class="font-bold text-xl" style="color: var(--text-primary)">
            {{ message.header }}
          </div>
          <div
            style="color: var(--text-primary)"
            v-html="message.message.replace(/\n/g, '<br>')"
          />
          <div class="flex justify-content-end gap-2">
            <Button
              class="p-2 border-round-lg"
              :label="message.rejectProps?.label || 'Cancel'"
              variant="outlined"
              style="
                color: var(--text-primary);
                border-color: var(--text-primary);
              "
              @click="rejectCallback"
            />
            <Button
              class="p-2 border-round-lg"
              :label="message.acceptProps?.label || 'Confirm'"
              :severity="message.acceptProps?.severity"
              style="color: var(--text-primary)"
              @click="acceptCallback"
            />
          </div>
        </div>
      </div>
    </template>
  </ConfirmDialog>

  <div id="app">
    <AppNavBar v-if="isAuthenticated && isInitialized && !hideNavigation" />

    <div
      class="flex-1 app-content"
      :style="{
        'margin-left':
          isAuthenticated && isInitialized && !hideNavigation ? '80px' : '0px',
      }"
    >
      <div
        v-if="requiresAuthView && !isInitialized"
        class="w-full h-full flex items-center justify-center"
      >
        <i class="pi pi-spin pi-spinner text-2xl" />
      </div>
      <div v-else>
        <div
          v-if="
            !isSettingsView &&
            isAuthenticated &&
            isInitialized &&
            !hideNavigation
          "
          id="breadcrumb"
          class="flex flex-row gap-2 mb-2 align-items-center justify-content-between"
          style="max-width: 1000px; margin: 0 auto; padding: 1rem 0.5rem 0 0"
        >
          <div id="crumbs" class="flex gap-1 text-center align-items-center">
            <i class="pi pi-ellipsis-v mobile-only text-xs hover-icon" />
            <template v-for="(part, index) in pageTitle" :key="index">
              <span
                class="text-sm"
                :style="{
                  color:
                    index === pageTitle.length - 1
                      ? 'var(--text-primary)'
                      : 'var(--text-secondary)',
                  cursor: part === 'Home' ? 'pointer' : 'default',
                }"
                @click="part === 'Home' && goHome()"
              >
                {{ part }}
              </span>
              <i
                v-if="index < pageTitle.length - 1"
                class="pi pi-angle-right"
              />
            </template>
          </div>

          <div id="sidebar-icon" class="flex flex-row gap-3">
            <i
              class="pi pi-mobile hover-icon"
              style="margin-left: 0"
              @click="notesRef?.toggle && notesRef.toggle()"
            />

            <i
              class="pi pi-book hover-icon"
              style="margin-left: 0"
              @click="sidebarRef?.toggle && sidebarRef.toggle()"
            />
          </div>
        </div>
        <router-view />
      </div>
    </div>

    <AppFooter />

    <AppSideBar
      v-if="isAuthenticated && isInitialized && !hideNavigation"
      ref="sidebarRef"
    />

    <AppNotesBar
      v-if="isAuthenticated && isInitialized && !hideNavigation"
      ref="notesRef"
    />
  </div>
</template>

<style scoped lang="scss">
main {
  @media (max-width: 768px) {
    padding-left: 0;
  }
}

#app {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

@media (max-width: 768px) {
  .app-content {
    margin-left: 0 !important;
    padding-bottom: 0;
  }
  .mobile-only {
    display: inline-block;
  }
  .settings {
    padding: 0 1rem 0 1rem !important;
  }
  .no-mobile {
    display: none !important;
  }
  #breadcrumb {
    padding: 1rem 0.7rem 0 0.7rem !important;
  }
  #crumbs {
    margin-left: 0.5rem;
  }
  #sidebar-icon {
    margin-right: 0.5rem;
  }
}
@media (max-width: 1111px) {
  #breadcrumb {
    padding: 1rem 0.5rem 0 0.5rem !important;
  }
}
.mobile-only {
  display: none;
}
</style>
