<script setup lang="ts">
type PaginatorState = {
  total: number;
  from: number;
  to: number;
  rowsPerPage: number;
};

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
      default: 'FirstPageLink PrevPageLink PageLinks NextPageLink LastPageLink RowsPerPageDropdown',
    }"
    @page="(e) => emit('onPage', e)"
  >
    <template #end>
      <div id="end" class="ml-2 text-sm">
        {{
          `Showing ${paginator.from} to ${paginator.to} out of ${paginator.total} records`
        }}
      </div>
    </template>
  </Paginator>
</template>

<style scoped>

</style>