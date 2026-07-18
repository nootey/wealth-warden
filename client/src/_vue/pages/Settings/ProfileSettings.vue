<script setup lang="ts">
import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import { computed, onMounted, ref } from "vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import { useThemeStore } from "../../../services/stores/theme_store.ts";
import ShowLoading from "../../components/base/ShowLoading.vue";
import type {
  CurrencyInfo,
  LanguageInfo,
  TimezoneInfo,
  UserSettings,
} from "../../../models/settings_models.ts";

const settingsStore = useSettingsStore();
const toastStore = useToastStore();
const themeStore = useThemeStore();

const userSettings = ref<UserSettings>();
const originalTimezone = ref<string>("");

const loading = ref<boolean>(true);

const languages = ref<LanguageInfo[]>([]);
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

const separatorOptions = ref([
  { value: ";", label: "Semicolon ( ; )" },
  { value: ",", label: "Comma ( , )" },
]);

const selectedSeparator = computed({
  get: () =>
    separatorOptions.value.find(
      (s) => s.value === userSettings.value?.default_sheet_separator,
    ),
  set: (newValue: { value: string; label: string } | null) => {
    if (userSettings.value && newValue) {
      userSettings.value.default_sheet_separator = newValue.value;
    }
  },
});

const selectedCurrency = computed({
  get: () =>
    currencies.value.find(
      (c) => c.value === userSettings.value?.default_currency,
    ),
  set: (newValue: CurrencyInfo | null) => {
    if (userSettings.value && newValue) {
      userSettings.value.default_currency = newValue.value;
    }
  },
});

const selectedLanguage = computed({
  get: () =>
    languages.value.find((lang) => lang.value === userSettings.value?.language),
  set: (newValue: LanguageInfo | null) => {
    if (userSettings.value && newValue) {
      userSettings.value.language = newValue.value;
    }
  },
});

const selectedTimezone = computed({
  get: () =>
    timezones.value.find((tz) => tz.value === userSettings.value?.timezone),
  set: (newValue: TimezoneInfo | null) => {
    if (userSettings.value && newValue) {
      userSettings.value.timezone = newValue.value;
    }
  },
});

const timezoneChanged = computed(
  () =>
    originalTimezone.value !== "" &&
    userSettings.value?.timezone !== originalTimezone.value,
);

const selectedTheme = computed({
  get: () =>
    themeOptions.value.find(
      (theme) => theme.value === userSettings.value?.theme,
    ),
  set: (newValue: { value: string; label: string } | null) => {
    if (userSettings.value && newValue) {
      userSettings.value.theme = newValue.value;
    }
  },
});

const selectedAccent = computed({
  get: () =>
    accentOptions.value.find(
      (accent) => accent.value === userSettings.value?.accent,
    ),
  set: (newValue: { value: string; label: string } | null) => {
    if (userSettings.value && newValue) {
      userSettings.value.accent = newValue.value;
    }
  },
});

onMounted(async () => {
  await initUserSettings();
  await getAvailableTimezones();
  await getAvailableLanguages();
  await getAvailableCurrencies();
});

