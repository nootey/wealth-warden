<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useAccountStore } from "../../../services/stores/account_store.ts";
import { useTransactionStore } from "../../../services/stores/transaction_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";

const router = useRouter();
const accountStore = useAccountStore();
const transactionStore = useTransactionStore();
const toastStore = useToastStore();

const loading = ref(true);
const hasCategories = ref(false);
const hasAccounts = ref(false);
const hasTransactions = ref(false);

const steps = computed(() => [
  {
    key: "categories",
    label: "Add categories",
    hint: "Group your transactions so your reports make sense.",
    done: hasCategories.value,
    cta: "Set up categories",
    action: () => router.push({ name: "settings.categories" }),
  },
  {
    key: "accounts",
    label: "Add an account",
    hint: "A bank, wallet or card to track balances.",
    done: hasAccounts.value,
    cta: "Add account",
    action: () => router.push({ name: "settings.accounts" }),
  },
  {
    key: "transactions",
    label: "Record your first transaction",
    hint: "Log income or spending to see your finances move.",
    done: hasTransactions.value,
    cta: "Add transaction",
    action: () => router.push({ name: "transactions" }),
  },
]);

const complete = computed(
  () => hasCategories.value && hasAccounts.value && hasTransactions.value,
);

onMounted(async () => {
  try {
    const [, , txnCount] = await Promise.all([
      transactionStore.getCategories(),
      accountStore.getAllAccounts(),
      transactionStore.getTransactionCount(),
    ]);
    hasCategories.value = transactionStore.categories.length > 0;
    hasAccounts.value = accountStore.accounts.length > 0;
    // Each account auto-creates one opening-balance transaction, so a real
    // user transaction only exists once the count exceeds the account count.
    hasTransactions.value = txnCount > accountStore.accounts.length;
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <Panel v-if="!loading && !complete" header="Getting started">
    <div class="w-full flex flex-col gap-3 p-1">
      <span class="text-sm" style="color: var(--text-secondary)">
        A few steps to set up your finances. This card goes away once you're
        done.
      </span>

      <div
        v-for="s in steps"
        :key="s.key"
        class="flex flex-row items-center justify-between gap-3 py-2 px-3 rounded-md"
        style="border: 1px solid var(--border-color)"
      >
        <div class="flex flex-row items-center gap-3">
          <i
            :class="s.done ? 'pi pi-check-circle' : 'pi pi-circle'"
            :style="{
              color: s.done ? 'var(--p-green-500)' : 'var(--text-secondary)',
            }"
          />
          <div class="flex flex-col">
            <span style="font-weight: 500; color: var(--text-primary)">
              {{ s.label }}
            </span>
            <span class="text-sm" style="color: var(--text-secondary)">
              {{ s.hint }}
            </span>
          </div>
        </div>

        <Button
          v-if="!s.done"
          :label="s.cta"
          size="small"
          class="main-button"
          @click="s.action()"
        />
        <span v-else class="text-sm" style="color: var(--text-secondary)">
          Done
        </span>
      </div>
    </div>
  </Panel>
</template>

<style scoped></style>
