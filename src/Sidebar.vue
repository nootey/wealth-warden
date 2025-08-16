<script setup lang="ts">
import {useThemeStore} from './services/stores/theme_store.ts';
import {useAuthStore} from './services/stores/auth_store.ts';
import {storeToRefs} from "pinia";
import {ref} from 'vue';

const themeStore = useThemeStore();
const authStore = useAuthStore();
const { user } = storeToRefs(authStore);

interface MenuItem {
  to: string;
  icon: string;
  text: string;
}

const menuItems: MenuItem[] = [
  {to: "/", icon: "pi-home", text: "Home"},
  {to: "/accounts", icon: "pi-hashtag", text: "Accounts"},
  {to: "/transactions", icon: "pi-credit-card", text: "Transactions"},
  {to: "/logs", icon: "pi-address-book", text: "Logging"},
];

const profileMenuRef = ref<any>(null);

function toggleProfileMenu(event: any) {
  if (profileMenuRef.value) {
    profileMenuRef.value.toggle(event);
  }
}

</script>

<template>
  <aside style="
    display: flex;
    flex-direction: column;
    width: 80px;
    min-height: 100vh;
    overflow: hidden;
    padding: 1rem 0.5rem;
    background-color: var(--background-secondary);
    color: var(--text-primary);">
    <div class="logo-section" style="
      display: flex;
      align-items: center;
      justify-content: center;
      margin-bottom: 2rem;
      padding: 0.5rem;">
      <div style="
        width: 32px;
        height: 32px;
        background: linear-gradient(135deg, var(--accent-primary) 0%, var(--accent-secondary) 100%);
        border-radius: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);">
        <i class="pi pi-wallet" style="
          font-size: 1rem;
          color: white;" />
      </div>
    </div>

    <div style="flex: 1;">
      <div class="menu" style="
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
        height: 100%;">
        <router-link
            v-for="(item, index) in menuItems"
            :key="index"
            :to="item.to"
            style="
              display: flex;
              flex-direction: column;
              align-items: center;
              text-decoration: none;
              padding: 0.5rem 0.25rem;
              border-radius: 12px;
              transition: all 0.2s ease;
              color: var(--text-secondary);"
            :style="{
              backgroundColor: $route.path === item.to ? 'var(--background-primary)' : 'transparent',
              color: $route.path === item.to ? 'var(--text-primary)' : 'var(--text-secondary)'}">
          <i :class="['pi', item.icon]" style="
            font-size: 1.1rem;
            margin-bottom: 0.25rem;
            transition: all 0.2s ease;">
          </i>
          <span style="
            font-size: 0.65rem;
            font-weight: 500;
            text-align: center;
            line-height: 1.1;">
            {{ item.text }}</span>
        </router-link>

        <div
            id="user-menu-trigger"
            style="
              display: flex;
              flex-direction: column;
              align-items: center;
              padding: 0.5rem 0.25rem;
              border-radius: 12px;
              transition: all 0.2s ease;
              cursor: pointer;
              color: var(--text-secondary);
              margin-top: auto;"
            @click="toggleProfileMenu($event)">
          <div style="
            width: 32px;
            height: 32px;
            border-radius: 50%;
            background: linear-gradient(135deg, var(--accent-primary) 0%, var(--accent-secondary) 100%);
            display: flex;
            align-items: center;
            justify-content: center;
            margin-bottom: 0.25rem;
            font-size: 0.75rem;
            font-weight: 600;
            color: white;">
            {{ user?.display_name.split(' ').map(n => n[0]).join('') }}
          </div>
        </div>
      </div>
    </div>

    <Popover ref="profileMenuRef">
      <div style="padding: 1rem;">
        <div style="
          display: flex;
          align-items: center;
          gap: 0.75rem;
          padding-bottom: 1rem;
          border-bottom: 1px solid var(--border-color);
          margin-bottom: 0.75rem;">
          <div style="
            width: 40px;
            height: 40px;
            border-radius: 50%;
            background: linear-gradient(135deg, var(--accent-primary) 0%, var(--accent-secondary) 100%);
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 0.875rem;
            font-weight: 600;
            color: white;">
            {{ user?.display_name.split(' ').map(n => n[0]).join('') }}
          </div>
          <div>
            <div style="
              font-weight: 600;
              color: var(--text-primary);
              margin-bottom: 0.25rem;">{{ user?.display_name }}
            </div>
            <div style="
              font-size: 0.875rem;
              color: var(--text-secondary);">{{ user?.email }}
            </div>
          </div>
        </div>

        <div style="display: flex; flex-direction: column; gap: 0.5rem;">

          <div id="profileMenuItem" style="
            display: flex;
            align-items: center;
            gap: 0.75rem;
            padding: 0.5rem;
            border-radius: 8px;
            cursor: pointer;
            transition: all 0.2s ease;
            color: var(--text-primary);">
            <i class="pi pi-cog" style="font-size: 1rem;"></i>
            <span style="font-size: 0.875rem;">Settings</span>
          </div>

          <div id="profileMenuItem" style="
            display: flex;
            align-items: center;
            gap: 0.75rem;
            padding: 0.5rem;
            border-radius: 8px;
            cursor: pointer;
            transition: all 0.2s ease;
            color: var(--text-primary);" @click="themeStore.toggleDarkMode()">
            <i class="pi" :class="themeStore.darkModeActive ? 'pi-sun' : 'pi-moon'" style="font-size: 1rem;"></i>
            <span style="font-size: 0.875rem;">Theme</span>
          </div>

          <div id="profileMenuItem" style="
            display: flex;
            align-items: center;
            gap: 0.75rem;
            padding: 0.5rem;
            border-radius: 8px;
            cursor: pointer;
            transition: all 0.2s ease;
            color: #ef4444;" @click="authStore.logoutUser()">
            <i class="pi pi-sign-out" style="font-size: 1rem;"></i>
            <span style="font-size: 0.875rem;">Sign out</span>
          </div>

        </div>
      </div>
    </Popover>
  </aside>
</template>

<style scoped lang="scss">

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