<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import AuthSkeleton from "../../components/layout/AuthSkeleton.vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useAuthStore } from "../../../services/stores/auth_store.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import { useThemeStore } from "../../../services/stores/theme_store.ts";
import { useRouter } from "vue-router";
import type {
  CurrencyInfo,
  LanguageInfo,
  TimezoneInfo,
} from "../../../models/settings_models.ts";

const authStore = useAuthStore();
const settingsStore = useSettingsStore();
const toastStore = useToastStore();
const themeStore = useThemeStore();
const router = useRouter();

const loading = ref(true);
const saving = ref(false);

const form = ref({
  language: "en",
  timezone: "",
  default_currency: "",
  theme: "system",
  accent: "blurple",
});

const languages = ref<LanguageInfo[]>([{ value: "en", label: "English" }]);
const filteredLanguages = ref<LanguageInfo[]>([]);

const timezones = ref<TimezoneInfo[]>([]);
const filteredTimezones = ref<TimezoneInfo[]>([]);

const currencies = ref<CurrencyInfo[]>([]);
const filteredCurrencies = ref<CurrencyInfo[]>([]);

const themeOptions = ref([
  { value: "system", label: "System" },
  { value: "dark", label: "Dark" },
  { value: "light", label: "Light" },
]);

const accentOptions = ref([{ value: "blurple", label: "Blurple" }]);

const selectedCurrency = computed({
  get: () =>
    currencies.value.find((c) => c.value === form.value.default_currency),
  set: (v: CurrencyInfo | null) => {
    if (v) form.value.default_currency = v.value;
  },
});

const selectedLanguage = computed({
  get: () => languages.value.find((l) => l.value === form.value.language),
  set: (v: LanguageInfo | null) => {
    if (v) form.value.language = v.value;
  },
});

const selectedTimezone = computed({
  get: () => timezones.value.find((t) => t.value === form.value.timezone),
  set: (v: TimezoneInfo | null) => {
    if (v) form.value.timezone = v.value;
  },
});

const selectedTheme = computed({
  get: () => themeOptions.value.find((t) => t.value === form.value.theme),
  set: (v: { value: string; label: string } | null) => {
    if (v) form.value.theme = v.value;
  },
});

const selectedAccent = computed({
  get: () => accentOptions.value.find((a) => a.value === form.value.accent),
  set: (v: { value: string; label: string } | null) => {
    if (v) form.value.accent = v.value;
  },
});

