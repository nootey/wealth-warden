<script setup lang="ts">
import { onMounted, ref } from "vue";
import type { DailyStats } from "../../../models/analytics_models.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import ShowLoading from "../base/ShowLoading.vue";
import { useAnalyticsStore } from "../../../services/stores/analytics_store.ts";

const analyticsStore = useAnalyticsStore();
const toastStore = useToastStore();

const loading = ref(false);

const dailyStats = ref<DailyStats | null>(null);

onMounted(async () => {
  await loadStats();
});

async function loadStats() {
  try {
    loading.value = true;
    const result = await analyticsStore.getTodayStats(null);

    if (!result) {
      dailyStats.value = null;
    } else {
      dailyStats.value = result;
    }
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <section
    class="flex flex-col gap-3 rounded-2xl border border-line bg-card p-5 shadow-[var(--shadow-card)]"
  >
    <div class="flex items-baseline justify-between gap-2">
      <span class="label">Today</span>
      <span class="text-xs text-faint">Checking accounts</span>
    </div>

    <ShowLoading v-if="loading" :num-fields="2" />

    <div v-else-if="dailyStats" class="grid grid-cols-2 gap-2">
      <div class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3">
        <span class="text-xs text-muted">Inflows</span>
        <span class="text-lg font-medium tracking-tight text-gain">
          {{
            vueHelper.displayAsCurrency(dailyStats.inflow, dailyStats.currency)
          }}
        </span>
      </div>
      <div class="flex flex-col gap-1 rounded-xl bg-sunken px-4 py-3">
        <span class="text-xs text-muted">Outflows</span>
        <span class="text-lg font-medium tracking-tight text-loss">
          {{
            vueHelper.displayAsCurrency(dailyStats.outflow, dailyStats.currency)
          }}
        </span>
      </div>
    </div>

    <span v-else class="text-sm text-muted">
      Currently, no stats can be shown.
    </span>
  </section>
</template>
