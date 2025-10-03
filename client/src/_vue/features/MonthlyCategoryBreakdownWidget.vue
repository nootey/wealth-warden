<script setup lang="ts">
import {computed, onMounted, ref, watch} from "vue";
import MultiSelect from "primevue/multiselect";
import CategoryBreakdownChart from "../components/charts/CategoryBreakdownChart.vue";

import { useToastStore } from "../../services/stores/toast_store.ts";
import { useChartStore } from "../../services/stores/chart_store.ts";
import type {Category} from "../../models/transaction_models.ts";
import {useTransactionStore} from "../../services/stores/transaction_store.ts";

const chartStore = useChartStore();
const toastStore = useToastStore();
const transactionStore = useTransactionStore();

const allYears = ref<number[]>([]);
const selectedYears = ref<number[]>([]);
const maxYears = 3;

const series = ref<{ name: string; data: number[] }[]>([]);

const yearOptions = computed(() =>
    allYears.value.map(y => ({ label: String(y), value: y }))
);

const ALL_CATEGORY = { id: undefined, name: 'All', display_name: 'All', parent_id: null } as unknown as Category;
const availableCategories = computed<Category[]>(() => [
    ALL_CATEGORY,
    ...transactionStore.categories.filter(c => !!c.parent_id)
]);
const selectedCategory = ref<Category | null>(ALL_CATEGORY);
const filteredCategories = ref<Category[]>([]);

const hasAnyData = computed(() =>
    series.value.some(s => s.data?.some(v => Number(v) > 0))
);

const searchCategory = (event: { query: string }) => {
    setTimeout(() => {
        const q = event.query.trim().toLowerCase();
        if (!q.length) {
            filteredCategories.value = [...availableCategories.value];
        } else {
            filteredCategories.value = availableCategories.value.filter((record) => {
                const name = (record.name ?? '').toLowerCase();
                const disp = (record.display_name ?? '').toLowerCase();
                return name.startsWith(q) || disp.startsWith(q);
            });
        }
    }, 250);
};

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
            class: "expense",
            percent: false,
            category: selectedCategory.value?.id ?? null,
        });

        const ys: number[] = res?.years ?? [];
        const by = res?.by_year ?? {};

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

        if (!selectedCategory.value) selectedCategory.value = ALL_CATEGORY;

    } catch (e) {
        toastStore.errorResponseToast(e);
    }
};

onMounted(async () => {
    await loadYears();
    await fetchData();
});

watch(selectedYears, async (arr) => {
    if (arr.length > maxYears) selectedYears.value = arr.slice(0, maxYears);
    await fetchData();
});

watch(selectedCategory, async () => {
    await fetchData();
});
</script>

<template>
    <div class="flex flex-column w-full p-3">
        <div id="mobile-row" class="flex flex-row gap-2 w-full justify-content-between align-items-center">

            <div id="filter-row" class="flex flex-row w-full align-items-center gap-2 justify-content-between">

                <div class="flex flex-column gap-1">
                <span class="text-sm" style="color: var(--text-secondary)">
                  View and compare category totals by month.
                </span>
                </div>

                <div class="flex flex-column">
                    <AutoComplete
                            v-model="selectedCategory"
                            size="small"
                            :suggestions="filteredCategories"
                            @complete="searchCategory"
                            optionLabel="display_name"
                            placeholder="Select category"
                            forceSelection
                            dropdown
                    />
                </div>

                <div class="flex flex-column">
                    <MultiSelect
                            v-model="selectedYears"
                            :options="yearOptions"
                            :maxSelectedLabels="3"
                            :selectionLimit="3"
                            display="chip"
                            placeholder="Years"
                            optionLabel="label"
                            optionValue="value"
                            style="width: 310px"
                    />
                </div>


            </div>
        </div>

        <div id="mobile-row" class="flex flex-row w-full justify-content-center align-items-center">
            <CategoryBreakdownChart v-if="hasAnyData" :series="series" />
            <div v-else class="flex flex-column align-items-center justify-content-center mt-3"
                 style="height: 400px">
                <span class="text-sm p-3" style="color: var(--text-secondary); border: 1px dashed var(--border-color); border-radius: 16px;">
                    Not enough data to display for:
                    {{ selectedCategory?.display_name ?? 'any category' }}
                    in {{ selectedYears.join(', ') }}.
                </span>
            </div>
        </div>

    </div>
</template>

<style scoped>
@media (max-width: 768px) {

    #filter-row {
        flex-direction: column !important;
        align-items: center !important;
        min-width: 0 !important;
    }
}
</style>