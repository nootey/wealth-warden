<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import AccountsPanel from "../../features/AccountsPanel.vue";
import {useAccountStore} from "../../../services/stores/account_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import type {Account} from "../../../models/account_models.ts";

const accountStore = useAccountStore();
const toastStore = useToastStore();

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
                        :advanced="true"
                        :allowEdit="true"
                        @toggle-enabled="toggleEnabled"
                />
            </div>
        </SettingsSkeleton>
    </div>
</template>

<style scoped>

</style>