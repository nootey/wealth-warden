<script setup lang="ts">
import TransactionRecords from "../features/transactions/TransactionRecords.vue";
import AddTransaction from "../components/forms/AddTransaction.vue";
import {computed, onMounted, ref} from "vue";
import type {Account} from "../../models/account_models.ts";
import vueHelper from "../../utils/vueHelper.ts";
import {useSharedStore} from "../../services/stores/shared_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {useTransactionStore} from "../../services/stores/transaction_store.ts";

const shared_store = useSharedStore();
const toast_store = useToastStore();
const transactionStore = useTransactionStore();

const apiPrefix = "transactions";

const addTransactionModal = ref(false);

onMounted(async () => {
  await getData();
  await transactionStore.getCategories();
})

const loadingTransactions = ref(true);
const transactions = ref<Account[]>([]);

const params = computed(() => {
  return {
    rowsPerPage: paginator.value.rowsPerPage,
    sort: sort.value,
    filters: [],
  }
});
const rows = ref([100]);
const default_rows = ref(rows.value[0]);
const paginator = ref({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: default_rows.value
});
const page = ref(1);
const sort = ref(vueHelper.initSort());

async function getData(new_page = null) {

  loadingTransactions.value = true;
  if(new_page)
    page.value = new_page;

  try {
    let paginationResponse = await shared_store.getRecordsPaginated(
        apiPrefix,
        { ...params.value },
        page.value
    );
    transactions.value = paginationResponse.data;
    paginator.value.total = paginationResponse.total_records;
    paginator.value.to = paginationResponse.to;
    paginator.value.from = paginationResponse.from;
    loadingTransactions.value = false;
  } catch (error) {
    toast_store.errorResponseToast(error);
  }
}

function manipulateDialog(modal: string, value: boolean) {
  switch (modal) {
    case 'addTransaction': {
      addTransactionModal.value = value;
      break;
    }
    default: {
      break;
    }
  }
}

async function handleEmit(emitType: any) {
  switch (emitType) {
    case 'addTransaction': {
      addTransactionModal.value = false;
      await getData();
      break;
    }
    default: {
      break;
    }
  }
}

</script>

<template>
  <Dialog class="rounded-dialog" v-model:visible="addTransactionModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '500px'}" header="Add transaction">
    <AddTransaction entity="account" @addTransaction="handleEmit('addTransaction')"></AddTransaction>
  </Dialog>

  <main style="display: flex;flex-direction: column;height: 100vh;width: 100%;padding: 1rem;align-items: center;">

    <div class="flex flex-row justify-content-between align-items-center p-3"
         style="border-top-right-radius: 8px; border-top-left-radius: 8px;
         border: 1px solid var(--border-color);background: var(--background-secondary);
         max-width: 1000px; width: 100%;">

      <div style="font-weight: bold;">Transactions</div>
      <Button label="New transaction" icon="pi pi-plus" @click="manipulateDialog('addTransaction', true)"
              style="background-color: var(--text-primary); color: var(--background-primary);
                border: none; border-radius: 6px; font-size: 0.875rem; padding: 0.5rem 1rem;"></Button>
    </div>

    <div class="flex flex-column justify-content-center p-3"
         style="border-bottom-right-radius: 8px; border-bottom-left-radius: 8px;
         border: 1px solid var(--border-color);background: var(--background-secondary);
         max-width: 1000px; width: 100%;">
      <div class="flex flex-row" style="width: 100%;">
        <InputText style="width: 100%; border-radius: 8px; font-size: 0.875rem; padding: 0.5rem 1rem;" placeholder="Search transactions ..." />
      </div>
      <div class="flex flex-row" style="width: 100%;">
        <TransactionRecords></TransactionRecords>
      </div>

    </div>
  </main>
</template>

<style scoped>

</style>