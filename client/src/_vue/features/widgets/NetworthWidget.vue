<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import NetworthChart from "../../components/charts/NetworthChart.vue";
import ShowLoading from "../../components/base/ShowLoading.vue";
import { useRouter } from "vue-router";
import { useAnalyticsStore } from "../../../services/stores/analytics_store.ts";
import { useAccountStore } from "../../../services/stores/account_store.ts";
import { useChartColors } from "../../../style/theme/chartColors.ts";
import type {
  ChartPoint,
  NetworthResponse,
} from "../../../models/analytics_models.ts";

const props = withDefaults(
  defineProps<{
    accountId?: number | null;
    title?: string;
    storageKeyPrefix?: string;
    chartHeight?: number;
    isRefreshing?: boolean;
  }>(),
  {
    accountId: null,
    title: "Net worth",
    storageKeyPrefix: "networth_range_key",
    chartHeight: 300,
    isRefreshing: false,
  },
);

const router = useRouter();

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

const periodLabels: Record<RangeKey, string> = {
  "1w": "week",
  "1m": "month",
  "3m": "3 months",
  "6m": "6 months",
  ytd: "year to date",
  "1y": "year",
  "5y": "5 years",
};

const toastStore = useToastStore();
const analyticsStore = useAnalyticsStore();
const accountStore = useAccountStore();

const hydrating = ref(true);
const payload = ref<NetworthResponse | null>(null);
const selectedDTO = ref<(typeof dateRanges)[number] | null>(
  dateRanges[4] ?? null,
);
const selectedKey = computed<RangeKey>(
  () => (selectedDTO.value?.key ?? "ytd") as RangeKey,
);

const orderedPoints = computed<ChartPoint[]>(() => {
  const arr = payload.value?.points ?? [];
  return [...arr].sort(
    (a, b) => new Date(a.date).getTime() - new Date(b.date).getTime(),
  );
});
const hasSeries = computed(() => (payload.value?.points?.length ?? 0) > 0);

const { colors } = useChartColors();
const activeColor = computed(() =>
  effectiveAbs.value >= 0 ? colors.value.pos : colors.value.neg,
);

const storageSuffix = computed(() =>
  props.accountId ? `acct_${props.accountId}` : "ALL",
);
const storageKey = computed(
  () => `${props.storageKeyPrefix}_${storageSuffix.value}`,
);

const startOfRange = computed(() => {
  const today = new Date();
  // normalize to UTC midnight like backend
  const dto = new Date(
    Date.UTC(today.getUTCFullYear(), today.getUTCMonth(), today.getUTCDate()),
  );

  switch (selectedKey.value) {
    case "1w":
      return new Date(dto.getTime() - 7 * 24 * 60 * 60 * 1000);
    case "1m":
      return new Date(
        Date.UTC(dto.getUTCFullYear(), dto.getUTCMonth() - 1, dto.getUTCDate()),
      );
    case "3m":
      return new Date(
        Date.UTC(dto.getUTCFullYear(), dto.getUTCMonth() - 3, dto.getUTCDate()),
      );
    case "6m":
      return new Date(
        Date.UTC(dto.getUTCFullYear(), dto.getUTCMonth() - 6, dto.getUTCDate()),
      );
    case "ytd":
      return new Date(Date.UTC(dto.getUTCFullYear(), 0, 1));
    case "1y":
      return new Date(
        Date.UTC(dto.getUTCFullYear() - 1, dto.getUTCMonth(), dto.getUTCDate()),
      );
    case "5y":
      return new Date(
        Date.UTC(dto.getUTCFullYear() - 5, dto.getUTCMonth(), dto.getUTCDate()),
      );
    default:
      return new Date(
        Date.UTC(dto.getUTCFullYear(), dto.getUTCMonth() - 1, dto.getUTCDate()),
      );
  }
});

const displayPoints = computed<ChartPoint[]>(() => {
  const pts = orderedPoints.value;
  if (pts.length === 1) {
    const first = pts[0];
    if (!first) return pts;

    const start = startOfRange.value;
    const firstDay = new Date(first.date);
    // if first point isn't already at start, prepend a phantom point at start with same value
    if (
      firstDay.getUTCFullYear() !== start.getUTCFullYear() ||
      firstDay.getUTCMonth() !== start.getUTCMonth() ||
      firstDay.getUTCDate() !== start.getUTCDate()
    ) {
      return [{ date: start.toISOString(), value: first.value }, ...pts];
    }
  }

  const isLiability = payload.value?.asset_type === "liability";
  return isLiability
    ? pts.map((p) => ({ ...p, value: Math.abs(Number(p.value)) }))
    : pts;
});

function displayNetworthChange(change: string) {
  if (change === "year to date") return change;
  return `vs. last ${change ?? "period"}`;
}

