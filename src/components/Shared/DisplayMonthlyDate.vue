<script setup lang="ts">

import vueHelper from "../../utils/vueHelper.ts";
import dateHelper from "../../utils/dateHelper.ts";

const props = defineProps<{
  groupedValues: any[];
}>();


</script>

<template>
  <div class="flex flex-row w-full">
    <div class="flex flex-column w-full">
      <DataTable :value="vueHelper.pivotedRecords(groupedValues)" size="small" showGridlines>
        <Column field="category_name" header="Category" style="max-width: 2rem;"/>

        <Column
            v-for="month in dateHelper.monthColumns.value"
            :key="month"
            :field="month.toString()"
            :header="dateHelper.formatMonth(month)"
            :body="(data: any) => data[month] ? data[month] : 0"
            style="max-width: 1rem;">
          <template #body="slotProps">
            {{ vueHelper.displayAsCurrency(slotProps.data[month])}}
          </template>
        </Column>
      </DataTable>
    </div>
  </div>
</template>

<style scoped>

</style>