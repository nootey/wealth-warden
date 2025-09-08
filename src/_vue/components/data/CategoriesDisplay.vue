<script setup lang="ts">

import type {Category} from "../../../models/transaction_models.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import {computed, ref} from "vue";
import type {Column} from "../../../services/filter_registry.ts";
import CategoryForm from "../forms/CategoryForm.vue";

const props = defineProps<{
    categories: Category[];
}>();

const emit = defineEmits<{
    (e: "completeOperation"): void;
    (e: "deleteCategory", id: number, name: string, deleted_at: Date | null): void;
}>();

const localCategories = computed(() => {
    return props.categories.filter((category) => !category.name.startsWith("("))
})

const updateModal = ref(false);
const selectedID = ref<number | null>(null);

const categoryColumns = computed<Column[]>(() => [
    { field: 'display_name', header: 'Name'},
    { field: 'is_default', header: 'Type'},
    { field: 'classification', header: 'Classification'},
]);

function openModal(type: string, data: any) {
    switch (type) {
        case "update": {
            updateModal.value = true;
            selectedID.value = data;
            break;
        }
    }
}

async function handleEmit(type: string, data?: any) {
    switch (type) {
        case "completeOperation": {
            updateModal.value = false;
            emit("completeOperation");
            break;
        }
        case "deleteCategory": {
            emit("deleteCategory", data.id, data.display_name, data.deleted_at);
            break;
        }
    }
}

function showDeleteButton(data: Category) {
        switch (data.is_default){
            case true: {
                return !data.deleted_at
            }
            default: {
                return true;
            }
        }
}

</script>

<template>

    <Dialog position="right" class="rounded-dialog" v-model:visible="updateModal"
            :breakpoints="{ '501px': '90vw' }" :modal="true" :style="{ width: '500px' }" header="Update category">
        <CategoryForm mode="update" :recordId="selectedID"
                     @completeOperation="handleEmit('completeOperation')"/>
    </Dialog>

    <DataTable class="w-full enhanced-table" dataKey="id" :value="localCategories"
               paginator :rows="10" :rowsPerPageOptions="[10, 25]" scrollable scroll-height="75vh"
               rowGroupMode="subheader" groupRowsBy="classification" :rowClass="vueHelper.deletedRowClass">
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
                    <i class="pi pi-pen-to-square hover-icon text-xs" v-tooltip="'Edit category'"
                       @click="openModal('update', data.id!)"/>
                    <i v-if="showDeleteButton(data)" class="pi pi-trash hover-icon text-xs" v-tooltip="'Delete category'"
                       style="color: var(--p-red-300);"
                       @click="handleEmit('deleteCategory', data)"></i>
                </div>
            </template>
        </Column>

    </DataTable>
</template>

<style scoped>

</style>