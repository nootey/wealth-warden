<script setup lang="ts">

import type {Account} from "../../../models/account_models.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import {useTransactionStore} from "../../../services/stores/transaction_store.ts";
import {computed, nextTick, onMounted, ref} from "vue";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import TransactionsPaginated from "./TransactionsPaginated.vue";
import type {Column} from "../../../services/filter_registry.ts";
import {useConfirm} from "primevue/useconfirm";
import NetworthWidget from "../../features/NetworthWidget.vue";
import AccountBasicStats from "../../features/AccountBasicStats.vue";
import SlotSkeleton from "../layout/SlotSkeleton.vue";
import dateHelper from "../../../utils/date_helper.ts";
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import ShowLoading from "../base/ShowLoading.vue";
import Decimal from "decimal.js";
import {useChartColors} from "../../../style/theme/chartColors.ts";

const props = defineProps<{
    accID: number;
    advanced: boolean;
}>();

onMounted(async () => {
    await loadRecord(props.accID);
})

const emit = defineEmits<{
    (event: 'closeAccount', id: number): void;
}>();

const toastStore = useToastStore();
const transactionStore = useTransactionStore();
const sharedStore = useSharedStore();

const confirm = useConfirm();
const nWidgetRef = ref<InstanceType<typeof NetworthWidget> | null>(null);
const account = ref<Account | null>(null);

const { colors } = useChartColors();

const transactionColumns = computed<Column[]>(() => [
    { field: 'category', header: 'Category'},
    { field: 'amount', header: 'Amount'},
    { field: 'created_at', header: 'Date'},
    { field: 'description', header: 'Description'},
]);

const expectedDifference = computed(() => {
    const expectedBalance = account.value?.expected_balance
    const endBalance = account.value?.balance?.end_balance

    if (!expectedBalance || !endBalance) {
        return null
    }

    return new Decimal(endBalance).minus(expectedBalance).toString()
})

const differenceColor = computed(() => {
    if (!expectedDifference.value) {
        return colors.value.dim
    }

    const diff = new Decimal(expectedDifference.value)

    if (diff.isZero()) {
        return colors.value.dim
    }

    return diff.isPositive() ? colors.value.pos : colors.value.neg
})

async function loadRecord(id: number) {
    try {
        account.value = await sharedStore.getRecordByID("accounts", id, {initial_balance: true});

        await nextTick();

    } catch (err) {
        toastStore.errorResponseToast(err);
    }
}

async function loadTransactionsPage({ page, rows, sort: s, filters: f, include_deleted }: any) {
    let response = null;

    try {
        response = await transactionStore.getPaginatedTransactionsForAccount(
            { rowsPerPage: rows, sort: s, filters: f, include_deleted },
            page,
            props.accID!
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
    <div v-if="account" class="flex flex-column w-full gap-3">
        <div class="flex flex-row gap-2 align-items-center text-center">
            <i :class="['pi', account.account_type.classification === 'liability' ? 'pi-credit-card' : 'pi-wallet']">
            </i>
            <h3>{{ account.name }}</h3>
            <Tag :severity="!account.is_active ? 'secondary' : 'success'" style="transform: scale(0.8)">
                {{ !account.is_active ? 'Inactive' : 'Active' }}
            </Tag>
            <Button v-if="advanced" size="small"
                    label="Close account" class="delete-button" style="margin-left: auto;"
                    @click="confirmCloseAccount(account.id!)">
                    <div class="flex flex-row gap-1 align-items-center">
                        <span> Close </span>
                        <span class="mobile-hide"> account </span>
                    </div>
            </Button>
        </div>

        <SlotSkeleton class="w-full" bg="opt">
            <div class="flex flex-column gap-2 p-3 w-full">
                <h4>KPI</h4>
                <span> Start balance: <b>{{ vueHelper.displayAsCurrency(account.balance.start_balance) }} </b> </span>
                <span> Currency: <b>{{ account.currency }} </b> </span>
                <span> Opened: <b>{{ dateHelper.formatDate(account.opened_at!, false) }} </b> </span>
                <span v-if="account.closed_at"> Closed: <b>{{ dateHelper.formatDate(account.closed_at!, true) }} </b> </span>
            </div>
        </SlotSkeleton>

        <SlotSkeleton class="w-full" bg="opt">
            <div class="flex flex-column gap-2 p-3 w-full">
                <h4>Details</h4>
                <span> Type: <b>{{ vueHelper.capitalize(vueHelper.denormalize(account.account_type.type)) }} </b> </span>
                <span> Subtype: <b>{{ vueHelper.capitalize(account.account_type.sub_type) }} </b> </span>
                <span> Classification:
                    <Tag :severity="account.account_type.classification === 'liability' ? 'danger' : 'success'" style="transform: scale(0.8)">
                        {{ vueHelper.capitalize(account.account_type.classification) }}
                    </Tag>
                </span>
            </div>
        </SlotSkeleton>

        <SlotSkeleton class="w-full" bg="opt">
            <div class="flex flex-column gap-2 p-3 w-full">
                <h4>Projections</h4>
                <span> Expectation type: <b> {{ account.balance_projection }} </b> </span>
                <span> Expected balance: <b> {{ vueHelper.displayAsCurrency(account.expected_balance! )}} </b> </span>
                <span> Difference:
                    <b :style="{ color: differenceColor }">
                    {{  vueHelper.displayAsCurrency(expectedDifference) }}
                    </b>
                </span>
            </div>
        </SlotSkeleton>

        <Divider />

        <SlotSkeleton class="w-full">
          <NetworthWidget ref="nWidgetRef" :accountId="account.id" :chartHeight="200"/>
        </SlotSkeleton>

        <div class="w-full flex flex-column gap-2">
            <h3 style="color: var(--text-primary)">Stats</h3>
        </div>
        <SlotSkeleton class="w-full">
            <AccountBasicStats :accID="account.id" :pieChartSize="250" />
        </SlotSkeleton>

        <div class="w-full flex flex-column gap-2">
            <h3 style="color: var(--text-primary)">Activity</h3>
        </div>
        <SlotSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-3">
                <div class="w-full flex flex-column gap-2">
                    <h4 style="color: var(--text-primary)">Transactions</h4>
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
    <ShowLoading v-else :numFields="7" />
</template>

<style scoped>

</style>