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
    <Toast position="bottom-right"/>
      <ConfirmDialog unstyled>
          <template #container="{ message, acceptCallback, rejectCallback }">
              <div class="flex justify-content-center align-items-center p-overlay-mask p-overlay-mask-enter">
                  <div class="flex flex-column p-5 gap-4 min-w-30rem border-round-lg" style="background-color: var(--background-secondary)">
                      <span class="font-bold text-xl">{{ message.header }}</span>
                      <span>{{ message.message }}</span>
                      <div class="flex justify-content-end gap-2"  >
                          <Button class="p-2 border-round-lg" label="Cancel" variant="outlined" style="color: var(--text-primary); border-color: var(--text-primary)" @click="rejectCallback" />
                          <Button class="p-2 border-round-lg" label="Delete" severity="danger" style="color: var(--text-primary);" @click="acceptCallback" />
                      </div>
                  </div>
              </div>
          </template>
      </ConfirmDialog>
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