const pctStr = computed(() => {
  const c = payload.value?.change;
  if (!c) return "0.0%";
  const prev = Number(c.prev_period_end_value || 0);
  const isLiability = payload.value?.asset_type === "liability";
  const denom = isLiability ? Math.abs(prev) : prev || 0;
  const pct = denom !== 0 ? (effectiveAbs.value / Math.abs(denom)) * 100 : 0;
  return pct.toFixed(1) + "%";
});

const effectiveAbs = computed(() => {
  const c = payload.value?.change;
  if (!c) return 0;
  const prev = Number(c.prev_period_end_value || 0);
  const curr = Number(c.current_end_value || 0);
  const isLiability = payload.value?.asset_type === "liability";

  return isLiability ? Math.abs(prev) - Math.abs(curr) : curr - prev;
});

async function getData() {
  const lastKey = localStorage.getItem(storageKey.value);
  if (lastKey) {
    const found = dateRanges.find((r) => r.key === lastKey);
    if (found) selectedDTO.value = found;
  }
  await accountStore.syncBalances().catch(() => {});
  await getNetworthData({ rangeKey: selectedKey.value });
  hydrating.value = false;
}

async function getNetworthData(opts?: {
  rangeKey?: RangeKey;
  from?: string;
  to?: string;
}) {
  try {
    const params: any = {};
    if (opts?.from || opts?.to) {
      if (opts.from) params.from = opts.from;
      if (opts.to) params.to = opts.to;
    } else if (opts?.rangeKey) {
      params.range = opts.rangeKey;
    }
    if (props.accountId) params.account = props.accountId;

    const res = await analyticsStore.getNetWorth(params);

    res.points = res.points.map((p: any) => ({ ...p, value: Number(p.value) }));
    res.current.value = Number(res.current.value);
    if (res.change) {
      res.change.prev_period_end_value = Number(
        res.change.prev_period_end_value,
      );
      res.change.current_end_value = Number(res.change.current_end_value);
      res.change.abs = Number(res.change.abs);
      res.change.pct = Number(res.change.pct);
    }

    payload.value = res;
  } catch (err) {
    toastStore.errorResponseToast(err);
  }
}

watch(selectedDTO, (val) => {
  if (!val || hydrating.value) return;
  localStorage.setItem(storageKey.value, val.key);
  getNetworthData({ rangeKey: val.key as RangeKey });
});

defineExpose({ refresh: getData });

onMounted(getData);
</script>

<template>
  <div
    v-if="payload && !isRefreshing"
    class="w-full flex flex-col justify-center gap-4"
  >
    <div
      class="flex flex-row flex-wrap gap-4 w-full justify-between items-start"
    >
      <div class="flex flex-col gap-2 min-w-0">
        <span class="label">{{ title }}</span>
        <span
          class="text-2xl font-semibold tracking-tight leading-none text-ink"
        >
          {{ vueHelper.displayAsCurrency(payload.current.value) }}
        </span>

        <div
          v-if="payload?.change && hasSeries"
          class="flex flex-row flex-wrap gap-2 items-center text-sm"
        >
          <span
            class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 font-medium"
            :style="{
              color: activeColor,
              backgroundColor: `color-mix(in srgb, ${activeColor}, transparent 88%)`,
            }"
          >
            <i
              class="text-xs"
              :class="
                effectiveAbs >= 0
                  ? 'pi pi-arrow-up-right'
                  : 'pi pi-arrow-down-right'
              "
            />
            {{ vueHelper.displayAsCurrency(Math.abs(effectiveAbs)) }}
            <span class="opacity-80">· {{ pctStr }}</span>
          </span>
          <span class="text-muted">
            {{ displayNetworthChange(periodLabels[selectedKey]) }}
          </span>
        </div>
      </div>

      <div class="max-w-full overflow-x-auto">
        <SelectButton
          v-model="selectedDTO"
          size="small"
          :options="dateRanges"
          option-label="name"
          :allow-empty="false"
        />
      </div>
    </div>

    <NetworthChart
      v-if="hasSeries"
      :height="chartHeight"
      :data-points="displayPoints"
      :currency="payload.currency"
      :active-color="activeColor"
      :is-liability="payload?.asset_type === 'liability'"
    />

    <div
      v-else
      class="flex flex-col items-center justify-center border border-dashed rounded-md border-surface"
      :style="{ height: chartHeight / 2 + 'px' }"
    >
      <i
        class="pi pi-inbox text-2xl mb-2"
        style="color: var(--text-secondary)"
      />
      <div class="text-sm" style="color: var(--text-secondary)">
        <span> No data yet - connect an</span>
        <span
          class="hover-icon font-bold text-base"
          @click="router.push({ name: 'settings.accounts' })"
        >
          account
        </span>
        <span> to see your net worth over time. </span>
      </div>
    </div>
  </div>
  <ShowLoading v-else :num-fields="6" class="mb-6" />
</template>
