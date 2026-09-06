<script setup lang="ts">
import { ref } from "vue";
import { useConfirm } from "primevue/useconfirm";
import ValidationError from "../components/validation/ValidationError.vue";
import { useAccountStore } from "../../services/stores/account_store.ts";
import { useUserStore } from "../../services/stores/user_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import type { AccountLookup } from "../../models/account_models.ts";
import type { UserLookup } from "../../models/user_models.ts";

const accountStore = useAccountStore();
const userStore = useUserStore();
const toastStore = useToastStore();
const confirm = useConfirm();

const purging = ref(false);
const userSuggestions = ref<UserLookup[]>([]);
const selectedUser = ref<UserLookup | null>(null);
const accounts = ref<AccountLookup[]>([]);
const selectedAccount = ref<AccountLookup | null>(null);

async function searchUsers(event: { query: string }): Promise<void> {
  const q = event.query.trim();
  if (q.length < 2) {
    userSuggestions.value = [];
    return;
  }

  try {
    userSuggestions.value = await userStore.searchUsers(q);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function loadAccounts(): Promise<void> {
  selectedAccount.value = null;
  accounts.value = [];

  if (!selectedUser.value) {
    return;
  }

  try {
    accounts.value = await accountStore.fetchAccountsForUser(
      selectedUser.value.id,
    );
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

function accountLabel(account: AccountLookup): string {
  const state = account.closed_at ? ", closed" : "";
  return `${account.name} (${account.currency}${state})`;
}

function confirmPurge(): void {
  confirm.require({
    header: "Confirm account purge",
    message: `You are about to permanently delete "${selectedAccount.value?.name}" and everything hanging off it. This action is irreversible. Are you sure?`,
    rejectProps: { label: "Cancel" },
    acceptProps: { label: "Purge", severity: "danger" },
    accept: () => doPurge(),
  });
}

async function doPurge(): Promise<void> {
  purging.value = true;
  try {
    const res = await accountStore.purgeAccount(selectedAccount.value!.id);
    toastStore.successResponseToast(res);
    await loadAccounts();
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    purging.value = false;
  }
}
</script>

<template>
  <div class="flex flex-col gap-1 p-4 border rounded-md border-surface">
    <div style="font-weight: bold">Purge Account</div>
    <div class="text-sm text-muted-color">
      Hard deletes one account and everything hanging off it - transactions,
      transfers on both sides, balances, snapshots, templates, saving goals and
      investment assets. Then it rebuilds the owner's whole balance history.
      Closing an account keeps its past; this erases it.
      <strong>There is no restore.</strong>
    </div>
    <div class="flex flex-col gap-2 mt-2 sm:flex-row sm:items-end">
      <div class="flex flex-col gap-1">
        <ValidationError :is-required="true">
          <label for="purge-user">Owner email</label>
        </ValidationError>
        <AutoComplete
          id="purge-user"
          v-model="selectedUser"
          size="small"
          option-label="email"
          placeholder="At least 2 characters"
          :suggestions="userSuggestions"
          @complete="searchUsers"
          @option-select="loadAccounts"
        />
      </div>
      <div class="flex flex-col gap-1">
        <ValidationError :is-required="true">
          <label for="purge-account">Account</label>
        </ValidationError>
        <Select
          id="purge-account"
          v-model="selectedAccount"
          size="small"
          :options="accounts"
          :option-label="accountLabel"
          :disabled="!selectedUser"
          placeholder="Select account"
        />
      </div>
      <Button
        label="Purge account"
        class="delete-button"
        style="height: 44px"
        :disabled="!selectedUser || !selectedAccount"
        :loading="purging"
        @click="confirmPurge"
      />
    </div>
  </div>
</template>

<style scoped></style>
