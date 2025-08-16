<script setup lang="ts">

import AddAccount from "../components/forms/AddAccount.vue";
import {computed, onMounted, ref} from "vue";
import {useAccountStore} from "../../services/stores/account_store.ts";
import vueHelper from "../../utils/vue_helper.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {useSharedStore} from "../../services/stores/shared_store.ts";
import type {Account} from "../../models/account_models.ts";
import filterHelper from "../../utils/filter_helper.ts";

const accountStore = useAccountStore();
const sharedStore = useSharedStore();
const toastStore = useToastStore();
import Decimal from "decimal.js";


const apiPrefix = "accounts";

const addAccountModal = ref(false);

onMounted(async () => {
  await accountStore.getAccountTypes();
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
const sort = ref(filterHelper.initSort());

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
accountStore.accountTypes.forEach(t => {
  typeMap[t.type] = t.classification;
});

async function getData(new_page = null) {

  loadingAccounts.value = true;
  if(new_page)
    page.value = new_page;

  try {
    let paginationResponse = await sharedStore.getRecordsPaginated(
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
    toastStore.errorResponseToast(error);
  }
}

const groupedAccounts = computed(() => {

  const groups = new Map<string, typeof accounts.value>();
  for (const acc of accounts.value) {
    const t = acc.account_type?.type || "other_asset";
    if (!groups.has(t)) groups.set(t, []);
    groups.get(t)!.push(acc);
  }

  return Array.from(groups.entries())
      .sort(([typeA], [typeB]) => {
        const ca = typeMap[typeA] ?? "asset";
        const cb = typeMap[typeB] ?? "asset";
        if (ca !== cb) {
          return ca === "asset" ? -1 : 1;
        }
        return typeA.localeCompare(typeB);
      });
});

const groupTotal = (group: Account[]) =>
    group.reduce((sum, acc) => sum.add(new Decimal(acc.balance.end_balance || 0)), new Decimal(0));

const totals = computed(() => {
  const vals = accounts.value.map(a => new Decimal(a.balance.end_balance || 0));

  const total    = vals.reduce((s, v) => s.add(v), new Decimal(0));
  const positive = vals.reduce((s, v) => (v.greaterThan(0) ? s.add(v) : s), new Decimal(0));
  const negative = vals.reduce((s, v) => (v.lessThan(0) ? s.add(v) : s), new Decimal(0));

  return {
    total: total.toString(),
    positive: positive.toString(),
    negative: negative.toString()
  };
});

function manipulateDialog(modal: string, value: boolean) {
  switch (modal) {
    case 'addAccount': {
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
    case 'addAccount': {
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
    <AddAccount entity="account" @addAccount="handleEmit('addAccount')"></AddAccount>
  </Dialog>

  <main class="flex flex-column w-full p-2 justify-content-center align-items-center" style="height: 100vh;">

    <div class="flex flex-row justify-content-between align-items-center p-3 w-full"
         style="border-radius: 8px; border: 1px solid var(--border-color);background: var(--background-secondary);
         max-width: 1000px;">
      
      <div style="font-weight: bold;">Accounts</div>
      <Button class="main-button" label="New Account" icon="pi pi-plus" @click="manipulateDialog('addAccount', true)"/>
    </div>

    <div style="display:flex;gap:0.75rem;max-width:1000px;width:100%;padding:0.75rem 0;">
      <div style="flex:1;border:1px solid var(--border-color);border-radius:8px;padding:0.75rem;background:var(--background-secondary);">
        <div style="font-size:0.75rem;color:var(--text-secondary);">Total</div>
        <div style="font-weight:bold;">
          {{ vueHelper.displayAsCurrency(totals.total) }}
        </div>
      </div>
      <div style="flex:1;border:1px solid var(--border-color);border-radius:8px;padding:0.75rem;background:var(--background-secondary);">
        <div style="font-size:0.75rem;color:var(--text-secondary);">Positive</div>
        <div style="font-weight:bold;color:green">
          {{ vueHelper.displayAsCurrency(totals.positive) }}
        </div>
      </div>
      <div style="flex:1;border:1px solid var(--border-color);border-radius:8px;padding:0.75rem;background:var(--background-secondary);">
        <div style="font-size:0.75rem;color:var(--text-secondary);">Negative</div>
        <div style="font-weight:bold;color:red">
          {{ vueHelper.displayAsCurrency(totals.negative) }}
        </div>
      </div>
    </div>

    <div style="flex: 1 1 auto;overflow-y: auto;padding: 1rem; border-radius: 8px;
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