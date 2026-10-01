<script setup lang="ts">
import { ref } from "vue";
import { required, email } from "@regle/rules";
import { useRegle } from "@regle/core";
import { useRoute, useRouter } from "vue-router";
import ValidationError from "../../components/validation/ValidationError.vue";
import { useAuthStore } from "../../../services/stores/auth_store.ts";
import AuthSkeleton from "../../components/layout/AuthSkeleton.vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import styleHelper from "../../../utils/style_helper.ts";
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
</script>

<template>
  <AuthSkeleton>
    <div class="w-full mx-auto" style="max-width: 400px">
      <div class="mb-8">
        <h2
          class="m-0 text-3xl font-medium"
          style="color: var(--text-primary); letter-spacing: -0.025em"
        >
          Welcome back
        </h2>
        <p
          class="mt-2 text-base leading-normal"
          style="color: var(--text-secondary)"
        >
          Sign in to your account to continue.
        </p>
      </div>

      <form class="flex flex-col gap-4" @submit.prevent="login">
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
            <Password
              id="password"
              v-model="form.password"
              :placeholder="'Password'"
              :feedback="false"
              toggle-mask
              fluid
              input-class="rounded-xl"
            />
          </div>
        </div>

        <div class="flex flex-row w-full justify-between">
          <div class="flex flex-row items-center gap-2">
            <Checkbox
              v-model="form.remember_me"
              input-id="rememberMe"
              :binary="true"
              :dt="styleHelper.neutralControlDt"
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

          <router-link
            :to="{ name: 'forgot.password' }"
            class="text-sm text-ink font-medium no-underline hover:opacity-80"
          >
            Forgot password?
          </router-link>
        </div>

        <Button
          :label="loading ? 'Signing in...' : 'Sign in'"
          :icon="loading ? 'pi pi-spin pi-spinner mr-2' : ''"
          class="w-full main-button"
          :disabled="loading || r$.$error"
          type="submit"
        />
      </form>

      <div
        class="flex items-center justify-center gap-2 mt-6 pt-4"
        style="border-top: 1px solid var(--border-color)"
      >
        <span class="text-sm" style="color: var(--text-secondary)">
          Don't have an account?
        </span>
        <router-link
          :to="{ name: 'sign.up' }"
          class="text-sm text-ink font-medium no-underline hover:opacity-80"
        >
          Create account
        </router-link>
      </div>
    </div>
  </AuthSkeleton>
</template>
