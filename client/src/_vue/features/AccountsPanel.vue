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
import EmptyState from "../components/base/EmptyState.vue";
import { colorForAccountType } from "../../style/theme/accountColors.ts";
import { usePermissions } from "../../utils/use_permissions.ts";

const props = withDefaults(
  defineProps<{
    advanced?: boolean;
    allowEdit?: boolean;
    onToggle?: (acc: Account, nextValue: boolean) => Promise<boolean>;
    maxHeight?: number;
  }>(),
  {
    advanced: false,
    allowEdit: true,
    onToggle: undefined,
    maxHeight: 75,
  },
);

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
  field: "opened_at",
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
      page.value,
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

accountStore.accountTypes.forEach((t) => {
  typeMap[t.type] = t.classification;
});

const groupedAccounts = computed(() => {
  const groups = new Map<string, typeof accounts.value>();
  const inactive: typeof accounts.value = [];
  for (const acc of accounts.value) {
    if (!acc.is_active) {
      inactive.push(acc);
      continue;
    }
    const t = acc.account_type?.type || "other_asset";
    if (!groups.has(t)) groups.set(t, []);
    groups.get(t)!.push(acc);
  }

  const sorted = Array.from(groups.entries()).sort(([typeA], [typeB]) => {
    const ca = typeMap[typeA] ?? "asset";
    const cb = typeMap[typeB] ?? "asset";
    if (ca !== cb) return ca === "asset" ? -1 : 1;
    return typeA.localeCompare(typeB);
  });

  if (inactive.length > 0) sorted.push(["inactive", inactive]);
  return sorted;
});

const groupTotal = (group: Account[]) =>
  group.reduce(
    (sum, acc) => sum.add(new Decimal(acc.balance.total_balance || 0)),
    new Decimal(0),
  );

const totals = computed(() => {
  const vals = accounts.value.map(
    (a) => new Decimal(a.balance.total_balance || 0),
  );
  const total = vals.reduce((s, v) => s.add(v), new Decimal(0));
  const positive = vals.reduce(
    (s, v) => (v.greaterThan(0) ? s.add(v) : s),
    new Decimal(0),
  );
  const negative = vals.reduce(
    (s, v) => (v.lessThan(0) ? s.add(v) : s),
    new Decimal(0),
  );

  return {
    total: total.toString(),
    positive: positive.toString(),
    negative: negative.toString(),
  };
});

function openModal(type: string, data: any) {
  switch (type) {
    case "update": {
      if (!hasPermission("manage_data")) {
        toastStore.createInfoToast(
          "Access denied",
          "You don't have permission to perform this action.",
        );
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

const hasAccounts = computed(() => accounts.value.length > 0);

defineExpose({ refresh: getData, hasAccounts });
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

  <div class="flex flex-col w-full gap-4">
    <div
      v-if="advanced && (loading || hasAccounts)"
      id="balance-row"
      class="grid grid-cols-3 w-full gap-2"
      style="max-width: 1000px"
    >
      <div class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3">
        <span class="text-xs text-muted">Total</span>
        <span class="text-base font-medium tracking-tight text-ink">
          {{ vueHelper.displayAsCurrency(totals.total) }}
        </span>
      </div>
      <div class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3">
        <span class="text-xs text-muted">Positive</span>
        <span class="text-base font-medium tracking-tight text-gain">
          {{ vueHelper.displayAsCurrency(totals.positive) }}
        </span>
      </div>
      <div class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3">
        <span class="text-xs text-muted">Negative</span>
        <span class="text-base font-medium tracking-tight text-loss">
          {{ vueHelper.displayAsCurrency(totals.negative) }}
        </span>
      </div>
    </div>

    <div
      class="flex-1 w-full rounded-xl overflow-y-auto"
      :style="{ maxWidth: '1000px', maxHeight: `${maxHeight}vh` }"
    >
      <template v-if="loading">
        <ShowLoading :num-fields="10" />
      </template>

      <EmptyState
        v-else-if="groupedAccounts.length === 0"
        icon="pi pi-wallet"
        title="No accounts yet."
        description="Create an account to start tracking your balances."
      />

      <TransitionGroup v-else name="list-anim" tag="div" class="relative">
        <div
          v-for="[type, group] in groupedAccounts"
          :key="type"
          class="w-full mb-4 rounded-2xl border border-line bg-card p-4 shadow-[var(--shadow-card)]"
        >
          <div class="flex mx-1 items-baseline justify-between gap-2">
            <div class="flex items-baseline gap-2">
              <span class="label">{{ vueHelper.formatString(type) }}</span>
              <span class="text-xs text-faint">{{ group.length }}</span>
            </div>
            <span class="text-sm font-medium tracking-tight text-ink">
              {{ vueHelper.displayAsCurrency(groupTotal(group)) }}
            </span>
          </div>

          <TransitionGroup name="list-anim" tag="div" class="relative">
            <div
              v-for="(account, i) in group"
              :key="account.id ?? i"
              class="account-row flex items-center justify-between px-3 py-2.5 rounded-xl mt-2 bg-sunken"
              :class="{ advanced, inactive: !account.is_active }"
            >
              <div class="flex items-center">
                <!-- Avatar -->
                <div
                  class="flex items-center justify-center font-bold"
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
                    class="font-medium text-ink clickable"
                    @click="openModal('details', account)"
                  >
                    {{ account.name }}
                  </div>

                  <div class="text-xs text-muted">
                    {{ vueHelper.formatString(account.account_type?.sub_type) }}
                    {{ !account.is_active ? " - Inactive" : "" }}
                  </div>
                </div>

                <!-- Edit icon -->
                <i
                  v-if="hasPermission('manage_data') && account.is_active"
                  v-tooltip="'Edit account'"
                  class="ml-4 pi pi-pen-to-square text-xs text-muted hover:text-ink hover-icon edit-icon"
                  @click="openModal('update', account.id!)"
                />
              </div>

              <div class="flex items-center gap-2">
                <div class="font-medium tracking-tight text-ink mr-1">
                  {{
                    vueHelper.displayAsCurrency(account.balance.total_balance)
                  }}
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
          </TransitionGroup>
        </div>
      </TransitionGroup>
    </div>
  </div>
</template>

<style scoped>
.clickable {
  cursor: pointer;
}

.account-row .font-medium.clickable:hover {
  text-decoration: underline;
}

.account-row .edit-icon {
  opacity: 0;
  transition: opacity 0.15s ease;
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

.list-anim-move,
.list-anim-enter-active,
.list-anim-leave-active {
  transition: all 0.35s ease;
}

.list-anim-enter-from,
.list-anim-leave-to {
  opacity: 0;
  transform: translateY(6px);
}

.list-anim-leave-active {
  position: absolute;
  left: 0;
  right: 0;
}

@media (max-width: 768px) {
  #balance-row {
    font-size: 75%;
  }
  .account-row {
    padding: 0.5rem !important;
  }

  .account-row > .flex:first-child > div:first-child {
    width: 26px !important;
    height: 26px !important;
  }

  .account-row > .flex:first-child .font-medium {
    font-size: 0.8rem !important;
  }
  .account-row > .flex:first-child .text-xs {
    font-size: 0.7rem !important;
  }

  .account-row > .flex:last-child .font-medium {
    font-size: 0.85rem !important;
    white-space: nowrap !important;
  }

  .account-row .ml-2 {
    margin-left: 0.5rem !important;
  }
  .account-row .ml-3 {
    margin-left: 0.4rem !important;
  }

  .account-row .edit-icon {
    opacity: 1 !important;
  }
}
</style>
