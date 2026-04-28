<script setup lang="ts">
import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import { useAuthStore } from "../../../services/stores/auth_store.ts";
import { computed, onMounted, ref } from "vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import type { User } from "../../../models/user_models.ts";
import { useConfirm } from "primevue/useconfirm";
import ShowLoading from "../../components/base/ShowLoading.vue";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import { email, required, helpers } from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {
  passwordMinLength,
  noSpaces,
  hasNumber,
  hasUppercase,
  hasSpecialChar,
} from "../../../utils/password_validators.ts";
import ValidationError from "../../components/validation/ValidationError.vue";

const authStore = useAuthStore();
const toastStore = useToastStore();
const settingsStore = useSettingsStore();

const confirm = useConfirm();

const currentUser = ref<User>();

const emailUpdated = ref(false);
const loading = ref(true);

const password = ref("");
const passwordConfirmation = ref("");

const rules = computed(() => ({
  currentUser: {
    display_name: { required, $autoDirty: true },
    email: { required, email, $autoDirty: true },
  },
  password: password.value
    ? {
        minLength: passwordMinLength,
        noSpaces,
        hasNumber,
        hasUppercase,
        hasSpecialChar,
        $autoDirty: true,
      }
    : {},
  passwordConfirmation: password.value
    ? {
        repeatPassword: helpers.withMessage(
          ": must match password",
          (value: string) => value === password.value,
        ),
        $autoDirty: true,
      }
    : {},
}));

const v$ = useVuelidate(rules, { currentUser, password, passwordConfirmation });

async function isRecordValid() {
  const isValid = await v$.value.$validate();
  if (!isValid) return false;
  return true;
}

onMounted(async () => {
  await initUser();
});

async function initUser() {
  loading.value = true;
  try {
    currentUser.value = await authStore.getAuthUser(false);
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
}

async function confirmUpdateSettings() {
  const changingEmail = emailUpdated.value;
  const changingPassword = !!password.value;

  if (changingEmail || changingPassword) {
    const parts = [];
    if (changingEmail) parts.push("your email address");
    if (changingPassword) parts.push("your password");
    const what = parts.join(" and ");

    confirm.require({
      header: "Confirm operation",
      message: `You're about to change ${what}. This will log you out.`,
      rejectProps: { label: "Cancel" },
      acceptProps: { label: "Continue" },
      accept: async () => await updateSettings(),
    });
  } else {
    await updateSettings();
  }
}

async function updateSettings() {
  if (!(await isRecordValid())) return;

  loading.value = true;
  const rec = {
    display_name: currentUser.value?.display_name,
    email_updated: emailUpdated.value,
    email: currentUser.value?.email,
    password: password.value || null,
    password_confirmation: passwordConfirmation.value || null,
  };

  try {
    let response = await settingsStore.updateProfileSettings(rec);
    toastStore.successResponseToast(response);
    if (rec.email_updated || rec.password) {
      authStore.logout();
    }
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="flex flex-column w-full gap-3">
    <SettingsSkeleton class="w-full">
      <div class="w-full flex flex-column gap-3 p-2">
        <div class="w-full flex flex-column gap-2">
          <h3>Profile</h3>
          <h5 style="color: var(--text-secondary)">
            Customize how your account details.
          </h5>
        </div>

        <div class="flex flex-row gap-2 w-50" style="margin: 0 auto">
          <div
            class="flex flex-column gap-3 justify-content-center align-items-center"
          >
            <div
              class="w-8rem h-8rem border-circle border-1 surface-border flex align-items-center justify-content-center cursor-pointer"
            >
              <i class="pi pi-image text-2xl" />
            </div>

            <Button
              class="main-button"
              label="Upload photo"
              icon="pi pi-image"
            />

            <span style="color: var(--text-secondary)"
              >JPG or PNG. 5MB max.</span
            >
          </div>
        </div>

        <div
          v-if="!loading && currentUser"
          class="w-full flex flex-column gap-2 w-full"
        >
          <div class="flex flex-row w-full">
            <div class="flex flex-column w-full">
              <ValidationError
                :is-required="true"
                :message="v$.currentUser.email.$errors[0]?.$message"
              >
                <label>Email</label>
              </ValidationError>
              <InputText
                v-model="currentUser.email"
                class="w-full"
                @update:model-value="emailUpdated = true"
              />
            </div>
          </div>
          <div class="flex flex-row w-full">
            <div class="flex flex-column w-full">
              <ValidationError
                :is-required="true"
                :message="v$.currentUser.display_name.$errors[0]?.$message"
              >
                <label>Display name</label>
              </ValidationError>
              <InputText
                id="in_label"
                v-model="currentUser.display_name"
                class="w-full"
              />
            </div>
          </div>
          <div class="flex flex-row w-full">
            <div class="flex flex-column gap-1 w-full">
              <ValidationError :message="v$.password.$errors[0]?.$message">
                <label>New password</label>
              </ValidationError>
              <InputText
                v-model="password"
                type="password"
                placeholder="Password (leave blank to keep)"
                class="w-full"
              />
            </div>
          </div>
          <div class="flex flex-row w-full">
            <div class="flex flex-column gap-1 w-full">
              <ValidationError
                :message="v$.passwordConfirmation.$errors[0]?.$message"
              >
                <label>Confirm new password</label>
              </ValidationError>
              <InputText
                v-model="passwordConfirmation"
                type="password"
                placeholder="Confirm new password"
                class="w-full"
              />
            </div>
          </div>
          <div class="w-full flex flex-row gap-2 w-full">
            <Button
              class="main-button ml-auto"
              label="Save"
              @click="confirmUpdateSettings"
            />
          </div>
        </div>
        <ShowLoading v-else :num-fields="2" />
      </div>
    </SettingsSkeleton>

    <SettingsSkeleton class="w-full">
      <div class="w-full flex flex-column gap-3 p-2">
        <div class="w-full flex flex-column gap-2">
          <h3>Danger zone</h3>
          <h5 style="color: var(--text-secondary)">Thread carefully.</h5>
        </div>

        <div class="w-full flex flex-row gap-3 align-items-center">
          <div class="flex flex-column w-full">
            <h4>Reset account</h4>
            <h5 style="color: var(--text-secondary)">
              Resetting your account will delete all your accounts, categories,
              and other data, but keep your user account intact.
            </h5>
          </div>
          <div class="flex flex-column w-3">
            <Button size="small" label="Reset account" class="delete-button" />
          </div>
        </div>

        <div class="w-full flex flex-row gap-3 align-items-center">
          <div class="flex flex-column w-full">
            <h4>Delete account</h4>
            <h5 style="color: var(--text-secondary)">
              Deleting your account will permanently remove all your data and
              cannot be undone.
            </h5>
          </div>
          <div class="flex flex-column w-3">
            <Button size="small" label="Delete account" class="delete-button" />
          </div>
        </div>
      </div>
    </SettingsSkeleton>
  </div>
</template>

<style scoped></style>
