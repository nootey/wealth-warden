<script setup lang="ts">
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
import TransactionTemplates from "../features/TransactionTemplates.vue";

const toastStore = useToastStore();
const transactionStore = useTransactionStore();
const accountStore = useAccountStore();

onMounted(async () => {
  await transactionStore.getCategories();
  await accountStore.getAllAccounts(false, true);
  await getTrTemplateCount();
});

const router = useRouter();
const { hasPermission } = usePermissions();

const trRef = ref<InstanceType<typeof TransfersPaginated> | null>(null);
const txRef = ref<InstanceType<typeof TransactionsPaginated> | null>(null);
const tpRef = ref<InstanceType<typeof TransactionTemplates> | null>(null);

const createModal = ref(false);
const updateModal = ref(false);
const templateModal = ref(false);
const updateTransactionID = ref(null);

const categories = computed<Category[]>(() => transactionStore.categories);
const accounts = computed<Account[]>(() => accountStore.accounts);
const trTemplateCount = ref<number>(0);

const activeTab = ref("transactions");

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
    hideOnMobile: true,
  },
  { field: "amount", header: "Amount", type: "number" },
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
      break;
    }
    case "completeTrOperation": {
      createModal.value = false;
      trRef.value?.refresh();
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
    :breakpoints="{ '901px': '90vw' }"
    :modal="true"
    :style="{ width: '900px' }"
    header="Transaction templates"
  >
    <TransactionTemplates
      ref="tpRef"
      @refresh-template-count="handleEmit('refreshTemplateCount')"
    />
  </Dialog>

  <main class="flex flex-column w-full align-items-center">
    <div
      id="mobile-container"
      class="flex flex-column justify-content-center w-full gap-3 border-round-xl"
    >
      <div
        class="w-full flex flex-row justify-content-between p-1 gap-2 align-items-center"
      >
        <div class="w-full flex flex-column gap-2">
          <div class="flex flex-row gap-2 align-items-center w-full">
            <div style="font-weight: bold">Activity</div>
            <i
              v-if="hasPermission('manage_data')"
              v-tooltip="'Go to categories settings.'"
              class="pi pi-external-link hover-icon mr-auto text-sm"
              @click="router.push('settings/categories')"
            />
          </div>
          <div>A complete record of your financial activity.</div>
        </div>
        <Button
          class="outline-button w-3"
          @click="manipulateDialog('openTemplateView', true)"
        >
          <div class="flex flex-row gap-1 align-items-center">
            <i class="pi pi-database" />
            <span
              ><span class="mobile-hide"> Templates </span>
              {{ "(" + trTemplateCount + ")" }}</span
            >
          </div>
        </Button>
        <Button
          class="main-button w-3"
          @click="manipulateDialog('addTransaction', true)"
        >
          <div class="flex flex-row gap-1 align-items-center">
            <i class="pi pi-plus" />
            <span> New </span>
            <span class="mobile-hide"> Transaction </span>
          </div>
        </Button>
      </div>

      <div class="flex flex-row gap-3 p-2">
        <div
          class="cursor-pointer pb-1"
          style="color: var(--text-secondary)"
          :style="
            activeTab === 'transactions'
              ? 'color: var(--text-primary); border-bottom: 2px solid var(--text-primary)'
              : ''
          "
          @click="activeTab = 'transactions'"
        >
          Transactions
        </div>
        <div
          class="cursor-pointer pb-1"
          style="color: var(--text-secondary)"
          :style="
            activeTab === 'transfers'
              ? 'color: var(--text-primary); border-bottom: 2px solid var(--text-primary)'
              : ''
          "
          @click="activeTab = 'transfers'"
        >
          Transfers
        </div>
      </div>

      <Transition name="fade" mode="out-in">
        <div
          v-if="activeTab === 'transactions'"
          key="transactions"
          class="flex flex-column justify-content-center w-full gap-3"
        >
          <div
            class="flex flex-column w-full p-3 gap-3 border-round-2xl"
            style="
              background-color: var(--background-secondary);
              border: 1px solid var(--border-color);
            "
          >
            <span class="font-bold">Transactions</span>
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
            class="flex flex-column w-full p-3 gap-3 border-round-2xl"
            style="
              background-color: var(--background-secondary);
              border: 1px solid var(--border-color);
            "
          >
            <span class="font-bold">Transfers</span>
            <TransfersPaginated ref="trRef" />
          </div>
        </div>
      </Transition>
    </div>
  </main>
</template>

<style scoped></style>
