<script setup lang="ts">
import type { Account, Balance } from "../../../models/account_models.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import { computed, nextTick, onMounted, ref } from "vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import TransactionsPaginated from "./TransactionsPaginated.vue";
import type { Column } from "../../../services/filter_registry.ts";
import { useConfirm } from "primevue/useconfirm";
import NetworthWidget from "../../features/widgets/NetworthWidget.vue";
import AccountBasicStats from "../../features/AccountBasicStats.vue";
import SlotSkeleton from "../layout/SlotSkeleton.vue";
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
const nWidgetRef = ref<InstanceType<typeof NetworthWidget> | null>(null);
const account = ref<Account | null>(null);
const projectionsModal = ref(false);
const latestBalance = ref<Balance | null>(null);

const { colors } = useChartColors();

const transactionColumns = computed<Column[]>(() => [
  { field: "category", header: "Category" },
  { field: "amount", header: "Amount" },
  { field: "txn_date", header: "Date" },
  { field: "description", header: "Description" },
]);

const expectedDifference = computed(() => {
  const expectedBalance = account.value?.expected_balance;
  const endBalance = latestBalance.value?.end_balance;

  if (!expectedBalance || !endBalance) {
    return null;
  }

  return new Decimal(endBalance).minus(expectedBalance).toString();
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
      "You are about to close this account. This action is irreversible. Are you sure?",
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

  <div v-if="account" class="flex flex-column w-full gap-3">
    <div class="flex flex-row gap-2 align-items-center text-center">
      <i
        :class="[
          'pi',
          account.account_type.classification === 'liability'
            ? 'pi-credit-card'
            : 'pi-wallet',
        ]"
      />
      <h3>{{ account.name }}</h3>
      <Tag
        :severity="!account.is_active ? 'secondary' : 'success'"
        style="transform: scale(0.8)"
      >
        {{ !account.is_active ? "Inactive" : "Active" }}
      </Tag>
      <Button
        v-if="advanced"
        size="small"
        label="Close account"
        class="delete-button"
        style="margin-left: auto"
        @click="confirmCloseAccount(account.id!)"
      >
        <div class="flex flex-row gap-1 align-items-center">
          <span> Close </span>
          <span class="mobile-hide"> account </span>
        </div>
      </Button>
    </div>

    <div
      v-if="!account.is_active"
      class="flex flex-row gap-2 align-items-center text-center pl-1"
    >
      <small style="color: var(--text-secondary)"
        >Account is inactive, some aspects will not be shown.</small
      >
    </div>

    <SlotSkeleton class="w-full" bg="opt">
      <div class="flex flex-column gap-2 p-3 w-full">
        <div class="flex flex-row gap-1 align-items-center">
          <h4>KPI</h4>
          ·
          <span style="color: var(--text-secondary)">{{
            account.currency
          }}</span>
        </div>
        <span>
          Start balance:
          <b
            >{{ vueHelper.displayAsCurrency(account.balance.start_balance) }}
          </b>
        </span>

        <span>
          Opened: <b>{{ dateHelper.formatDate(account.opened_at!, false) }} </b>
        </span>
        <span v-if="account.closed_at">
          Closed: <b>{{ dateHelper.formatDate(account.closed_at!, true) }} </b>
        </span>
      </div>
    </SlotSkeleton>

    <SlotSkeleton class="w-full" bg="opt">
      <div class="flex flex-column gap-2 p-3 w-full">
        <div class="flex flex-row gap-1 align-items-center">
          <h4>Details</h4>
          ·
          <span style="color: var(--text-secondary)">
            <Tag
              :severity="
                account.account_type.classification === 'liability'
                  ? 'danger'
                  : 'success'
              "
              style="transform: scale(0.8)"
            >
              {{ vueHelper.capitalize(account.account_type.classification) }}
            </Tag>
          </span>
        </div>
        <span>
          Type:
          <b
            >{{
              vueHelper.capitalize(
                vueHelper.denormalize(account.account_type.type),
              )
            }}
          </b>
        </span>
        <span>
          Subtype:
          <b>{{ vueHelper.capitalize(account.account_type.sub_type) }} </b>
        </span>
      </div>
    </SlotSkeleton>

    <SlotSkeleton v-if="account.is_active" class="w-full" bg="opt">
      <div class="flex flex-column gap-2 p-3 w-full">
        <div class="flex flex-row gap-1 align-items-center text-center">
          <h4>Projections</h4>
          ·
          <i
            v-tooltip="'Edit account projections'"
            class="pi pi-pen-to-square hover-icon text-xs"
            @click="openModal('editProjection')"
          />
        </div>
        <span>
          Expected balance:
          <b> {{ vueHelper.displayAsCurrency(account.expected_balance!) }} </b>
        </span>
        <span>
          Difference:
          <b :style="{ color: differenceColor }">
            {{ vueHelper.displayAsCurrency(expectedDifference) }}
          </b>
        </span>
      </div>
    </SlotSkeleton>

    <Divider />

    <SlotSkeleton class="w-full">
      <NetworthWidget
        ref="nWidgetRef"
        :account-id="account.id"
        :chart-height="200"
      />
    </SlotSkeleton>

    <div v-if="account.is_active" class="w-full flex flex-column gap-2">
      <h3 style="color: var(--text-primary)">Stats</h3>
    </div>
    <SlotSkeleton v-if="account.is_active" class="w-full">
      <AccountBasicStats :acc-i-d="account.id" :pie-chart-size="250" />
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
            :acc-i-d="accID"
            :read-only="true"
            :columns="transactionColumns"
          />
        </div>
      </div>
    </SlotSkeleton>

    <SlotSkeleton class="w-full">
      <div class="w-full flex flex-column gap-3 p-3">
        <div class="w-full flex flex-column gap-2">
          <h4 style="color: var(--text-primary)">Transfers</h4>
        </div>

        <div class="flex flex-row gap-2">
          <TransfersPaginated ref="trRef" :acc-i-d="accID" />
        </div>
      </div>
    </SlotSkeleton>
  </div>
  <ShowLoading v-else :num-fields="7" />
</template>

<style scoped></style>
