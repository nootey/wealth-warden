<script setup lang="ts">
import {computed, nextTick, onMounted, ref, watch} from "vue";
import MultiSelect from "primevue/multiselect";
import CategoryBreakdownChart from "../components/charts/CategoryBreakdownChart.vue";

import { useToastStore } from "../../services/stores/toast_store.ts";
import { useChartStore } from "../../services/stores/chart_store.ts";
import type {Category} from "../../models/transaction_models.ts";
import {useTransactionStore} from "../../services/stores/transaction_store.ts";
import Select from "primevue/select";
import type {YearlyCategoryStats} from "../../models/chart_models.ts";
import vueHelper from "../../utils/vue_helper.ts";
import type {Account} from "../../models/account_models.ts";
import {useAccountStore} from "../../services/stores/account_store.ts";

const chartStore = useChartStore();
const toastStore = useToastStore();
const transactionStore = useTransactionStore();
const accStore = useAccountStore();

const allYears = ref<number[]>([]);
const selectedYears = ref<number[]>([]);
const maxYears = 5;

const series = ref<{ name: string; data: number[] }[]>([]);
const stats = ref<YearlyCategoryStats | null>(null);
const accounts = ref<Account[]>([]);
const selectedAccountID = ref<number | null>(null);
const selectedClassification = ref<'income' | 'expense'>('expense');

type OptionItem = { label: string; value: number | undefined; meta: Category }
const yearOptions = computed(() =>
    allYears.value.map(y => ({ label: String(y), value: y }))
);

const ALL_CATEGORY = {
    id: undefined,
    name: 'All',
    display_name: 'All',
    parent_id: null
} as unknown as Category

const availableCategories = computed<Category[]>(() => [
    ALL_CATEGORY,
    ...transactionStore.categories.filter(c =>
        (c.classification == selectedClassification.value && c.parent_id) ||
        c.classification == "uncategorized"
    )
])

const selectedCategoryId = ref<number | undefined>(ALL_CATEGORY.id!)

const selectedCategory = computed<Category>(() =>
    availableCategories.value.find(c => c.id === selectedCategoryId.value) ?? ALL_CATEGORY
)

const categoryOptions = computed((): OptionItem[] => {
    const order = ['_general', 'income', 'expense', 'investments', 'savings'] as const
    const keyOf = (c: Category) => (c === ALL_CATEGORY ? '_general' : (c.classification ?? 'other'))

    return availableCategories.value
        .map((c): OptionItem => ({
            label: (c.display_name ?? c.name ?? '') as string,
            value: c.id!,
            meta: c
        }))
        .sort((a, b) => {
            const ak = order.indexOf(keyOf(a.meta) as any)
            const bk = order.indexOf(keyOf(b.meta) as any)
            const ai = ak === -1 ? Number.POSITIVE_INFINITY : ak
            const bi = bk === -1 ? Number.POSITIVE_INFINITY : bk
            return ai - bi || a.label.localeCompare(b.label)
        })
})

const hasAnyData = computed(() =>
    series.value.some(s => s.data?.some(v => Number(v) > 0))
);

const loadYears = async () => {
    const now = new Date().getFullYear();
    allYears.value = [now, now - 1, now - 2, now - 3, now - 4];
    selectedYears.value = [now, now - 1, now - 2].filter(y => y >= Math.min(...allYears.value)).slice(0, maxYears);
};

const fetchData = async () => {
    try {
        if (!selectedYears.value.length) return;

        const res = await chartStore.getMultiYearMonthlyCategoryBreakdown({
            years: selectedYears.value.slice(0, maxYears),
            class: selectedClassification.value,
            percent: false,
            category: selectedCategory.value?.id ?? null,
            account: selectedAccountID.value ?? null
        });

        const ys: number[] = res?.years ?? [];
        const by = res?.by_year ?? {};
        stats.value = res?.stats ?? null;

        series.value = ys.map((y: number) => {
            const data = new Array(12).fill(0);

            if (Array.isArray(by?.[y]?.series)) {
                for (const row of by[y].series) {
                    const m = Number(row?.month);
                    const idx = m >= 1 && m <= 12 ? m - 1 : -1;
                    const n = typeof row?.amount === "string" ? Number(row.amount) : Number(row?.amount ?? 0);

                    if (idx >= 0 && isFinite(n)) {
                        data[idx] += n;
                    }
                }
            }

            return { name: String(y), data };
        });

        await nextTick();

    } catch (e) {
        toastStore.errorResponseToast(e);
    }
};

async function loadAccounts() {
    try {
        accounts.value = await accStore.getAccountsBySubtype("checking");
    } catch (e) {
        toastStore.errorResponseToast(e);
    }
}

onMounted(async () => {
    await loadYears();
    await fetchData();
    await loadAccounts();
    const defaultChecking = accounts.value.find(
        acc => acc.is_default && acc.account_type?.sub_type === 'checking'
    );
    if (defaultChecking) {
        selectedAccountID.value = defaultChecking.id;
    }
});

watch(
    () => [selectedYears.value, selectedCategoryId.value, selectedAccountID.value, selectedClassification.value] as const,
    async ([years, category, account, classification], [oldYears, oldCategory, oldAccount, oldClassification]) => {
        if (years.length > maxYears) {
            selectedYears.value = years.slice(0, maxYears);
            return;
        }

        if (
            JSON.stringify(years) !== JSON.stringify(oldYears) ||
            category !== oldCategory ||
            account !== oldAccount ||
            classification !== oldClassification
        ) {
            await fetchData();
        }
    },
    { deep: true }
);

