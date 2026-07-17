<script setup lang="ts">
import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import { computed, onMounted, ref } from "vue";
import { useAuthStore } from "../../../services/stores/auth_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import { useWsStore } from "../../../services/stores/ws_store.ts";
import { usePermissions } from "../../../utils/use_permissions.ts";
import { useConfirm } from "primevue/useconfirm";
import ShowLoading from "../../components/base/ShowLoading.vue";
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
import dateHelper from "../../../utils/date_helper.ts";
import type { User } from "../../../models/user_models.ts";
import type { SessionInfo } from "../../../models/auth_models.ts";

const authStore = useAuthStore();
const toastStore = useToastStore();
const settingsStore = useSettingsStore();
const wsStore = useWsStore();
const { hasPermission } = usePermissions();

const confirm = useConfirm();

const isAdmin = hasPermission("access_backoffice");
const endpoint = wsStore.endpoint();

const currentUser = ref<User>();

const emailUpdated = ref(false);
const loading = ref(true);

const password = ref("");
const passwordConfirmation = ref("");

const sessions = ref<SessionInfo[]>([]);
const sessionsLoading = ref(true);

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
  await initSessions();
});

async function initSessions() {
  sessionsLoading.value = true;
  try {
    const response = await authStore.getSessions();
    sessions.value = response.data ?? [];
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    sessionsLoading.value = false;
  }
}

async function revokeSession(session: SessionInfo) {
  try {
    const response = await authStore.revokeSession(session.id);
    toastStore.successResponseToast(response);
    await initSessions();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

function confirmRevokeAllSessions() {
  confirm.require({
    header: "Confirm operation",
    message: "This will log you out on every device, including this one.",
    rejectProps: { label: "Cancel" },
    acceptProps: { label: "Continue" },
    accept: async () => await revokeAllSessions(),
  });
}

async function revokeAllSessions() {
  // detach first so this tab doesn't also react to its own revoked-close frame
  wsStore.disconnect();
  try {
    const response = await authStore.revokeAllSessions();
    toastStore.successResponseToast(response);
    await authStore.logoutUser();
  } catch (error) {
    toastStore.errorResponseToast(error);
    wsStore.connect();
  }
}

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
          <h3>Account</h3>
          <h5 style="color: var(--text-secondary)">
            Customize how your account details.
          </h5>
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
          <h3>Sessions</h3>
          <h5 style="color: var(--text-secondary)">
            Devices currently signed in to your account. Revoking a session logs
            that device out.
          </h5>
        </div>

        <div v-if="!sessionsLoading" class="w-full flex flex-column gap-3">
          <div
            v-for="session in sessions"
            :key="session.id"
            class="w-full flex flex-row gap-3 align-items-center"
          >
            <div class="flex flex-column flex-1 gap-1">
              <div class="flex flex-row align-items-center">
                <span class="text-sm">{{ session.device }}</span>
                <i
                  v-if="session.current"
                  v-tooltip="'This device'"
                  class="pi pi-check-circle ml-2 text-sm"
                  style="color: var(--p-green-400)"
                ></i>
                <i
                  v-else
                  v-tooltip="'Revoke'"
                  class="pi pi-trash ml-2 text-sm"
                  style="color: var(--p-red-300); cursor: pointer"
                  @click="revokeSession(session)"
                ></i>
              </div>
              <span class="text-xs" style="color: var(--text-secondary)">
                {{ session.ip }} &middot; Last seen
                {{ dateHelper.formatDate(session.last_seen, true) }}
              </span>
            </div>
          </div>

          <div class="flex flex-row gap-2">
            <Button
              label="Log out everywhere"
              class="delete-button"
              size="small"
              @click="confirmRevokeAllSessions"
            />
          </div>
        </div>
        <ShowLoading v-else :num-fields="2" />
      </div>
    </SettingsSkeleton>

    <SettingsSkeleton class="w-full">
      <div class="w-full flex flex-column gap-3 p-2">
        <div class="w-full flex flex-column gap-2">
          <h3>Connection</h3>
          <h5 style="color: var(--text-secondary)">
            The live connection that delivers notifications and report updates
            without a page refresh.
          </h5>
        </div>

        <div class="w-full flex flex-column gap-2">
          <div class="flex flex-row align-items-center gap-2">
            <span class="text-sm w-10rem" style="color: var(--text-secondary)"
              >Status</span
            >
            <Tag
              :severity="wsStore.connected ? 'success' : 'danger'"
              :value="wsStore.connected ? 'Connected' : 'Disconnected'"
            />
          </div>

          <div class="flex flex-row align-items-center gap-2">
            <span class="text-sm w-10rem" style="color: var(--text-secondary)"
              >Reconnect attempts</span
            >
            <span class="text-sm">{{ wsStore.attempts }}</span>
          </div>

          <div v-if="isAdmin" class="flex flex-row align-items-center gap-2">
            <span class="text-sm w-10rem" style="color: var(--text-secondary)"
              >Endpoint</span
            >
            <code class="text-sm">{{ endpoint }}</code>
          </div>
        </div>

        <div class="text-sm" style="color: var(--text-secondary)">
          If updates stop arriving, disconnect and connect again to force a
          fresh connection. The app already retries on its own with a growing
          delay, so reach for this only when it has given up.
        </div>

        <div class="text-sm" style="color: var(--text-secondary)">
          Connecting and disconnecting here only lasts for as long as this page
          is open. Refreshing the page reopens the connection automatically, and
          logging out closes it.
        </div>

        <div class="flex flex-row gap-2">
          <Button
            class="main-button"
            size="small"
            label="Connect"
            :disabled="wsStore.connected"
            @click="wsStore.reconnect()"
          />
          <Button
            label="Disconnect"
            class="delete-button"
            size="small"
            :disabled="!wsStore.connected"
            @click="wsStore.disconnect(true)"
          />
        </div>
      </div>
    </SettingsSkeleton>

    <SettingsSkeleton class="w-full">
      <div class="w-full flex flex-column gap-3 p-2">
        <div class="w-full flex flex-column gap-2">
          <h3>Danger zone</h3>
          <h5 style="color: var(--text-secondary)">Tread carefully.</h5>
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
