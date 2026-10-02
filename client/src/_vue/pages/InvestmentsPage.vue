<script setup lang="ts">
import SegmentedTabs from "../components/layout/SegmentedTabs.vue";
import { useToastStore } from "../../services/stores/toast_store.ts";
import PageHeader from "../components/layout/PageHeader.vue";
import { onMounted, ref } from "vue";
import { usePermissions } from "../../utils/use_permissions.ts";
import InvestmentAssetForm from "../components/forms/InvestmentAssetForm.vue";
import InvestmentAssetsPaginated from "../components/data/InvestmentAssetsPaginated.vue";
import InvestmentTradeForm from "../components/forms/InvestmentTradeForm.vue";
import InvestmentTradesPaginated from "../components/data/InvestmentTradesPaginated.vue";
import InvestmentAllocationPanel from "../components/InvestmentAllocationPanel.vue";
import InvestmentReturnsPanel from "../components/InvestmentReturnsPanel.vue";
import InvestmentTaxBracketsPanel from "../components/InvestmentTaxBracketsPanel.vue";
import EmptyState from "../components/base/EmptyState.vue";
import ShowLoading from "../components/base/ShowLoading.vue";
import { useAccountStore } from "../../services/stores/account_store.ts";
import { useInvestmentStore } from "../../services/stores/investment_store.ts";
import type { Account } from "../../models/account_models.ts";

const toastStore = useToastStore();
const accountStore = useAccountStore();
const investmentStore = useInvestmentStore();

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

const hasInvestmentAccount = ref<boolean | null>(null);
const hasAssets = ref<boolean | null>(null);

onMounted(async () => {
  await Promise.all([checkInvestmentAccount(), checkHasAssets()]);
});

async function checkInvestmentAccount() {
  try {
    const all: Account[] = await accountStore.getAllAccounts(true, true);
    hasInvestmentAccount.value = all.some((a) =>
      ["investment", "crypto"].includes(a.account_type?.type),
    );
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function checkHasAssets() {
  try {
    hasAssets.value = (await investmentStore.getAssetCount()) > 0;
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

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
      await checkHasAssets();
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
      await checkHasAssets();
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

  <main class="flex flex-col w-full items-center">
    <div
      id="mobile-container"
      class="flex flex-col justify-center w-full gap-6"
    >
      <PageHeader
        eyebrow="Portfolio"
        title="Investments"
        description="A detailed look into your investments."
      >
        <template #actions>
          <Button
            v-if="hasInvestmentAccount"
            class="outline-button"
            @click="manipulateDialog('addAsset', true)"
          >
            <div class="flex flex-row gap-2 items-center">
              <i class="pi pi-plus" />
              <span><span class="mobile-hide">Add </span>asset</span>
            </div>
          </Button>
          <Button
            v-if="hasInvestmentAccount && hasAssets"
            class="main-button"
            @click="manipulateDialog('addTrade', true)"
          >
            <div class="flex flex-row gap-2 items-center">
              <i class="pi pi-plus" />
              <span><span class="mobile-hide">Add </span>trade</span>
            </div>
          </Button>
        </template>
      </PageHeader>

      <ShowLoading
        v-if="hasInvestmentAccount === null || hasAssets === null"
        :num-fields="5"
      />

      <EmptyState
        v-else-if="!hasInvestmentAccount"
        icon="pi pi-briefcase"
        title="No investment account yet."
        description="Create an investment or crypto account to start tracking assets."
      />

      <EmptyState
        v-else-if="!hasAssets"
        icon="pi pi-chart-line"
        title="No assets yet."
        description="Add an asset to start tracking your portfolio."
      />

      <SegmentedTabs
        v-else
        v-model="activeTab"
        :options="[
          { key: 'assets', label: 'Assets' },
          { key: 'trades', label: 'Trades' },
          { key: 'allocation', label: 'Allocation' },
          { key: 'returns', label: 'Return' },
          { key: 'tax', label: 'Tax' },
        ]"
      />

      <Transition
        v-if="hasInvestmentAccount && hasAssets"
        name="fade"
        mode="out-in"
      >
        <div
          v-if="activeTab === 'assets'"
          key="assets"
          class="flex flex-col justify-center w-full gap-4"
        >
          <div
            class="flex flex-col w-full p-4 gap-4 rounded-2xl"
            style="
              background-color: var(--background-secondary);
              border: 1px solid var(--border-color);
            "
          >
            <span class="font-bold">Assets</span>
            <InvestmentAssetsPaginated
              ref="holdRef"
              @update-asset="(id) => manipulateDialog('updateAsset', id)"
            />
          </div>
        </div>
        <div v-else-if="activeTab === 'trades'" key="trades" class="w-full">
          <div
            class="flex flex-col w-full p-4 gap-4 rounded-2xl"
            style="
              background-color: var(--background-secondary);
              border: 1px solid var(--border-color);
            "
          >
            <span class="font-bold">Trades</span>
            <InvestmentTradesPaginated
              ref="txnRef"
              @update-trade="(id) => manipulateDialog('updateTrade', id)"
            />
          </div>
        </div>
        <div
          v-else-if="activeTab === 'allocation'"
          key="allocation"
          class="w-full"
        >
          <InvestmentAllocationPanel />
        </div>
        <div v-else-if="activeTab === 'returns'" key="returns" class="w-full">
          <InvestmentReturnsPanel />
        </div>
        <div v-else key="tax" class="w-full">
          <InvestmentTaxBracketsPanel />
        </div>
      </Transition>
    </div>
  </main>
</template>

<style scoped></style>
