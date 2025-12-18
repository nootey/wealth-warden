<script setup lang="ts">
import {computed, onMounted, ref, watch} from "vue";
import type {BasicAccountStats, CategoryStat} from "../../models/statistics_models.ts";
import {useStatisticsStore} from "../../services/stores/statistics_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import ShowLoading from "../components/base/ShowLoading.vue";
import vueHelper from "../../utils/vue_helper.ts";
import ComparativePieChart from "../components/charts/ComparativePieChart.vue";
import type {Account} from "../../models/account_models.ts";
import {useAccountStore} from "../../services/stores/account_store.ts";

const props = defineProps<{
  accID?: number | null;
  pieChartSize: number;
}>();

const statsStore = useStatisticsStore();
const accStore = useAccountStore();
const toastStore = useToastStore();

const accBasicStats = ref<BasicAccountStats | null>(null);
const years = ref<number[]>([]);
const selectedYear = ref<number>(new Date().getFullYear());
const accounts = ref<Account[]>([]);
const selectedAccountID = ref<number | null>(props.accID ?? null);

const isLoadingYears = ref(false);
const isLoadingStats = ref(false);
const isLoadingAccounts = ref(false);
const isLoading = computed(() => isLoadingYears.value || isLoadingStats.value || isLoadingAccounts.value);

onMounted(async () => {
    try {
        if(!props.accID) {
            await loadAccounts();
            const defaultChecking = accounts.value.find(
                acc => acc.is_default && acc.account_type?.sub_type === 'checking'
            );
            if (defaultChecking) {
                selectedAccountID.value = defaultChecking.id;
            }
        }
        await loadYears();
        await loadStats();
    } catch (e) {
        toastStore.errorResponseToast(e);
    }
});

watch(selectedYear, async (newVal, oldVal) => {
    if (newVal !== oldVal) {
        try {
            await loadStats();
        } catch (e) {
            toastStore.errorResponseToast(e);
        }
    }
});

watch(selectedAccountID, async (newVal, oldVal) => {
    if (newVal !== oldVal) {
        try {
            await loadYears();
            await loadStats();
        } catch (e) {
            toastStore.errorResponseToast(e);
        }
    }
});

const loadStats = async () => {
    isLoadingStats.value = true;
    try {
        accBasicStats.value = await statsStore.getBasicStatisticsForAccount(
            selectedAccountID.value ?? null,
            selectedYear.value
        );
    } finally {
        isLoadingStats.value = false;
    }
};

const loadYears = async () => {
    isLoadingAccounts.value = true;
    try {
        const result = await statsStore.getAvailableStatsYears(selectedAccountID.value ?? null);
        years.value = Array.isArray(result) ? result : [];

        const current = new Date().getFullYear();
        selectedYear.value = years.value.includes(current)
            ? current
            : (years.value[0] ?? current);

    } finally {
        isLoadingAccounts.value = false;
    }
};

const loadAccounts = async () => {
    isLoadingYears.value = true;
    try {
        accounts.value = await accStore.getAccountsBySubtype("checking");
    } finally {
        isLoadingYears.value = false;
    }
};

const toNumber = (val?: string | null) => {
    if (val == null) return 0;
    const n = Number(val);
    return isNaN(n) ? 0 : n;
};

const inflowLabels = computed<string[]>(() => {
    if (!accBasicStats.value?.categories?.length) return [];
    return accBasicStats.value.categories
        .filter((c: CategoryStat) => toNumber(c.inflow) > 0)
        .map((c: CategoryStat) => c.category_name ?? `Category ${c.category_id}`);
});

const inflowValues = computed<number[]>(() => {
    if (!accBasicStats.value?.categories?.length) return [];
    return accBasicStats.value.categories
        .filter((c: CategoryStat) => toNumber(c.inflow) > 0)
        .map((c: CategoryStat) => toNumber(c.inflow));
});

const outflowLabels = computed<string[]>(() => {
    if (!accBasicStats.value?.categories?.length) return [];
    return accBasicStats.value.categories
        .filter((c: CategoryStat) => toNumber(c.outflow) !== 0)
        .map((c: CategoryStat) => c.category_name ?? `Category ${c.category_id}`);
});

