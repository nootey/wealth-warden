<script setup lang="ts">
import type { PaginatorState } from "../../../models/shared_models.ts";

defineProps<{
  paginator: PaginatorState;
  rows: number[];
}>();

const emit = defineEmits<{
  onPage: [value: any];
}>();
</script>

<template>
  <Paginator
    id="custom-paginator"
    class="small"
    :first="paginator.from"
    :rows="paginator.rowsPerPage"
    :rows-per-page-options="rows"
    :total-records="paginator.total"
    :template="{
      '640px': 'PrevPageLink CurrentPageReport NextPageLink',
      '960px':
        'FirstPageLink PrevPageLink CurrentPageReport NextPageLink LastPageLink',
      '1300px':
        'FirstPageLink PrevPageLink PageLinks NextPageLink LastPageLink RowsPerPageDropdown',
      default:
        'FirstPageLink PrevPageLink PageLinks NextPageLink LastPageLink RowsPerPageDropdown',
    }"
    @page="(e) => emit('onPage', e)"
  >
    <template #start>
      <span class="text-xs text-muted whitespace-nowrap">
        {{ paginator.from.toLocaleString() }}–{{
          paginator.to.toLocaleString()
        }}
        of {{ paginator.total.toLocaleString() }}
      </span>
    </template>
  </Paginator>
</template>

<style scoped>
@media (max-width: 960px) {
  #custom-paginator :deep(.p-paginator-content-start) {
    display: none;
  }
}
</style>
