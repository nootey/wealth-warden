<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useAnalyticsStore } from "../../../services/stores/analytics_store.ts";
import NetworthChart from "../../components/charts/NetworthChart.vue";
import ShowLoading from "../../components/base/ShowLoading.vue";
import vueHelper from "../../../utils/vue_helper.ts";
import type {
  AssetChartResponse,
  ChartPoint,
} from "../../../models/analytics_models.ts";

const props = defineProps<{
  assetId: number;
  chartHeight?: number;
}>();

const toastStore = useToastStore();
const analyticsStore = useAnalyticsStore();

type RangeKey = "1w" | "1m" | "3m" | "6m" | "ytd" | "1y" | "5y";

const dateRanges: { name: string; key: RangeKey }[] = [
  { name: "1W", key: "1w" },
  { name: "1M", key: "1m" },
  { name: "3M", key: "3m" },
  { name: "6M", key: "6m" },
  { name: "YTD", key: "ytd" },
  { name: "1Y", key: "1y" },
  { name: "5Y", key: "5y" },
];

const hydrating = ref(true);
const payload = ref<AssetChartResponse | null>(null);
const selectedDTO = ref<(typeof dateRanges)[number] | null>(
  dateRanges[4] ?? null,
);
const selectedKey = computed<RangeKey>(
  () => (selectedDTO.value?.key ?? "ytd") as RangeKey,
);

const storageKey = computed(() => `asset_chart_range_${props.assetId}`);

const marketValuePoints = computed<ChartPoint[]>(() => {
  const arr = payload.value?.market_value_points ?? [];
  return [...arr].sort(
    (a, b) => new Date(a.date).getTime() - new Date(b.date).getTime(),
  );
});

const costBasisPoints = computed<ChartPoint[]>(() => {
  const arr = payload.value?.cost_basis_points ?? [];
  return [...arr].sort(
    (a, b) => new Date(a.date).getTime() - new Date(b.date).getTime(),
  );
});

const hasSeries = computed(() => marketValuePoints.value.length > 0);

const currentValue = computed(() => {
  const pts = marketValuePoints.value;
  return pts.length > 0 ? Number(pts[pts.length - 1]!.value) : 0;
});

const periodChange = computed(() => {
  const pts = marketValuePoints.value;
  if (pts.length < 2) return null;
  const first = Number(pts[0]!.value);
  const last = Number(pts[pts.length - 1]!.value);
  const abs = last - first;
  const pct = first !== 0 ? (abs / Math.abs(first)) * 100 : null;
  return { abs, pct };
});

const periodLabels: Record<RangeKey, string> = {
  "1w": "week",
  "1m": "month",
  "3m": "3 months",
  "6m": "6 months",
  ytd: "year to date",
  "1y": "year",
  "5y": "5 years",
};

const activeColor = computed(() => {
  const cb = costBasisPoints.value;
  const mv = marketValuePoints.value;
  if (cb.length === 0 || mv.length === 0) return "#22c55e";
  const latestCb = Number(cb[cb.length - 1]!.value);
  const latestMv = Number(mv[mv.length - 1]!.value);
  return latestMv >= latestCb ? "#22c55e" : "#ef4444";
});

async function getData() {
  const lastKey = localStorage.getItem(storageKey.value);
  if (lastKey) {
    const found = dateRanges.find((r) => r.key === lastKey);
    if (found) selectedDTO.value = found;
  }
  await fetchChart();
  hydrating.value = false;
}

async function fetchChart() {
  try {
    payload.value = await analyticsStore.getAssetChart(
      props.assetId,
      selectedKey.value,
    );
  } catch (err) {
    toastStore.errorResponseToast(err);
  }
}

watch(selectedDTO, (val) => {
  if (!val || hydrating.value) return;
  localStorage.setItem(storageKey.value, val.key);
  fetchChart();
});

defineExpose({ refresh: getData });

onMounted(getData);
</script>

<template>
  <div
    v-if="payload && !hydrating"
    class="w-full flex flex-column justify-content-center p-2 gap-1"
  >
    <div class="flex flex-row gap-2 w-full justify-content-between">
      <div class="flex flex-column gap-1">
        <strong>{{ vueHelper.displayAsCurrency(currentValue) }}</strong>
        <div
          v-if="periodChange && hasSeries"
          class="flex flex-row gap-2 align-items-center"
          :style="{ color: activeColor }"
        >
          <span>{{
            vueHelper.displayAsCurrency(Math.abs(periodChange.abs))
          }}</span>
          <div class="flex flex-row gap-1 align-items-center">
            <i
              class="text-sm"
              :class="
                periodChange.abs >= 0
                  ? 'pi pi-angle-double-up'
                  : 'pi pi-angle-double-down'
              "
            />
            <span v-if="periodChange.pct !== null"
              >({{ Math.abs(periodChange.pct).toFixed(1) }}%)</span
            >
          </div>
          <span class="text-sm" style="color: var(--text-secondary)">
            {{
              selectedKey === "ytd"
                ? "year to date"
                : `vs. last ${periodLabels[selectedKey]}`
            }}
          </span>
        </div>
      </div>
      <Select
        v-model="selectedDTO"
        size="small"
        style="width: 90px"
        :options="dateRanges"
        option-label="name"
      />
    </div>

    <NetworthChart
      v-if="hasSeries"
      :height="chartHeight ?? 200"
      :data-points="marketValuePoints"
      :secondary-points="costBasisPoints"
      secondary-label="Cost basis"
      :currency="payload.currency"
      :active-color="activeColor"
    />

    <div
      v-else
      class="flex flex-column align-items-center justify-content-center border-1 border-dashed border-round-md surface-border"
      :style="{ height: (chartHeight ?? 200) / 2 + 'px' }"
    >
      <i
        class="pi pi-inbox text-2xl mb-2"
        style="color: var(--text-secondary)"
      />
      <span class="text-sm" style="color: var(--text-secondary)">
        No price history available yet
      </span>
    </div>
  </div>
  <ShowLoading v-else :num-fields="4" class="mb-4" />
</template>
