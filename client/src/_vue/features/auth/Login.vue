<script setup lang="ts">
import { ref } from "vue";
import { required, email } from "@regle/rules";
import { useRegle } from "@regle/core";
import { useRoute, useRouter } from "vue-router";
import ValidationError from "../../components/validation/ValidationError.vue";
import { useAuthStore } from "../../../services/stores/auth_store.ts";
import AuthSkeleton from "../../components/layout/AuthSkeleton.vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import type { AuthForm } from "../../../models/auth_models.ts";

const authStore = useAuthStore();
const toastStore = useToastStore();

const router = useRouter();
const route = useRoute();

const loading = ref<boolean>(false);

const form = ref<AuthForm>({
  email: "",
  password: "",
  remember_me: false,
});

const { r$ } = useRegle(form, {
  email: {
    required,
    email,
  },
  password: {
    required,
  },
});

function resolveRedirect(): string {
  const q = route.query.redirect as string | string[] | undefined;
  const redirect = Array.isArray(q) ? q[0] : q;

  if (typeof redirect !== "string") return "/";

  // Disallow absolute URLs or protocol-relative
  if (/^https?:\/\//i.test(redirect) || redirect.startsWith("//")) return "/";

  // Allow only root-relative paths
  if (!redirect.startsWith("/")) return "/";

  // Avoid looping back to login
  if (redirect === "/login") return "/";

  return redirect;
}

async function login() {
  const { valid } = await r$.$validate();
  if (!valid) return;

  loading.value = true;
  try {
    await authStore.login(form.value);

    if (authStore.authenticated) {
      const target = resolveRedirect();
      await router.replace(target);
    }
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
}

function signUp() {
  router.push({ name: "sign.up" });
}

function forgotPassword() {
  router.push({ name: "forgot.password" });
}
</script>

<template>
  <AuthSkeleton>
    <div class="w-full mx-auto px-4 sm:px-0" style="max-width: 400px">
      <div id="hideOnMobile" class="text-center mb-6">
        <h2
          class="m-0 text-2xl sm:text-3xl font-bold"
          style="color: var(--text-primary); letter-spacing: -0.025em"
        >
          Welcome back
        </h2>
        <p
          class="mt-2 text-base leading-normal"
          style="color: var(--text-secondary)"
        >
          Sign in to your account to continue
        </p>
      </div>

      <div class="flex flex-col gap-4">
        <div class="flex flex-row w-full">
          <div class="flex flex-col gap-1 w-full">
            <ValidationError :is-required="true" :message="r$.email.$errors[0]">
              <label>Email</label>
            </ValidationError>
            <InputText
              id="email"
              v-model="form.email"
              type="email"
              :placeholder="'Email'"
              class="w-full rounded-xl"
            />
          </div>
        </div>

        <div class="flex flex-row w-full">
          <div class="flex flex-col gap-1 w-full">
            <ValidationError
              :is-required="true"
              :message="r$.password.$errors[0]"
            >
              <label>Password</label>
            </ValidationError>
            <InputText
              id="password"
              v-model="form.password"
              type="password"
              :placeholder="'Password'"
              class="w-full rounded-xl"
              @keydown.enter="login"
            />
          </div>
        </div>

        <div class="flex flex-row w-full justify-between">
          <div class="flex flex-row items-center gap-2">
            <Checkbox
              v-model="form.remember_me"
              input-id="rememberMe"
              :binary="true"
              class="scale-90"
            />
            <label
              for="rememberMe"
              class="text-sm cursor-pointer"
              style="color: var(--text-secondary)"
            >
              Remember me
            </label>
          </div>

          <span class="text-sm hover-icon hover-dim" @click="forgotPassword">
            Forgot password?</span
          >
        </div>

        <Button
          :label="loading ? 'Signing in...' : 'Sign in'"
          :icon="loading ? 'pi pi-spin pi-spinner mr-2' : ''"
          class="w-full auth-accent-button"
          :disabled="loading || r$.$error"
          @click="login"
        />
      </div>

      <div
        class="flex items-center justify-center gap-2 mt-6 pt-4"
        style="border-top: 1px solid var(--border-color)"
      >
        <span class="text-sm" style="color: var(--text-secondary)">
          Don't have an account?
        </span>
        <span class="text-sm hover-icon hover-dim" @click="signUp">
          Create account</span
        >
      </div>
    </div>
  </AuthSkeleton>
</template>

<style scoped>
@media (max-width: 768px) {
  #hideOnMobile {
    display: none;
  }
}

.hover-dim {
  color: var(--accent-primary);
}
.hover-dim:hover {
  color: var(--accent-secondary);
}
</style>
