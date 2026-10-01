<script setup lang="ts">
import { ref } from "vue";
import AuthSkeleton from "../../components/layout/AuthSkeleton.vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useAuthStore } from "../../../services/stores/auth_store.ts";
import { useRouter } from "vue-router";

const authStore = useAuthStore();
const toastStore = useToastStore();

const router = useRouter();
const loading = ref(false);
const emailInput = ref(null);

async function requestPasswordReset() {
  loading.value = true;
  try {
    const response = await authStore.requestPasswordReset(emailInput.value!);
    toastStore.successResponseToast(response);
    await router.push("/login");
  } catch (error) {
    toastStore.errorResponseToast(error);
    loading.value = false;
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <AuthSkeleton>
    <div class="w-full mx-auto" style="max-width: 400px">
      <div class="mb-8">
        <h2
          class="m-0 text-3xl font-medium"
          style="color: var(--text-primary); letter-spacing: -0.025em"
        >
          Reset password
        </h2>
        <p
          class="mt-2 text-base leading-normal"
          style="color: var(--text-secondary)"
        >
          Request a password reset for your account.
        </p>
      </div>

      <form class="flex flex-col gap-4" @submit.prevent="requestPasswordReset">
        <div class="flex flex-row w-full">
          <div class="flex flex-col gap-1 w-full">
            <label>Email</label>
            <InputText
              id="email"
              v-model="emailInput"
              type="email"
              :disabled="loading"
              class="w-full rounded-xl"
            />
          </div>
        </div>

        <Button
          label="Request password reset"
          class="w-full main-button"
          :disabled="loading"
          type="submit"
        />
      </form>

      <div
        class="flex items-center justify-center gap-2 mt-6 pt-4"
        style="border-top: 1px solid var(--border-color)"
      >
        <span class="text-sm" style="color: var(--text-secondary)">
          Sign in with a different account?
        </span>
        <router-link
          :to="{ name: 'login' }"
          class="text-sm text-ink font-medium no-underline hover:opacity-80"
        >
          Log in
        </router-link>
      </div>
    </div>
  </AuthSkeleton>
</template>
