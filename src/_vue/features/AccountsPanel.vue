<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import Decimal from "decimal.js";
import AccountForm from "../components/forms/AccountForm.vue";
import { useAccountStore } from "../../services/stores/account_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { useSharedStore } from "../../services/stores/shared_store.ts";
import vueHelper from "../../utils/vue_helper.ts";
import filterHelper from "../../utils/filter_helper.ts";
import type { Account } from "../../models/account_models.ts";
import AccountDetails from "../components/AccountDetails.vue";

const props = withDefaults(defineProps<{
    advanced?: boolean;
    allowEdit?: boolean;
}>(), {
    advanced: false,
    allowEdit: true,
});

const emit = defineEmits<{
    (e: "toggle-enabled", account: Account): void;
    (e: "refresh"): void;
}>();

const accountStore = useAccountStore();
const sharedStore = useSharedStore();
const toastStore = useToastStore();

const apiPrefix = "accounts";

const detailsModal = ref(false);
const updateModal = ref(false);
const selectedID = ref<number | null>(null);
const selectedAccount = ref<Account>();

const loadingAccounts = ref(true);
const accounts = ref<Account[]>([]);

const rows = ref([10, 25]);
const default_rows = ref(rows.value[0]);
const paginator = ref({
    total: 0,
    from: 0,
    to: 0,
    rowsPerPage: default_rows.value,
});
const page = ref(1);
const sort = ref(filterHelper.initSort());

const params = computed(() => ({
    rowsPerPage: paginator.value.rowsPerPage,
    sort: sort.value,
    filters: [],
}));

onMounted(async () => {
    await accountStore.getAccountTypes();
    await getData();
});

async function getData(new_page: number | null = null) {
    loadingAccounts.value = true;
    if (new_page) page.value = new_page;

    try {
        const paginationResponse = await sharedStore.getRecordsPaginated(
            apiPrefix,
            { ...params.value },
            page.value
        );
        accounts.value = paginationResponse.data;
        paginator.value.total = paginationResponse.total_records;
        paginator.value.to = paginationResponse.to;
        paginator.value.from = paginationResponse.from;
    } catch (error) {
        toastStore.errorResponseToast(error);
    } finally {
        loadingAccounts.value = false;
    }
}

const typeColors: Record<string, { bg: string; fg: string }> = {
    cash: { bg: "#9b59b6", fg: "#6c3483" },
    investment: { bg: "#2980b9", fg: "#1c5980" },
    crypto: { bg: "#16a085", fg: "#0d6655" },
    property: { bg: "#8e44ad", fg: "#5b2c6f" },
    vehicle: { bg: "#3498db", fg: "#21618c" },
    other_asset: { bg: "#7d3c98", fg: "#4a235a" },
    credit_card: { bg: "#e74c3c", fg: "#922b21" },
    loan: { bg: "#e67e22", fg: "#9a531c" },
    other_liability: { bg: "#f1c40f", fg: "#9a7d0a" },
};
const logoColor = (type: string) => typeColors[type] ?? { bg: "#444", fg: "#222" };

const typeMap: Record<string, string> = {};

accountStore.accountTypes.forEach(t => {
    typeMap[t.type] = t.classification;
});

const groupedAccounts = computed(() => {
    const groups = new Map<string, typeof accounts.value>();
    for (const acc of accounts.value) {
        const t = acc.account_type?.type || "other_asset";
        if (!groups.has(t)) groups.set(t, []);
        groups.get(t)!.push(acc);
    }

    return Array.from(groups.entries())
        .sort(([typeA], [typeB]) => {
            const ca = typeMap[typeA] ?? "asset";
            const cb = typeMap[typeB] ?? "asset";
            if (ca !== cb) return ca === "asset" ? -1 : 1;
            return typeA.localeCompare(typeB);
        });
});

const groupTotal = (group: Account[]) =>
    group.reduce((sum, acc) => sum.add(new Decimal(acc.balance.end_balance || 0)), new Decimal(0));

const totals = computed(() => {
    const vals = accounts.value.map(a => new Decimal(a.balance.end_balance || 0));
    const total = vals.reduce((s, v) => s.add(v), new Decimal(0));
    const positive = vals.reduce((s, v) => (v.greaterThan(0) ? s.add(v) : s), new Decimal(0));
    const negative = vals.reduce((s, v) => (v.lessThan(0) ? s.add(v) : s), new Decimal(0));
    return {
        total: total.toString(),
        positive: positive.toString(),
        negative: negative.toString(),
    };
});

function openModal(type: string, data: any) {

    switch (type) {
        case "update": {
            if (!props.allowEdit) return;
            updateModal.value = true;
            selectedID.value = data;
            break;
        }

        case "details": {
            detailsModal.value = true;
            selectedAccount.value = data;
            break;
        }

    }

}

