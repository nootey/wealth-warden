<script setup lang="ts">

import type {Category} from "../../../models/transaction_models.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import {computed} from "vue";
import type {Column} from "../../../services/filter_registry.ts";

const props = defineProps<{
    categories: Category[];

}>();

const emits = defineEmits<{
    (e: "deleteCategory", payload: { id: number }): void;
}>();

const localCategories = computed(() => {
    return props.categories.filter((category) => !category.name.startsWith("("))
})

const categoryColumns = computed<Column[]>(() => [
    { field: 'display_name', header: 'Name'},
    { field: 'is_default', header: 'Type'},
    { field: 'classification', header: 'Classification'},
]);

</script>

<template>
    <DataTable class="w-full enhanced-table" dataKey="id" :value="localCategories"
               paginator :rows="10" :rowsPerPageOptions="[10, 25]" scrollable scroll-height="75vh"
               rowGroupMode="subheader" groupRowsBy="classification">
        <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
        <template #loading> <LoadingSpinner></LoadingSpinner> </template>

        <template #groupheader="slotProps">
            <div class="flex items-center gap-2">
                <span class="font-bold text-lg">{{ vueHelper.capitalize(slotProps.data.classification) }}</span>
            </div>
        </template>

        <Column v-for="col of categoryColumns" :key="col.field"
                :field="col.field" :header="col.header"
                :sortable="col.field === 'is_default'">
            <template #body="{ data, field }">
                <template v-if="field === 'is_default'">
                    {{ data.user_id ? "Custom" : "Default" }}
                </template>
                <template v-else>
                    {{ data[field] }}
                </template>
            </template>
        </Column>

        <Column header="Actions">
            <template #body="{ data }">
                <div class="flex flex-row align-items-center gap-2">
                    <i class="pi pi-pencil" style="font-size: 0.875rem;"></i>
                    <i class="pi pi-trash hover-icon" style="font-size: 0.875rem; color: var(--p-red-300);"
                       @click="$emit('deleteCategory', { id: data.id })"></i>
                </div>
            </template>
        </Column>

    </DataTable>
</template>

<style scoped>

</style>