<script setup lang="ts">
import type {
  AccountBalance,
  AccountWithOpening,
} from "../../../models/account_models.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import { computed, nextTick, onMounted, ref } from "vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import TransactionsPaginated from "./TransactionsPaginated.vue";
import type { Column } from "../../../services/filter_registry.ts";
import { useConfirm } from "primevue/useconfirm";
import NetworthWidget from "../../features/widgets/NetworthWidget.vue";
import AccountBasicStats from "../../features/AccountBasicStats.vue";
import dateHelper from "../../../utils/date_helper.ts";
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import ShowLoading from "../base/ShowLoading.vue";
import Decimal from "decimal.js";
import { useChartColors } from "../../../style/theme/chartColors.ts";
import AccountProjectionForm from "../forms/AccountProjectionForm.vue";
import { useAccountStore } from "../../../services/stores/account_store.ts";
import TransfersPaginated from "./TransfersPaginated.vue";

const props = defineProps<{
  accID: number;
  advanced: boolean;
}>();

const emit = defineEmits<{
  (event: "closeAccount", id: number): void;
}>();

const toastStore = useToastStore();
const sharedStore = useSharedStore();
const accountStore = useAccountStore();

const confirm = useConfirm();
const account = ref<AccountWithOpening | null>(null);
const projectionsModal = ref(false);
const latestBalance = ref<AccountBalance | null>(null);

const { colors } = useChartColors();

const transactionColumns = computed<Column[]>(() => [
  { field: "category", header: "Category" },
  { field: "amount", header: "Amount" },
  { field: "txn_date", header: "Date" },
  { field: "description", header: "Description" },
]);

const expectedDifference = computed(() => {
  const expectedBalance = account.value?.expected_balance;
  const currentBalance = latestBalance.value?.total_balance;

  if (!expectedBalance || !currentBalance) {
    return null;
  }

  return new Decimal(currentBalance).minus(expectedBalance).toString();
});

const differenceColor = computed(() => {
  if (!expectedDifference.value) {
    return colors.value.dim;
  }

  const diff = new Decimal(expectedDifference.value);

  if (diff.isZero()) {
    return colors.value.dim;
  }

  return diff.isPositive() ? colors.value.pos : colors.value.neg;
});

onMounted(async () => {
  await accountStore.syncBalances().catch(() => {});
  await loadRecord(props.accID);
  await loadLatestBalance(props.accID);
});

async function loadRecord(id: number) {
  try {
    account.value = await sharedStore.getRecordByID("accounts", id, {
      initial_balance: true,
    });

    await nextTick();
  } catch (err) {
    toastStore.errorResponseToast(err);
  }
}

async function loadLatestBalance(id: number) {
  try {
    latestBalance.value = await accountStore.getLatestBalance(id);
  } catch (err) {
    toastStore.errorResponseToast(err);
  }
}

async function confirmCloseAccount(id: number) {
  confirm.require({
    header: "Confirm account close",
    message:
      "You are about to close this account. The balance must be zero first, so move the money out with a transfer, a withdrawal, or a correction. This action is irreversible. Are you sure?",
    rejectProps: { label: "Cancel" },
    acceptProps: { label: "Close account", severity: "danger" },
    accept: () => emit("closeAccount", id),
  });
}

function openModal(type: string) {
  switch (type) {
    case "editProjection": {
      projectionsModal.value = true;
      break;
    }
  }
}

async function handleEmit(type: string) {
  switch (type) {
    case "completeOperation": {
      projectionsModal.value = false;
      await loadRecord(props.accID);
      break;
    }
  }
}
</script>

