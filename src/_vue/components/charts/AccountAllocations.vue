<script setup lang="ts">
import {computed, onMounted, ref, watch} from "vue";
import type {Account} from "../../../models/account_models.ts";
import {useAccountStore} from "../../../services/stores/account_store.ts";
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import vueHelper from "../../../utils/vue_helper.ts";

const props = defineProps<{
    title: string;
    classification: string;
}>();

const accountStore = useAccountStore();
const sharedStore  = useSharedStore();
const toastStore   = useToastStore();

const loading  = ref(true);
const accounts = ref<Account[]>([]);

// nice, distinct colors (extend if needed)
const palette = ["#8B5CF6","#3B82F6","#A3A3A3","#22C55E","#F59E0B","#EF4444","#06B6D4","#E879F9"];

type Bucket = { key: string; amount: number; color: string; percent: number };

// robust string→number for Decimal strings
function toNumber(v: unknown): number {
    if (v == null) return 0;
    if (typeof v === "number") return v;
    if (typeof v === "string") {
        // strip spaces/commas; keep minus and dot
        const cleaned = v.replace(/[^0-9.\-]/g, "");
        const n = Number(cleaned);
        return Number.isFinite(n) ? n : 0;
    }
    return 0;
}

const buckets = computed<Bucket[]>(() => {
    const map = new Map<string, number>();
    for (const a of accounts.value) {
        const key = a?.account_type?.type || "Other";
        const amt = toNumber(a?.balance?.end_balance);
        map.set(key, (map.get(key) ?? 0) + amt);
    }

    const absValues = Array.from(map.values()).map(v => Math.abs(v));
    const totalAbs = absValues.reduce((s, n) => s + n, 0);

    const keys = Array.from(map.keys());
    return keys.map((k, i) => {
        const amount = map.get(k) ?? 0;
        const percent = totalAbs > 0 ? (Math.abs(amount) / totalAbs) * 100 : 0;
        return { key: k, amount, color: palette[i % palette.length], percent };
    }).sort((a, b) => Math.abs(b.amount) - Math.abs(a.amount));
});

const totalAmount = computed(() =>
    buckets.value.reduce((s, b) => s + b.amount, 0)
);

onMounted(async () => {
    await accountStore.getAccountTypes();
    await getData();
});
watch(() => props.classification, () => { getData(); });

async function getData(page = 1) {
    loading.value = true;
    try {
        const { data } = await sharedStore.getRecordsPaginated(
            "accounts",
            { rowsPerPage: 25, sort: { field: "id", order: "asc" }, filters: [], classification: props.classification },
            page
        );
        accounts.value = data ?? [];
    } catch (err) {
        toastStore.errorResponseToast(err);
    } finally {
        loading.value = false;
    }
}
</script>

<template>
    <div class="flex flex-column gap-2 w-full p-3">

        <div class="flex align-items-center gap-2">
            <span class="font-semibold">{{ title }}</span>
            <span class="opacity-60">·</span>
            <span class="opacity-90">{{ vueHelper.displayAsCurrency(totalAmount) }}</span>
        </div>

        <div class="px-2 pt-1 w-full" :class="{ 'opacity-60': loading }">
            <div v-if="!loading && totalAmount !== 0"
                    class="flex w-full"
                    :style="{ height: '8px', borderRadius: '9999px', overflow: 'hidden', gap: '2px' }"
                    role="progressbar">

                <div v-for="b in buckets"
                        :key="b.key"
                        class="flex"
                        :style="{
                                  width: Math.max(b.percent, 0.5) + '%',
                                  backgroundColor: b.color,
                                  borderRadius: '9999px'
                                }"
                        :title="`${b.key}: ${vueHelper.displayAsCurrency(b.amount)} (${b.percent.toFixed(0)}%)`"
                />

            </div>

            <div v-else class="w-full"
                 :style="{ height: '8px', borderRadius: '9999px', background: 'rgba(128,128,128,.25)' }"
            />

        </div>

        <div class="flex align-items-center flex-wrap gap-3 px-0 pt-1">
            <span v-if="loading" class="opacity-70 text-sm">Loading…</span>

            <template v-else-if="buckets.length">
                <div v-for="b in buckets" :key="b.key" class="flex align-items-center gap-2 text-sm">
                    <span class="inline-block" :style="{ width: '8px', height: '8px', borderRadius: '9999px', backgroundColor: b.color }"></span>
                    <span class="opacity-90">{{ vueHelper.capitalize(vueHelper.denormalize(b.key)) }}</span>
                    <span class="font-semibold opacity-95">{{ b.percent.toFixed(0) }}%</span>
                </div>
            </template>

            <span v-else class="opacity-70 text-sm">No accounts found.</span>
        </div>
    </div>
</template>