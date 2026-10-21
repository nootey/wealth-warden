<script setup lang="ts">
import { onMounted, ref } from "vue";
import { required, email, sameAs } from "@regle/rules";
import { useRegle } from "@regle/core";
import {
  passwordMinLength,
  noSpaces,
  hasNumber,
  hasUppercase,
  hasSpecialChar,
} from "../../../utils/password_validators.ts";
import { useRouter } from "vue-router";
import ValidationError from "../../components/validation/ValidationError.vue";
import AuthSkeleton from "../../components/layout/AuthSkeleton.vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import type { AuthForm } from "../../../models/auth_models.ts";
import { useAuthStore } from "../../../services/stores/auth_store.ts";
import { useUserStore } from "../../../services/stores/user_store.ts";

const authStore = useAuthStore();
const toastStore = useToastStore();
const userStore = useUserStore();

const router = useRouter();

const loading = ref(false);
const token = ref("");

const form = ref<AuthForm>({
  display_name: "",
  email: "",
  password: "",
  password_confirmation: "",
});

const { r$ } = useRegle(form, {
  email: {
    required,
    email,
  },
  password: {
    required,
    minLength: passwordMinLength,
    noSpaces,
    hasNumber,
    hasUppercase,
    hasSpecialChar,
  },
  password_confirmation: {
    required,
    sameAs: sameAs(() => form.value.password, "password"),
  },
});

onMounted(async () => {
  loading.value = true;
  token.value = window.location.pathname.substring(
    window.location.pathname.lastIndexOf("/") + 1,
    window.location.pathname.length,
  );
  if (!token.value || token.value === "") {
    await router.push("/");
  }
  await getUser();
});

async function getUser() {
  loading.value = true;

  try {
    const response = await userStore.getUserByToken(
      "password-reset",
      token.value,
    );
    form.value.email = response.data.email;
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
}

async function resetPassword() {
  const { valid } = await r$.$validate();
  if (!valid) return;

  loading.value = true;

  try {
    const response = await authStore.resetPassword({
      ...form.value,
      token: token.value,
    });
    toastStore.successResponseToast(response);
    await router.push({ name: "login" });
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
          Reset password
        </h2>
      </div>

      <form class="flex flex-col gap-4" @submit.prevent="resetPassword">
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
              :disabled="loading"
              :readonly="true"
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
              <label>New password</label>
            </ValidationError>
            <Password
              id="password"
              v-model="form.password"
              placeholder="New password"
              :disabled="loading"
              :feedback="false"
              toggle-mask
              fluid
              input-class="rounded-xl"
            />
          </div>
        </div>

        <div class="flex flex-row w-full">
          <div class="flex flex-col gap-1 w-full">
            <ValidationError
              :is-required="true"
              :message="r$.password_confirmation.$errors[0]"
            >
              <label>Confirm new password</label>
            </ValidationError>
            <Password
              id="password_confirmation"
              v-model="form.password_confirmation"
              placeholder="Confirm new password"
              :feedback="false"
              toggle-mask
              fluid
              input-class="rounded-xl"
              :disabled="loading"
            />
          </div>
        </div>

        <Button
          label="Reset password"
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
          Already have an account?
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
