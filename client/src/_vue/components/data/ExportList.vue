<script setup lang="ts">

import vueHelper from "../../../utils/vue_helper.ts";
import dateHelper from "../../../utils/date_helper.ts";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import type {Column} from "../../../services/filter_registry.ts";
import {useDataStore} from "../../../services/stores/data_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {usePermissions} from "../../../utils/use_permissions.ts";
import {useConfirm} from "primevue/useconfirm";
import {computed, onMounted, ref} from "vue";
import type {Export} from "../../../models/dataio_models.ts";

const dataStore = useDataStore();
const toastStore = useToastStore();
const sharedStore = useSharedStore();

const { hasPermission } = usePermissions();
const confirm = useConfirm();

const exports = ref<Export[]>([]);
const loading = ref(false);

onMounted(async () => {
    await getData()
})

async function getData() {
    try {
        exports.value = await dataStore.getExports();
    } catch (e) {
        toastStore.errorResponseToast(e)
    }
}

function refresh() { getData(); }

defineExpose({ refresh });

const activeColumns = computed<Column[]>(() => [
    { field: 'name', header: 'Name'},
    { field: 'status', header: 'Status'},
    { field: 'currency', header: 'Currency'},
]);

async function deleteConfirmation(id: number, name: string) {
    confirm.require({
        header: 'Delete record?',
        message: `This will delete export: ${name}".`,
        icon: "pi pi-exclamation-triangle",
        acceptLabel: "Delete",
        rejectLabel: "Cancel",
        acceptClass: "p-button-danger",
        rejectClass: "p-button-text",
        accept: () => deleteRecord(id),
    });
}

async function downloadExport(id: number) {
    try {
        await dataStore.downloadExport(id);
        toastStore.successResponseToast({title: "Success", message: `Data exported`});
        await getData();
    } catch (error) {
        toastStore.errorResponseToast(error);
    }
}

async function deleteRecord(id: number) {

    if(!hasPermission("delete_export")) {
        toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
        return;
    }

    try {
        let response = await sharedStore.deleteRecord(
            "exports",
            id
        );
        toastStore.successResponseToast(response);
        await getData();
    } catch (error) {
        toastStore.errorResponseToast(error);
    }
}

</script>

<template>
  <div class="w-full flex flex-row gap-2 justify-content-center">
    <DataTable
      data-key="id"
      class="w-full enhanced-table"
      :loading="loading"
      :value="exports"
      scrollable
      scroll-height="50vh"
      column-resize-mode="fit"
      scroll-direction="both"
    >
      <template #empty>
        <div style="padding: 10px;">
          No records found.
        </div>
      </template>
      <template #loading>
        <LoadingSpinner />
      </template>
      <Column header="Actions">
        <template #body="{ data }">
          <div class="flex flex-row align-items-center gap-2">
            <i
              v-if="hasPermission('manage_data')"
              class="pi pi-download hover-icon"
              style="font-size: 0.875rem;"
              @click="downloadExport(data?.id)"
            />
            <i
              v-if="hasPermission('manage_data')"
              class="pi pi-trash hover-icon"
              style="font-size: 0.875rem; color: var(--p-red-300);"
              @click="deleteConfirmation(data?.id, data?.name)"
            />
          </div>
        </template>
      </Column>
      <Column
        v-for="col of activeColumns"
        :key="col.field"
        :header="col.header"
        :field="col.field"
        :header-class="col.hideOnMobile ? 'mobile-hide ' : ''"
        :body-class="col.hideOnMobile ? 'mobile-hide ' : ''"
      >
        <template #body="{ data }">
          <template v-if="col.field === 'amount'">
            {{ vueHelper.displayAsCurrency(data.transaction_type == "expense" ? (data.amount*-1) : data.amount) }}
          </template>
          <template v-else-if="col.field === 'started_at' || col.field === 'completed_at'">
            {{ dateHelper.formatDate(data[col.field], true) }}
          </template>
          <template v-else>
            {{ data[col.field] }}
          </template>
        </template>
      </Column>
    </DataTable>
  </div>
</template>

<style scoped>

</style>