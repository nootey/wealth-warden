<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import {onMounted, ref} from "vue";
import type { UserSettings } from "../../../models/settings_models.ts";
import {useUserStore} from "../../../services/stores/user_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";

const userStore = useUserStore();
const toastStore = useToastStore();

const userSettings = ref<UserSettings>();

onMounted(async () => {
    await initUserSettings();
})

async function initUserSettings() {
    try {
        let response = await userStore.getUserSettings();
        userSettings.value = response.data;
    } catch (error) {
        toastStore.errorResponseToast(error)
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

                <div v-if="userSettings" class="w-full flex flex-column gap-2 w-full">
                    <div class="w-full flex flex-row gap-2 w-full">
                        <IftaLabel class="w-full" variant="in">
                            <InputText class="w-full" id="in_label" :value="userSettings.language" />
                            <label for="in_label">Language</label>
                        </IftaLabel>
                    </div>

                    <div class="w-full flex flex-row gap-2 w-full">
                        <IftaLabel class="w-full" variant="in">
                            <InputText class="w-full" id="in_label" :value="userSettings.timezone" />
                            <label for="in_label">Timezone</label>
                        </IftaLabel>
                    </div>

                    <div class="w-full flex flex-row gap-2 w-full">
                        <Button class="main-button ml-auto" label="Save"></Button>
                    </div>
                </div>
            </div>
        </SettingsSkeleton>

        <SettingsSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-2">
                <div class="w-full flex flex-column gap-2">
                    <h3>Theme</h3>
                    <h5 style="color: var(--text-secondary)">Choose a preferred theme for the app.</h5>
                </div>

                <div v-if="userSettings" class="w-full flex flex-column gap-2 w-full">
                    <div class="w-full flex flex-row gap-2 w-full">
                        <IftaLabel class="w-full" variant="in">
                            <InputText class="w-full" id="in_label" :value="userSettings.theme" />
                            <label for="in_label">Theme</label>
                        </IftaLabel>
                    </div>

                    <div class="w-full flex flex-row gap-2 w-full">
                        <IftaLabel class="w-full" variant="in">
                            <InputText class="w-full" id="in_label" :value="userSettings.accent" />
                            <label for="in_label">Accent</label>
                        </IftaLabel>
                    </div>

                    <div class="w-full flex flex-row gap-2 w-full">
                        <Button class="main-button ml-auto" label="Save"></Button>
                    </div>
                </div>
            </div>
        </SettingsSkeleton>
    </div>
</template>

<style scoped>

</style>