<script setup lang="ts">
import { ref } from "vue";
import AuthSkeleton from "../../components/layout/AuthSkeleton.vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useAuthStore } from "../../../services/stores/auth_store.ts";

const authStore = useAuthStore();
const toastStore = useToastStore();

const loading = ref(false);

async function resendConfirmationEmail() {
  loading.value = true;
  try {
    const response = await authStore.resendConfirmationEmail(
      authStore.user?.email,
    );
    toastStore.successResponseToast(response);
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
          Confirm email
        </h2>
        <p
          class="mt-2 text-base leading-normal"
          style="color: var(--text-secondary)"
        >
          You need to confirm your email to continue using the app.
        </p>
      </div>

      <div class="flex flex-col gap-4">
        <div class="flex flex-row w-full">
          <div class="flex flex-col gap-1 w-full">
            <label>Email</label>
            <InputText
              id="email"
              :value="authStore.user?.email"
              type="email"
              :disabled="loading"
              :readonly="true"
              class="w-full rounded-xl"
            />
          </div>
        </div>

        <Button
          label="Resend email"
          class="w-full main-button"
          :disabled="loading"
          @click="resendConfirmationEmail"
        />
      </div>

      <div
        class="flex items-center justify-center gap-2 mt-6 pt-4"
        style="border-top: 1px solid var(--border-color)"
      >
        <span class="text-sm" style="color: var(--text-secondary)">
          Sign in with a different account?
        </span>
        <button
          type="button"
          class="text-sm text-ink font-medium bg-transparent border-0 p-0 cursor-pointer hover:opacity-80"
          @click="authStore.logoutUser()"
        >
          Log in
        </button>
      </div>
    </div>
  </AuthSkeleton>
</template>