const outflowValues = computed<number[]>(() => {
    if (!accBasicStats.value?.categories?.length) return [];
    return accBasicStats.value.categories
        .filter((c: CategoryStat) => toNumber(c.outflow) !== 0)
        .map((c: CategoryStat) => Math.abs(toNumber(c.outflow)));
});

const chartItems = [
    { type: "inflows" },
    { type: "outflows" },
];

const hasInflowData = computed(() => inflowValues.value?.length > 0);
const hasOutflowData = computed(() => outflowValues.value?.length > 0);

const pieOptions = computed(() => ({
    plugins: {
        legend: { display: false, position: "bottom" },
        tooltip: {
            callbacks: {
                label: (ctx: any) => {
                    const label = ctx.label ?? "";
                    const value = Number(ctx.parsed);
                    const data = (ctx.dataset?.data ?? []) as (number | string)[];
                    const total = (data as (number | string)[])
                        .map(v => Number(v))
                        .reduce((a, b) => a + b, 0);
                    const pct = total ? (value / total) : 0;

                    return `${label}: ${vueHelper.displayAsCurrency(value)} · ${vueHelper.displayAsPercentage(pct)}`;
                }
            }
        }
    }
}));

</script>

<template>
  <div
    v-if="accBasicStats"
    class="w-full flex flex-column gap-3 p-3"
  >
    <h3 style="color: var(--text-primary)">
      Basic
    </h3>
    <div
      v-if="years.length > 0"
      class="flex flex-row gap-2 w-full justify-content-between align-items-center"
    >
      <div class="flex flex-column gap-2">
        <div class="flex flex-row">
          <span
            class="text-sm"
            style="color: var(--text-secondary)"
          >
            Select which year you want to display statistics for. Current year will be used as a default.
          </span>
        </div>
      </div>

      <div class="flex flex-column gap-2">
        <Select
          v-model="selectedYear"
          size="small"
          style="width: 150px;"
          :options="years"
        />
      </div>
    </div>

    <div
      v-if="!accID"
      class="flex flex-row gap-2 w-full justify-content-between align-items-center"
    >
      <div class="flex flex-column gap-2">
        <div class="flex flex-row">
          <span
            class="text-sm"
            style="color: var(--text-secondary)"
          >
            A default checking account was found. The stats are representative of the cash flow to this account.
          </span>
        </div>
      </div>

      <div class="flex flex-column gap-2">
        <Select
          v-model="selectedAccountID"
          size="small"
          style="width: 150px;"
          :options="accounts"
          option-value="id"
          placeholder="All accounts"
          show-clear
        >
          <template #value="slotProps">
            <span v-if="slotProps.value">
              {{ accounts.find(a => a.id === slotProps.value)?.name }}
            </span>
            <span v-else>All accounts</span>
          </template>
          <template #option="slotProps">
            <div class="flex flex-column">
              <span class="font-semibold">{{ slotProps.option.name }}</span>
              <span
                class="text-xs"
                style="color: var(--text-secondary)"
              >
                {{ vueHelper.formatString(slotProps.option.account_type?.sub_type) }}
              </span>
            </div>
          </template>
        </Select>
      </div>
    </div>

    <div
      id="stats-row"
      class="flex flex-row w-full justify-content-center p-1"
    >
      <div class="flex flex-column w-6 gap-3">
        <div class="flex flex-row gap-2">
          <span>Total inflows:</span>
          <b>{{ vueHelper.displayAsCurrency(accBasicStats.inflow) }}</b>
        </div>
        <div class="flex flex-row gap-2">
          <span>Total outflows:</span>
          <b>{{ vueHelper.displayAsCurrency(accBasicStats.outflow) }}</b>
        </div>
        <div class="flex flex-row gap-2">
          <span>Avg. monthly inflows:</span>
          <b>{{ vueHelper.displayAsCurrency(accBasicStats.avg_monthly_inflow) }}</b>
        </div>
        <div class="flex flex-row gap-2">
          <span>Avg. monthly outflows:</span>
          <b>{{ vueHelper.displayAsCurrency(accBasicStats.avg_monthly_outflow) }}</b>
        </div>
        <div class="flex flex-row gap-2">
          <span>Take home:</span>
          <b>{{ vueHelper.displayAsCurrency(accBasicStats.take_home) }}</b>
        </div>
        <div class="flex flex-row gap-2">
          <span>Overflow:</span>
          <b>{{ vueHelper.displayAsCurrency(accBasicStats.overflow) }}</b>
        </div>
        <div class="flex flex-row gap-2">
          <span>Avg. monthly take home:</span>
          <b>{{ vueHelper.displayAsCurrency(accBasicStats.avg_monthly_take_home) }}</b>
        </div>
        <div class="flex flex-row gap-2">
          <span>Avg. monthly overflow:</span>
          <b>{{ vueHelper.displayAsCurrency(accBasicStats.avg_monthly_overflow) }}</b>
        </div>
      </div>

      <div class="flex flex-column w-6 justify-content-center align-items-center gap-2">
        <div class="flex flex-column justify-content-center w-12">
          <ShowLoading
            v-if="isLoading"
            :num-fields="4"
          />
          <template v-else>
            <Carousel
              v-if="hasInflowData && hasOutflowData"
              id="stats-carousel"
              :value="chartItems"
              :num-visible="1"
              :num-scroll="1"
            >
              <template #item="slotProps">
                <div class="flex flex-column justify-content-center align-items-center">
                  {{ vueHelper.capitalize(slotProps.data.type) }}
                </div>
                <div class="flex flex-column justify-content-center align-items-center">
                  <ComparativePieChart
                    v-if="slotProps.data.type === 'inflows'"
                    :size="pieChartSize"
                    :show-legend="false"
                    :options="pieOptions"
                    :values="inflowValues"
                    :labels="inflowLabels"
                  />
                  <ComparativePieChart
                    v-else
                    :size="pieChartSize"
                    :show-legend="false"
                    :options="pieOptions"
                    :values="outflowValues"
                    :labels="outflowLabels"
                  />
                </div>
              </template>
            </Carousel>
            <div
              v-else
              class="flex flex-column align-items-center justify-content-center p-3"
              style="border: 1px dashed var(--border-color); border-radius: 16px;"
            >
              <span
                class="text-sm"
                style="color: var(--text-secondary);"
              >
                Not enough transactions found for {{ selectedYear }}.
              </span>
              <span
                class="text-sm"
                style="color: var(--text-secondary);"
              >
                Keep inserting transactions to see the chart.
              </span>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
  <ShowLoading
    v-else
    :num-fields="5"
  />