onMounted(async () => {
  try {
    const [settingsRes, timezonesRes, currenciesRes] = await Promise.all([
      settingsStore.getUserSettings(),
      settingsStore.getAvailableTimezones(),
      settingsStore.getAvailableCurrencies(),
    ]);

    timezones.value = timezonesRes.data;
    currencies.value = currenciesRes.data;

    if (settingsRes.data) {
      form.value.language = settingsRes.data.language || "en";
      form.value.timezone = settingsRes.data.timezone || "";
      form.value.default_currency = settingsRes.data.default_currency || "";
      form.value.theme = settingsRes.data.theme || "system";
      form.value.accent = settingsRes.data.accent || "blurple";
    }
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
});

function searchTimezone(event: { query: string }) {
  const query = event.query.toLowerCase();
  filteredTimezones.value = !query
    ? timezones.value
    : timezones.value.filter(
        (tz) =>
          tz.label.toLowerCase().includes(query) ||
          tz.value.toLowerCase().includes(query),
      );
}

function searchCurrency(event: { query: string }) {
  const query = event.query.toLowerCase();
  filteredCurrencies.value = !query
    ? currencies.value
    : currencies.value.filter(
        (c) =>
          c.label.toLowerCase().includes(query) ||
          c.value.toLowerCase().includes(query),
      );
}

function searchLanguage(event: { query: string }) {
  const query = event.query.toLowerCase();
  filteredLanguages.value = !query
    ? languages.value
    : languages.value.filter(
        (l) =>
          l.label.toLowerCase().includes(query) ||
          l.value.toLowerCase().includes(query),
      );
}

async function completeSetup() {
  saving.value = true;
  try {
    await authStore.completeSetup(form.value);
    themeStore.setTheme(
      form.value.theme as "system" | "dark" | "light",
      form.value.accent,
    );
    await authStore.getAuthUser(true);
    await router.push({ name: "dashboard" });
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <AuthSkeleton>
    <div class="w-full mx-auto px-3 sm:px-0" style="max-width: 420px">
      <div class="text-center mb-4">
        <h2
          class="m-0 text-2xl sm:text-3xl font-bold"
          style="color: var(--text-primary); letter-spacing: -0.025em"
        >
          Welcome, {{ authStore.user?.display_name ?? "there" }}
        </h2>
        <p
          class="mt-2 line-height-3 text-base"
          style="color: var(--text-secondary)"
        >
          Let's get a few things set up before you dive in.
        </p>
      </div>

      <div v-if="!loading" class="flex flex-column gap-3">
        <IftaLabel class="w-full" variant="in">
          <AutoComplete
            id="currency_input"
            v-model="selectedCurrency"
            dropdown
            size="small"
            :suggestions="filteredCurrencies"
            option-label="label"
            class="w-full"
            :input-class="'w-full'"
            placeholder="Search currency..."
            force-selection
            @complete="searchCurrency"
          />
          <label for="currency_input">Default Currency</label>
        </IftaLabel>

        <IftaLabel class="w-full" variant="in">
          <AutoComplete
            id="timezone_input"
            v-model="selectedTimezone"
            dropdown
            size="small"
            :suggestions="filteredTimezones"
            option-label="label"
            class="w-full"
            :input-class="'w-full'"
            placeholder="Search timezone..."
            force-selection
            @complete="searchTimezone"
          />
          <label for="timezone_input">Timezone</label>
        </IftaLabel>

        <IftaLabel class="w-full" variant="in">
          <AutoComplete
            id="language_input"
            v-model="selectedLanguage"
            dropdown
            size="small"
            :suggestions="filteredLanguages"
            option-label="label"
            class="w-full"
            :input-class="'w-full'"
            placeholder="Search language..."
            force-selection
            @complete="searchLanguage"
          />
          <label for="language_input">Language</label>
        </IftaLabel>

        <IftaLabel class="w-full" variant="in">
          <Select
            id="theme_input"
            v-model="selectedTheme"
            :options="themeOptions"
            option-label="label"
            class="w-full"
            placeholder="Select theme..."
          />
          <label for="theme_input">Theme</label>
        </IftaLabel>

        <IftaLabel class="w-full" variant="in">
          <Select
            id="accent_input"
            v-model="selectedAccent"
            :options="accentOptions"
            option-label="label"
            class="w-full"
            placeholder="Select accent..."
          />
          <label for="accent_input">Accent</label>
        </IftaLabel>

        <Button
          label="Complete setup"
          class="w-full auth-accent-button mt-2"
          :disabled="saving || !form.default_currency || !form.timezone"
          :loading="saving"
          @click="completeSetup"
        />
      </div>

      <div v-else class="flex flex-column gap-3">
        <Skeleton v-for="n in 5" :key="n" height="3rem" border-radius="8px" />
      </div>

      <div
        class="flex align-items-center justify-content-center gap-2 mt-4 pt-3"
        style="border-top: 1px solid var(--border-color)"
      >
        <span class="text-sm" style="color: var(--text-secondary)">
          Wrong account?
        </span>
        <span
          class="text-sm hover-dim"
          style="cursor: pointer"
          @click="authStore.logoutUser()"
        >
          Log out
        </span>
      </div>
    </div>
  </AuthSkeleton>
</template>

<style scoped>
.hover-dim {
  color: var(--accent-primary);
}
.hover-dim:hover {
  color: var(--accent-secondary);
}
</style>