async function handleEmit(emitType: string) {
    if (emitType === "completeOperation") {
        updateModal.value = false;
        await getData();
        emit("refresh");
    }
}

function onToggleEnabled(acc: Account) {
    emit("toggle-enabled", acc);
}

function handlePrimaryClick(acc: Account) {
    if (props.advanced) {
        openModal("details", acc);
    }
    // non-advanced: do nothing
}

defineExpose({ refresh: getData });

</script>

<template>

    <Dialog position="right" class="rounded-dialog" v-model:visible="updateModal"
            :breakpoints="{ '501px': '90vw' }" :modal="true" :style="{ width: '500px' }" header="Update account">
        <AccountForm mode="update" :recordId="selectedID"
                @completeOperation="handleEmit('completeOperation')"/>
    </Dialog>

    <Dialog position="top" class="rounded-dialog" v-model:visible="detailsModal"
            :breakpoints="{ '851px': '90vw' }" :modal="true" :style="{ width: '850px' }" header="Account details">
        <AccountDetails :account="selectedAccount"></AccountDetails>
    </Dialog>

    <div class="flex w-full p-3 gap-2 border-round-md bordered justify-content-between align-items-center"
         style="max-width: 1000px">
        <div>
            <div class="text-xs" style="color: var(--text-secondary)">Total</div>
            <div class="font-bold">{{ vueHelper.displayAsCurrency(totals.total) }}</div>
        </div>
        <div>
            <div class="text-xs" style="color: var(--text-secondary)">Positive</div>
            <div class="font-bold" style="color: green">
                {{ vueHelper.displayAsCurrency(totals.positive) }}
            </div>
        </div>
        <div>
            <div class="text-xs" style="color: var(--text-secondary)">Negative</div>
            <div class="font-bold" style="color: red">
                {{ vueHelper.displayAsCurrency(totals.negative) }}
            </div>
        </div>
    </div>

    <div class="flex-1 w-full border-round-md p-2 bordered overflow-y-auto" style="max-width: 1000px;" >
        <div v-for="[type, group] in groupedAccounts" :key="type"
             class="w-full p-3 mb-2 border-round-md"
             style="background: var(--background-primary)">
            <div class="flex p-2 mb-2 pb-21 align-items-center justify-content-between"
                 style="border-bottom: 1px solid var(--border-color)">
                <div class="text-sm" style="color: var(--text-secondary)">
                    {{ vueHelper.formatString(type) }} · {{ group.length }}
                </div>
                <div class="font-bold text-sm" style="color: var(--text-secondary)">
                    {{ vueHelper.displayAsCurrency(groupTotal(group)) }}
                </div>
            </div>

            <div v-for="(account, i) in group" :key="account.id ?? i"
                 class="account-row flex align-items-center justify-content-between p-2 border-round-md mt-1 bordered"
                 :class="{ advanced }">

                <div class="flex align-items-center">
                    <!-- Avatar -->
                    <div class="flex align-items-center justify-content-center font-bold"
                         :style="{
                                    width: '32px',
                                    height: '32px',
                                    border: '1px solid',
                                    borderColor: logoColor(account.account_type.type).fg,
                                    borderRadius: '50%',
                                    background: logoColor(account.account_type.type).bg,
                                    color: logoColor(account.account_type.type).fg,
                                }">
                        {{ account.name.charAt(0).toUpperCase() }}
                    </div>

                    <!-- Name + subtype -->
                    <div class="ml-2">
                        <div class="font-bold"
                             :class="{ clickable: advanced }"
                             @click="handlePrimaryClick(account)">
                            {{ account.name }}
                        </div>

                        <div class="text-sm" style="color: var(--text-secondary)">
                            {{ vueHelper.formatString(account.account_type?.sub_type) }}
                        </div>
                    </div>

                    <!-- Edit icon -->
                    <i class="ml-3 pi pi-pen-to-square text-xs hover-icon edit-icon"
                       style="color: var(--text-secondary)"
                       @click="openModal('update', account.id!)"
                       v-tooltip="'Edit account'" />

                </div>

                <div class="flex align-items-center gap-2">
                    <div class="font-bold mr-1">
                        {{ vueHelper.displayAsCurrency(account.balance.end_balance) }}
                    </div>

                    <template v-if="advanced">
                        <ToggleSwitch style="transform: scale(0.675)" v-model="account.is_active" @update:modelValue="onToggleEnabled(account)" />
                    </template>

                </div>
            </div>

        </div>
    </div>


</template>

<style scoped>

.bordered {
    border: 1px solid var(--border-color);
    background: var(--background-secondary);
}

.clickable { cursor: pointer; }

.account-row.advanced .font-bold.clickable:hover {
    text-decoration: underline;
}

.account-row .edit-icon {
    opacity: 0;
    transition: opacity .15s ease;
}
.account-row:hover .edit-icon {
    opacity: 1;
}

.account-row.advanced .edit-icon {
    opacity: 1;
}
</style>
