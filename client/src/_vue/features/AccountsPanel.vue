<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import Decimal from "decimal.js";
import AccountForm from "../components/forms/AccountForm.vue";
import { useAccountStore } from "../../services/stores/account_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { useSharedStore } from "../../services/stores/shared_store.ts";
import vueHelper from "../../utils/vue_helper.ts";
import type { Account } from "../../models/account_models.ts";
import AccountDetails from "../components/data/AccountDetails.vue";
import ShowLoading from "../components/base/ShowLoading.vue";
import {colorForAccountType} from "../../style/theme/accountColors.ts";
import {usePermissions} from "../../utils/use_permissions.ts";

const props = withDefaults(defineProps<{
    advanced?: boolean;
    allowEdit?: boolean;
    onToggle?: (acc: Account, nextValue: boolean) => Promise<boolean>;
    maxHeight?: number;
}>(), {
    advanced: false,
    allowEdit: true,
    onToggle: undefined,
    maxHeight: 75,
});

const emit = defineEmits<{
    (e: "refresh"): void;
    (e: "closeAccount", id: number): void;
}>();

const accountStore = useAccountStore();
const sharedStore = useSharedStore();
const toastStore = useToastStore();
const { hasPermission } = usePermissions();

const apiPrefix = "accounts";

const detailsModal = ref(false);
const updateModal = ref(false);
const selectedID = ref<number | null>(null);
const selectedAccount = ref<Account>();

const loading = ref(true);
const accounts = ref<Account[]>([]);

const rows = ref([25]);
const default_rows = ref(rows.value[0]);
const paginator = ref({
    total: 0,
    from: 0,
    to: 0,
    rowsPerPage: default_rows.value,
});
const page = ref(1);
const sort = ref({
    order: -1,
    field: 'opened_at'
});

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
    loading.value = true;
    if (new_page) page.value = new_page;

    try {
        const paginationResponse = await sharedStore.getRecordsPaginated(
            apiPrefix,
            { ...params.value, inactive: true }, // props.advanced
            page.value
        );
        accounts.value = paginationResponse.data;
        paginator.value.total = paginationResponse.total_records;
        paginator.value.to = paginationResponse.to;
        paginator.value.from = paginationResponse.from;
    } catch (error) {
        toastStore.errorResponseToast(error);
    } finally {
        loading.value = false;
    }
}

const logoColor = (type?: string) => colorForAccountType(type);

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
    // const activeAccounts = accounts.value.filter(a => a.is_active);

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
            if(!hasPermission("manage_data")) {
                toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
                return;
            }

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

async function handleEmit(type: string, data?: any) {
    switch (type) {
        case "completeOperation": {
            updateModal.value = false;
            await getData();
            emit("refresh");
            break;
        }
        case "closeAccount": {
            emit("closeAccount", data);
            detailsModal.value = false;
            break;
        }
    }
}

async function onToggleEnabled(acc: Account, nextValue: boolean) {
    const prev = !nextValue;
    if (props.onToggle) {
        const ok = await props.onToggle(acc, nextValue);
        if (!ok) acc.is_active = prev;
    }
}

defineExpose({ refresh: getData });

</script>

