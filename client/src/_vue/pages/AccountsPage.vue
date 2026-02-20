<script setup lang="ts">
import AccountsPanel from "../features/AccountsPanel.vue";
import AccountForm from "../components/forms/AccountForm.vue";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { usePermissions } from "../../utils/use_permissions.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import SlotSkeleton from "../components/layout/SlotSkeleton.vue";

const createModal = ref(false);
const router = useRouter();
const { hasPermission } = usePermissions();
const toastStore = useToastStore();

const accountsPanelRef = ref<InstanceType<typeof AccountsPanel> | null>(null);

function openCreate() {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }

  createModal.value = true;
}

async function handleCreate() {
  createModal.value = false;
  await accountsPanelRef.value?.refresh?.();
}
</script>

<template>
  <Dialog
    v-model:visible="createModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Create account"
  >
    <AccountForm mode="create" @complete-operation="handleCreate" />
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
            <div class="font-bold">Accounts</div>
            <i
              v-if="hasPermission('manage_data')"
              v-tooltip="'Go to accounts settings.'"
              class="pi pi-external-link hover-icon mr-auto text-sm"
              @click="router.push('settings/accounts')"
            />
          </div>
          <div>Manage and monitor all your accounts.</div>
        </div>
        <Button class="main-button" @click="openCreate">
          <div class="flex flex-row gap-1 align-items-center">
            <i class="pi pi-plus" />
            <span> New </span>
            <span class="mobile-hide"> Account </span>
          </div>
        </Button>
      </div>

      <Panel :collapsed="false" header="Assets">
        <AccountsPanel
          ref="accountsPanelRef"
          :advanced="false"
          :allow-edit="true"
          :max-height="72"
        />
      </Panel>
    </div>
  </main>
</template>

<style scoped lang="scss">
@media (max-width: 768px) {
  #inner-row {
    padding: 0.75rem !important;
    margin-bottom: -7px !important;
  }
}
</style>
