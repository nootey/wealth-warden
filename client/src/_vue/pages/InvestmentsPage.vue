<script setup lang="ts">
import { useToastStore } from "../../services/stores/toast_store.ts";
import { ref } from "vue";
import { usePermissions } from "../../utils/use_permissions.ts";
import InvestmentForm from "../components/forms/InvestmentForm.vue";
import InvestmentHoldingsPaginated from "../components/data/InvestmentHoldingsPaginated.vue";
import InvestmentTransactionForm from "../components/forms/InvestmentTransactionForm.vue";
import InvestmentTransactionsPaginated from "../components/data/InvestmentTransactionsPaginated.vue";

const toastStore = useToastStore();

const { hasPermission } = usePermissions();

const holdRef = ref<InstanceType<typeof InvestmentHoldingsPaginated> | null>(
  null,
);
const txnRef = ref<InstanceType<typeof InvestmentTransactionsPaginated> | null>(
  null,
);

const createHoldingModal = ref(false);
const updateHoldingModal = ref(false);
const updateHoldingID = ref(null);

const createTxnModal = ref(false);
const updateTxnModal = ref(false);
const updateTxnID = ref(null);

function manipulateDialog(modal: string, value: any) {
  switch (modal) {
    case "addHolding": {
      if (!hasPermission("manage_data")) {
        toastStore.createInfoToast(
          "Access denied",
          "You don't have permission to perform this action.",
        );
        return;
      }
      createHoldingModal.value = value;
      break;
    }
    case "updateHolding": {
      if (!hasPermission("manage_data")) {
        toastStore.createInfoToast(
          "Access denied",
          "You don't have permission to perform this action.",
        );
        return;
      }
      updateHoldingModal.value = true;
      updateHoldingID.value = value;
      break;
    }
    case "addTransaction": {
      if (!hasPermission("manage_data")) {
        toastStore.createInfoToast(
          "Access denied",
          "You don't have permission to perform this action.",
        );
        return;
      }
      createTxnModal.value = value;
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
      updateTxnModal.value = true;
      updateTxnID.value = value;
      break;
    }
    default: {
      break;
    }
  }
}

async function handleEmit(emitType: any) {
  switch (emitType) {
    case "completeHoldingOperation": {
      createHoldingModal.value = false;
      updateHoldingModal.value = false;
      holdRef.value?.refresh();
      break;
    }
    case "completeTxnOperation": {
      createTxnModal.value = false;
      updateTxnModal.value = false;
      holdRef.value?.refresh();
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
    v-model:visible="createHoldingModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Add asset"
  >
    <InvestmentForm
      mode="create"
      @complete-operation="handleEmit('completeHoldingOperation')"
    />
  </Dialog>

  <Dialog
    v-model:visible="updateHoldingModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Asset details"
  >
    <InvestmentForm
      mode="update" :record-id="updateHoldingID"
      @complete-operation="handleEmit('completeHoldingOperation')"
    />
  </Dialog>

  <Dialog
    v-model:visible="createTxnModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Add transaction"
  >
    <InvestmentTransactionForm
      mode="create"
      @complete-operation="handleEmit('completeTxnOperation')"
    />
  </Dialog>

  <Dialog
    v-model:visible="updateTxnModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Transaction details"
  >
    <InvestmentTransactionForm
      mode="update" :record-id="updateTxnID"
      @complete-operation="handleEmit('completeTxnOperation')"
    />
  </Dialog>

  <main class="flex flex-column w-full p-2 align-items-center" style="height: 100%">
    <div
      id="mobile-container"
      class="flex flex-column justify-content-center p-3 w-full gap-3 border-round-md"
      style="
        border: 1px solid var(--border-color);
        background: var(--background-secondary);
      "
    >
      <div
        class="flex flex-row align-items-center text-center gap-2 w-full"
      >
        <div style="font-weight: bold" class="mr-auto">Investments</div>
        <Button
          class="main-button"
          @click="manipulateDialog('addHolding', true)"
        >
          <div class="flex flex-row gap-1 align-items-center">
            <i class="pi pi-plus" />
            <span class="mobile-hide"> Add </span>
            <span> Holding </span>
          </div>
        </Button>
        <Button
          class="main-button"
          @click="manipulateDialog('addTransaction', true)"
        >
          <div class="flex flex-row gap-1 align-items-center">
            <i class="pi pi-plus" />
            <span class="mobile-hide"> Add </span>
            <span> Transaction </span>
          </div>
        </Button>
      </div>

      <div id="mobile-row" class="flex flex-row w-full">
        <InvestmentHoldingsPaginated ref="holdRef"
          @update-holding="(id) => manipulateDialog('updateHolding', id)"
        />
      </div>

      <label>Transactions</label>
      <div id="mobile-row" class="flex flex-row w-full">
        <InvestmentTransactionsPaginated ref="txnRef"
          @update-transaction="(id) => manipulateDialog('updateTransaction', id)"/>
      </div>

    </div>
  </main>
</template>

<style scoped></style>
