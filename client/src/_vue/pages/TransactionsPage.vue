<script setup lang="ts">
import SegmentedTabs from "../components/layout/SegmentedTabs.vue";
import TransactionForm from "../components/forms/TransactionForm.vue";
import { computed, onMounted, ref } from "vue";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { useTransactionStore } from "../../services/stores/transaction_store.ts";
import type { Category } from "../../models/transaction_models.ts";
import type { Column } from "../../services/filter_registry.ts";
import { useAccountStore } from "../../services/stores/account_store.ts";
import type { Account } from "../../models/account_models.ts";
import TransfersPaginated from "../components/data/TransfersPaginated.vue";
import TransactionsPaginated from "../components/data/TransactionsPaginated.vue";
import { useRouter } from "vue-router";
import { usePermissions } from "../../utils/use_permissions.ts";
import TransactionTemplatesPaginated from "../components/data/TransactionTemplatesPaginated.vue";
import PageHeader from "../components/layout/PageHeader.vue";
import EmptyState from "../components/base/EmptyState.vue";
import ShowLoading from "../components/base/ShowLoading.vue";

const toastStore = useToastStore();
const transactionStore = useTransactionStore();
const accountStore = useAccountStore();

onMounted(async () => {
  await transactionStore.getCategories();
  await accountStore.getAllAccounts(false, true);
  await getTrTemplateCount();
  await checkHasRecords();
});

const router = useRouter();
const { hasPermission } = usePermissions();

const trRef = ref<InstanceType<typeof TransfersPaginated> | null>(null);
const txRef = ref<InstanceType<typeof TransactionsPaginated> | null>(null);

const createModal = ref(false);
const updateModal = ref(false);
const templateModal = ref(false);
const updateTransactionID = ref(null);

const categories = computed<Category[]>(() => transactionStore.categories);
const accounts = computed<Account[]>(() => accountStore.accounts);
const trTemplateCount = ref<number>(0);
const hasRecords = ref<boolean | null>(null);

const activeTab = ref("transactions");

const directionOptions = [
  { label: "Income", value: "income" },
  { label: "Expense", value: "expense" },
];

const activeColumns = computed<Column[]>(() => [
  {
    field: "account",
    header: "Account",
    type: "enum",
    options: accounts.value,
    optionLabel: "name",
  },
  {
    field: "category",
    header: "Category",
    type: "enum",
    options: categories.value,
    optionLabel: "name",
    optionHint: "classification",
    hideOnMobile: true,
  },
  { field: "amount", header: "Amount", type: "number" },
  {
    field: "direction",
    header: "Direction",
    type: "enum",
    options: directionOptions,
    optionLabel: "label",
    optionValue: "value",
    hideOnMobile: true,
  },
  { field: "txn_date", header: "Date", type: "date" },
  {
    field: "description",
    header: "Description",
    type: "text",
    hideOnMobile: true,
  },
]);