<template>
  <Dialog
    v-model:visible="projectionsModal"
    position="right"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Edit account projections"
  >
    <AccountProjectionForm
      :acc-i-d="accID"
      @complete-operation="handleEmit('completeOperation')"
    />
  </Dialog>

  <div v-if="account" class="flex flex-col w-full gap-4">
    <div class="flex flex-row items-center gap-3">
      <div
        class="flex items-center justify-center w-10 h-10 shrink-0 rounded-xl bg-sunken text-muted"
      >
        <i
          :class="[
            'pi',
            account.account_type.classification === 'liability'
              ? 'pi-credit-card'
              : 'pi-wallet',
          ]"
        />
      </div>
      <div class="flex flex-col gap-1 min-w-0">
        <span class="text-lg font-medium tracking-tight text-ink truncate">
          {{ account.name }}
        </span>
        <div class="flex flex-row flex-wrap items-center gap-2 text-xs">
          <span
            class="rounded-full px-2 py-0.5 font-medium"
            :class="
              account.is_active ? 'bg-sunken text-gain' : 'bg-sunken text-muted'
            "
          >
            {{ account.is_active ? "Active" : "Inactive" }}
          </span>
          <span
            class="rounded-full px-2 py-0.5 font-medium bg-sunken"
            :class="
              account.account_type.classification === 'liability'
                ? 'text-loss'
                : 'text-gain'
            "
          >
            {{ vueHelper.capitalize(account.account_type.classification) }}
          </span>
          <span class="text-faint">{{ account.currency }}</span>
          <span
            v-if="
              ['investment', 'crypto', 'other_asset'].includes(
                account.account_type.type,
              )
            "
            class="text-muted"
          >
            · Cash
            <span class="font-medium text-ink">
              {{ vueHelper.displayAsCurrency(latestBalance?.balance ?? null) }}
            </span>
          </span>
        </div>
      </div>
      <Button
        v-if="advanced"
        size="small"
        label="Close account"
        class="delete-button ml-auto"
        @click="confirmCloseAccount(account.id!)"
      >
        <div class="flex flex-row gap-1 items-center">
          <span> Close </span>
          <span class="mobile-hide"> account </span>
        </div>
      </Button>
    </div>

    <span v-if="!account.is_active" class="text-xs text-muted">
      Account is inactive, some aspects will not be shown.
    </span>

    <div id="account-tiles" class="grid grid-cols-3 gap-2">
      <div class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3 min-w-0">
        <span class="label">Type</span>
        <span class="text-base font-medium tracking-tight text-ink truncate">
          {{
            vueHelper.capitalize(
              vueHelper.denormalize(account.account_type.type),
            )
          }}
          <span class="text-muted">
            · {{ vueHelper.capitalize(account.account_type.sub_type) }}
          </span>
        </span>
      </div>
      <div class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3 min-w-0">
        <span class="label">Start balance</span>
        <span class="text-base font-medium tracking-tight text-ink truncate">
          {{ vueHelper.displayAsCurrency(account.start_balance) }}
        </span>
      </div>
      <div class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3 min-w-0">
        <span class="label">Opened</span>
        <span class="text-base font-medium tracking-tight text-ink truncate">
          {{ dateHelper.formatDate(account.opened_at!, false) }}
        </span>
      </div>
      <div
        v-if="account.closed_at"
        class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3 min-w-0"
      >
        <span class="label">Closed</span>
        <span class="text-base font-medium tracking-tight text-ink truncate">
          {{ dateHelper.formatDate(account.closed_at!, true) }}
        </span>
      </div>
    </div>

    <section
      v-if="account.is_active"
      class="flex flex-col gap-3 rounded-2xl border border-line p-4"
    >
      <div class="flex flex-row items-center justify-between">
        <span class="label">Projections</span>
        <button
          v-tooltip="'Edit account projections'"
          type="button"
          class="flex items-center justify-center w-7 h-7 rounded-lg text-muted hover:text-ink hover:bg-sunken transition-colors cursor-pointer"
          aria-label="Edit account projections"
          @click="openModal('editProjection')"
        >
          <i class="pi pi-pen-to-square text-xs" />
        </button>
      </div>
      <div class="grid grid-cols-2 gap-2">
        <div class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3 min-w-0">
          <span class="text-xs text-muted">Expected balance</span>
          <span class="text-base font-medium tracking-tight text-ink truncate">
            {{ vueHelper.displayAsCurrency(account.expected_balance!) }}
          </span>
        </div>
        <div class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3 min-w-0">
          <span class="text-xs text-muted">Difference</span>
          <span
            class="text-base font-medium tracking-tight truncate"
            :style="{ color: differenceColor }"
          >
            {{ vueHelper.displayAsCurrency(expectedDifference) }}
          </span>
        </div>
      </div>
    </section>

    <section class="flex flex-col rounded-2xl border border-line p-4">
      <NetworthWidget
        :account-id="account.id"
        title="Balance"
        :chart-height="200"
      />
    </section>

    <section
      v-if="account.is_active"
      class="flex flex-col gap-2 rounded-2xl border border-line p-4"
    >
      <span class="label">Stats</span>
      <AccountBasicStats :acc-i-d="account.id" :pie-chart-size="250" />
    </section>

    <section class="flex flex-col gap-3 rounded-2xl border border-line p-4">
      <span class="label">Transactions</span>
      <TransactionsPaginated
        ref="txRef"
        :acc-i-d="accID"
        :read-only="true"
        :columns="transactionColumns"
      />
    </section>

    <section class="flex flex-col gap-3 rounded-2xl border border-line p-4">
      <span class="label">Transfers</span>
      <TransfersPaginated ref="trRef" :acc-i-d="accID" :read-only="true" />
    </section>
  </div>
  <ShowLoading v-else :num-fields="7" />
</template>

<style scoped>
@media (max-width: 640px) {
  #account-tiles {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
