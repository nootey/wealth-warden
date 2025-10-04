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
    (e: 'onPage', value: any): void;
}>();
</script>

<template>
    <Paginator v-model:first="paginator.from" v-model:rows="paginator.rowsPerPage"
            :rowsPerPageOptions="rows" :totalRecords="paginator.total" @page="e => emit('onPage', e)"
               :template="{
                        '640px': 'PrevPageLink CurrentPageReport NextPageLink',
                        '960px': 'FirstPageLink PrevPageLink CurrentPageReport NextPageLink LastPageLink',
                        '1300px': 'FirstPageLink PrevPageLink PageLinks NextPageLink LastPageLink',
                        default: 'FirstPageLink PrevPageLink PageLinks NextPageLink LastPageLink'
                    }">
        <template #end>
            <div id="end" class="ml-2 text-sm">
                {{ `Showing ${paginator.from} to ${paginator.to} out of ${paginator.total} records` }}
            </div>
        </template>
    </Paginator>
</template>

<style scoped lang="scss">
@media (max-width: 640px) {
  :deep(.p-paginator-content-end) {
    flex: 1 1 100% !important;
    display: flex !important;
    justify-content: center !important;
    text-align: center !important;
    margin-top: 0.25rem !important;
  }

  #end {
    margin: 0 auto !important;
    text-align: center !important;
  }
}
</style>
