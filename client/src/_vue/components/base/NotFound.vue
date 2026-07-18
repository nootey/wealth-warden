<script setup lang="ts">
import { useRouter } from "vue-router";
import { useAuthStore } from "../../../services/stores/auth_store.ts";
import { storeToRefs } from "pinia";

const router = useRouter();
const authStore = useAuthStore();

const { isAuthenticated } = storeToRefs(authStore);

function handlePrimary() {
  router.push({ name: isAuthenticated.value ? "dashboard" : "login" });
}
</script>

<template>
  <div
    class="min-h-screen w-full flex flex-col items-center justify-center text-center p-12"
  >
    <div class="flex flex-col gap-4" role="region" aria-label="Page not found">
      <h1 ref="headingRef" tabindex="-1" class="text-3xl font-bold">
        404 Page not found
      </h1>
      <p style="color: var(--text-secondary)">
        The page you're looking for doesn't exist or was moved.
      </p>
      <div class="flex flex-row gap-4 justify-center">
        <Button
          :label="isAuthenticated ? 'Go Home' : 'Login'"
          class="main-button w-4/12"
          @click="handlePrimary"
        />
      </div>
    </div>
  </div>
</template>
