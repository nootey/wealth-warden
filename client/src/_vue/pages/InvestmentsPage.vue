<script setup lang="ts">
import { useToastStore } from "../../services/stores/toast_store.ts";
import { ref } from "vue";
import { usePermissions } from "../../utils/use_permissions.ts";
import InvestmentAssetForm from "../components/forms/InvestmentAssetForm.vue";
import InvestmentAssetsPaginated from "../components/data/InvestmentAssetsPaginated.vue";
import InvestmentTradeForm from "../components/forms/InvestmentTradeForm.vue";
import InvestmentTradesPaginated from "../components/data/InvestmentTradesPaginated.vue";

const toastStore = useToastStore();

const { hasPermission } = usePermissions();

const holdRef = ref<InstanceType<typeof InvestmentAssetsPaginated> | null>(
  null,
);
const txnRef = ref<InstanceType<typeof InvestmentTradesPaginated> | null>(null);

const createAssetModal = ref(false);
const updateAssetModal = ref(false);
const updateAssetID = ref(null);

const createTxnModal = ref(false);
const updateTxnModal = ref(false);
const updateTxnID = ref(null);

function manipulateDialog(modal: string, value: any) {
  switch (modal) {
    case "addAsset": {
      if (!hasPermission("manage_data")) {
        toastStore.createInfoToast(
          "Access denied",
          "You don't have permission to perform this action.",
        );
        return;
      }
      createAssetModal.value = value;
      break;
    }
    case "updateAsset": {
      if (!hasPermission("manage_data")) {
        toastStore.createInfoToast(
          "Access denied",
          "You don't have permission to perform this action.",
        );
        return;
      }
      updateAssetModal.value = true;
      updateAssetID.value = value;
      break;
    }
    case "addTrade": {
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
    case "updateTrade": {
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
    case "completeAssetOperation": {
      createAssetModal.value = false;
      updateAssetModal.value = false;
      holdRef.value?.refresh();
      break;
    }
    case "completeTxnOperation": {
      createTxnModal.value = false;
      updateTxnModal.value = false;
      holdRef.value?.refresh();
      txnRef.value?.refresh();
      break;
    }
    case "completeTxnDelete": {
      updateTxnModal.value = false;
      holdRef.value?.refresh();
      txnRef.value?.refresh();
      break;
    }
    case "completeAssetDelete": {
      updateAssetModal.value = false;
      holdRef.value?.refresh();
      txnRef.value?.refresh();
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
    v-model:visible="createAssetModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Add asset"
  >
    <InvestmentAssetForm
      mode="create"
      @complete-operation="handleEmit('completeAssetOperation')"
    />
  </Dialog>

  <Dialog
    v-model:visible="updateAssetModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Asset details"
  >
    <InvestmentAssetForm
      mode="update"
      :record-id="updateAssetID"
      @complete-operation="handleEmit('completeAssetOperation')"
      @complete-delete="handleEmit('completeAssetDelete')"
    />
  </Dialog>

  <Dialog
    v-model:visible="createTxnModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Add Trade"
  >
    <InvestmentTradeForm
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
    header="Trade details"
  >
    <InvestmentTradeForm
      mode="update"
      :record-id="updateTxnID"
      @complete-operation="handleEmit('completeTxnOperation')"
      @complete-delete="handleEmit('completeTxnDelete')"
    />
  </Dialog>

  <main
    class="flex flex-column w-full p-2 align-items-center"
    style="height: 100%"
  >
    <div
      id="mobile-container"
      class="flex flex-column justify-content-center p-3 w-full gap-3 border-round-md"
      style="
        border: 1px solid var(--border-color);
        background: var(--background-secondary);
      "
    >
      <div class="flex flex-row align-items-center text-center gap-2 w-full">
        <div style="font-weight: bold" class="mr-auto">Investments</div>
        <Button class="main-button" @click="manipulateDialog('addAsset', true)">
          <div class="flex flex-row gap-1 align-items-center">
            <i class="pi pi-plus" />
            <span class="mobile-hide"> Add </span>
            <span> Asset </span>
          </div>
        </Button>
        <Button class="main-button" @click="manipulateDialog('addTrade', true)">
          <div class="flex flex-row gap-1 align-items-center">
            <i class="pi pi-plus" />
            <span class="mobile-hide"> Add </span>
            <span> Trade </span>
          </div>
        </Button>
      </div>

      <div id="mobile-row" class="flex flex-row w-full">
        <InvestmentAssetsPaginated
          ref="holdRef"
          @update-asset="(id) => manipulateDialog('updateAsset', id)"
        />
      </div>

      <div style="font-weight: bold">Trades</div>
      <div id="mobile-row" class="flex flex-row w-full">
        <InvestmentTradesPaginated
          ref="txnRef"
          @update-trade="(id) => manipulateDialog('updateTrade', id)"
        />
      </div>
    </div>
  </main>
</template>

<style scoped></style>
