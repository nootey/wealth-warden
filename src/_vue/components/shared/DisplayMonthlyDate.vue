<script setup lang="ts">

import vueHelper from "../../../utils/vueHelper.ts";
import dateHelper from "../../../utils/dateHelper.ts";
import {onMounted} from "vue";

const props = defineProps<{
  groupedValues: any[];
}>();

</script>

<template>
  <div class="flex flex-row w-full">
    <div class="flex flex-column w-full">

      <DataTable :value="vueHelper.pivotedRecords(groupedValues, (item) => item.category_type)" size="small"
                 showGridlines rowGroupMode="subheader" groupRowsBy="category_type" scrollable scrollHeight="550px">

        <Column field="category_name" header="Category" style="max-width: 2rem;"></Column>

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

        <template #groupheader="slotProps">
          <div class="flex items-center gap-2">
            <b v-if="slotProps.data.category_type && slotProps.data.category_type !== 'Unknown'">
              {{ slotProps.data.category_type.charAt(0).toUpperCase() + slotProps.data.category_type.slice(1) }}
            </b>
          </div>
        </template>

      </DataTable>
    </div>
  </div>
</template>

<style scoped>

</style>