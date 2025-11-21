<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import {computed, onMounted, ref} from "vue";
import type {LanguageInfo, TimezoneInfo, UserSettings} from "../../../models/settings_models.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useSettingsStore} from "../../../services/stores/settings_store.ts";
import ShowLoading from "../../components/base/ShowLoading.vue";
import {useThemeStore} from "../../../services/stores/theme_store.ts";

const settingsStore = useSettingsStore();
const toastStore = useToastStore();
const themeStore = useThemeStore();

const userSettings = ref<UserSettings>();

const loading = ref<boolean>(true);

const languages = ref<LanguageInfo[]>([]);
const filteredLanguages = ref<LanguageInfo[]>([]);

const timezones = ref<TimezoneInfo[]>([]);
const filteredTimezones = ref<TimezoneInfo[]>([]);

const themeOptions = ref([
    { value: "system", label: "System" },
    { value: "dark", label: "Dark" },
    { value: "light", label: "Light" }
]);

const accentOptions = ref([
    { value: "blurple", label: "Blurple" }
]);

const selectedLanguage = computed({
    get: () => languages.value.find(lang => lang.value === userSettings.value?.language),
    set: (newValue: LanguageInfo | null) => {
        if (userSettings.value && newValue) {
            userSettings.value.language = newValue.value;
        }
    }
});

const selectedTimezone = computed({
    get: () => timezones.value.find(tz => tz.value === userSettings.value?.timezone),
    set: (newValue: TimezoneInfo | null) => {
        if (userSettings.value && newValue) {
            userSettings.value.timezone = newValue.value;
        }
    }
});

const selectedTheme = computed({
    get: () => themeOptions.value.find(theme => theme.value === userSettings.value?.theme),
    set: (newValue: { value: string, label: string } | null) => {
        if (userSettings.value && newValue) {
            userSettings.value.theme = newValue.value;
        }
    }
});

const selectedAccent = computed({
    get: () => accentOptions.value.find(accent => accent.value === userSettings.value?.accent),
    set: (newValue: { value: string, label: string } | null) => {
        if (userSettings.value && newValue) {
            userSettings.value.accent = newValue.value;
        }
    }
});

onMounted(async () => {
    await initUserSettings();
    await getAvailableTimezones();
    await getAvailableLanguages();
})

async function initUserSettings() {
    loading.value = true;
    try {
        let response = await settingsStore.getUserSettings();
        userSettings.value = response.data;
    } catch (error) {
        toastStore.errorResponseToast(error)
    } finally {
        loading.value = false;
    }
}

async function getAvailableTimezones() {
    try {
        let response = await settingsStore.getAvailableTimezones();
        timezones.value = response.data;
    } catch (error) {
        toastStore.errorResponseToast(error)
    }
}

async function getAvailableLanguages() {
    try {
        languages.value = [{ value: "en", label: "English" },]
    } catch (error) {
        toastStore.errorResponseToast(error)
    }
}

function searchTimezone(event: any) {
    const query = event.query.toLowerCase();

    if (!query) {
        filteredTimezones.value = timezones.value;
    } else {
        filteredTimezones.value = timezones.value.filter(tz =>
            tz.label.toLowerCase().includes(query) ||
            tz.value.toLowerCase().includes(query)
        );
    }
}

function searchLanguage(event: { query: string }) {
    const query = event.query.toLowerCase();

    if (!query) {
        filteredLanguages.value = languages.value;
    } else {
        filteredLanguages.value = languages.value.filter(lang =>
            lang.label.toLowerCase().includes(query) ||
            lang.value.toLowerCase().includes(query)
        );
    }
}

async function updateSettings() {
    loading.value = true;
    const settings = {
        language: userSettings.value?.language,
        timezone: userSettings.value?.timezone,
        theme: userSettings.value?.theme as 'system' | 'dark' | 'light',
        accent: userSettings.value?.accent,
    }
    try {
        let response = await settingsStore.updatePreferenceSettings(settings);
        themeStore.setTheme(settings?.theme!, settings.accent);
        toastStore.successResponseToast(response);
    } catch (error) {
        toastStore.errorResponseToast(error)
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
                    <h3>General</h3>
                    <h5 style="color: var(--text-secondary)">Configure your preferences.</h5>
                </div>

                <div v-if="!loading" class="w-full flex flex-column gap-2 w-full">
                    <div class="w-full flex flex-row gap-2 w-full">
                        <IftaLabel class="w-full" variant="in">
                            <AutoComplete
                                    id="language_input" dropdown size="small"
                                    v-model="selectedLanguage"
                                    :suggestions="filteredLanguages"
                                    @complete="searchLanguage"
                                    optionLabel="label"
                                    optionValue="value"
                                    class="w-full"
                                    :inputClass="'w-full'"
                                    placeholder="Search language..."
                                    forceSelection
                            />
                            <label for="in_label">Language</label>
                        </IftaLabel>
                    </div>

                    <div class="w-full flex flex-row gap-2 w-full">
                        <IftaLabel class="w-full" variant="in">
                            <AutoComplete id="in_label" dropdown size="small"
                                    v-model="selectedTimezone"
                                    :suggestions="filteredTimezones"
                                    @complete="searchTimezone"
                                    optionLabel="label"
                                    optionValue="value"
                                    class="w-full"
                                    :inputClass="'w-full'"
                                    placeholder="Search timezone..."
                                    forceSelection
                            />
                            <label for="in_label">Timezone</label>
                        </IftaLabel>
                    </div>
                </div>
                <ShowLoading v-else :numFields="2" />
            </div>
        </SettingsSkeleton>

        <SettingsSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-2">
                <div class="w-full flex flex-column gap-2">
                    <h3>Theme</h3>
                    <h5 style="color: var(--text-secondary)">Choose a preferred theme for the app.</h5>
                </div>

                <div v-if="!loading" class="w-full flex flex-column gap-2 w-full">
                    <div class="w-full flex flex-row gap-2 w-full">
                        <IftaLabel class="w-full" variant="in">
                            <Select id="theme_input"
                                    v-model="selectedTheme"
                                    :options="themeOptions"
                                    optionLabel="label"
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
                                    optionLabel="label"
                                    class="w-full"
                                    placeholder="Select accent..."
                            />
                            <label for="in_label">Accent</label>
                        </IftaLabel>
                    </div>

                </div>
                <ShowLoading v-else :numFields="2" />
            </div>
        </SettingsSkeleton>

        <div class="w-full flex flex-row gap-2 w-full">
            <Button class="main-button ml-auto" label="Save" @click="updateSettings"></Button>
        </div>
    </div>
</template>

<style scoped>

</style>