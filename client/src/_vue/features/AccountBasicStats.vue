<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import type {
  BasicAccountStats,
  CategoryStat,
} from "../../models/analytics_models.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import ShowLoading from "../components/base/ShowLoading.vue";
import vueHelper from "../../utils/vue_helper.ts";
import ComparativePieChart from "../components/charts/ComparativePieChart.vue";
import type { Account } from "../../models/account_models.ts";
import { useAccountStore } from "../../services/stores/account_store.ts";
import { useAnalyticsStore } from "../../services/stores/analytics_store.ts";

const props = defineProps<{
  accID?: number | null;
  pieChartSize: number;
}>();

const analyticsStore = useAnalyticsStore();
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
const isLoading = computed(
  () => isLoadingYears.value || isLoadingStats.value || isLoadingAccounts.value,
);

onMounted(async () => {
  try {
    if (!props.accID) {
      await loadAccounts();
      const defaultChecking = accounts.value.find(
        (acc) => acc.is_default && acc.account_type?.sub_type === "checking",
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

async function loadStats() {
  isLoadingStats.value = true;
  try {
    accBasicStats.value = await analyticsStore.getBasicStatisticsForAccount(
      selectedAccountID.value ?? null,
      selectedYear.value,
    );
  } finally {
    isLoadingStats.value = false;
  }
}

async function loadYears() {
  isLoadingYears.value = true;
  try {
    const result = await analyticsStore.getAvailableStatsYears(
      selectedAccountID.value ?? null,
    );
    years.value = Array.isArray(result) ? result.map((y) => y.year) : [];

    const current = new Date().getFullYear();
    selectedYear.value = years.value.includes(current)
      ? current
      : (years.value[0] ?? current);
  } finally {
    isLoadingYears.value = false;
  }
}

async function loadAccounts() {
  isLoadingAccounts.value = true;
  try {
    accounts.value = await accStore.getAccountsBySubtype("checking");
  } finally {
    isLoadingAccounts.value = false;
  }
}

const toNumber = (val?: string | null) => {
  if (val == null) return 0;
  const n = Number(val);
  return isNaN(n) ? 0 : n;
};

const MAX_SLICES = 8;

const inflowData = computed(() =>
  vueHelper.groupPieSlices(
    (accBasicStats.value?.categories ?? [])
      .filter((c: CategoryStat) => toNumber(c.inflow) > 0)
      .map((c: CategoryStat) => ({
        label: c.category_name ?? `Category ${c.category_id}`,
        value: toNumber(c.inflow),
      })),
    MAX_SLICES,
  ),
);

const outflowData = computed(() =>
  vueHelper.groupPieSlices(
    (accBasicStats.value?.categories ?? [])
      .filter((c: CategoryStat) => toNumber(c.outflow) !== 0)
      .map((c: CategoryStat) => ({
        label: c.category_name ?? `Category ${c.category_id}`,
        value: Math.abs(toNumber(c.outflow)),
      })),
    MAX_SLICES,
  ),
);

const pieViews = [
  { label: "Inflows", value: "inflows" },
  { label: "Outflows", value: "outflows" },
];
const pieView = ref<"inflows" | "outflows">("inflows");

const hasInflowData = computed(() => inflowData.value.values.length > 0);
const hasOutflowData = computed(() => outflowData.value.values.length > 0);

const pieData = computed(() =>
  pieView.value === "inflows" ? inflowData.value : outflowData.value,
);

const statTiles = computed(() => {
  const s = accBasicStats.value;
  if (!s) return [];
  return [
    { label: "Total inflows", value: s.inflow, tone: "text-gain" },
    { label: "Total outflows", value: s.outflow, tone: "text-loss" },
    {
      label: "Avg. monthly inflows",
      value: s.avg_monthly_inflow,
      tone: "text-gain",
    },
    {
      label: "Avg. monthly outflows",
      value: s.avg_monthly_outflow,
      tone: "text-loss",
    },
    { label: "Take home", value: s.take_home, tone: "text-ink" },
    { label: "Overflow", value: s.overflow, tone: "text-ink" },
    {
      label: "Avg. monthly take home",
      value: s.avg_monthly_take_home,
      tone: "text-ink",
    },
    {
      label: "Avg. monthly overflow",
      value: s.avg_monthly_overflow,
      tone: "text-ink",
    },
  ];
});

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
            .map((v) => Number(v))
            .reduce((a, b) => a + b, 0);
          const pct = total ? value / total : 0;

          return `${label}: ${vueHelper.displayAsCurrency(value)} · ${vueHelper.displayAsPercentage(pct)}`;
        },
      },
    },
  },
}));
</script>

