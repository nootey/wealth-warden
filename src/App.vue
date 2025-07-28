<script setup lang="ts">
import {computed, onMounted} from 'vue';
import { useAuthStore } from './services/stores/auth_store.ts';
import { useThemeStore } from './services/stores/theme_store.ts';
import Sidebar from "./Sidebar.vue";

const authStore = useAuthStore();
const themeStore = useThemeStore();

themeStore.initializeTheme();

const authenticated = computed(() => authStore.isAuthenticated);
const initialized = computed(() => authStore.isInitialized);

onMounted(async () => {
  if (authenticated.value) {
    await authStore.init();
  }
});

</script>

<template>
  <div id="app" class="app">
    <Toast position="top-right"/>
    <ConfirmPopup></ConfirmPopup>
    <Sidebar v-if="authenticated && initialized" />
    <router-view/>
  </div>
</template>

<style scoped lang="scss">

.app {
  display: flex;
  main {
    flex: 1 1 0;
    padding: 2rem;

    @media (max-width: 768px) {
      padding-left: 6rem;
    }
  }
}
</style>
