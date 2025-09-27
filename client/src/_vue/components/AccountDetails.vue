<script setup lang="ts">

import type {Account} from "../../models/account_models.ts";
import vueHelper from "../../utils/vue_helper.ts";
import {useTransactionStore} from "../../services/stores/transaction_store.ts";
import {computed, ref} from "vue";
import {useToastStore} from "../../services/stores/toast_store.ts";
import TransactionsPaginated from "./data/TransactionsPaginated.vue";
import type {Column} from "../../services/filter_registry.ts";
import {useConfirm} from "primevue/useconfirm";
import NetworthWidget from "./widgets/NetworthWidget.vue";
import BasicStats from "./charts/BasicStats.vue";
import SlotSkeleton from "./layout/SlotSkeleton.vue";

const props = defineProps<{
    account: Account;
    advanced: boolean;
}>();

const emit = defineEmits<{
    (event: 'closeAccount', id: number): void;
}>();

const toastStore = useToastStore();
const transactionStore = useTransactionStore();

const confirm = useConfirm();
const nWidgetRef = ref<InstanceType<typeof NetworthWidget> | null>(null);

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
            <h3>{{ account.name }}</h3>
            <Button v-if="advanced" size="small" label="Close account" severity="danger" style="color: white;" @click="confirmCloseAccount(account.id!)"></Button>
        </div>

        <SlotSkeleton class="w-full">
          <NetworthWidget ref="nWidgetRef" :accountId="account.id" :chartHeight="200"/>
        </SlotSkeleton>

        <SlotSkeleton class="w-full">
            <BasicStats :accID="account.id" />
        </SlotSkeleton>

        <SlotSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-3">
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
        </SlotSkeleton>
    </div>
</template>

<style scoped>

</style>