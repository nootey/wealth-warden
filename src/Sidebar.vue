<script setup lang="ts">
import {ref} from "vue";
import { useThemeStore } from './services/stores/theme';

const sidebarExpanded = ref(false);
const darkModeActive = ref(false);
const themeStore = useThemeStore();

const toggleMenu = () => {
    sidebarExpanded.value = !sidebarExpanded.value;
}

</script>

<template>
<!--  <img src="./assets/images/temp_logo.png" alt="Temp" />-->
  <aside :class="`${sidebarExpanded ? 'sidebar-expanded' : ''}`">
    <div class="logo">

      <div class="menu-toggle-wrap">
        <div class="menu-toggle" >
          <i class="pi pi-angle-double-right sidebar-icon" style="font-size: 1.5rem" @click="toggleMenu"></i>
        </div>
      </div>
    </div>

    <h3>Menu</h3>
    <div class="menu">
      <router-link to="/" class="sidebar-item" v-tooltip="'Dashboard'">
        <i class="pi pi-home sidebar-icon"></i>
        <span class="text">Dashboard</span>
      </router-link>
      <router-link to="/inflows" class="sidebar-item" v-tooltip="'Inflows'">
        <i class="pi pi-cart-plus sidebar-icon"></i>
        <span class="text">Inflows</span>
      </router-link>
    </div>

    <div class="flex"></div>

    <div class="menu">
      <div class="sidebar-item">
        <i class="pi pi-cog sidebar-icon"></i>
        <span class="text">Settings</span>
      </div>
      <div class="sidebar-item" @click="themeStore.toggleDarkMode()">
        <i class="sidebar-icon pi" :class="darkModeActive ?  'pi-sun' : 'pi-moon'"></i>
        <span class="text">Theme</span>
      </div>

    </div>

  </aside>
</template>

<style scoped lang="scss">
  aside {
    display: flex;
    flex-direction: column;
    width: calc(2rem + 32px);
    min-height: 100vh;
    overflow: hidden;
    padding: 1rem;

    background-color: var(--background-secondary);
    color: var(--text-primary);

    transition: 0.2s ease-out;

    .flex {
      flex: 1 1 0;
    }

    .logo {
      margin-bottom: 1rem;
      img {
        width: 2rem;
      }
    }

    .menu-toggle-wrap {
      display: flex;
      justify-content: flex-start;
      margin-bottom: 1rem;
      top: 0;
      position: relative;
      transition: 0.2s ease-out;
      padding-bottom: 0.5rem;

      .menu-toggle {
        transition: 0.2s ease-out;
        .sidebar-icon {
          font-size: 2rem;
          color: var(--text-primary);
          transition: 0.2s ease-out;
        }

        &:hover {
          .sidebar-icon {
            color: var(--accent-primary);
            transform: translate(0.5rem);
          }
        }
      }
    }

    h3, .sidebar-item .text {
      opacity: 0;
      transition: 0.3s ease-out;
    }

    .menu {
      margin: 0 -1rem;

      .sidebar-item {
        display: flex;
        align-items: center;
        text-decoration: none;

        padding: 0.5rem 1rem;
        transition: 0.2s ease-out;

        .sidebar-icon {
          font-size: 1.35rem;
          color: var(--text-primary);
          margin-right: 1rem;
          transition: 0.2s ease-out;
        }

        .text {
          color: var(--text-primary);
          transition: 0.2s ease-out;
        }

        &:hover, &.router-link-exact-active {
          background-color: var(--background-primary);

          .sidebar-icon, .text {
            color: var(--text-primary);
          }
        }

        &.router-link-exact-active {
          border-right: 5px solid var(--accent-primary)
        }
      }
    }

    &.sidebar-expanded {
      width: var(--sidebar-width);

      .menu-toggle-wrap {
        //top: -3rem;
        .menu-toggle {
          transform: rotate(-180deg);
        }
      }

      h3, .sidebar-item .text {
        opacity: 1;
      }

      h3 {
        color: grey;
        font-size: 0.875rem;
        margin-bottom: 0.5rem;
        text-transform: uppercase;
      }

      .sidebar-item {
        .sidebar-icon {
          margin-right: 1rem;
        }
      }

    }

    @media (max-width: 768px) {
      position: fixed;
      //z-index: 99;
    }
  }
</style>