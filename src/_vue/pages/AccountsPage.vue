<script setup lang="ts">

import InsertAccount from "../components/forms/InsertAccount.vue";
import {computed, onMounted, ref} from "vue";
import {useAccountStore} from "../../services/stores/account_store.ts";
import vueHelper from "../../utils/vueHelper.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {useSharedStore} from "../../services/stores/shared_store.ts";
import type {Account} from "../../models/account_models.ts";

const account_store = useAccountStore();
const shared_store = useSharedStore();
const toast_store = useToastStore();

const apiPrefix = "accounts";

const addAccountModal = ref(false);

onMounted(async () => {
  await account_store.getAccountTypes();
  await getData();
})

const loadingAccounts = ref(true);
const accounts = ref<Account[]>([]);

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

const typeColors: Record<string, { bg: string; fg: string }> = {
  cash:         { bg: "#9b59b6", fg: "#6c3483" },
  investment:   { bg: "#2980b9", fg: "#1c5980" },
  crypto:       { bg: "#16a085", fg: "#0d6655" },
  property:     { bg: "#8e44ad", fg: "#5b2c6f" },
  vehicle:      { bg: "#3498db", fg: "#21618c" },
  other_asset:  { bg: "#7d3c98", fg: "#4a235a" },

  credit_card:    { bg: "#e74c3c", fg: "#922b21" },
  loan:           { bg: "#e67e22", fg: "#9a531c" },
  other_liability:{ bg: "#f1c40f", fg: "#9a7d0a" },
};

const logoColor = (type: string) =>
    typeColors[type] ?? { bg: "#444", fg: "#222" };

const typeMap: Record<string, string> = {};
account_store.accountTypes.forEach(t => {
  typeMap[t.type] = t.classification;
});

async function getData(new_page = null) {

  loadingAccounts.value = true;
  if(new_page)
    page.value = new_page;

  try {
    let paginationResponse = await shared_store.getRecordsPaginated(
        apiPrefix,
        { ...params.value },
        page.value
    );
    accounts.value = paginationResponse.data;
    paginator.value.total = paginationResponse.total_records;
    paginator.value.to = paginationResponse.to;
    paginator.value.from = paginationResponse.from;
    loadingAccounts.value = false;
  } catch (error) {
    toast_store.errorResponseToast(error);
  }
}

const groupedAccounts = computed(() => {
  // group into a Map<type, Account[]>
  const groups = new Map<string, typeof accounts.value>();
  for (const acc of accounts.value) {
    const t = acc.account_type?.type || "other_asset";
    if (!groups.has(t)) groups.set(t, []);
    groups.get(t)!.push(acc);
  }

  // turn into [type, group[]] array and sort
  return Array.from(groups.entries())
      .sort(([typeA], [typeB]) => {
        const ca = typeMap[typeA] ?? "asset";    // default asset
        const cb = typeMap[typeB] ?? "asset";
        if (ca !== cb) {
          // assets (-1) before liabilities (1)
          return ca === "asset" ? -1 : 1;
        }
        // same classification → alphabetical by type
        return typeA.localeCompare(typeB);
      });
});

const groupTotal = (group: Account[]) =>
    group.reduce((sum, acc) => sum + (acc.balance.end_balance || 0), 0);

function manipulateDialog(modal: string, value: boolean) {
  switch (modal) {
    case 'add-account': {
      addAccountModal.value = value;
      break;
    }
    default: {
      break;
    }
  }
}

async function handleEmit(emitType: any) {
  switch (emitType) {
    case 'insertAccount': {
      addAccountModal.value = false;
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
  <Dialog class="rounded-dialog" v-model:visible="addAccountModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '500px'}" header="Add account">
    <InsertAccount entity="account" @insertAccount="handleEmit('insertAccount')"></InsertAccount>
  </Dialog>
  <main style="display: flex;flex-direction: column;height: 100vh;width: 100%;padding: 1rem;justify-content: center;align-items: center;">

    <div style="flex: 0 0 auto;display: flex;justify-content: space-between;align-items: center;padding: 1rem;border-top-right-radius: 8px;
        border-top-left-radius: 8px;border: 1px solid var(--border-color);background: var(--background-secondary);max-width: 1000px;width: 100%;">

      <div style="font-weight: bold;">Accounts</div>
      <Button class="main-button" label="New Account" icon="pi pi-plus" @click="manipulateDialog('add-account', true)"/>
    </div>

    <div style="flex: 1 1 auto;overflow-y: auto;padding: 1rem;border-bottom-right-radius: 8px;border-bottom-left-radius: 8px;
        border: 1px solid var(--border-color);background: var(--background-secondary);max-width: 1000px;width: 100%; ">

        <div v-for="[type, group] in groupedAccounts" :key="type"
             style="margin-bottom:1.5rem;background:var(--background-primary);border-radius:8px;padding:1rem; width: 100%;">

          <div style="display:flex;justify-content:space-between;align-items:center;padding-bottom:0.5rem;
              border-bottom:1px solid var(--border-color);margin-bottom:0.5rem;">
            <div style="font-size:0.875rem; color:var(--text-secondary);">
              {{ vueHelper.formatString(type) }} · {{ group.length }}
            </div>
            <div style="font-weight:bold; font-size:0.875rem; color:var(--text-secondary);">
              {{ vueHelper.displayAsCurrency(groupTotal(group)) }}
            </div>
          </div>

          <div v-for="(account, i) in group" :key="account.id ?? i" style="display:flex;align-items:center;justify-content:space-between;
                padding:0.5rem;border:1px solid var(--border-color);border-radius:8px;margin-top:0.5rem;background:var(--background-secondary);">

            <div style="display:flex; align-items:center;">
              <div :style="{
                    width: '32px',
                    height: '32px',
                    border: '1px solid',
                    borderColor:       logoColor(account.account_type.type).fg,
                    borderRadius: '50%',
                    background:  logoColor(account.account_type.type).bg,
                    color:       logoColor(account.account_type.type).fg,
                    display:     'flex',
                    alignItems:  'center',
                    justifyContent: 'center',
                    fontWeight:  'bold'
              }">
                {{ account.name.charAt(0).toUpperCase() }}
              </div>
              <div style="margin-left:1rem;">
                <div style="font-weight:bold;">{{ account.name }}</div>
                <div style="font-size:0.875rem; color:var(--text-secondary);">
                  {{ vueHelper.formatString(account.account_type?.subtype) }}
                </div>
              </div>
            </div>

            <div style="display:flex; align-items:center;">
              <div style="font-weight:bold; margin-right:0.5rem;">
                {{ vueHelper.displayAsCurrency(account.balance.end_balance) }}
              </div>
            </div>

          </div>
        </div>


    </div>
  </main>
</template>

<style scoped>

</style>