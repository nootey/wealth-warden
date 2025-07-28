<script setup lang="ts">
import { useThemeStore } from './services/stores/theme_store.ts';
import { useAuthStore } from './services/stores/auth_store.ts';

const themeStore = useThemeStore();
const authStore = useAuthStore();

interface MenuItem {
  to: string;
  icon: string;
  text: string;
}

const menuItems: MenuItem[] = [
  { to: "/", icon: "pi-home", text: "Home"},
  { to: "/logs", icon: "pi-address-book", text: "Logging"},
];

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
    color: var(--text-primary);
  ">
    <!-- Logo Section - Hidden on mobile -->
    <div class="logo-section" style="
      display: flex;
      align-items: center;
      justify-content: center;
      margin-bottom: 2rem;
      padding: 0.5rem;
    ">
      <div style="
        width: 32px;
        height: 32px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        border-radius: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
      ">
        <i class="pi pi-wallet" style="
          font-size: 1rem;
          color: white;
        "></i>
      </div>
    </div>

    <!-- Navigation Menu -->
    <div style="flex: 1;">
      <div class="menu navigation-menu" style="
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
      ">
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
            color: var(--text-primary);
          "
          :style="{
            backgroundColor: $route.path === item.to ? 'var(--background-primary)' : 'transparent',
            color: $route.path === item.to ? 'var(--accent-primary)' : 'var(--text-primary)'
          }"
        >
          <i :class="['pi', item.icon]" style="
            font-size: 1.1rem;
            margin-bottom: 0.25rem;
            transition: all 0.2s ease;">
          </i>
          <span style="
            font-size: 0.65rem;
            font-weight: 500;
            text-align: center;
            line-height: 1.1;
          ">{{ item.text }}</span>
        </router-link>
      </div>
    </div>

    <!-- Bottom Actions -->
    <div class="menu bottom-menu" style="
      display: flex;
      flex-direction: column;
      gap: 0.75rem;
      margin-top: auto;
    ">
      <!-- Theme Button -->
      <div style="
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 0.5rem 0.25rem;
        border-radius: 12px;
        transition: all 0.2s ease;
        cursor: pointer;
        color: var(--text-primary);
      " @click="themeStore.toggleDarkMode()">
        <i class="pi" :class="themeStore.darkModeActive ? 'pi-sun' : 'pi-moon'" style="
          font-size: 1.1rem;
          margin-bottom: 0.25rem;
          transition: all 0.2s ease;
        "></i>
        <span style="
          font-size: 0.65rem;
          font-weight: 500;
          text-align: center;
        ">Theme</span>
      </div>
      
      <!-- Sign Out Button -->
      <div style="
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 0.5rem 0.25rem;
        border-radius: 12px;
        transition: all 0.2s ease;
        cursor: pointer;
        color: var(--text-primary);
      " @click="authStore.logoutUser()">
        <i class="pi pi-sign-out" style="
          font-size: 1.1rem;
          margin-bottom: 0.25rem;
          transition: all 0.2s ease;
        "></i>
        <span style="
          font-size: 0.65rem;
          font-weight: 500;
          text-align: center;
        ">Sign out</span>
      </div>
    </div>
  </aside>
</template>

<style scoped lang="scss">

/* Hover effects for navigation items */
aside .menu a:hover,
aside .menu div:hover {
  background-color: var(--background-primary) !important;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

/* Active state for router links */
.router-link-exact-active {
  background-color: var(--background-primary) !important;
  color: var(--accent-primary) !important;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

/* Mobile responsive design */
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

  /* Hide only logo on mobile */
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
  }
}
</style>