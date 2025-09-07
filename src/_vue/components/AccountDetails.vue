<script setup lang="ts">

import SettingsSkeleton from "./layout/SettingsSkeleton.vue";
import type {Account} from "../../models/account_models.ts";
import vueHelper from "../../utils/vue_helper.ts";
import {useTransactionStore} from "../../services/stores/transaction_store.ts";
import {computed} from "vue";
import {useToastStore} from "../../services/stores/toast_store.ts";
import TransactionsPaginated from "./TransactionsPaginated.vue";
import type {Column} from "../../services/filter_registry.ts";
import {useConfirm} from "primevue/useconfirm";

const props = defineProps<{
    account: Account;
}>();

const emit = defineEmits<{
    (event: 'closeAccount', id: number): void;
}>();

const toastStore = useToastStore();
const transactionStore = useTransactionStore();

const confirm = useConfirm();

const transactionColumns = computed<Column[]>(() => [
    { field: 'category', header: 'Category'},
    { field: 'amount', header: 'Amount'},
    { field: 'txn_date', header: 'Date'},
    { field: 'description', header: 'Description'},
]);

async function loadTransactionsPage({ page, rows, sort: s, filters: f, include_deleted }: any) {
    let response = null;

    try {
        response = await transactionStore.getPaginatedTransactionsForAccount(
            { rowsPerPage: rows, sort: s, filters: f, include_deleted },
            page,
            props.account.id!
        );
    } catch (e) {
        toastStore.errorResponseToast(e);
    }
    return { data: response?.data, total: response?.total_records };
}

async function confirmCloseAccount(id: number) {
    confirm.require({
        header: 'Confirm account close',
        message: 'You are about to close this account. This action is irreversible. Are you sure?',
        rejectProps: { label: 'Cancel' },
        acceptProps: { label: 'Close account', severity: 'danger' },
        accept: () => emit("closeAccount", id),
    });
}

</script>

<template>
    <div class="flex flex-column w-full gap-3">
        <div class="flex flex-row gap-2 align-items-center justify-content-between">
            <span>{{ account.name }}</span>
            <Button size="small" label="Close account" severity="danger" style="color: white;" @click="confirmCloseAccount(account.id!)"></Button>
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

        <SettingsSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-2">
                <div class="w-full flex flex-column gap-2">
                    <h3 style="color: var(--text-primary)">Activity</h3>
                </div>

                <div class="flex flex-row gap-2">
                    <TransactionsPaginated
                            ref="txRef"
                            :readOnly="true"
                            :columns="transactionColumns"
                            :fetchPage="loadTransactionsPage"
                            :rowClass="vueHelper.deletedRowClass"
                    />
                </div>

            </div>
        </SettingsSkeleton>
    </div>
</template>

<style scoped>

</style>