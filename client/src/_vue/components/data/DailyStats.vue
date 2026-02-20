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
  <div v-if="!loading" class="flex flex-column p-2 gap-2">
    <span style="color: var(--text-secondary)"
      >View your daily spending activity.</span
    >
    <div v-if="dailyStats" class="flex flex-column mt-2">
      <div class="flex flex-column w-full gap-2">
        <div class="flex flex-row gap-2 align-items-center">
          <span>Inflows:</span>
          <span
            ><b>{{ vueHelper.displayAsCurrency(dailyStats?.inflow!) }}</b></span
          >
        </div>
        <div class="flex flex-row gap-2 align-items-center">
          <span>Outflows:</span>
          <span
            ><b>{{
              vueHelper.displayAsCurrency(dailyStats?.outflow!)
            }}</b></span
          >
        </div>
      </div>
    </div>
    <div v-else>
      <span>Currently, no stats can be shown.</span>
    </div>
  </div>
  <ShowLoading v-else :num-fields="7" />
</template>

<style scoped></style>