async function initUserSettings() {
  loading.value = true;
  try {
    let response = await settingsStore.getUserSettings();
    userSettings.value = response.data;
    originalTimezone.value = response.data?.timezone ?? "";
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
}

async function getAvailableTimezones() {
  try {
    let response = await settingsStore.getAvailableTimezones();
    timezones.value = response.data;
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function getAvailableLanguages() {
  try {
    languages.value = [{ value: "en", label: "English" }];
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function getAvailableCurrencies() {
  try {
    let response = await settingsStore.getAvailableCurrencies();
    currencies.value = response.data;
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

function searchTimezone(event: any) {
  const query = event.query.toLowerCase();

  if (!query) {
    filteredTimezones.value = timezones.value;
  } else {
    filteredTimezones.value = timezones.value.filter(
      (tz) =>
        tz.label.toLowerCase().includes(query) ||
        tz.value.toLowerCase().includes(query),
    );
  }
}

function searchCurrency(event: { query: string }) {
  const query = event.query.toLowerCase();

  if (!query) {
    filteredCurrencies.value = currencies.value;
  } else {
    filteredCurrencies.value = currencies.value.filter(
      (c) =>
        c.label.toLowerCase().includes(query) ||
        c.value.toLowerCase().includes(query),
    );
  }
}

function searchLanguage(event: { query: string }) {
  const query = event.query.toLowerCase();

  if (!query) {
    filteredLanguages.value = languages.value;
  } else {
    filteredLanguages.value = languages.value.filter(
      (lang) =>
        lang.label.toLowerCase().includes(query) ||
        lang.value.toLowerCase().includes(query),
    );
  }
}

async function updateSettings() {
  loading.value = true;
  const settings = {
    language: userSettings.value?.language,
    timezone: userSettings.value?.timezone,
    theme: userSettings.value?.theme as "system" | "dark" | "light",
    accent: userSettings.value?.accent,
    default_currency: userSettings.value?.default_currency,
    default_sheet_separator: userSettings.value?.default_sheet_separator,
  };
  try {
    let response = await settingsStore.updatePreferenceSettings(settings);
    themeStore.setTheme(settings.theme!, settings.accent);
    originalTimezone.value = settings.timezone ?? "";
    toastStore.successResponseToast(response);
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="flex flex-col w-full gap-4">
    <SettingsSkeleton class="w-full">
      <div class="w-full flex flex-col gap-4 p-2">
        <div class="w-full flex flex-col gap-2">
          <h3>Avatar</h3>
          <h5 style="color: var(--text-secondary)">
            Customize your profile picture.
          </h5>
        </div>

        <div class="flex flex-row gap-2 w-50" style="margin: 0 auto">
          <div class="flex flex-col gap-4 justify-center items-center">
            <div
              class="w-32 h-32 rounded-full border border-surface flex items-center justify-center cursor-pointer"
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
      </div>
    </SettingsSkeleton>

    <SettingsSkeleton class="w-full">
      <div class="w-full flex flex-col gap-4 p-2">
        <div class="w-full flex flex-col gap-2">
          <h3>General</h3>
          <h5 style="color: var(--text-secondary)">
            Configure your preferences.
          </h5>
        </div>

        <div v-if="!loading" class="w-full flex flex-col gap-2 w-full">
          <div class="w-full flex flex-row gap-2 w-full">
            <IftaLabel class="w-full" variant="in">
              <AutoComplete
                id="language_input"
                v-model="selectedLanguage"
                dropdown
                size="small"
                :suggestions="filteredLanguages"
                option-label="label"
                option-value="value"
                class="w-full"
                :input-class="'w-full'"
                placeholder="Search language..."
                force-selection
                @complete="searchLanguage"
              />
              <label for="in_label">Language</label>
            </IftaLabel>
          </div>

          <div class="w-full flex flex-col gap-1">
            <IftaLabel class="w-full" variant="in">
              <AutoComplete
                id="currency_input"
                v-model="selectedCurrency"
                dropdown
                size="small"
                :suggestions="filteredCurrencies"
                option-label="label"
                option-value="value"
                class="w-full"
                :input-class="'w-full'"
                placeholder="Search currency..."
                force-selection
                disabled
                @complete="searchCurrency"
              />
              <label for="currency_input">Default Currency</label>
            </IftaLabel>
            <span class="text-xs" style="color: var(--text-secondary)">
              Default currency cannot be changed after initial setup.
            </span>
          </div>

          <div class="w-full flex flex-col gap-1">
            <IftaLabel class="w-full" variant="in">
              <AutoComplete
                id="in_label"
                v-model="selectedTimezone"
                dropdown
                size="small"
                :suggestions="filteredTimezones"
                option-label="label"
                option-value="value"
                class="w-full"
                :input-class="'w-full'"
                placeholder="Search timezone..."
                force-selection
                @complete="searchTimezone"
              />
              <label for="in_label">Timezone</label>
            </IftaLabel>
            <span
              v-if="timezoneChanged"
              class="text-xs"
              style="color: var(--p-red-300)"
            >
              Active templates will be rescheduled to match the same dates in
              the new timezone.
            </span>
          </div>

          <div class="w-full flex flex-col gap-1">
            <IftaLabel class="w-full" variant="in">
              <Select
                id="sheet_separator_input"
                v-model="selectedSeparator"
                :options="separatorOptions"
                option-label="label"
                class="w-full"
                placeholder="Select separator..."
              />
              <label for="sheet_separator_input">Sheet Separator</label>
            </IftaLabel>
            <span class="text-xs" style="color: var(--text-secondary)">
              Column separator used when exporting spreadsheet files. (Currently
              unused)
            </span>
          </div>
        </div>
        <ShowLoading v-else :num-fields="2" />
      </div>
    </SettingsSkeleton>

    <SettingsSkeleton class="w-full">
      <div class="w-full flex flex-col gap-4 p-2">
        <div class="w-full flex flex-col gap-2">
          <h3>Theme</h3>
          <h5 style="color: var(--text-secondary)">
            Choose a preferred theme for the app.
          </h5>
        </div>

        <div v-if="!loading" class="w-full flex flex-col gap-2 w-full">
          <div class="w-full flex flex-row gap-2 w-full">
            <IftaLabel class="w-full" variant="in">
              <Select
                id="theme_input"
                v-model="selectedTheme"
                :options="themeOptions"
                option-label="label"
                class="w-full"
                placeholder="Select theme..."
              />
              <label for="in_label">Theme</label>
            </IftaLabel>
          </div>

          <div class="w-full flex flex-row gap-2 w-full">
            <IftaLabel class="w-full" variant="in">
              <Select
                id="accent_input"
                v-model="selectedAccent"
                :options="accentOptions"
                option-label="label"
                class="w-full"
                placeholder="Select accent..."
              />
              <label for="in_label">Accent</label>
            </IftaLabel>
          </div>
        </div>
        <ShowLoading v-else :num-fields="2" />
      </div>
    </SettingsSkeleton>

    <div class="w-full flex flex-row gap-2 w-full">
      <Button
        class="main-button ml-auto"
        label="Save"
        @click="updateSettings"
      />
    </div>
  </div>
</template>

<style scoped></style>
