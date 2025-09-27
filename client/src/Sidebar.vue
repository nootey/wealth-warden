<script setup lang="ts">
import {useThemeStore} from './services/stores/theme_store.ts';
import {useAuthStore} from './services/stores/auth_store.ts';
import {storeToRefs} from "pinia";
import {computed, ref} from 'vue';
import {useRouter} from "vue-router";
import {usePermissions} from "./utils/use_permissions.ts";

const themeStore = useThemeStore();
const authStore = useAuthStore();
const { user } = storeToRefs(authStore);
const { hasPermission } = usePermissions();

const router = useRouter();

interface MenuItem {
  to: string;
  icon: string;
  text: string;
  block?: boolean;
}

const menuItems: MenuItem[] = [
  {to: "/", icon: "pi-home", text: "Home"},
  {to: "/accounts", icon: "pi-hashtag", text: "Accounts"},
  {to: "/transactions", icon: "pi-credit-card", text: "Transactions"},
  {to: "/users", icon: "pi-users", text: "Users", block: !hasPermission('manage_users')},
  {to: "/logs", icon: "pi-address-book", text: "Logging", block: !hasPermission('view_activity_logs')},
];

const visibleMenuItems = computed(() =>
    menuItems.filter(item => !item.block)
);

const profileMenuRef = ref<any>(null);

function toggleProfileMenu(event: any) {
  if (profileMenuRef.value) {
    profileMenuRef.value.toggle(event);
  }
}

</script>

<template>
  <aside v-if="authStore.authenticated && authStore.isValidated"
         class="flex flex-column overflow-hidden h-screen fixed left-0 top-0"
         style="width: 80px; background-color: var(--background-secondary); color: var(--text-primary); padding: 1rem 0;">
    <div class="logo-section flex align-items-center justify-content-center mb-3 p-2">
      <div class="flex align-items-center justify-content-center"
           style="
            width: 32px; height: 32px; border-radius: 8px;box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
            background: linear-gradient(135deg, var(--accent-primary) 0%, var(--accent-secondary) 100%);">
        <i class="pi pi-wallet" style="color: white;" />
      </div>
    </div>
    <div class="flex-1">
      <div class="menu flex flex-column h-full">
        <router-link v-for="(item, index) in visibleMenuItems" :key="index" :to="item.to"
            class="flex flex-column align-items-center p-1 border-round-lg"
            style="text-decoration: none; transition: all 0.2s ease; color: var(--text-secondary);"
            :style="{
              backgroundColor: $route.path === item.to ? 'var(--background-primary)' : 'transparent',
              color: $route.path === item.to ? 'var(--text-primary)' : 'var(--text-secondary)'}">
          <i :class="['pi', item.icon]" class="text-0" style="transition: all 0.2s ease;" />
          <span class="text-xs font-bold text-align-center">
            {{ item.text }}</span>
        </router-link>

        <div id="user-menu-trigger" @click="toggleProfileMenu($event)"
            class="flex flex-column align-items-center p-1 border-round-lg mt-auto"
            style="transition: all 0.2s ease; cursor: pointer; color: var(--text-secondary);">

          <div class="flex align-items-center justify-content-center font-bold text-sm mb-1"
               style="width: 32px; height: 32px; border-radius: 50%; background: linear-gradient(135deg, var(--accent-primary) 0%, var(--accent-secondary) 100%);color: white;">
            {{ user?.display_name.split(' ').map(n => n[0]).join('') }}
          </div>
        </div>
      </div>
    </div>

    <Popover ref="profileMenuRef">
      <div style="padding: 1rem;">
        <div class="flex align-items-center gap-3 pb-1 mb-2" style="border-bottom: 1px solid var(--border-color);">
          <div class="flex align-items-center justify-content-center text-sm font-bold"
               style="
                  width: 40px;
                  height: 40px;
                  border-radius: 50%;
                  background: linear-gradient(135deg, var(--accent-primary) 0%, var(--accent-secondary) 100%);
                  color: white;">
            {{ user?.display_name.split(' ').map(n => n[0]).join('') }}
          </div>
          <div>
            <div class="font-bold mb-1" style="color: var(--text-primary);">{{ user?.display_name }}
            </div>
            <div class="text-sm" style="color: var(--text-secondary);">{{ user?.email }}
            </div>
          </div>
        </div>

        <div class="flex flex-column gap-2 p-1">

          <div id="profileMenuItem" class="flex align-items-center gap-2 p-1 border-round-md"
               style="cursor: pointer; transition: all 0.2s ease; color: var(--text-primary);"
               @click="router.push('/settings')">
            <i class="pi pi-cog"></i>
            <span class="text-sm">Settings</span>
          </div>

          <div id="profileMenuItem" class="flex align-items-center gap-2 p-1 border-round-md"
               style="cursor: pointer; transition: all 0.2s ease;"
               @click="themeStore.toggleDarkMode()">
            <i class="pi" :class="themeStore.darkModeActive ? 'pi-sun' : 'pi-moon'"></i>
            <span class="text-sm">Theme</span>
          </div>

          <div id="profileMenuItem" class="flex align-items-center gap-2 p-1 border-round-md"
               style="cursor: pointer; transition: all 0.2s ease;color: #ef4444;"
               @click="authStore.logoutUser()">
            <i class="pi pi-sign-out"></i>
            <span class="text-sm">Sign out</span>
          </div>

        </div>
      </div>
    </Popover>
  </aside>
</template>

<style scoped lang="scss">

.menu {
  gap: 1rem;
  padding: 0 0.5rem;
}

aside .menu a,
aside .menu div {
  padding: 0.75rem 0.5rem;
  border-radius: 0.5rem;
}

aside .menu a:hover,
aside .menu div:hover {
  background-color: var(--background-primary) !important;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.router-link-exact-active {
  background-color: var(--background-primary) !important;
  color: var(--text-primary) !important;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

#user-menu-trigger {
  padding: 0.75rem 0.5rem !important;
}

#profileMenuItem:hover {
  background-color: var(--background-primary);
}

@media (max-width: 768px) {
  aside {
    position: fixed !important;
    bottom: 0 !important;
    left: 0 !important;
    right: 0 !important;
    top: auto !important;
    width: 100% !important;
    height: auto !important;
    min-height: auto !important;
    padding: 0.75rem 1rem !important;
    background-color: var(--background-secondary) !important;
    border-top: 1px solid var(--border-color);
    z-index: 1000;
  }

  .logo-section {
    display: none !important;
  }

  /* Convert all menus to horizontal layout */
  aside .menu {
    flex-direction: row !important;
    justify-content: space-around !important;
    gap: 0 !important;
  }

  aside .menu a,
  aside .menu div {
    flex: 1 !important;
    max-width: 80px !important;
    padding: 0.5rem 0.25rem !important;
    margin-top: 0 !important;
  }
}
</style>