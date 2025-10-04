<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import {onMounted, ref} from "vue";
import type { GeneralSettings } from "../../../models/settings_models.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useSettingsStore} from "../../../services/stores/settings_store.ts";

const settingsStore = useSettingsStore();
const toastStore = useToastStore();

const settings = ref<GeneralSettings>();

onMounted(async () => {
    await initSettings();
})

async function initSettings() {
    try {
        let response = await settingsStore.getGeneralSettings();
        settings.value = response.data;
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
                    <h5 style="color: var(--text-secondary)">Configure general app preferences.</h5>
                </div>

                <div v-if="settings" class="w-full flex flex-column gap-2 w-full">
                    <div class="w-full flex flex-row gap-2 w-full">
                        <IftaLabel class="w-full" variant="in">
                            <InputText class="w-full" id="in_label" :value="settings.default_locale" />
                            <label for="in_label">Default locale</label>
                        </IftaLabel>
                    </div>

                    <div class="w-full flex flex-row gap-2 w-full">
                        <IftaLabel class="w-full" variant="in">
                            <InputText class="w-full" id="in_label" :value="settings.default_timezone" />
                            <label for="in_label">Default timezone</label>
                        </IftaLabel>
                    </div>

                    <div class="w-full flex flex-row gap-2 w-full">
                        <IftaLabel class="w-full" variant="in">
                            <InputText class="w-full" id="in_label" :value="settings.support_email" />
                            <label for="in_label">Support email</label>
                        </IftaLabel>
                    </div>

                    <div class="w-full flex flex-row gap-2 w-full">
                        <IftaLabel class="w-full" variant="in">
                            <InputNumber class="w-full" id="in_label" :modelValue="settings.max_user_accounts" />
                            <label for="in_label">Max user accounts</label>
                        </IftaLabel>
                    </div>

                    <div class="w-full flex flex-row gap-2 w-full align-items-center">
                        <label for="in_label">Allow signups</label>
                        <ToggleSwitch style="transform: scale(0.675)"
                                      v-model="settings.allow_signups"/>
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