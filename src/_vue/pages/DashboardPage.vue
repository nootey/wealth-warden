<script setup lang="ts">
import {useAuthStore} from "../../services/stores/auth_store.ts";
import {useAccountStore} from "../../services/stores/account_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {computed, onMounted, ref, watch} from "vue";
import NetworthChart from "../components/charts/NetworthChart.vue";
import {useChartStore} from "../../services/stores/chart_store.ts";
import SlotSkeleton from "../components/layout/SlotSkeleton.vue";
import ShowLoading from "../components/base/ShowLoading.vue";
import vueHelper from "../../utils/vue_helper.ts";
import type {NetworthResponse, ChartPoint} from "../../models/chart_models.ts";

const authStore = useAuthStore();
const accountStore = useAccountStore();
const toastStore = useToastStore();
const chatStore = useChartStore();

const STORAGE_KEY = 'networth_range_key'
const hydrating = ref(true);

onMounted(async () => {
    const lastKey = localStorage.getItem(STORAGE_KEY)
    if (lastKey) {
        const found = dateRanges.find(r => r.key === lastKey)
        if (found) selectedDTO.value = found
    }

    const initialKey = (selectedDTO.value as any)?.key || '1m'
    await getNetworthData({ rangeKey: initialKey })
    hydrating.value = false
})

const payload = ref<NetworthResponse | null>(null);

const dateRanges = [
    { name: '1W',  key: '1w'  },
    { name: '1M',  key: '1m'  },
    { name: '3M',  key: '3m'  },
    { name: '6M',  key: '6m'  },
    { name: 'YTD', key: 'ytd' },
    { name: '1Y',  key: '1y'  },
    { name: '5Y',  key: '5y'  },
] as const;

type RangeOption = typeof dateRanges[number];

const filteredDateRanges = ref<RangeOption[]>([...dateRanges]);
const selectedDTO = ref<RangeOption | null>(dateRanges[1]); // '1M'
const selectedKey = computed(() => selectedDTO.value?.key ?? '1m');

const orderedPoints = computed<ChartPoint[]>(() => {
    const arr = payload.value?.points ?? []
    return [...arr].sort(
        (a, b) => new Date(a.date).getTime() - new Date(b.date).getTime()
    )
});

const periodLabels: Record<string,string> = {
    "1w": "week",
    "1m": "month",
    "3m": "3 months",
    "6m": "6 months",
    "ytd": "year to date",
    "1y": "year",
    "5y": "5 years"
}

async function getNetworthData(opts?: { rangeKey?: string; from?: string; to?: string }) {
    try {
        const params: any = {};
        if (opts?.from || opts?.to) {
            if (opts.from) params.from = opts.from;
            if (opts.to) params.to = opts.to;
        } else if (opts?.rangeKey) {
            params.range = opts.rangeKey;
        }

        const res = await chatStore.getNetWorth(params);


        res.points = res.points.map((p: any) => ({ ...p, value: Number(p.value) }));
        res.current.value = Number(res.current.value);

        if (res.change) {
            res.change.prev_period_end_value = Number(res.change.prev_period_end_value);
            res.change.current_end_value    = Number(res.change.current_end_value);
            res.change.abs                  = Number(res.change.abs);
            res.change.pct                  = Number(res.change.pct);
        }

        payload.value = res;

    } catch (err) {
        toastStore.errorResponseToast(err);
    }
}

async function backfillBalances(){
    try {
        const response = await accountStore.backfillBalances();
        toastStore.successResponseToast(response.data);
    } catch (err) {
        toastStore.errorResponseToast(err)
    }
}

const searchDaterange = (event: any) => {
    const q = (event.query ?? '').trim().toLowerCase();
    filteredDateRanges.value = q
        ? dateRanges.filter(o => o.name.toLowerCase().startsWith(q))
        : [...dateRanges];
};

watch(selectedDTO, (val: any) => {
    if (!val) return
    if (hydrating.value) return
    if (val.key) localStorage.setItem(STORAGE_KEY, val.key)
    getNetworthData({ rangeKey: val.key })
});

</script>

<template>

    <main class="flex flex-column w-full p-2 align-items-center" style="height: 100vh;">

        <div class="flex flex-column justify-content-center p-2 w-full gap-3 border-round-md"
             style="max-width: 1200px;">

            <SlotSkeleton bg="transparent">
                <div class="w-full flex flex-row justify-content-between p-2 gap-2">
                    <div class="w-full flex flex-column gap-2">
                        <div style="font-weight: bold;"> Welcome back {{ authStore?.user?.display_name }} </div>
                        <div>{{ "Here's what's happening with your finances." }} </div>
                    </div>
                    <Button label="Refresh" icon="pi pi-refresh" class="main-button" @click="backfillBalances"></Button>
                </div>
            </SlotSkeleton>

            <SlotSkeleton bg="secondary">
                <div v-if="payload" class="w-full flex flex-column justify-content-center p-3 gap-3">

                    <div class="flex flex-row p-2 gap-2 w-full justify-content-between">
                        <div class="flex flex-column gap-2">
                            <div class="flex flex-row">
                                <span class="text-sm" style="color: var(--text-secondary)">Net worth</span>
                            </div>
                            <div class="flex flex-row">
                                <strong>{{ vueHelper.displayAsCurrency(payload.current.value) }}</strong>
                            </div>
                        </div>

                        <div class="flex flex-column gap-2">
                            <AutoComplete size="small" style="width: 90px;" v-model="selectedDTO"
                                          :suggestions="filteredDateRanges" dropdown
                                          @complete="searchDaterange" optionLabel="name" forceSelection />
                        </div>
                    </div>

                    <div v-if="payload?.change" class="flex flex-row gap-2 align-items-center" :style="{ color: activeColor }">
                        <span>
                        {{ vueHelper.displayAsCurrency(payload.change.current_end_value) }}
                        </span>

                        <div class="flex flex-row gap-1 align-items-center">
                            <i :class="payload.change.abs >= 0 ? 'pi pi-angle-up' : 'pi pi-angle-down'"></i>
                            <span>
                            ({{ (payload.change.pct * 100).toFixed(1) }}%)
                          </span>
                        </div>

                        <span class="text-sm" style="color: var(--text-secondary)">
                          vs. last {{ periodLabels[selectedKey] ?? 'period' }}
                        </span>
                    </div>

                    <NetworthChart
                            :dataPoints="orderedPoints"
                            :currency="payload.currency"
                            @point-select="p => console.log('selected', p)"
                    />

                </div>
                <ShowLoading v-else :numFields="6" />
            </SlotSkeleton>

            <SlotSkeleton bg="secondary">
                <div class="w-full flex flex-row justify-content-between p-2 gap-2">
                    Assets - WIP
                </div>
            </SlotSkeleton>

            <SlotSkeleton bg="secondary">
                <div class="w-full flex flex-row justify-content-between p-2 gap-2">
                    Liabilities - WIP
                </div>
            </SlotSkeleton>

    </div>
    </main>

</template>

<style scoped>
.main-item {
  width: 100%;
  max-width: 1000px;
  align-items: center;
  padding: 1rem;
  border-radius: 8px; border: 1px solid var(--border-color); background-color: var(--background-secondary)
}
</style>