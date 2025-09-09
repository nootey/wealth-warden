<script setup lang="ts">
import {useAuthStore} from "../../services/stores/auth_store.ts";
import {useAccountStore} from "../../services/stores/account_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {computed, onMounted, ref} from "vue";
import NetworthChart from "../components/charts/NetworthChart.vue";
import {useChartStore} from "../../services/stores/chart_store.ts";

const authStore = useAuthStore();
const accountStore = useAccountStore();
const toastStore = useToastStore();
const chatStore = useChartStore();

type ChartPoint = { date: string; value: number | string }
type NetworthResponse = {
    currency: string
    points: ChartPoint[]
    current: ChartPoint // 👈 new
}

const payload = ref<NetworthResponse | null>(null)

const currencyFmt = computed(() =>
    new Intl.NumberFormat('de-DE', {
        style: 'currency',
        currency: payload.value?.currency || 'EUR'
    })
)

const orderedPoints = computed<ChartPoint[]>(() => {
    const arr = payload.value?.points ?? []
    return [...arr].sort(
        (a, b) => new Date(a.date).getTime() - new Date(b.date).getTime()
    )
})

onMounted(async () => {
    const raw = await chatStore.getNetWorth()
    const res: any = raw?.data?.points ?? raw?.points

    console.log(res)

    res.points = res.points.map(p => ({ ...p, value: Number(p.value) }))
    res.current.value = Number(res.current.value)
    payload.value = res
    console.log(payload.value)
})

async function backfillBalances(){
    try {
        const response = await accountStore.backfillBalances();
        toastStore.successResponseToast(response.data);
    } catch (err) {
        toastStore.errorResponseToast(err)
    }
}

</script>

<template>
  <main>
    <div class="flex flex-column w-100 gap-3 justify-content-center align-items-center">

      <div class="main-item flex flex-column justify-content-center gap-1">
        <div style="font-weight: bold;">WealthWarden </div>
        <br>
        <div> Welcome back {{ authStore?.user?.display_name }} </div>
        <div>{{ "Here's what's happening with your finances." }} </div>
      </div>

        <Button label="magic" @click="backfillBalances"></Button>

        <div v-if="payload" class="flex align-items-center gap-2" style="margin-top:.5rem">
            <span style="opacity:.7">As of {{ new Date(payload.current.date).toLocaleDateString() }}:</span>
            <strong>{{ currencyFmt.format(Number(payload.current.value)) }}</strong>
        </div>

        <div v-if="payload">
            <NetworthChart
                    :data-points="orderedPoints"
                    :currency="payload.currency"
                    @point-select="p => console.log('selected', p)"
            />
        </div>
        <div v-else>Loading net worth …</div>

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