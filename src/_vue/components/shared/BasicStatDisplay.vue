<script setup lang="ts">

import vueHelper from "../../../utils/vueHelper.ts";
import ComparativePieChart from "./charting/ComparativePieChart.vue";
import type {Statistics} from "../../../models/shared.ts";

const props = defineProps<{
  basicStats: Statistics[];
  limit: boolean;
  dataCount: number
}>();

</script>

<template>
  <div class="flex flex-row w-full">
    <div v-if="props.dataCount > 0" class="flex flex-column w-full">
      <DataTable  :value="basicStats" size="small" showGridlines groupRowsBy="category_type" scrollable scrollHeight="450px">
        <Column field="category" header="Category" style="max-width: 2rem;"/>
        <Column field="total" header="Total" style="max-width: 2rem;">
          <template #body="slotProps">
            {{ vueHelper.displayAsCurrency(slotProps.data.total) }}
          </template>
        </Column>
        <Column field="average" header="Average" style="max-width: 2rem;">
          <template #body="slotProps">
            {{ vueHelper.displayAsCurrency(slotProps.data.average) }}
          </template>
        </Column>
        <Column v-if="limit" field="spending_limit" header="Limit" style="max-width: 1.5rem;">
          <template #body="slotProps">
            {{ vueHelper.displayAsCurrency(slotProps.data.spending_limit) }}
          </template>
        </Column>
      </DataTable>
    </div>
    <div v-else class="flex flex-column w-full p-2 gap-2">
      {{ "No data to display yet"}}
    </div>
  </div>

  <div class="flex flex-row w-full">
    <div class="flex flex-column w-full">
      <ComparativePieChart
          :values="basicStats.filter(item => item.category !== 'Total').map(item => item.total)"
          :labels="basicStats.filter(item => item.category !== 'Total').map(item => item.category)"
      />
    </div>
  </div>
</template>

<style scoped>

</style>