async function getTrTemplateCount() {
  try {
    let res = await transactionStore.getTransactionTemplateCount();
    trTemplateCount.value = res.data;
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function checkHasRecords() {
  try {
    const [txCount, trCount] = await Promise.all([
      transactionStore.getTransactionCount(),
      transactionStore.getTransferCount(),
    ]);
    hasRecords.value = txCount > 0 || trCount > 0;
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

function manipulateDialog(modal: string, value: any) {
  switch (modal) {
    case "addTransaction": {
      if (!hasPermission("manage_data")) {
        toastStore.createInfoToast(
          "Access denied",
          "You don't have permission to perform this action.",
        );
        return;
      }
      createModal.value = value;
      break;
    }
    case "openTemplateView": {
      if (!hasPermission("manage_data")) {
        toastStore.createInfoToast(
          "Access denied",
          "You don't have permission to perform this action.",
        );
        return;
      }
      templateModal.value = value;
      break;
    }
    case "updateTransaction": {
      if (!hasPermission("manage_data")) {
        toastStore.createInfoToast(
          "Access denied",
          "You don't have permission to perform this action.",
        );
        return;
      }
      updateModal.value = true;
      updateTransactionID.value = value;
      break;
    }
    default: {
      break;
    }
  }
}

async function handleEmit(emitType: any) {
  switch (emitType) {
    case "completeTxOperation": {
      createModal.value = false;
      updateModal.value = false;
      txRef.value?.refresh();
      await checkHasRecords();
      break;
    }
    case "completeTrOperation": {
      createModal.value = false;
      trRef.value?.refresh();
      await checkHasRecords();
      break;
    }
    case "refreshTemplateCount": {
      await getTrTemplateCount();
      break;
    }
    case "deleteTxn": {
      createModal.value = false;
      updateModal.value = false;
      txRef.value?.refresh();
      await checkHasRecords();
      break;
    }
    default: {
      break;
    }
  }
}
</script>

<template>
  <Dialog
    v-model:visible="createModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Add transaction"
  >
    <TransactionForm
      mode="create"
      @complete-tx-operation="handleEmit('completeTxOperation')"
      @complete-tr-operation="handleEmit('completeTrOperation')"
    />
  </Dialog>

  <Dialog
    v-model:visible="updateModal"
    position="right"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Transaction details"
  >
    <TransactionForm
      mode="update"
      :record-id="updateTransactionID"
      @complete-tx-operation="handleEmit('completeTxOperation')"
      @complete-tx-delete="handleEmit('deleteTxn')"
    />
  </Dialog>

  <Dialog
    v-model:visible="templateModal"
    class="rounded-dialog"
    :breakpoints="{ '1001px': '90vw' }"
    :modal="true"
    :style="{ width: '1000px' }"
    header="Transaction templates"
  >
    <TransactionTemplatesPaginated
      @refresh-template-count="handleEmit('refreshTemplateCount')"
    />
  </Dialog>

  <main class="flex flex-col w-full items-center">
    <div
      id="mobile-container"
      class="flex flex-col justify-center w-full gap-6"
    >
      <PageHeader
        eyebrow="Ledger"
        title="Transactions"
        description="A complete record of your financial activity."
      >
        <template #title-suffix>
          <button
            v-if="hasPermission('manage_data')"
            v-tooltip="'Go to categories settings.'"
            type="button"
            class="size-8 grid place-items-center rounded-lg text-muted hover:bg-sunken hover:text-ink cursor-pointer"
            @click="router.push({ name: 'settings.categories' })"
          >
            <i class="pi pi-external-link text-sm" />
          </button>
        </template>
        <template #actions>
          <Button
            v-if="hasRecords"
            class="outline-button"
            @click="manipulateDialog('openTemplateView', true)"
          >
            <div class="flex flex-row gap-2 items-center">
              <i class="pi pi-clone" />
              <span class="mobile-hide">Templates</span>
              <span
                class="rounded-full bg-sunken px-1.5 text-xs font-medium tabular-nums"
              >
                {{ trTemplateCount }}
              </span>
            </div>
          </Button>
          <Button
            class="main-button"
            @click="manipulateDialog('addTransaction', true)"
          >
            <div class="flex flex-row gap-2 items-center">
              <i class="pi pi-plus" />
              <span>New<span class="mobile-hide"> transaction</span></span>
            </div>
          </Button>
        </template>
      </PageHeader>

      <ShowLoading v-if="hasRecords === null" :num-fields="5" />

      <EmptyState
        v-else-if="!hasRecords"
        icon="pi pi-receipt"
        title="No transactions yet."
        description="Add your first transaction to start tracking your activity."
      />

      <SegmentedTabs
        v-else
        v-model="activeTab"
        :options="[
          { key: 'transactions', label: 'Transactions' },
          { key: 'transfers', label: 'Transfers' },
        ]"
      />

      <Transition v-if="hasRecords" name="fade" mode="out-in">
        <div
          v-if="activeTab === 'transactions'"
          key="transactions"
          class="flex flex-col justify-center w-full gap-4"
        >
          <div
            class="flex flex-col w-full p-4 gap-4 rounded-2xl border border-line bg-card shadow-[var(--shadow-card)]"
          >
            <TransactionsPaginated
              ref="txRef"
              :read-only="false"
              :columns="activeColumns"
              @row-click="(id) => manipulateDialog('updateTransaction', id)"
            />
          </div>
        </div>
        <div v-else key="transfers" class="w-full">
          <div
            class="flex flex-col w-full p-4 gap-4 rounded-2xl border border-line bg-card shadow-[var(--shadow-card)]"
          >
            <TransfersPaginated ref="trRef" />
          </div>
        </div>
      </Transition>
    </div>
  </main>
</template>

<style scoped></style>
