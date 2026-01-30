<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import type { YearlyBreakdownStats } from "../../models/statistics_models.ts";
import { useStatisticsStore } from "../../services/stores/statistics_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import ShowLoading from "../components/base/ShowLoading.vue";
import vueHelper from "../../utils/vue_helper.ts";
import type { Account } from "../../models/account_models.ts";
import { useAccountStore } from "../../services/stores/account_store.ts";
import { useChartColors } from "../../style/theme/chartColors.ts";

const props = defineProps<{
  accID?: number | null;
}>();

const statsStore = useStatisticsStore();
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
const { colors } = useChartColors();

const comparisonYearOptions = computed(() => {
  return years.value.filter((y) => y !== selectedYear.value);
});

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
    const res = await statsStore.getYearlyBreakdownStats(
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
    const result = await statsStore.getAvailableStatsYears(
      selectedAccountID.value ?? null,
    );
    years.value = Array.isArray(result) ? result : [];

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

const formatPct = (val: number) => {
  return `${val.toFixed(1)}%`;
};

const calcDiff = (current: string, comparison: string) => {
  const curr = Number(current);
  const comp = Number(comparison);
  return curr - comp;
};

const calcPctDiff = (current: number, comparison: number) => {
  return current - comparison;
};

const getDiffColor = (diff: number) => {
  if (diff === 0) return colors.value.dim;
  return diff > 0 ? colors.value.pos : colors.value.neg;
};
</script>

<template>
  <div v-if="breakdownStats" class="w-full flex flex-column gap-3 p-3">
    <div
      class="flex flex-row gap-3 w-full justify-content-between align-items-center"
    >
      <div class="mobile-hide flex flex-column gap-2 flex-1">
        <span class="text-sm" style="color: var(--text-secondary)">
          Select year and optional comparison year
        </span>
      </div>

      <div class="flex flex-row gap-2">
        <div class="flex flex-column gap-1">
          <label class="text-xs" style="color: var(--text-secondary)"
            >Year</label
          >
          <Select
            v-model="selectedYear"
            size="small"
            style="width: 150px"
            :options="years"
          />
        </div>

        <div class="flex flex-column gap-1">
          <label class="text-xs" style="color: var(--text-secondary)"
            >Compare to</label
          >
          <Select
            v-model="selectedComparisonYear"
            size="small"
            style="width: 150px"
            :options="comparisonYearOptions"
            placeholder="None"
            show-clear
          />
        </div>
      </div>
    </div>

    <div
      v-if="!accID"
      class="flex flex-row gap-2 w-full justify-content-between align-items-center"
    >
      <div class="mobile-hide flex flex-column gap-2 w-full">
        <span class="text-sm" style="color: var(--text-secondary)">
          Select checking account for cash flow analysis
        </span>
      </div>

      <div id="wide" class="flex flex-column gap-2">
        <Select
          id="wide"
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

    <ShowLoading v-if="isLoading" :num-fields="5" />
    <div v-else id="stats-container" class="flex flex-row gap-4 w-full">
      <div class="flex flex-column gap-3 flex-1">
        <h4 style="color: var(--text-primary)">
          {{ breakdownStats.current_year.year }}
          <span
            v-if="breakdownStats.comparison_year"
            class="mobile-comparison-header text-sm"
            style="color: var(--text-secondary)"
          >
            (vs {{ breakdownStats.comparison_year.year }})
          </span>
        </h4>

        <div
          class="flex flex-column gap-2 p-3"
          style="background: var(--surface-50); border-radius: 8px"
        >
          <span
            class="font-semibold text-sm"
            style="color: var(--text-secondary)"
            >Cash Flow</span
          >

          <div class="flex flex-row justify-content-between mobile-small">
            <span>Total Inflows:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(breakdownStats.current_year.inflow)
              }}</b>
              <span
                v-if="breakdownStats.comparison_year"
                class="mobile-comparison text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.inflow,
                      breakdownStats.comparison_year.inflow,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.inflow,
                    breakdownStats.comparison_year.inflow,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.inflow,
                      breakdownStats.comparison_year.inflow,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between mobile-small">
            <span>Total Outflows:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(breakdownStats.current_year.outflow)
              }}</b>
              <span
                v-if="breakdownStats.comparison_year"
                class="mobile-comparison text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.outflow,
                      breakdownStats.comparison_year.outflow,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.outflow,
                    breakdownStats.comparison_year.outflow,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.outflow,
                      breakdownStats.comparison_year.outflow,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between mobile-small">
            <span>Avg Monthly Inflows:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.current_year.avg_monthly_inflow,
                )
              }}</b>
              <span
                v-if="breakdownStats.comparison_year"
                class="mobile-comparison text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_inflow,
                      breakdownStats.comparison_year.avg_monthly_inflow,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.avg_monthly_inflow,
                    breakdownStats.comparison_year.avg_monthly_inflow,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_inflow,
                      breakdownStats.comparison_year.avg_monthly_inflow,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between mobile-small">
            <span>Avg Monthly Outflows:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.current_year.avg_monthly_outflow,
                )
              }}</b>
              <span
                v-if="breakdownStats.comparison_year"
                class="mobile-comparison text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_outflow,
                      breakdownStats.comparison_year.avg_monthly_outflow,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.avg_monthly_outflow,
                    breakdownStats.comparison_year.avg_monthly_outflow,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_outflow,
                      breakdownStats.comparison_year.avg_monthly_outflow,
                    ),
                  )
                }})
              </span>
            </div>
          </div>
        </div>

        <div
          class="flex flex-column gap-2 p-3"
          style="background: var(--surface-50); border-radius: 8px"
        >
          <span
            class="font-semibold text-sm"
            style="color: var(--text-secondary)"
            >Allocations</span
          >

          <div class="flex flex-row justify-content-between mobile-small">
            <span>Savings:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.current_year.savings_allocated,
                )
              }}</b>
              <span class="text-sm" style="color: var(--text-secondary)">
                ({{ formatPct(breakdownStats.current_year.savings_pct) }})
              </span>
              <span
                v-if="breakdownStats.comparison_year"
                class="mobile-comparison text-xs"
                :style="{
                  color: getDiffColor(
                    calcPctDiff(
                      breakdownStats.current_year.savings_pct,
                      breakdownStats.comparison_year.savings_pct,
                    ),
                  ),
                }"
              >
                ({{
                  calcPctDiff(
                    breakdownStats.current_year.savings_pct,
                    breakdownStats.comparison_year.savings_pct,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  formatPct(
                    calcPctDiff(
                      breakdownStats.current_year.savings_pct,
                      breakdownStats.comparison_year.savings_pct,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between mobile-small">
            <span>Investments:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.current_year.investment_allocated,
                )
              }}</b>
              <span class="text-sm" style="color: var(--text-secondary)">
                ({{ formatPct(breakdownStats.current_year.investment_pct) }})
              </span>
              <span
                v-if="breakdownStats.comparison_year"
                class="mobile-comparison text-xs"
                :style="{
                  color: getDiffColor(
                    calcPctDiff(
                      breakdownStats.current_year.investment_pct,
                      breakdownStats.comparison_year.investment_pct,
                    ),
                  ),
                }"
              >
                ({{
                  calcPctDiff(
                    breakdownStats.current_year.investment_pct,
                    breakdownStats.comparison_year.investment_pct,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  formatPct(
                    calcPctDiff(
                      breakdownStats.current_year.investment_pct,
                      breakdownStats.comparison_year.investment_pct,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between mobile-small">
            <span>Debt Payments:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.current_year.debt_allocated,
                )
              }}</b>
              <span class="text-sm" style="color: var(--text-secondary)">
                ({{ formatPct(breakdownStats.current_year.debt_pct) }})
              </span>
              <span
                v-if="breakdownStats.comparison_year"
                class="mobile-comparison text-xs"
                :style="{
                  color: getDiffColor(
                    calcPctDiff(
                      breakdownStats.current_year.debt_pct,
                      breakdownStats.comparison_year.debt_pct,
                    ),
                  ),
                }"
              >
                ({{
                  calcPctDiff(
                    breakdownStats.current_year.debt_pct,
                    breakdownStats.comparison_year.debt_pct,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  formatPct(
                    calcPctDiff(
                      breakdownStats.current_year.debt_pct,
                      breakdownStats.comparison_year.debt_pct,
                    ),
                  )
                }})
              </span>
            </div>
          </div>
        </div>

        <div
          class="flex flex-column gap-2 p-3"
          style="background: var(--surface-50); border-radius: 8px"
        >
          <span
            class="font-semibold text-sm"
            style="color: var(--text-secondary)"
            >Net Position</span
          >

          <div class="flex flex-row justify-content-between mobile-small">
            <span>Take Home:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.current_year.take_home,
                )
              }}</b>
              <span
                v-if="breakdownStats.comparison_year"
                class="mobile-comparison text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.take_home,
                      breakdownStats.comparison_year.take_home,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.take_home,
                    breakdownStats.comparison_year.take_home,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.take_home,
                      breakdownStats.comparison_year.take_home,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between mobile-small">
            <span>Overflow:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.current_year.overflow,
                )
              }}</b>
              <span
                v-if="breakdownStats.comparison_year"
                class="mobile-comparison text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.overflow,
                      breakdownStats.comparison_year.overflow,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.overflow,
                    breakdownStats.comparison_year.overflow,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.overflow,
                      breakdownStats.comparison_year.overflow,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between mobile-small">
            <span>Avg Monthly Take Home:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.current_year.avg_monthly_take_home,
                )
              }}</b>
              <span
                v-if="breakdownStats.comparison_year"
                class="mobile-comparison text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_take_home,
                      breakdownStats.comparison_year.avg_monthly_take_home,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.avg_monthly_take_home,
                    breakdownStats.comparison_year.avg_monthly_take_home,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_take_home,
                      breakdownStats.comparison_year.avg_monthly_take_home,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between mobile-small">
            <span>Avg Monthly Overflow:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.current_year.avg_monthly_overflow,
                )
              }}</b>
              <span
                v-if="breakdownStats.comparison_year"
                class="mobile-comparison text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_overflow,
                      breakdownStats.comparison_year.avg_monthly_overflow,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.avg_monthly_overflow,
                    breakdownStats.comparison_year.avg_monthly_overflow,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_overflow,
                      breakdownStats.comparison_year.avg_monthly_overflow,
                    ),
                  )
                }})
              </span>
            </div>
          </div>
        </div>
      </div>

      <div
        v-if="breakdownStats.comparison_year"
        id="desktop-comparison"
        class="flex flex-column gap-3 flex-1"
      >
        <h4 style="color: var(--text-primary)">
          {{ breakdownStats.comparison_year.year }}
          <span class="text-sm" style="color: var(--text-secondary)"
            >(vs {{ breakdownStats.current_year.year }})</span
          >
        </h4>

        <div
          class="flex flex-column gap-2 p-3"
          style="background: var(--surface-50); border-radius: 8px"
        >
          <span
            class="font-semibold text-sm"
            style="color: var(--text-secondary)"
            >Cash Flow</span
          >

          <div class="flex flex-row justify-content-between">
            <span>Total Inflows:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.comparison_year.inflow,
                )
              }}</b>
              <span
                class="text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.inflow,
                      breakdownStats.comparison_year.inflow,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.inflow,
                    breakdownStats.comparison_year.inflow,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.inflow,
                      breakdownStats.comparison_year.inflow,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between">
            <span>Total Outflows:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.comparison_year.outflow,
                )
              }}</b>
              <span
                class="text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.outflow,
                      breakdownStats.comparison_year.outflow,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.outflow,
                    breakdownStats.comparison_year.outflow,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.outflow,
                      breakdownStats.comparison_year.outflow,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between">
            <span>Avg Monthly Inflows:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.comparison_year.avg_monthly_inflow,
                )
              }}</b>
              <span
                class="text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_inflow,
                      breakdownStats.comparison_year.avg_monthly_inflow,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.avg_monthly_inflow,
                    breakdownStats.comparison_year.avg_monthly_inflow,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_inflow,
                      breakdownStats.comparison_year.avg_monthly_inflow,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between">
            <span>Avg Monthly Outflows:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.comparison_year.avg_monthly_outflow,
                )
              }}</b>
              <span
                class="text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_outflow,
                      breakdownStats.comparison_year.avg_monthly_outflow,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.avg_monthly_outflow,
                    breakdownStats.comparison_year.avg_monthly_outflow,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_outflow,
                      breakdownStats.comparison_year.avg_monthly_outflow,
                    ),
                  )
                }})
              </span>
            </div>
          </div>
        </div>

        <div
          class="flex flex-column gap-2 p-3"
          style="background: var(--surface-50); border-radius: 8px"
        >
          <span
            class="font-semibold text-sm"
            style="color: var(--text-secondary)"
            >Allocations</span
          >

          <div class="flex flex-row justify-content-between">
            <span>Savings:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.comparison_year.savings_allocated,
                )
              }}</b>
              <span class="text-sm" style="color: var(--text-secondary)">
                ({{ formatPct(breakdownStats.comparison_year.savings_pct) }})
              </span>
              <span
                class="text-xs"
                :style="{
                  color: getDiffColor(
                    calcPctDiff(
                      breakdownStats.current_year.savings_pct,
                      breakdownStats.comparison_year.savings_pct,
                    ),
                  ),
                }"
              >
                ({{
                  calcPctDiff(
                    breakdownStats.current_year.savings_pct,
                    breakdownStats.comparison_year.savings_pct,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  formatPct(
                    calcPctDiff(
                      breakdownStats.current_year.savings_pct,
                      breakdownStats.comparison_year.savings_pct,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between">
            <span>Investments:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.comparison_year.investment_allocated,
                )
              }}</b>
              <span class="text-sm" style="color: var(--text-secondary)">
                ({{ formatPct(breakdownStats.comparison_year.investment_pct) }})
              </span>
              <span
                class="text-xs"
                :style="{
                  color: getDiffColor(
                    calcPctDiff(
                      breakdownStats.current_year.investment_pct,
                      breakdownStats.comparison_year.investment_pct,
                    ),
                  ),
                }"
              >
                ({{
                  calcPctDiff(
                    breakdownStats.current_year.investment_pct,
                    breakdownStats.comparison_year.investment_pct,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  formatPct(
                    calcPctDiff(
                      breakdownStats.current_year.investment_pct,
                      breakdownStats.comparison_year.investment_pct,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between">
            <span>Debt Payments:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.comparison_year.debt_allocated,
                )
              }}</b>
              <span class="text-sm" style="color: var(--text-secondary)">
                ({{ formatPct(breakdownStats.comparison_year.debt_pct) }})
              </span>
              <span
                class="text-xs"
                :style="{
                  color: getDiffColor(
                    calcPctDiff(
                      breakdownStats.current_year.debt_pct,
                      breakdownStats.comparison_year.debt_pct,
                    ),
                  ),
                }"
              >
                ({{
                  calcPctDiff(
                    breakdownStats.current_year.debt_pct,
                    breakdownStats.comparison_year.debt_pct,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  formatPct(
                    calcPctDiff(
                      breakdownStats.current_year.debt_pct,
                      breakdownStats.comparison_year.debt_pct,
                    ),
                  )
                }})
              </span>
            </div>
          </div>
        </div>

        <div
          class="flex flex-column gap-2 p-3"
          style="background: var(--surface-50); border-radius: 8px"
        >
          <span
            class="font-semibold text-sm"
            style="color: var(--text-secondary)"
            >Net Position</span
          >

          <div class="flex flex-row justify-content-between">
            <span>Take Home:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.comparison_year.take_home,
                )
              }}</b>
              <span
                class="text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.take_home,
                      breakdownStats.comparison_year.take_home,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.take_home,
                    breakdownStats.comparison_year.take_home,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.take_home,
                      breakdownStats.comparison_year.take_home,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between">
            <span>Overflow:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.comparison_year.overflow,
                )
              }}</b>
              <span
                class="text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.overflow,
                      breakdownStats.comparison_year.overflow,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.overflow,
                    breakdownStats.comparison_year.overflow,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.overflow,
                      breakdownStats.comparison_year.overflow,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between">
            <span>Avg Monthly Take Home:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.comparison_year.avg_monthly_take_home,
                )
              }}</b>
              <span
                class="text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_take_home,
                      breakdownStats.comparison_year.avg_monthly_take_home,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.avg_monthly_take_home,
                    breakdownStats.comparison_year.avg_monthly_take_home,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_take_home,
                      breakdownStats.comparison_year.avg_monthly_take_home,
                    ),
                  )
                }})
              </span>
            </div>
          </div>

          <div class="flex flex-row justify-content-between">
            <span>Avg Monthly Overflow:</span>
            <div class="flex flex-row gap-2 align-items-center">
              <b>{{
                vueHelper.displayAsCurrency(
                  breakdownStats.comparison_year.avg_monthly_overflow,
                )
              }}</b>
              <span
                class="text-xs"
                :style="{
                  color: getDiffColor(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_overflow,
                      breakdownStats.comparison_year.avg_monthly_overflow,
                    ),
                  ),
                }"
              >
                ({{
                  calcDiff(
                    breakdownStats.current_year.avg_monthly_overflow,
                    breakdownStats.comparison_year.avg_monthly_overflow,
                  ) >= 0
                    ? "+"
                    : ""
                }}{{
                  vueHelper.displayAsCurrency(
                    calcDiff(
                      breakdownStats.current_year.avg_monthly_overflow,
                      breakdownStats.comparison_year.avg_monthly_overflow,
                    ),
                  )
                }})
              </span>
            </div>
          </div>
        </div>
      </div>

      <div
        v-else
        class="flex flex-column gap-3 flex-1 justify-content-center align-items-center p-4"
        style="border: 1px dashed var(--border-color); border-radius: 8px"
      >
        <span class="text-sm" style="color: var(--text-secondary)">
          Select a comparison year to see side-by-side stats
        </span>
      </div>
    </div>
  </div>
  <ShowLoading v-else :num-fields="5" />
</template>

<style scoped>
.mobile-comparison,
.mobile-comparison-header {
  display: none;
}

@media (max-width: 768px) {
  #desktop-comparison {
    display: none !important;
  }

  .mobile-comparison,
  .mobile-comparison-header {
    display: inline;
  }

  #stats-container {
    flex-direction: column;
  }

  #wide {
    width: 100% !important;
  }

  .mobile-small {
    font-size: 0.75rem !important;
  }

  .mobile-small b,
  .mobile-small span {
    font-size: 0.75rem !important;
  }
}
</style>
