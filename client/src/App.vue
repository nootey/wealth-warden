<script setup lang="ts">
import {computed, onMounted} from 'vue';
import { useAuthStore } from './services/stores/auth_store.ts';
import { useThemeStore } from './services/stores/theme_store.ts';
import Sidebar from "./Sidebar.vue";
import {storeToRefs} from "pinia";
import {useRoute} from 'vue-router';

const authStore = useAuthStore();
const themeStore = useThemeStore();
const route = useRoute();

themeStore.initializeTheme();

const {isAuthenticated, isInitialized} = storeToRefs(authStore);

const requiresAuthView = computed<boolean>(() => route.matched.some(r => r.meta.requiresAuth));
const isGuestOnlyView = computed<boolean>(() => route.matched.some(r => r.meta.guestOnly));

onMounted(async () => {
  if (isAuthenticated.value) {
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
                          <div class="flex flex-column p-5 gap-4 border-round-lg"
                               style="background-color: var(--background-secondary);">
                              <span class="font-bold text-xl" style="color: var(--text-primary);">{{ message.header }}</span>
                              <span style="color: var(--text-primary);">{{ message.message }}</span>
                              <div class="flex justify-content-end gap-2"  >
                                  <Button class="p-2 border-round-lg" :label="message.rejectProps?.label || 'Cancel'" variant="outlined" style="color: var(--text-primary); border-color: var(--text-primary)" @click="rejectCallback" />
                                  <Button class="p-2 border-round-lg" :label="message.acceptProps?.label || 'Confirm'" :severity="message.acceptProps?.severity" style="color: var(--text-primary);" @click="acceptCallback" />
                              </div>
                          </div>
                      </div>
                  </template>
              </ConfirmDialog>
        <Sidebar v-if="isAuthenticated && isInitialized && !isGuestOnlyView" />
        <div class="flex-1 app-content" :style="{ 'margin-left': (isAuthenticated && isInitialized && !isGuestOnlyView) ? '80px' : '0px' }">
            <div v-if="requiresAuthView && !isInitialized" class="w-full h-full flex items-center justify-center">
                <i class="pi pi-spin pi-spinner text-2xl"></i>
            </div>
            <router-view v-else/>
        </div>
    </div>
</template>

<style scoped lang="scss">

.app {
  display: flex;

  main {
    flex: 1 1 0;
    padding: 2rem;

    @media (max-width: 768px) {
      padding-left: 0;
    }
  }
}

@media (max-width: 768px) {
  .app-content {
    margin-left: 0 !important;
    padding-bottom: 72px;
  }
}
</style>
