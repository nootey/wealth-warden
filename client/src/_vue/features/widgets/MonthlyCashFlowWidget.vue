<script setup lang="ts">
import MonthlyCashFlowChart from "../../components/charts/MonthlyCashFlowChart.vue";
import type { MonthlyCashFlowResponse } from "../../../models/chart_models.ts";
import { onMounted, ref, watch } from "vue";
import { useStatisticsStore } from "../../../services/stores/statistics_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useChartStore } from "../../../services/stores/chart_store.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import type { Account } from "../../../models/account_models.ts";
import { useAccountStore } from "../../../services/stores/account_store.ts";
import ShowLoading from "../../components/base/ShowLoading.vue";

const statsStore = useStatisticsStore();
const toastStore = useToastStore();
const chartStore = useChartStore();
const accStore = useAccountStore();

const years = ref<number[]>([]);
const selectedYear = ref<number>(new Date().getFullYear());
const monthlyCashFlow = ref<MonthlyCashFlowResponse>({ year: 0, series: [] });
const accounts = ref<Account[]>([]);
const selectedAccountID = ref<number | null>(null);

const isLoadingStats = ref(false);

async function fetchMonthlyCashFlows(
  year: number | null,
  account: number | null = null,
) {
  isLoadingStats.value = true;
  // Clear data first to prevent chart rendering with stale data
  monthlyCashFlow.value = { year: 0, series: [] };

  if (!year) {
    year = new Date().getFullYear();
  }

  try {
    const params: any = { year: year };
    if (account) {
      params.account = account;
    }

    monthlyCashFlow.value = await chartStore.getMonthlyCashFlowForYear(params);
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    isLoadingStats.value = false;
  }
}

async function loadYears() {
  try {
    const result = await statsStore.getAvailableStatsYears(null);

    years.value = Array.isArray(result) ? result : [];

    const current = new Date().getFullYear();
    selectedYear.value = years.value.includes(current)
      ? current
      : (years.value[0] ?? current);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function loadAccounts() {
  try {
    accounts.value = await accStore.getAccountsBySubtype("checking");
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
}

onMounted(async () => {
  await loadAccounts();
  await loadYears();
  const defaultChecking = accounts.value.find(
    (acc) => acc.is_default && acc.account_type?.sub_type === "checking",
  );
  if (defaultChecking) {
    selectedAccountID.value = defaultChecking.id;
  }
  await fetchMonthlyCashFlows(null, selectedAccountID.value);
});

watch(
  () => [selectedYear.value, selectedAccountID.value] as const,
  async ([year, account]) => {
    await fetchMonthlyCashFlows(year, account);
  },
  { flush: "post" },
);
</script>

<template>
  <div class="flex flex-column w-full p-3 gap-3">
    <div
      v-if="years.length > 0"
      id="mobile-row"
      class="flex flex-row gap-2 w-full justify-content-between align-items-center"
    >
      <div class="flex flex-column gap-1">
        <div class="flex flex-row">
          <span class="text-sm" style="color: var(--text-secondary)">
            Select which year you want to display statistics for. Current year
            will be used as a default.
          </span>
        </div>
      </div>

      <div class="flex flex-column gap-2">
        <Select
          v-model="selectedYear"
          size="small"
          style="width: 150px"
          :options="years"
        />
      </div>
    </div>

    <div
      class="flex flex-row gap-2 w-full justify-content-between align-items-center"
    >
      <div class="flex flex-column gap-2">
        <div class="flex flex-row">
          <span class="text-sm" style="color: var(--text-secondary)">
            A default checking account was found. The stats are representative
            of the cash flow to this account.
          </span>
        </div>
      </div>

      <div class="flex flex-column gap-2">
        <Select
          v-model="selectedAccountID"
          size="small"
          style="width: 150px"
          :options="accounts"
          option-value="id"
          placeholder="All accounts"
          show-clear
        >
          <template #value="slotProps">
            <span v-if="slotProps.value">
              {{ accounts.find((a) => a.id === slotProps.value)?.name }}
            </span>
            <span v-else>All accounts</span>
          </template>
          <template #option="slotProps">
            <div class="flex flex-column">
              <span class="font-semibold">{{ slotProps.option.name }}</span>
              <span class="text-xs" style="color: var(--text-secondary)">
                {{
                  vueHelper.formatString(
                    slotProps.option.account_type?.sub_type,
                  )
                }}
              </span>
            </div>
          </template>
        </Select>
      </div>
    </div>

    <ShowLoading v-if="isLoadingStats" :num-fields="7" />
    <MonthlyCashFlowChart
      v-else-if="monthlyCashFlow.series.length > 0"
      :key="`chart-${selectedYear}-${selectedAccountID ?? 'all'}-${monthlyCashFlow.series.length}`"
      :data="monthlyCashFlow"
    />
  </div>
</template>

<style scoped></style>
