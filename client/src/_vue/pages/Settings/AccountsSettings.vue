<script setup lang="ts">
import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import AccountsPanel from "../../features/AccountsPanel.vue";
import { useAccountStore } from "../../../services/stores/account_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import type { Account } from "../../../models/account_models.ts";
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { ref } from "vue";
import { usePermissions } from "../../../utils/use_permissions.ts";
import DefaultAccounts from "../../features/DefaultAccounts.vue";
import AccountForm from "../../components/forms/AccountForm.vue";

const accountStore = useAccountStore();
const toastStore = useToastStore();
const sharedStore = useSharedStore();
const { hasPermission } = usePermissions();

const accRef = ref<InstanceType<typeof AccountsPanel> | null>(null);

const createModal = ref(false);

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
  await accRef.value?.refresh?.();
}

async function toggleEnabled(
  acc: Account,
  nextValue: boolean,
): Promise<boolean> {
  const previous = acc.is_active;
  acc.is_active = nextValue;
  try {
    const response = await accountStore.toggleActiveState(acc.id!);
    toastStore.successResponseToast(response);
    return true;
  } catch (error) {
    // add a small delay for the toggle animation to complete
    await new Promise((resolve) => setTimeout(resolve, 300));
    acc.is_active = previous;
    toastStore.errorResponseToast(error);
    return false;
  }
}

async function closeAccount(id: number) {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }

  try {
    let response = await sharedStore.deleteRecord("accounts", id);
    toastStore.successResponseToast(response);
    accRef.value?.refresh();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}
</script>

<template>
  <div class="flex flex-column w-full gap-3">
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

    <SettingsSkeleton class="w-full">
      <div id="main-col" class="w-full flex flex-column gap-3 p-2">
        <div
          class="w-full flex flex-row justify-content-between p-1 gap-2 align-items-center"
        >
          <div class="w-full flex flex-column gap-2">
            <div class="flex flex-row gap-2 align-items-center w-full">
              <div class="font-bold">Account management</div>
            </div>
            <div>
              Manage administrative details and monitor all your accounts.
            </div>
          </div>
          <Button class="main-button" @click="openCreate">
            <div class="flex flex-row gap-1 align-items-center">
              <i class="pi pi-plus" />
              <span> New </span>
              <span class="mobile-hide"> Account </span>
            </div>
          </Button>
        </div>

        <AccountsPanel
          ref="accRef"
          :advanced="true"
          :allow-edit="true"
          :on-toggle="toggleEnabled"
          :max-height="57"
          @close-account="closeAccount"
        />
      </div>
    </SettingsSkeleton>

    <SettingsSkeleton class="w-full">
      <div id="main-col" class="w-full flex flex-column gap-3 p-2">
        <DefaultAccounts />
      </div>
    </SettingsSkeleton>
  </div>
</template>

<style scoped>
@media (max-width: 768px) {
  #main-col {
    padding: 0 !important;
  }
}
</style>
