<script setup lang="ts">
import { useToastStore } from "../../services/stores/toast_store.ts";
import { ref } from "vue";
import { usePermissions } from "../../utils/use_permissions.ts";
import InvestmentAssetForm from "../components/forms/InvestmentAssetForm.vue";
import InvestmentAssetsPaginated from "../components/data/InvestmentAssetsPaginated.vue";
import InvestmentTradeForm from "../components/forms/InvestmentTradeForm.vue";
import InvestmentTradesPaginated from "../components/data/InvestmentTradesPaginated.vue";
import SlotSkeleton from "../components/layout/SlotSkeleton.vue";
import YearlyBreakdownStats from "../features/YearlyBreakdownStats.vue";
import AccountBasicStats from "../features/AccountBasicStats.vue";

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

const activeTab = ref("assets");

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
    :breakpoints="{ '651px': '90vw' }"
    :modal="true"
    :style="{ width: '650px' }"
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

  <main class="flex flex-column w-full align-items-center">
    <div
      id="mobile-container"
      class="flex flex-column justify-content-center w-full gap-3 border-round-xl"
    >
      <div
        class="w-full flex flex-row justify-content-between p-1 gap-2 align-items-center"
      >
        <div class="w-full flex flex-column gap-2">
          <div style="font-weight: bold">Investments</div>
          <div>Comprehensive insights into your investment vehicles.</div>
        </div>
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

      <div class="flex flex-row gap-3 p-2">
        <div
          class="cursor-pointer pb-1"
          style="color: var(--text-secondary)"
          :style="
            activeTab === 'assets'
              ? 'color: var(--text-primary); border-bottom: 2px solid var(--text-primary)'
              : ''
          "
          @click="activeTab = 'assets'"
        >
          Assets
        </div>
        <div
          class="cursor-pointer pb-1"
          style="color: var(--text-secondary)"
          :style="
            activeTab === 'trades'
              ? 'color: var(--text-primary); border-bottom: 2px solid var(--text-primary)'
              : ''
          "
          @click="activeTab = 'trades'"
        >
          Trades
        </div>
      </div>

      <Transition name="fade" mode="out-in">
        <div
          v-if="activeTab === 'assets'"
          key="assets"
          class="flex flex-column justify-content-center w-full gap-3"
        >
          <Panel :collapsed="false" header="Assets">
            <div id="mobile-row" class="flex flex-row w-full">
              <InvestmentAssetsPaginated
                ref="holdRef"
                @update-asset="(id) => manipulateDialog('updateAsset', id)"
              />
            </div>
          </Panel>
        </div>
        <div v-else key="trades" class="w-full">
          <Panel :collapsed="false" header="Trades">
            <div id="mobile-row" class="flex flex-row w-full">
              <InvestmentTradesPaginated
                ref="txnRef"
                @update-trade="(id) => manipulateDialog('updateTrade', id)"
              />
            </div>
          </Panel>
        </div>
      </Transition>
    </div>
  </main>
</template>

<style scoped></style>
