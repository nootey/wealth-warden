<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import AccountsPanel from "../../features/AccountsPanel.vue";
import {useAccountStore} from "../../../services/stores/account_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import type {Account} from "../../../models/account_models.ts";
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {ref} from "vue";

const accountStore = useAccountStore();
const toastStore = useToastStore();
const sharedStore = useSharedStore();

const accRef = ref<InstanceType<typeof AccountsPanel> | null>(null);

async function toggleEnabled(acc: Account) {
    const previous = acc.is_active;

    try {
        const response = await accountStore.toggleActiveState(acc.id!);
        toastStore.successResponseToast(response);
    } catch (error) {
        acc.is_active = previous;
        toastStore.errorResponseToast(error);
    }
}

async function closeAccount(id: number) {
    try {
        let response = await sharedStore.deleteRecord(
            "accounts",
            id,
        );
        toastStore.successResponseToast(response);
        accRef.value?.refresh();
    } catch (error) {
        toastStore.errorResponseToast(error);
    }
}

</script>

<template>
    <div class="flex flex-column w-full gap-3">
        <SettingsSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-2">
                <div class="w-full flex flex-column gap-2">
                    <h3>Accounts</h3>
                    <h5 style="color: var(--text-secondary)">Manage administrative details for your accounts, like status and closure.</h5>
                </div>

                <AccountsPanel
                        ref="accRef"
                        :advanced="true"
                        :allowEdit="true"
                        @toggleEnabled="toggleEnabled"
                        @closeAccount="closeAccount"
                />
            </div>
        </SettingsSkeleton>
    </div>
</template>

<style scoped>

</style>