<template>
  <div v-if="accBasicStats" class="w-full flex flex-col gap-2 p-2">
    <div
      v-if="years.length > 0"
      class="flex flex-row gap-2 w-full justify-between items-center"
    >
      <div class="flex flex-col gap-2">
        <div class="flex flex-row">
          <span class="text-sm" style="color: var(--text-secondary)">
            Select which year you want to display statistics for. Current year
            will be used as a default.
          </span>
        </div>
      </div>

      <div class="flex flex-col gap-2">
        <Select
          v-model="selectedYear"
          size="small"
          style="width: 150px"
          :options="years"
        />
      </div>
    </div>

    <div
      v-if="!accID"
      class="flex flex-row gap-2 w-full justify-between items-center"
    >
      <div class="flex flex-col gap-2">
        <div class="flex flex-row">
          <span class="text-sm" style="color: var(--text-secondary)">
            A default checking account was found. The stats are representative
            of the cash flow to this account.
          </span>
        </div>
      </div>

      <div class="flex flex-col gap-2">
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
            <div class="flex flex-col">
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

    <div id="stats-row" class="flex flex-row w-full gap-6 p-1">
      <div class="grid grid-cols-2 gap-3 flex-1 min-w-0 content-start">
        <div
          v-for="tile in statTiles"
          :key="tile.label"
          class="flex flex-col gap-1.5 rounded-xl bg-sunken px-4 py-3.5 min-w-0"
        >
          <span class="text-xs text-muted leading-snug">{{ tile.label }}</span>
          <span class="text-base font-medium truncate" :class="tile.tone">
            {{ vueHelper.displayAsCurrency(tile.value) }}
          </span>
        </div>
      </div>

      <div
        class="flex flex-col flex-1 min-w-0 items-center justify-center gap-5"
      >
        <ShowLoading v-if="isLoading" :num-fields="4" />
        <template v-else-if="hasInflowData || hasOutflowData">
          <SelectButton
            v-model="pieView"
            size="small"
            :options="pieViews"
            option-label="label"
            option-value="value"
            :allow-empty="false"
          />
          <ComparativePieChart
            v-if="pieData.values.length"
            :key="pieView"
            :size="pieChartSize"
            :show-legend="false"
            :show-total="true"
            :options="pieOptions"
            :values="pieData.values"
            :labels="pieData.labels"
          />
          <div
            v-else
            class="flex items-center justify-center w-full rounded-xl border border-dashed border-line p-6"
          >
            <span class="text-sm text-muted">
              No {{ pieView }} found for {{ selectedYear }}.
            </span>
          </div>
        </template>
        <div
          v-else
          class="flex flex-col items-center justify-center gap-1 w-full rounded-xl border border-dashed border-line p-6"
        >
          <span class="text-sm text-muted">
            Not enough transactions found for {{ selectedYear }}.
          </span>
          <span class="text-xs text-faint">
            Keep inserting transactions to see the chart.
          </span>
        </div>
      </div>
    </div>
  </div>
  <ShowLoading v-else :num-fields="5" />
</template>

<style scoped>
@media (max-width: 768px) {
  #stats-row {
    flex-direction: column;
  }
  #stats-row > div {
    width: 100%;
  }
}
</style>
