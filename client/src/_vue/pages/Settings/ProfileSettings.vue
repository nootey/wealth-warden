<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import {useAuthStore} from "../../../services/stores/auth_store.ts";
import {onMounted, ref} from "vue";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import type {User} from "../../../models/user_models.ts";

const authStore = useAuthStore();
const toastStore = useToastStore();

const currentUser = ref<User>();

onMounted(async () => {
    await initUser();
})

async function initUser() {
    try {
        currentUser.value = await authStore.getAuthUser(false);
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
                    <h3>Profile</h3>
                    <h5 style="color: var(--text-secondary)">Customize how your account details.</h5>
                </div>

                <div class="flex flex-row gap-2 w-50" style="margin: 0 auto;">
                    <div class="flex flex-column gap-3 justify-content-center align-items-center">
                        <div class="w-8rem h-8rem border-circle border-1 surface-border flex align-items-center justify-content-center cursor-pointer">
                            <i class="pi pi-image text-2xl"></i>
                        </div>

                        <Button class="main-button" label="Upload photo" icon="pi pi-image"></Button>

                        <span style="color: var(--text-secondary)">JPG or PNG. 5MB max.</span>
                    </div>
                </div>


                <div v-if="currentUser" class="w-full flex flex-column gap-2 w-full">
                    <div class="w-full flex flex-row gap-2 w-full">
                        <IftaLabel class="w-full" variant="in">
                            <InputText class="w-full" id="in_label" :value="currentUser.email" />
                            <label for="in_label">Email</label>
                        </IftaLabel>
                    </div>
                    <div class="w-full flex flex-row gap-2 w-full">
                        <div class="flex flex-column flex-1 min-w-0">
                            <IftaLabel class="w-full" variant="in">
                                <InputText class="w-full" id="in_label" :value="currentUser.display_name" />
                                <label for="in_label">Display name</label>
                            </IftaLabel>
                        </div>
                    </div>
                    <div class="w-full flex flex-row gap-2 w-full">
                        <Button class="main-button ml-auto" label="Save"></Button>
                    </div>
                </div>
                <div v-else class="w-full flex flex-column gap-3">
                    <div class="w-full flex flex-row gap-2 w-full">
                        <Skeleton class="w-full" borderRadius="16px"></Skeleton>
                    </div>

                    <div class="w-full flex flex-row gap-2 w-full">
                        <div class="flex flex-column flex-1 min-w-0">
                            <Skeleton class="w-50" borderRadius="16px"></Skeleton>
                        </div>
                        <div class="flex flex-column flex-1 min-w-0">
                            <Skeleton class="w-50" borderRadius="16px"></Skeleton>
                        </div>
                    </div>
                </div>


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
                        <h5 style="color: var(--text-secondary)">Resetting your account will delete all your accounts, categories, and other data, but keep your user account intact.</h5>
                    </div>
                    <div class="flex flex-column w-3">
                        <Button size="small" label="Reset account" class="delete-button"></Button>
                    </div>
                </div>

                <div class="w-full flex flex-row gap-3 align-items-center">
                    <div class="flex flex-column w-full">
                        <h4>Delete account</h4>
                        <h5 style="color: var(--text-secondary)">Deleting your account will permanently remove all your data and cannot be undone.</h5>
                    </div>
                    <div class="flex flex-column w-3">
                        <Button size="small" label="Delete account" class="delete-button"></Button>
                    </div>
                </div>

            </div>
        </SettingsSkeleton>
    </div>
</template>

<style scoped>

</style>