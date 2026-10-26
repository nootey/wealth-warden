<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useToastStore } from "../../services/stores/toast_store.ts";
import ShowLoading from "../components/base/ShowLoading.vue";
import vueHelper from "../../utils/vue_helper.ts";
import { useAccountStore } from "../../services/stores/account_store.ts";
import { useAnalyticsStore } from "../../services/stores/analytics_store.ts";
import type {
  YearlyBreakdownStats,
  YearStatsWithAllocations,
} from "../../models/analytics_models.ts";
import type { Account } from "../../models/account_models.ts";

const props = defineProps<{
  accID?: number | null;
}>();

const analyticsStore = useAnalyticsStore();
const accStore = useAccountStore();
const toastStore = useToastStore();

const breakdownStats = ref<YearlyBreakdownStats | null>(null);
const years = ref<number[]>([]);
const selectedYear = ref<number>(new Date().getFullYear());
const selectedComparisonYear = ref<number | null>(null);
const accounts = ref<Account[]>([]);
const selectedAccountID = ref<number | null>(props.accID ?? null);

const isLoadingYears = ref(false);
const isLoadingStats = ref(false);
const isLoadingAccounts = ref(false);
const isLoading = computed(
  () => isLoadingYears.value || isLoadingStats.value || isLoadingAccounts.value,
);

const comparisonYearOptions = computed(() => {
  return years.value.filter((y) => y !== selectedYear.value);
});

const changeClass = ref<"expense" | "income">("expense");
const changeClassOptions = [
  { label: "Expenses", value: "expense" },
  { label: "Income", value: "income" },
];

const categoryChanges = computed(
  () => breakdownStats.value?.category_changes?.[changeClass.value] ?? [],
);

const changesPeriod = computed(() => {
  const month = breakdownStats.value?.category_changes?.through_month ?? 12;
  if (month >= 12) return "";
  const name = (m: number) =>
    new Date(2000, m - 1).toLocaleString("default", { month: "short" });
  return month === 1 ? name(1) : `${name(1)}–${name(month)}`;
});

const gridCols = computed(() =>
  breakdownStats.value?.comparison_year
    ? "grid-cols-[minmax(0,1fr)_6rem_5.5rem] md:grid-cols-[minmax(0,1fr)_8rem_8rem_8rem]"
    : "grid-cols-[minmax(0,1fr)_6rem] md:grid-cols-[minmax(0,1fr)_8rem]",
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
    if (selectedComparisonYear.value === newVal) {
      selectedComparisonYear.value = null;
    }
    try {
      await loadStats();
    } catch (e) {
      toastStore.errorResponseToast(e);
    }
  }
});

