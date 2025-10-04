<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import AccountsPanel from "../../features/AccountsPanel.vue";
import {useAccountStore} from "../../../services/stores/account_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import type {Account} from "../../../models/account_models.ts";
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {ref} from "vue";
import {usePermissions} from "../../../utils/use_permissions.ts";

const accountStore = useAccountStore();
const toastStore = useToastStore();
const sharedStore = useSharedStore();
const { hasPermission } = usePermissions();

const accRef = ref<InstanceType<typeof AccountsPanel> | null>(null);

async function toggleEnabled(acc: Account, nextValue: boolean): Promise<boolean> {
    const previous = acc.is_active;
    acc.is_active = nextValue;
    try {
        const response = await accountStore.toggleActiveState(acc.id!);
        toastStore.successResponseToast(response);
        return true;
    } catch (error) {
        // add a small delay for the toggle animation to complete
        await new Promise(resolve => setTimeout(resolve, 300));
        acc.is_active = previous;
        toastStore.errorResponseToast(error);
        return false;
    }
}

async function closeAccount(id: number) {

    if(!hasPermission("manage_data")) {
        toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
        return;
    }

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
            <div id="main-col" class="w-full flex flex-column gap-3 p-2">
                <div class="w-full flex flex-column gap-2">
                    <h3>Accounts</h3>
                    <h5 style="color: var(--text-secondary)">Manage administrative details for your accounts, like status and closure.</h5>
                </div>

                <AccountsPanel
                        ref="accRef"
                        :advanced="true"
                        :allowEdit="true"
                        :onToggle="toggleEnabled"
                        :maxHeight="70"
                        @closeAccount="closeAccount"
                />
            </div>
        </SettingsSkeleton>
    </div>
</template>

<style scoped>
@media (max-width: 768px) {
    #main-col {
        padding: 0 !important;
    }
}
</style>