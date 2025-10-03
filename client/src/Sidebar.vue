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
  {to: "/accounts", icon: "pi-hashtag", text: "Acc"},
  {to: "/transactions", icon: "pi-credit-card", text: "Txn"},
  {to: "/charts", icon: "pi-chart-bar", text: "Charts"},
  {to: "/users", icon: "pi-users", text: "Users", block: !hasPermission('manage_users')},
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

function checkAccess(route: string){
    switch(route){
        case 'logs':
            if(hasPermission('view_activity_logs')) {
                router.push('/logs');
            }
            break;
    }
}

</script>

<template>
  <aside v-if="authStore.authenticated && authStore.isValidated"
         class="flex flex-column overflow-hidden h-screen fixed left-0 top-0"
         style="width: 80px; color: var(--text-primary); padding: 1rem 0;">

      <div class="logo-section flex align-items-center justify-content-center mb-2 p-1">
          <div class="flex align-items-center justify-content-center"
           style="
            width: 32px; height: 32px; border-radius: 8px;box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
            background: linear-gradient(135deg, var(--accent-primary) 0%, var(--accent-secondary) 100%);">
        <i class="pi pi-wallet" style="color: white;" />
      </div>
      </div>

      <div class="menu flex flex-column h-full">
          <router-link v-for="(item, index) in visibleMenuItems" :key="index" :to="item.to"
                       class="flex flex-column align-items-center p-1"
                       style="text-decoration: none; transition: all 0.2s ease; color: var(--text-secondary);">
              <i :class="['pi', item.icon]" class="text-sm" style="transition: all 0.2s ease;" />
              <span class="text-xs text-align-center">{{ item.text }}</span>
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

            <div id="profileMenuItem" v-if="hasPermission('view_activity_logs')"
               class="flex align-items-center gap-2 p-1 border-round-md"
               style="cursor: pointer; transition: all 0.2s ease; color: var(--text-primary);"
               @click="checkAccess('logs')">
            <i class="pi pi-address-book"></i>
            <span class="text-sm">Activity logs</span>
          </div>

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
}

aside .menu a:hover,
aside .menu div:hover {
  background-color: var(--background-primary) !important;
  transform: translateY(-1px);
}

.router-link-exact-active {
  font-weight: bold;
  color: var(--text-primary) !important;
  position: relative;
}

.router-link-exact-active::before {
  content: '';
  position: absolute;
  left: 1px;
  top: 50%;
  transform: translateY(-50%);
  width: 4px;
  height: 16px;
  border-radius: 25px;
  background-color: var(--text-secondary);
}

#user-menu-trigger {
  padding: 0.75rem 0.5rem !important;
}

#profileMenuItem:hover {
  background-color: var(--background-primary);
}

@media (max-width: 768px) {
  aside {
    contain: layout;
    backface-visibility: hidden;
    transform: translateZ(0);
    position: fixed !important;
    bottom: 0 !important;
    left: 0 !important;
    right: 0 !important;
    top: auto !important;
    width: 100% !important;
    height: 66px !important;
    min-height: 66px !important;
    padding: 0.75rem 1rem !important;
    background-color: var(--background-primary) !important;
    border-top: 1px solid var(--border-color);
    z-index: 1000;
  }

  .router-link-exact-active::before {
    left: 50%;
    top: 90%;
    transform: translateX(-50%);
    width: 16px;
    height: 4px;
  }

  .logo-section {
    display: none !important;
  }

  aside .menu {
    flex-direction: row !important;
    justify-content: center !important;
    align-items: center;
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