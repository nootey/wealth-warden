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
               :pt="{
                      root: { class: 'p-0 text-sm' },
                      pages: { class: 'm-0 p-0' },
                      pageButton: { class: 'p-0 text-sm' },
                      firstPageButton: { class: 'p-0 text-sm' },
                      prevPageButton:  { class: 'p-0 text-sm' },
                      nextPageButton:  { class: 'p-0 text-sm' },
                      lastPageButton:  { class: 'p-0 text-sm' },
                      rowsPerPageDropdown:        { class: 'p-0 text-sm' },
                      rowsPerPageDropdownTrigger: { class: 'p-0' },
                      rowsPerPageDropdownLabel:   { class: 'p-0 m-0' },
                      rowsPerPageDropdownPanel:   { class: 'p-0 text-sm' },
                      rowsPerPageDropdownItem:    { class: 'py-0 px-1 text-sm' },
                      currentPageReport: { class: 'ml-1 text-sm' }
                }">
        <template #end>
            <div class="ml-2 text-sm">
                {{ `Showing ${paginator.from} to ${paginator.to} out of ${paginator.total} records` }}
            </div>
        </template>
    </Paginator>
</template>
