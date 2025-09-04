<script setup lang="ts">

import AccountForm from "../components/forms/AccountForm.vue";
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

const createModal = ref(false);
const detailsModal = ref(false);
const accountDetailsID = ref(null);

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

function manipulateDialog(modal: string, value: any) {
  switch (modal) {
    case 'addAccount': {
      createModal.value = value;
      break;
    }
    case 'accountDetails': {
      detailsModal.value = true;
      accountDetailsID.value = value;
      break;
    }
    default: {
      break;
    }
  }
}

async function handleEmit(emitType: any) {
  switch (emitType) {
    case 'completeOperation': {
      createModal.value = false;
      detailsModal.value = false;
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
    <Dialog class="rounded-dialog" v-model:visible="createModal" :breakpoints="{'801px': '90vw'}"
          :modal="true" :style="{width: '500px'}" header="Create account">
    <AccountForm mode="create" @completeOperation="handleEmit('completeOperation')"></AccountForm>
    </Dialog>

    <Dialog position="right" class="rounded-dialog" v-model:visible="detailsModal" :breakpoints="{'801px': '90vw'}"
            :modal="true" :style="{width: '500px'}" header="Account details">
        <AccountForm mode="update" :recordId="accountDetailsID" @completeOperation="handleEmit('completeOperation')"></AccountForm>
    </Dialog>

  <main class="flex flex-column w-full p-2 justify-content-center align-items-center h-screen gap-2">

    <div class="flex flex-row justify-content-between align-items-center p-3 w-full border-round-md bordered"
         style="max-width: 1000px;">

      <div class="font-bold">Accounts</div>
      <Button class="main-button" label="New Account" icon="pi pi-plus" @click="manipulateDialog('addAccount', true)"/>
    </div>

    <div class="flex w-full p-3 gap-2 border-round-md bordered justify-content-between align-items-center" style="max-width:1000px;">
        <div>
            <div class="text-xs" style="color:var(--text-secondary);">Total</div>
            <div class="font-bold">
              {{ vueHelper.displayAsCurrency(totals.total) }}
            </div>
        </div>
        <div>
            <div class="text-xs" style="color:var(--text-secondary);">Positive</div>
            <div class="font-bold" style="color:green">
              {{ vueHelper.displayAsCurrency(totals.positive) }}
            </div>
        </div>
        <div>
            <div class="text-xs" style="color:var(--text-secondary);">Negative</div>
            <div class="font-bold" style="color:red">
              {{ vueHelper.displayAsCurrency(totals.negative) }}
            </div>
        </div>
    </div>

    <div class="flex-1 w-full border-round-md p-2 bordered overflow-y-auto" style="max-width: 1000px;">

        <div v-for="[type, group] in groupedAccounts" :key="type" class="w-full p-3 mb-2 border-round-md"
             style="background:var(--background-primary);">

          <div class="flex p-2 mb-2 pb-21 align-items-center justify-content-between" style="border-bottom:1px solid var(--border-color);">
            <div class="text-sm" style="color:var(--text-secondary);">
              {{ vueHelper.formatString(type) }} · {{ group.length }}
            </div>
            <div class="font-bold text-sm" style="color:var(--text-secondary);">
              {{ vueHelper.displayAsCurrency(groupTotal(group)) }}
            </div>
          </div>

          <div v-for="(account, i) in group" :key="account.id ?? i"
               class="flex align-items-center justify-content-between p-2 border-round-md mt-1 bordered">

            <div class="flex align-items-center">
              <div class="flex align-items-center justify-content-center font-bold hover"
                   @click="manipulateDialog('accountDetails', account.id)"
                   :style="{
                        width: '32px',
                        height: '32px',
                        border: '1px solid',
                        borderColor:       logoColor(account.account_type.type).fg,
                        borderRadius: '50%',
                        background:  logoColor(account.account_type.type).bg,
                        color:       logoColor(account.account_type.type).fg,
                    }">
                {{ account.name.charAt(0).toUpperCase() }}
              </div>
              <div class="ml-2">
                <div class="font-bold hover" @click="manipulateDialog('accountDetails', account.id)">{{ account.name }}</div>
                <div class="text-sm" style="color:var(--text-secondary);">
                  {{ vueHelper.formatString(account.account_type?.subtype) }}
                </div>
              </div>
            </div>

            <div class="flex align-items-center">
              <div class="font-bold mr-1">
                {{ vueHelper.displayAsCurrency(account.balance.end_balance) }}
              </div>
            </div>

          </div>
        </div>


    </div>
  </main>
</template>

<style scoped>
.bordered {
    border: 1px solid var(--border-color);
    background: var(--background-secondary);
}
.hover {
    font-weight: bold;
}
.hover:hover {
    cursor: pointer;
    text-decoration: underline;
}
</style>