watch(selectedComparisonYear, async (newVal, oldVal) => {
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
    const res = await analyticsStore.getYearlyBreakdownStats(
      selectedAccountID.value ?? null,
      selectedYear.value,
      selectedComparisonYear.value,
    );
    breakdownStats.value = res.data;
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

const formatRate = (val: number) => {
  return `${val.toFixed(2)}%`;
};

const formatRateDiff = (diff: number) => {
  return `${diff > 0 ? "+" : ""}${diff.toFixed(2)} pp`;
};

const formatAmountDiff = (diff: number) => {
  return `${diff > 0 ? "+" : ""}${vueHelper.displayAsCurrency(diff)}`;
};

const calcDiff = (current: number, comparison: number) => {
  return Number((current - comparison).toFixed(2));
};

// Round before subtracting so the diff matches the rates shown on screen.
const calcRateDiff = (current: number, comparison: number) => {
  const diff = Number(current.toFixed(2)) - Number(comparison.toFixed(2));
  return Number(diff.toFixed(2));
};

const changeTone = (change: number | null, better?: "up" | "down") => {
  if (!change || !better) return "text-muted";
  return change > 0 === (better === "up") ? "text-gain" : "text-loss";
};

type RowDef = {
  key: string;
  label: string;
  value: (y: YearStatsWithAllocations) => number;
  share?: (y: YearStatsWithAllocations) => number;
  trackShare?: boolean;
  better?: "up" | "down";
};

const spend = (v: string) => Math.abs(Number(v));

const sectionDefs: {
  key: string;
  label: string;
  hint: string;
  rows: RowDef[];
}[] = [
  {
    key: "cash-flow",
    label: "Cash flow",
    hint: "Money coming in and going out",
    rows: [
      {
        key: "inflow",
        label: "Total inflows",
        value: (y) => Number(y.inflow),
        better: "up",
      },
      {
        key: "outflow",
        label: "Total outflows",
        value: (y) => spend(y.outflow),
        better: "down",
      },
      {
        key: "avg-inflow",
        label: "Avg monthly inflows",
        value: (y) => Number(y.avg_monthly_inflow),
        better: "up",
      },
      {
        key: "avg-outflow",
        label: "Avg monthly outflows",
        value: (y) => spend(y.avg_monthly_outflow),
        better: "down",
      },
    ],
  },
  {
    key: "allocations",
    label: "Allocations",
    hint: "Moved to savings, investments and debt · % of income",
    rows: [
      {
        key: "savings",
        label: "Savings",
        value: (y) => Number(y.savings_allocated),
        share: (y) => y.savings_pct,
        trackShare: true,
        better: "up",
      },
      {
        key: "investments",
        label: "Investments",
        value: (y) => Number(y.investment_allocated),
        share: (y) => y.investment_pct,
        trackShare: true,
        better: "up",
      },
      {
        key: "debt",
        label: "Debt payments",
        value: (y) => Number(y.debt_allocated),
        share: (y) => y.debt_pct,
        trackShare: true,
      },
    ],
  },
  {
    key: "necessity",
    label: "Needs vs wants",
    hint: "Expenses by category type · % of expenses",
    rows: [
      {
        key: "needs",
        label: "Needs",
        value: (y) => Number(y.necessity.needs),
        share: (y) => y.necessity.needs_pct,
        better: "down",
      },
      {
        key: "wants",
        label: "Wants",
        value: (y) => Number(y.necessity.wants),
        share: (y) => y.necessity.wants_pct,
        better: "down",
      },
      {
        key: "other",
        label: "Other",
        value: (y) => Number(y.necessity.other),
        share: (y) => y.necessity.other_pct,
        better: "down",
      },
    ],
  },
  {
    key: "net",
    label: "Net position",
    hint: "Left over or overspent after allocations",
    rows: [
      {
        key: "net",
        label: "Net",
        value: (y) => Number(y.net),
        better: "up",
      },
      {
        key: "avg-net",
        label: "Avg monthly net",
        value: (y) => Number(y.avg_monthly_net),
        better: "up",
      },
    ],
  },
];

const sections = computed(() => {
  const cur = breakdownStats.value?.current_year;
  if (!cur) return [];
  const cmp = breakdownStats.value?.comparison_year ?? null;

  return sectionDefs.map((s) => ({
    key: s.key,
    label: s.label,
    hint: s.hint,
    rows: s.rows
      .filter(
        (r) =>
          r.key !== "other" ||
          r.value(cur) > 0 ||
          (cmp !== null && r.value(cmp) > 0),
      )
      .map((r) => {
        const change = cmp ? calcDiff(r.value(cur), r.value(cmp)) : null;
        const shareChange =
          cmp && r.share && r.trackShare
            ? calcRateDiff(r.share(cur), r.share(cmp))
            : null;
        return {
          key: r.key,
          label: r.label,
          current: vueHelper.displayAsCurrency(r.value(cur)),
          comparison: cmp ? vueHelper.displayAsCurrency(r.value(cmp)) : null,
          currentShare: r.share ? formatRate(r.share(cur)) : null,
          comparisonShare: r.share && cmp ? formatRate(r.share(cmp)) : null,
          change: change === null ? null : formatAmountDiff(change),
          tone: changeTone(change, r.better),
          shareChange:
            shareChange === null ? null : formatRateDiff(shareChange),
          shareTone: changeTone(shareChange, r.better),
        };
      }),
  }));
});

const formatChangePct = (pct?: number | null) => {
  if (pct === null || pct === undefined) return "new";
  const rounded = Math.round(pct);
  return `${rounded >= 0 ? "+" : ""}${rounded}%`;
};
</script>

<template>
  <div v-if="breakdownStats" class="w-full flex flex-col gap-4 p-2">
    <div class="flex flex-row flex-wrap gap-3 w-full justify-between items-end">
      <span class="mobile-hide text-sm text-muted max-w-md">
        Compare a year with another one, side by side. Cash flow comes from the
        selected checking account.
      </span>

      <div class="grid grid-cols-2 gap-2 w-full md:flex md:flex-row md:w-auto">
        <div class="flex flex-col gap-1 min-w-0 md:w-40">
          <label class="text-xs text-muted">Year</label>
          <Select
            v-model="selectedYear"
            size="small"
            class="w-full"
            :options="years"
          />
        </div>

        <div class="flex flex-col gap-1 min-w-0 md:w-40">
          <label class="text-xs text-muted">Compare to</label>
          <Select
            v-model="selectedComparisonYear"
            size="small"
            class="w-full"
            :options="comparisonYearOptions"
            placeholder="None"
            show-clear
          />
        </div>

        <div v-if="!accID" class="col-span-2 flex flex-col gap-1 md:w-40">
          <label class="text-xs text-muted">Account</label>
          <Select
            v-model="selectedAccountID"
            size="small"
            class="w-full"
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
                <span class="text-xs text-muted">
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
    </div>

    <ShowLoading v-if="isLoading" :num-fields="5" />
    <template v-else>
      <section
        class="flex flex-col rounded-2xl border border-line overflow-hidden divide-y divide-line"
      >
        <div
          class="grid items-center gap-3 bg-sunken px-4 py-2.5 text-xs text-muted"
          :class="gridCols"
        >
          <span v-if="breakdownStats.comparison_year" />
          <span v-else class="text-faint">Pick a year to compare against</span>
          <span class="text-right font-medium text-ink">
            {{ breakdownStats.current_year.year }}
          </span>
          <template v-if="breakdownStats.comparison_year">
            <span class="hidden md:block text-right">
              {{ breakdownStats.comparison_year.year }}
            </span>
            <span class="text-right">
              <span class="md:hidden">
                vs {{ breakdownStats.comparison_year.year }}
              </span>
              <span class="hidden md:inline">Change</span>
            </span>
          </template>
        </div>

        <div
          v-for="section in sections"
          :key="section.key"
          class="flex flex-col px-4 pt-4 pb-1"
        >
          <div class="flex flex-col gap-0.5 pb-1">
            <span
              class="text-sm md:text-base font-medium tracking-tight text-ink"
            >
              {{ section.label }}
            </span>
            <span class="text-xs text-faint">{{ section.hint }}</span>
          </div>
          <div class="flex flex-col divide-y divide-line/60">
            <div
              v-for="row in section.rows"
              :key="row.key"
              class="grid items-baseline gap-3 py-2.5 text-xs md:text-sm"
              :class="gridCols"
            >
              <span class="text-muted">{{ row.label }}</span>
              <div class="flex flex-col items-end">
                <span
                  class="font-medium text-ink tabular-nums whitespace-nowrap"
                >
                  {{ row.current }}
                </span>
                <span
                  v-if="row.currentShare"
                  class="text-xs text-faint tabular-nums"
                >
                  {{ row.currentShare }}
                </span>
              </div>
              <template v-if="breakdownStats.comparison_year">
                <div class="hidden md:flex flex-col items-end">
                  <span class="text-muted tabular-nums whitespace-nowrap">
                    {{ row.comparison }}
                  </span>
                  <span
                    v-if="row.comparisonShare"
                    class="text-xs text-faint tabular-nums"
                  >
                    {{ row.comparisonShare }}
                  </span>
                </div>
                <div class="flex flex-col items-end">
                  <span
                    class="font-medium tabular-nums whitespace-nowrap"
                    :class="row.tone"
                  >
                    {{ row.change }}
                  </span>
                  <span
                    v-if="row.shareChange"
                    class="text-xs tabular-nums whitespace-nowrap"
                    :class="row.shareTone"
                  >
                    {{ row.shareChange }}
                  </span>
                </div>
              </template>
            </div>
          </div>
        </div>
      </section>

      <section
        v-if="breakdownStats.category_changes && breakdownStats.comparison_year"
        class="flex flex-col rounded-2xl border border-line overflow-hidden divide-y divide-line"
      >
        <div
          class="flex flex-col items-center gap-3 px-4 py-3 text-center md:flex-row md:justify-between md:text-left"
        >
          <div class="flex flex-col gap-0.5">
            <span
              class="text-sm md:text-base font-medium tracking-tight text-ink"
            >
              Biggest changes
            </span>
            <span class="text-xs text-faint">
              Top categories that moved the most vs
              {{ breakdownStats.comparison_year.year }}
              <template v-if="changesPeriod">· {{ changesPeriod }}</template>
            </span>
          </div>
          <SelectButton
            v-model="changeClass"
            size="small"
            :options="changeClassOptions"
            option-label="label"
            option-value="value"
            :allow-empty="false"
          />
        </div>

        <div
          class="grid items-center gap-3 bg-sunken px-4 py-2.5 text-xs text-muted"
          :class="gridCols"
        >
          <span>Category</span>
          <span class="text-right font-medium text-ink">
            {{ breakdownStats.current_year.year }}
          </span>
          <span class="hidden md:block text-right">
            {{ breakdownStats.comparison_year.year }}
          </span>
          <span class="text-right">
            <span class="md:hidden">
              vs {{ breakdownStats.comparison_year.year }}
            </span>
            <span class="hidden md:inline">Change</span>
          </span>
        </div>

        <div class="flex flex-col px-4 divide-y divide-line/60">
          <div
            v-for="item in categoryChanges"
            :key="item.category_id"
            class="grid items-baseline gap-3 py-2.5 text-xs md:text-sm"
            :class="gridCols"
          >
            <span class="text-ink truncate">{{ item.category }}</span>
            <span
              class="text-right font-medium text-ink tabular-nums whitespace-nowrap"
            >
              {{ vueHelper.displayAsCurrency(item.current) }}
            </span>
            <span
              class="hidden md:block text-right text-muted tabular-nums whitespace-nowrap"
            >
              {{ vueHelper.displayAsCurrency(item.comparison) }}
            </span>
            <div
              class="flex flex-col items-end"
              :class="
                changeTone(
                  Number(item.change),
                  changeClass === 'expense' ? 'down' : 'up',
                )
              "
            >
              <span class="font-medium tabular-nums whitespace-nowrap">
                {{ formatAmountDiff(Number(item.change)) }}
              </span>
              <span class="text-xs tabular-nums">
                {{ formatChangePct(item.change_pct) }}
              </span>
            </div>
          </div>

          <span v-if="!categoryChanges.length" class="py-3 text-sm text-muted">
            No category changes between these years.
          </span>
        </div>
      </section>
    </template>
  </div>
  <ShowLoading v-else :num-fields="5" />
</template>
