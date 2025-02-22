<script setup lang="ts">
import type {ReoccurringAction} from "../../models/actions.ts";
import vueHelper from "../../utils/vueHelper.ts";
import dateHelper from "../../utils/dateHelper.ts";

const props = defineProps<{
  categoryItems: ReoccurringAction[];
}>();

async function editAction(id: number) {
  console.log(id)
}

async function removeAction(id: number) {
  console.log(id)
}
</script>

<template>
  <div class="flex flex-row w-full">
    <div class="flex flex-column w-full">

      <DataTable :value="categoryItems" size="small">
        <Column header="Actions">
          <template #body="slotProps">
            <div class="flex flex-row align-items-center gap-2">
              <i class="pi pi-pencil hover_icon"
                 @click="editAction(slotProps.data?.id)"></i>
              <i class="pi pi-trash hover_icon" style="color: var(--accent-primary)"
                 @click="removeAction(slotProps.data?.id)"></i>
            </div>
          </template>
        </Column>
        <Column field="category_type" header="Category"></Column>
        <Column field="amount" header="Amount">
          <template #body="slotProps">
            {{ vueHelper.displayAsCurrency(slotProps.data.amount)}}
          </template>
        </Column>
        <Column field="interval_value" header="Value"></Column>
        <Column field="interval_unit" header="Unit"></Column>
        <Column field="start_date" header="Start date">
          <template #body="slotProps">
            {{ dateHelper.formatDate(slotProps.data?.start_date, false) }}
          </template>
        </Column>
        <Column field="end_date" header="End date">
          <template #body="slotProps">
            {{ slotProps.data?.end_date ? dateHelper.formatDate(slotProps.data?.end_date, false) : "∞"}}
          </template>
        </Column>
      </DataTable>
    </div>

  </div>
</template>

<style scoped>

</style>