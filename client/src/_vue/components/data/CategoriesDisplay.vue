<script setup lang="ts">

import type {Category} from "../../../models/transaction_models.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import {computed, ref} from "vue";
import type {Column} from "../../../services/filter_registry.ts";
import CategoryForm from "../forms/CategoryForm.vue";
import {usePermissions} from "../../../utils/use_permissions.ts";
import {useConfirm} from "primevue/useconfirm";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useSharedStore} from "../../../services/stores/shared_store.ts";

const props = defineProps<{
    categories: Category[];
}>();

const emit = defineEmits<{
    (e: "completeOperation"): void;
    (e: "completeDelete"): void;
}>();

const toastStore = useToastStore();
const sharedStore = useSharedStore();

const { hasPermission } = usePermissions();
const confirm = useConfirm();

const localCategories = computed(() => {
    return props.categories.filter(
        (c) =>
            !c.name.startsWith("(") &&
            c.display_name !== "Expense" &&
            c.display_name !== "Income"
    )
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
            await deleteConfirmation(data.id, data.display_name, data.deleted_at);
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

async function deleteConfirmation(id: number, name: string, deleted: Date | null) {
    confirm.require({
        header: 'Confirm operation',
        message: `You are about to ${!deleted ? 'archive' : 'delete'} category: "${name}". ${!deleted ? '' : 'This action is irreversible!'}`,
        rejectProps: { label: 'Cancel' },
        acceptProps: { label: 'Continue', severity: 'danger' },
        accept: () => deleteRecord(id),
    });
}

async function deleteRecord(id: number) {

    if(!hasPermission("manage_data")) {
        toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
        return;
    }

    try {
        let response = await sharedStore.deleteRecord(
            "transactions/categories",
            id,
        );
        toastStore.successResponseToast(response);
        emit("completeDelete")

    } catch (err) {
        toastStore.errorResponseToast(err)
    }
}

</script>

<template>
  <Dialog
    v-model:visible="updateModal"
    position="right"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Update category"
  >
    <CategoryForm
      mode="update"
      :record-id="selectedID"
      @complete-operation="handleEmit('completeOperation')"
    />
  </Dialog>

  <DataTable
    class="w-full enhanced-table"
    data-key="id"
    :value="localCategories"
    paginator
    :rows="10"
    :rows-per-page-options="[10, 25]"
    scrollable
    scroll-height="75vh"
    row-group-mode="subheader"
    group-rows-by="classification"
    :row-class="vueHelper.deletedRowClass"
  >
    <template #empty>
      <div style="padding: 10px;">
        No records found.
      </div>
    </template>
    <template #loading>
      <LoadingSpinner />
    </template>

    <template #groupheader="slotProps">
      <div class="flex items-center gap-2">
        <span class="font-bold text-lg">{{ vueHelper.capitalize(slotProps.data.classification) }}</span>
      </div>
    </template>

    <Column
      v-for="col of categoryColumns"
      :key="col.field"
      :field="col.field"
      :header="col.header"
      :sortable="col.field === 'is_default'"
    >
      <template #body="{ data }">
        <template v-if="col.field === 'is_default'">
          {{ data.user_id ? "Custom" : "Default" }}
        </template>
        <template v-else>
          {{ data[col.field] }}
        </template>
      </template>
    </Column>

    <Column header="Actions">
      <template #body="{ data }">
        <div class="flex flex-row align-items-center gap-2">
          <i
            v-if="hasPermission('manage_data')"
            v-tooltip="'Edit category'"
            class="pi pi-pen-to-square hover-icon text-xs"
            @click="openModal('update', data.id!)"
          />
          <i
            v-if="hasPermission('manage_data') && showDeleteButton(data)"
            v-tooltip="'Delete category'"
            class="pi pi-trash hover-icon text-xs"
            style="color: var(--p-red-300);"
            @click="handleEmit('deleteCategory', data)"
          />
          <i
            v-if="!hasPermission('manage_data')"
            v-tooltip="'No action currently available.'"
            class="pi pi-ban hover-icon"
          />
        </div>
      </template>
    </Column>
  </DataTable>
</template>

<style scoped>

</style>