<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import AccountsPanel from "../../features/AccountsPanel.vue";

function toggleEnabled(acc) {

}
function viewHistory(acc) {

}
function closeAccount(acc) {

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
                        @view-history="viewHistory"
                        @close-account="closeAccount"
                />
            </div>
        </SettingsSkeleton>
    </div>
</template>

<style scoped>

</style>