</script>

<template>
  <div class="flex flex-column w-full p-3 gap-3">
    <div class="flex flex-row gap-2 w-full justify-content-between align-items-center">
      <div class="mobile-hide flex flex-column gap-1 flex-grow-1">
        <span
          class="text-sm"
          style="color: var(--text-secondary)"
        >
          View and compare category totals by month.
        </span>
      </div>

      <div class="flex flex-row gap-2 flex-shrink-0 select-container">
        <Select
          v-model="selectedCategoryId"
          size="small"
          filter
          class="select-width"
          :options="categoryOptions"
          option-label="label"
          option-value="value"
        >
          <template #value="{ value }">
            {{
              availableCategories.find(c => c.id === value)?.display_name
                ?? availableCategories.find(c => c.id === value)?.name
                ?? (value === undefined ? 'All' : 'Select category')
            }}
          </template>

          <template #option="{ option }">
            <div class="flex justify-content-between w-full">
              <span>{{ option.label }}</span>
              <small class="text-color-secondary">
                {{ option.meta?.classification }}
              </small>
            </div>
          </template>
        </Select>
        <MultiSelect
          v-model="selectedYears"
          :options="yearOptions"
          :max-selected-labels="5"
          :selection-limit="5"
          size="small"
          class="select-width"
          placeholder="Years"
          option-label="label"
          option-value="value"
        />
      </div>
    </div>

    <div class="flex flex-row gap-2 w-full justify-content-between align-items-center">
      <div class="mobile-hide flex flex-column gap-2 flex-grow-1">
        <span
          class="text-sm"
          style="color: var(--text-secondary)"
        >
          A default checking account was found. The stats are representative of the cash flow to this account.
        </span>
      </div>

      <div class="flex flex-row gap-2 flex-shrink-0 select-container">
        <Select
          v-model="selectedAccountID"
          size="small"
          class="select-width"
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
        <Select
          v-model="selectedClassification"
          size="small"
          class="select-width"
          :options="[
            { label: 'Expenses', value: 'expense' },
            { label: 'Income', value: 'income' }
          ]"
          option-label="label"
          option-value="value"
        />
      </div>
    </div>

    <div class="flex flex-row w-full justify-content-center align-items-center">
      <CategoryBreakdownChart
        v-if="hasAnyData"
        :series="series"
      />
      <div
        v-else
        class="flex flex-column align-items-center justify-content-center mt-3"
        style="height: 400px"
      >
        <span
          class="text-sm p-3"
          style="color: var(--text-secondary); border: 1px dashed var(--border-color); border-radius: 16px;"
        >
          Not enough data to display for:
          {{ selectedCategory?.display_name ?? 'any category' }}
          in {{ selectedYears.join(', ') }}.
        </span>
      </div>
    </div>

    <div
      v-if="hasAnyData && stats"
      class="flex flex-column gap-3 mt-4"
    >
      <h5>Totals and averages</h5>

      <div
        class="flex flex-row w-full justify-content-between p-3"
        style="border: 1px solid var(--border-color); border-radius: 16px;"
      >
        <div class="flex flex-column">
          <div class="mb-1">
            Total over time
          </div>
          <div class="font-bold text-xl">
            {{ vueHelper.displayAsCurrency(stats.all_time_total) }}
          </div>
        </div>
        <div class="flex flex-column text-right">
          <div>Average</div>
          <div class="font-bold text-xl">
            {{ vueHelper.displayAsCurrency(stats.all_time_avg) }}
          </div>
          <div
            class="text-xs"
            style="color: var(--text-secondary)"
          >
            ({{ stats.all_time_months }} months)
          </div>
        </div>
      </div>

      <div
        class="flex flex-row flex-wrap w-full gap-3 p-3"
        style="border: 1px solid var(--border-color); border-radius: 16px;"
      >
        <div
          v-for="year in selectedYears"
          :key="year"
          class="flex-1 flex flex-column align-items-center text-center p-3 year-box"
        >
          <div class="mb-2 font-bold text-xl">
            {{ year }}
          </div>
          <div class="mb-1">
            {{ (selectedClassification === "expense" ? "Spent: " : "Earned: ") + vueHelper.displayAsCurrency(stats.year_stats[year]?.total ?? 0) }}
          </div>
          <div
            class="text-sm"
            style="color: var(--text-secondary)"
          >
            Avg: {{ vueHelper.displayAsCurrency(stats.year_stats[year]?.monthly_avg ?? 0) }}/mo
          </div>
          <div
            class="text-xs"
            style="color: var(--text-secondary)"
          >
            ({{ stats.year_stats[year]?.months_with_data ?? 0 }} months)
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">

.select-width {
  width: 200px;
}

@media (max-width: 768px) {

    .select-container {
        width: 100% !important;
    }

    .select-width {
        width: 100% !important;
    }

    .year-box {
        min-width: calc(50% - 0.75rem);
        border-right: none !important;
        border-bottom: 1px solid var(--border-color);
    }

    .year-box:last-child {
    border-bottom: none !important;
    }

    .year-box:nth-last-child(2):nth-child(odd) {
    border-bottom: none !important;
    }
}

@media (min-width: 769px) {
    .year-box:not(:last-child) {
        border-right: 1px solid var(--border-color);
    }
}
</style>