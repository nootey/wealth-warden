<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useConfirm } from "primevue/useconfirm";
import type { Account } from "../../models/account_models.ts";
import { useAccountStore } from "../../services/stores/account_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";

const accStore = useAccountStore();
const toastStore = useToastStore();
const confirm = useConfirm();

const merging = ref(false);
const accounts = ref<Account[]>([]);
const sourceAccount = ref<Account | null>(null);
const destinationAccount = ref<Account | null>(null);

onMounted(async () => {
  try {
    accounts.value = await accStore.getAllAccounts(true);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
});

function confirmMerge() {
  confirm.require({
    header: "Confirm account merge",
    message: `You are about to merge "${sourceAccount.value?.name}" into "${destinationAccount.value?.name}". This action is irreversible. Are you sure?`,
    rejectProps: { label: "Cancel" },
    acceptProps: { label: "Merge", severity: "danger" },
    accept: () => doMerge(),
  });
}

async function doMerge() {
  merging.value = true;
  try {
    const res = await accStore.mergeAccounts(
      sourceAccount.value!.id!,
      destinationAccount.value!.id!,
    );
    toastStore.successResponseToast(res);
    sourceAccount.value = null;
    destinationAccount.value = null;
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    merging.value = false;
  }
}
</script>

<template>
  <div class="w-full flex flex-column gap-2">
    <div class="flex flex-row justify-content-between align-items-center gap-3">
      <div class="w-full flex flex-column gap-2">
        <h3>Merge accounts</h3>
        <h5 style="color: var(--text-secondary)">
          Merge one account into another. All transactions and transfers will be
          moved to the destination account.
        </h5>
      </div>
    </div>

    <div class="flex flex-row gap-3 w-full">
      <div class="flex flex-column gap-1 w-full">
        <label>Source account</label>
        <Select
          v-model="sourceAccount"
          :options="accounts.filter((a) => a.id !== destinationAccount?.id)"
          filter
          option-label="name"
          placeholder="Select source"
          class="w-full"
          size="small"
        />
      </div>
      <div class="flex flex-column gap-1 w-full">
        <label>Destination account</label>
        <Select
          v-model="destinationAccount"
          :options="accounts.filter((a) => a.id !== sourceAccount?.id)"
          filter
          option-label="name"
          placeholder="Select destination"
          class="w-full"
          size="small"
        />
      </div>
    </div>

    <div class="flex flex-row gap-2 w-full">
      <div id="expand" class="flex flex-column gap-2 ml-auto">
        <Button
          class="main-button"
          label="Merge"
          :disabled="!sourceAccount || !destinationAccount"
          :loading="merging"
          style="height: 42px"
          @click="confirmMerge"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
@media (max-width: 768px) {
  #expand {
    width: 100% !important;
  }
}
</style>
