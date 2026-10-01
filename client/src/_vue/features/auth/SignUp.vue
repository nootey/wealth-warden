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
import { useRoute, useRouter } from "vue-router";
import ValidationError from "../../components/validation/ValidationError.vue";
import AuthSkeleton from "../../components/layout/AuthSkeleton.vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import type { AuthForm } from "../../../models/auth_models.ts";
import { useAuthStore } from "../../../services/stores/auth_store.ts";
import { useUserStore } from "../../../services/stores/user_store.ts";
import type { Invitation } from "../../../models/user_models.ts";

const authStore = useAuthStore();
const userStore = useUserStore();
const toastStore = useToastStore();

const router = useRouter();
const route = useRoute();

const token = ref(route.query.token as string);
const loading = ref(false);
const invitation = ref<Invitation | null>(null);
const wasInvited = ref<boolean>(false);

const form = ref<AuthForm>({
  display_name: "",
  email: "",
  password: "",
  password_confirmation: "",
});

const { r$ } = useRegle(form, {
  display_name: {
    required,
  },
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
  await loadInvitation();
});

async function loadInvitation() {
  if (!token.value) {
    return;
  }

  loading.value = true;
  try {
    invitation.value = await userStore.getInvitationByHash(token.value);
    if (invitation.value?.email) {
      form.value.email = invitation.value.email;
      wasInvited.value = true;
    }
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    loading.value = false;
  }
}

async function signUp() {
  const { valid } = await r$.$validate();
  if (!valid) return;
  loading.value = true;

  try {
    await authStore.signUp(form.value, invitation.value?.id ?? null);
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
          Create an account
        </h2>
        <p
          class="mt-2 text-base leading-normal"
          style="color: var(--text-secondary)"
        >
          Fill in your details to get started.
        </p>
      </div>

      <form class="flex flex-col gap-4" @submit.prevent="signUp">
        <div class="flex flex-row w-full">
          <div class="flex flex-col gap-1 w-full">
            <ValidationError
              :is-required="true"
              :message="r$.display_name.$errors[0]"
            >
              <label>Display name</label>
            </ValidationError>
            <InputText
              id="display_name"
              v-model="form.display_name"
              type="text"
              :placeholder="'Display name'"
              :disabled="loading"
              :readonly="loading"
              class="w-full rounded-xl"
            />
          </div>
        </div>

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
              :disabled="loading || wasInvited"
              :readonly="loading || wasInvited"
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
              <label>Confirm password</label>
            </ValidationError>
            <Password
              id="password_confirmation"
              v-model="form.password_confirmation"
              :placeholder="'Confirm password'"
              :feedback="false"
              toggle-mask
              fluid
              input-class="rounded-xl"
              :disabled="loading"
            />
          </div>
        </div>

        <Button
          label="Sign up"
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
