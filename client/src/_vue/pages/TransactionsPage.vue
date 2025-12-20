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
const includeDeleted = ref(false);

const categories = computed<Category[]>(() => transactionStore.categories);
const accounts = computed<Account[]>(() => accountStore.accounts);
const trTemplateCount = ref<number>(0);



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

  <main
    class="flex flex-column w-full p-2 align-items-center"
    style="height: 100%"
  >
    <div id="mobile-container"
      class="flex flex-column justify-content-center p-3 w-full gap-3 border-round-md"
      style="
        border: 1px solid var(--border-color);
        background: var(--background-secondary);
        max-width: 1000px;
      ">
      <div class="flex flex-row justify-content-between align-items-center text-center gap-2 w-full">
        <div style="font-weight: bold">Transactions</div>
        <i
          v-if="hasPermission('manage_data')"
          v-tooltip="'Go to categories settings.'"
          class="pi pi-external-link hover-icon mr-auto text-sm"
          @click="router.push('settings/categories')"
        />
        <Button
          class="outline-button"
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
          class="main-button"
          @click="manipulateDialog('addTransaction', true)"
        >
          <div class="flex flex-row gap-1 align-items-center">
            <i class="pi pi-plus" />
            <span> New </span>
            <span class="mobile-hide"> Transaction </span>
          </div>
        </Button>
      </div>

      <div id="mobile-row" class="flex flex-row w-full">
        <TransactionsPaginated
          ref="txRef"
          :read-only="false"
          :columns="activeColumns"
          @row-click="(id) => manipulateDialog('updateTransaction', id)"
        />
      </div>

      <label>Transfers</label>
      <div id="mobile-row" class="flex flex-row w-full">
        <TransfersPaginated ref="trRef" />
      </div>
    </div>
  </main>
</template>

<style scoped></style>