<template>
  <Dialog
    v-model:visible="updateModal"
    position="right"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Update account"
  >
    <AccountForm
      mode="update"
      :record-id="selectedID"
      @complete-operation="handleEmit('completeOperation')"
    />
  </Dialog>

  <Dialog
    v-model:visible="detailsModal"
    position="top"
    class="rounded-dialog"
    :breakpoints="{ '851px': '90vw' }"
    :modal="true"
    :style="{ width: '850px' }"
    header="Account details"
  >
    <AccountDetails
      :acc-i-d="selectedAccount?.id!"
      :advanced="advanced"
      @close-account="(id) => handleEmit('closeAccount', id)"
    />
  </Dialog>

  <div
    class="flex w-full p-3 gap-2 border-round-md bordered justify-content-between align-items-center"
    style="max-width: 1000px"
  >
    <div>
      <div
        class="text-xs"
        style="color: var(--text-secondary)"
      >
        Total
      </div>
      <div class="font-bold">
        {{ vueHelper.displayAsCurrency(totals.total) }}
      </div>
    </div>
    <div>
      <div
        class="text-xs"
        style="color: var(--text-secondary)"
      >
        Positive
      </div>
      <div
        class="font-bold"
        style="color: green"
      >
        {{ vueHelper.displayAsCurrency(totals.positive) }}
      </div>
    </div>
    <div>
      <div
        class="text-xs"
        style="color: var(--text-secondary)"
      >
        Negative
      </div>
      <div
        class="font-bold"
        style="color: red"
      >
        {{ vueHelper.displayAsCurrency(totals.negative) }}
      </div>
    </div>
  </div>

  <div
    class="flex-1 w-full border-round-md overflow-y-auto"
    :style="{ maxWidth: '1000px', maxHeight: `${maxHeight}vh` }"
  >
    <template v-if="loading">
      <ShowLoading :num-fields="10" />
    </template>

    <div
      v-else-if="groupedAccounts.length === 0"
      class="flex flex-row p-2 w-full justify-content-center"
    >
      <div class="flex flex-column gap-2 justify-content-center align-items-center">
        <i
          style="color: var(--text-secondary)"
          class="pi pi-eye-slash text-4xl"
        />
        <span>No accounts available</span>
      </div>
    </div>

    <div
      v-for="[type, group] in groupedAccounts"
      v-else
      :key="type"
      class="w-full p-3 mb-2 border-round-md"
      style="background: var(--background-primary)"
    >
      <div
        class="flex p-2 mb-2 pb-21 align-items-center justify-content-between"
        style="border-bottom: 1px solid var(--border-color)"
      >
        <div
          class="text-sm"
          style="color: var(--text-secondary)"
        >
          {{ vueHelper.formatString(type) }} · {{ group.length }}
        </div>
        <div
          class="font-bold text-sm"
          style="color: var(--text-secondary)"
        >
          {{ vueHelper.displayAsCurrency(groupTotal(group)) }}
        </div>
      </div>

      <div
        v-for="(account, i) in group"
        :key="account.id ?? i"
        class="account-row flex align-items-center justify-content-between p-2 border-round-md mt-1 bordered"
        :class="{ advanced, inactive: !account.is_active }"
      >
        <div class="flex align-items-center">
          <!-- Avatar -->
          <div
            class="flex align-items-center justify-content-center font-bold"
            :style="{
              width: '32px',
              height: '32px',
              border: '1px solid',
              borderColor: logoColor(account.account_type?.type).border,
              borderRadius: '50%',
              background: logoColor(account.account_type.type).bg,
              color: logoColor(account.account_type.type).fg,
            }"
          >
            {{ account.name.charAt(0).toUpperCase() }}
          </div>

          <!-- Name + subtype -->
          <div class="ml-2">
            <div
              class="font-bold clickable"
              @click="openModal('details', account)"
            >
              {{ account.name }}
            </div>

            <div
              class="text-sm"
              style="color: var(--text-secondary)"
            >
              {{ vueHelper.formatString(account.account_type?.sub_type) }}  {{ !account.is_active ? " - Inactive" : "" }}
            </div>
          </div>

          <!-- Edit icon -->
          <i
            v-if="hasPermission('manage_data') && account.is_active"
            v-tooltip="'Edit account'"
            class="ml-3 pi pi-pen-to-square text-xs hover-icon edit-icon"
            style="color: var(--text-secondary)"
            @click="openModal('update', account.id!)"
          />
        </div>

        <div class="flex align-items-center gap-2">
          <div class="font-bold mr-1">
            {{ vueHelper.displayAsCurrency(account.balance.end_balance) }}
          </div>

          <template v-if="advanced">
            <ToggleSwitch
              v-if="hasPermission('manage_data')"
              v-model="account.is_active"
              style="transform: scale(0.675)"
              @update:model-value="(v) => onToggleEnabled(account, v)"
            />
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

.account-row .font-bold.clickable:hover {
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
.account-row.inactive {
    filter: grayscale(100%);
    opacity: 0.6;
}

@media (max-width: 768px) {

    .account-row { padding: .5rem !important; }

    .account-row > .flex:first-child > div:first-child {
        width: 26px !important; height: 26px !important;
    }

    .account-row > .flex:first-child .font-bold {
        font-size: 0.8rem !important;
    }
    .account-row > .flex:first-child .text-sm {
        font-size: 0.7rem !important;
    }

    .account-row > .flex:last-child .font-bold {
        font-size: 0.85rem !important;
        white-space: nowrap !important;
    }

    .account-row .ml-2 { margin-left: .5rem !important; }
    .account-row .ml-3 { margin-left: .4rem !important; }

    .account-row .edit-icon { opacity: 1 !important; }
}
</style>
