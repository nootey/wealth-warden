<script setup lang="ts">
import type { YearlySankeyData } from "../../../models/chart_models.ts";
import { onMounted, ref, watch } from "vue";
import { useStatisticsStore } from "../../../services/stores/statistics_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import ShowLoading from "../../components/base/ShowLoading.vue";
import YearlySankeyCashFlowChart from "../../components/charts/YearlySankeyCashFlowChart.vue";
import { useAccountStore } from "../../../services/stores/account_store.ts";
import type { Account } from "../../../models/account_models.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import { useChartStore } from "../../../services/stores/chart_store.ts";

withDefaults(
  defineProps<{
    isMobile?: boolean;
  }>(),
  {
    isMobile: false,
  },
);

const chartStore = useChartStore();
const statsStore = useStatisticsStore();
const toastStore = useToastStore();
const accStore = useAccountStore();

const years = ref<number[]>([]);
const selectedYear = ref<number>(new Date().getFullYear());
const sankeyData = ref<YearlySankeyData | null>(null);
const isLoadingStats = ref(false);

const accounts = ref<Account[]>([]);
const selectedAccountID = ref<number | null>(null);

async function fetchSankeyData(year: number, account: number | null = null) {
  isLoadingStats.value = true;
  sankeyData.value = null;

  try {
    const params: any = { year };
    if (account) {
      params.account = account;
    }
    sankeyData.value = await chartStore.getYearlySankeyData(params);
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    isLoadingStats.value = false;
  }
}

async function loadAccounts() {
  try {
    accounts.value = await accStore.getAccountsBySubtype("checking");
  } catch (e) {
    toastStore.errorResponseToast(e);
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

onMounted(async () => {
  await loadAccounts();
  await loadYears();
  const defaultChecking = accounts.value.find(
    (acc) => acc.is_default && acc.account_type?.sub_type === "checking",
  );
  if (defaultChecking) {
    selectedAccountID.value = defaultChecking.id;
  }
  await fetchSankeyData(selectedYear.value, selectedAccountID.value);
});

watch(
  () => [selectedYear.value, selectedAccountID.value] as const,
  async ([year, account]) => {
    await fetchSankeyData(year, account);
  },
  { flush: "post" },
);
</script>

<template>
  <div class="flex flex-column w-full p-3 gap-3">
    <div
      v-if="years.length > 0"
      class="flex flex-row gap-2 w-full justify-content-between align-items-center"
    >
      <div class="flex flex-column gap-1">
        <span class="text-sm" style="color: var(--text-secondary)">
          View your yearly cash flow breakdown from checking accounts.
        </span>
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
        <span class="text-sm" style="color: var(--text-secondary)">
          A default checking account was found. The stats are representative of
          the cash flow to this account.
        </span>
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
    <YearlySankeyCashFlowChart
      v-else-if="sankeyData"
      :key="`sankey-${selectedYear}-${selectedAccountID ?? 'all'}`"
      :data="sankeyData"
    />
  </div>
</template>

<style scoped></style>
