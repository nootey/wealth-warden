<script setup lang="ts">
import type {Statistics} from "../../../models/shared.ts";
import vueHelper from "../../../utils/vueHelper.ts";
import ComparativePieChart from "../../components/shared/charting/ComparativePieChart.vue";

const props = defineProps<{
  savingsStats: Statistics[];
  dataCount: number
}>();
</script>

<template>
  <div class="flex flex-row w-full">
    <div v-if="props.dataCount > 0" class="flex flex-column w-full">
      <DataTable  :value="savingsStats" size="small" showGridlines groupRowsBy="category_type" scrollable scrollHeight="450px">
        <Column field="category" header="Category" style="max-width: 2rem;"/>
        <Column field="goal_progress" header="Progress" style="max-width: 2rem;">
          <template #body="slotProps">
            {{ vueHelper.displayAsCurrency(slotProps.data.goal_progress) }}
          </template>
        </Column>
        <Column field="goal_target" header="Target" style="max-width: 2rem;">
          <template #body="slotProps">
            {{ vueHelper.displayAsCurrency(slotProps.data.goal_target) }}
          </template>
        </Column>
        <Column field="goal_spent" header="Spent" style="max-width: 2rem;">
          <template #body="slotProps">
            {{ vueHelper.displayAsCurrency(slotProps.data.goal_spent) }}
          </template>
        </Column>
      </DataTable>
    </div>
    <div v-else class="flex flex-column w-full p-2 gap-2">
      {{ "No data to display yet"}}
    </div>
  </div>

<!--  <div class="flex flex-row w-full">-->
<!--    <div class="flex flex-column w-full">-->
<!--      <ComparativePieChart-->
<!--          :values="basicStats.filter(item => item.category !== 'Total').map(item => item.total)"-->
<!--          :labels="basicStats.filter(item => item.category !== 'Total').map(item => item.category)"-->
<!--      />-->
<!--    </div>-->
<!--  </div>-->
</template>

<style scoped>

</style>