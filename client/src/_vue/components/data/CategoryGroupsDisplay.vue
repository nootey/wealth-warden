<script setup lang="ts">

import type {Category, CategoryGroup} from "../../../models/transaction_models.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import {computed, ref} from "vue";
import type {Column} from "../../../services/filter_registry.ts";
import {usePermissions} from "../../../utils/use_permissions.ts";
import {useConfirm} from "primevue/useconfirm";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import CategoryGroupForm from "../forms/CategoryGroupForm.vue";

defineProps<{
    categoryGroups: CategoryGroup[];
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

const updateModal = ref(false);
const selectedID = ref<number | null>(null);

const activeColumns = computed<Column[]>(() => [
    { field: 'name', header: 'Name'},
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
        case "deleteCategoryGroup": {
            await deleteConfirmation(data.id, data.name);
            break;
        }
    }
}

async function deleteConfirmation(id: number, name: string) {
    confirm.require({
        header: 'Confirm operation',
        message: `You are about to delete category group: "${name}". 'This action is irreversible!'`,
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
            "transactions/categories/groups",
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
    header="Update group"
  >
    <CategoryGroupForm
      mode="update"
      :record-id="selectedID"
      :categories="categories"
      @complete-operation="handleEmit('completeOperation')"
    />
  </Dialog>

  <DataTable
    class="w-full enhanced-table"
    data-key="id"
    :value="categoryGroups"
    paginator
    :rows="10"
    :rows-per-page-options="[10, 25]"
    scrollable
    scroll-height="75vh"
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

    <Column
      v-for="col of activeColumns"
      :key="col.field"
      :field="col.field"
      :header="col.header"
    >
      <template #body="{ data }">
        {{ data[col.field] }}
      </template>
    </Column>

    <Column header="Categories">
      <template #body="{ data }">
        <div
          v-tooltip="'This group has ' + (data?.categories?.length ?? 0) + ' categories'"
          class="flex flex-row align-items-center gap-2"
        >
          <i class="pi pi-eye" />
          <span>{{ data?.categories?.length ?? 0 }}</span>
        </div>
      </template>
    </Column>

    <Column header="Actions">
      <template #body="{ data }">
        <div class="flex flex-row align-items-center gap-2">
          <i
            v-if="hasPermission('manage_data')"
            v-tooltip="'Edit category group'"
            class="pi pi-pen-to-square hover-icon text-xs"
            @click="openModal('update', data.id!)"
          />
          <i
            v-if="hasPermission('manage_data')"
            v-tooltip="'Delete group'"
            class="pi pi-trash hover-icon text-xs"
            style="color: var(--p-red-300);"
            @click="handleEmit('deleteCategoryGroup', data)"
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