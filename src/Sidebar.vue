<script setup lang="ts">
import {ref} from "vue";

const sidebarExpanded = ref(false);

const toggleMenu = () => {
    sidebarExpanded.value = !sidebarExpanded.value;
}

function toggleDarkMode() {
  document.documentElement.classList.toggle('my-app-dark');
}

</script>

<template>
  <aside :class="`${sidebarExpanded ? 'sidebar-expanded' : ''}`">
    <div class="logo">
      <img src="./assets/images/temp_logo.png" alt="Temp" />
      <div class="menu-toggle-wrap">
        <div class="menu-toggle" >
          <i class="pi pi-angle-double-right" style="font-size: 1.5rem" @click="toggleMenu"></i>
        </div>
      </div>
    </div>

    <h3>Menu</h3>
    <div class="menu">
      <router-link to="/" class="button">
        <i class="pi pi-home material-icons"></i>
        <span class="text">Home</span>
      </router-link>
      <router-link to="/about" class="button">
        <i class="pi pi-wallet material-icons"></i>
        <span class="text">About</span>
      </router-link>
    </div>

    <div class="flex"></div>

    <div class="menu">
      <router-link to="/" class="button">
        <i class="pi pi-cog material-icons"></i>
        <span class="text">Settings</span>
      </router-link>
      <i class="pi pi-sun material-icons" @click="toggleDarkMode()" />
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
      justify-content: flex-end;
      margin-bottom: 1rem;

      position: relative;
      top: 0;
      transition: 0.2s ease-out;

      .menu-toggle {
        transition: 0.2s ease-out;
        .material-icons {
          font-size: 2rem;
          color: var(--text-primary);
          transition: 0.2s ease-out;
        }

        &:hover {
          .material-icons {
            color: var(--accent-primary);
            transform: translate(0.5rem);
          }
        }
      }
    }

    h3, .button .text {
      opacity: 0;
      transition: 0.3s ease-out;
    }

    .menu {
      margin: 0 -1rem;

      .button {
        display: flex;
        align-items: center;
        text-decoration: none;

        padding: 0.5rem 1rem;
        transition: 0.2s ease-out;

        .material-icons {
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

          .material-icons, .text {
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
        top: -3rem;
        .menu-toggle {
          transform: rotate(-180deg);
        }
      }

      h3, .button .text {
        opacity: 1;
      }

      h3 {
        color: grey;
        font-size: 0.875rem;
        margin-bottom: 0.5rem;
        text-transform: uppercase;
      }

      .button {
        .material-icons {
          margin-right: 1rem;
        }
      }

    }

    @media (max-width: 768px) {
      position: fixed;
      z-index: 99;
    }
  }
</style>