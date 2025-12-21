<script setup lang="ts">
import { useSharedStore } from "../../services/stores/shared_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { ref } from "vue";
import { usePermissions } from "../../utils/use_permissions.ts";
import InvestmentForm from "../components/forms/InvestmentForm.vue";
import InvestmentHoldingsPaginated from "../components/data/InvestmentHoldingsPaginated.vue";

const toastStore = useToastStore();

const { hasPermission } = usePermissions();

const holdRef = ref<InstanceType<typeof InvestmentHoldingsPaginated> | null>(null);

const createModal = ref(false);
const updateModal = ref(false);
const updateRecordID = ref(null);

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
      createModal.value = value;
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
      updateModal.value = true;
      updateRecordID.value = value;
      break;
    }
    default: {
      break;
    }
  }
}

async function handleEmit(emitType: any) {
  switch (emitType) {
    case "completeOperation": {
      createModal.value = false;
      updateModal.value = false;
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
    v-model:visible="createModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Add asset"
  >
    <InvestmentForm
      mode="create"
      @complete-operation="handleEmit('completeOperation')"
    />
  </Dialog>

  <Dialog
    v-model:visible="updateModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Update asset"
  >
    <InvestmentForm
      mode="update"
      @complete-operation="handleEmit('completeOperation')"
    />
  </Dialog>

  <main class="flex flex-column w-full p-2 align-items-center">
    <div
      id="mobile-container"
      class="flex flex-column justify-content-center p-3 w-full gap-3 border-round-md"
      style="
        border: 1px solid var(--border-color);
        background: var(--background-secondary);
      "
    >
      <div
        class="flex flex-row justify-content-between align-items-center text-center gap-2 w-full"
      >
        <div style="font-weight: bold">Investments</div>
        <Button
          class="main-button"
          @click="manipulateDialog('addHolding', true)"
        >
          <div class="flex flex-row gap-1 align-items-center">
            <i class="pi pi-plus" />
            <span> New </span>
            <span class="mobile-hide"> Holding </span>
          </div>
        </Button>
      </div>

      <div id="mobile-row" class="flex flex-row w-full">
        <InvestmentHoldingsPaginated  @update-holding="(id) => manipulateDialog('updateHolding', id)" />
      </div>


    </div>
  </main>
</template>

<style scoped></style>