</template>

<style scoped>

:deep([data-pc-section="indicatorlist"]) { margin-top: 6px; gap: 6px; }
:deep([data-pc-section="content"]) { padding: 0; }
:deep([data-pc-section="indicatorbutton"]) {
    transform: scale(0.6);
    background-color: var(--border-color);
    border-radius: 50%;
}
:deep([data-p-active="true"] [data-pc-section="indicatorbutton"]) {
    background-color: var(--text-primary);
}

@media (max-width: 768px) {

    #stats-row {
        flex-direction: column !important;
        align-items: stretch !important;
        min-width: 0 !important;
    }
    #stats-row > div {
        width: 100% !important;
        min-width: 0 !important;
    }
    #stats-row > div:last-child { margin-top: 1rem; }

    :deep(#stats-carousel button[aria-label="Previous Page"]),
    :deep(#stats-carousel button[aria-label="Next Page"]),
    :deep(#stats-carousel button[data-pc-group="navigator"]) {
        display: none !important;
    }

    :deep(#stats-carousel .flex.justify-content-center.align-items-center) {
        width: 100% !important;
        height: auto !important;
        padding: 0 !important;
        transform: scale(0.9);
        transform-origin: center top;
    }

    :deep(#stats-carousel canvas) {
        width: 100% !important;
        height: auto !important;
    }
}
</style>