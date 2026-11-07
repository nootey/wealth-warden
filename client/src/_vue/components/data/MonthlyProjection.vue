<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useTransactionStore } from "../../../services/stores/transaction_store.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import ShowLoading from "../base/ShowLoading.vue";
import type { TemplateSummary } from "../../../models/transaction_models.ts";

const props = defineProps<{
  period: { year: number; month: number } | null;
}>();

const transactionStore = useTransactionStore();
const toastStore = useToastStore();

const loading = ref(true);
const summary = ref<TemplateSummary | null>(null);

const periodLabel = computed(() =>
  props.period
    ? new Date(props.period.year, props.period.month - 1).toLocaleString(
        "default",
        { month: "long", year: "numeric" },
      )
    : "",
);

watch(
  () => props.period,
  async (period) => {
    if (period) await loadSummary(period.year, period.month);
  },
  { immediate: true },
);

async function loadSummary(year: number, month: number) {
  try {
    loading.value = true;
    const response = await transactionStore.getTemplateSummary(year, month);
    summary.value = response.data;
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    loading.value = false;
  }
}

const tiles = computed<
  {
    label: string;
    value: number;
    avg?: number;
    showAvg?: boolean;
    color: string;
  }[]
>(() => {
  const amount = (key: keyof TemplateSummary) =>
    Number(summary.value?.[key] ?? 0);
  const projected = (
    flow: "income" | "expense" | "transfer",
    label: string,
    color = "text-ink",
  ) => ({
    label,
    value: amount(`monthly_${flow}`) + amount(`this_month_${flow}`),
    avg: amount(`monthly_${flow}`),
    showAvg: amount(`this_month_${flow}`) > 0,
    color,
  });
  return [
    projected("income", "Income", "text-gain"),
    projected("expense", "Expenses", "text-loss"),
    projected("transfer", "Transfers"),
    {
      label: "Current income",
      value: amount("current_income"),
      color: "text-gain",
    },
    {
      label: "Current expenses",
      value: amount("current_expense"),
      color: "text-loss",
    },
    {
      label: "Net",
      value: amount("projected_net"),
      color: "text-ink",
    },
  ];
});
</script>

<template>
  <section
    class="flex flex-col gap-3 rounded-2xl border border-line bg-card p-5 shadow-[var(--shadow-card)]"
  >
    <div class="flex items-baseline justify-between gap-2">
      <span class="label">Projection</span>
      <span class="text-xs text-faint">{{ periodLabel }}</span>
    </div>

    <ShowLoading v-if="loading" :num-fields="2" />

    <div v-else-if="summary" class="grid grid-cols-3 gap-2">
      <div
        v-for="tile in tiles"
        :key="tile.label"
        class="flex flex-col justify-between gap-1 rounded-xl bg-sunken px-3 py-2"
      >
        <span class="text-xs leading-tight text-muted">{{ tile.label }}</span>
        <div class="flex flex-col">
          <span
            class="text-xs sm:text-sm font-medium tracking-tight"
            :class="tile.color"
          >
            {{ vueHelper.displayAsCurrency(tile.value) }}
          </span>
          <span v-if="tile.showAvg" class="text-xs text-faint">
            {{ vueHelper.displayAsCurrency(tile.avg ?? null) }} avg.
          </span>
        </div>
      </div>
    </div>

    <span v-else class="text-sm text-muted">
      Currently, no projection can be shown.
    </span>
  </section>
</template>
