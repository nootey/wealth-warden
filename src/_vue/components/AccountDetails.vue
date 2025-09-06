<script setup lang="ts">

import SettingsSkeleton from "./layout/SettingsSkeleton.vue";
import type {Account} from "../../models/account_models.ts";
import vueHelper from "../../utils/vue_helper.ts";

const props = defineProps<{
    account: Account;
}>();

</script>

<template>
    <div class="flex flex-column w-full gap-3">
        <div class="flex flex-row gap-2 align-items-center">
            <span>{{ account.name }}</span>
        </div>
        <SettingsSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-2">
                <div class="w-full flex flex-column gap-2" style="height: 300px;">
                    <h5 style="color: var(--text-secondary)">Balance</h5>
                    <h3 style="color: var(--text-primary)">{{ vueHelper.displayAsCurrency(account.balance.end_balance)}}</h3>
                    <span>Work in progress ...</span>
                </div>
            </div>
        </SettingsSkeleton>
    </div>
</template>

<style scoped>

</style>