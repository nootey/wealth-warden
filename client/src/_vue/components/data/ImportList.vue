<script setup lang="ts">
import type { Import } from "../../../models/dataio_models.ts";
import { computed, onMounted, ref } from "vue";
import { useDataStore } from "../../../services/stores/data_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import vueHelper from "../../../utils/vue_helper.ts";
import type { Column } from "../../../services/filter_registry.ts";
import dateHelper from "../../../utils/date_helper.ts";
import { usePermissions } from "../../../utils/use_permissions.ts";
import { useConfirm } from "primevue/useconfirm";
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import DisplayStatus from "../base/DisplayStatus.vue";

const dataStore = useDataStore();
const toastStore = useToastStore();
const sharedStore = useSharedStore();

const { hasPermission } = usePermissions();
const confirm = useConfirm();

const imports = ref<Import[]>([]);
const loading = ref(false);
const expandedRows = ref<Record<string, boolean>>({});

function isExpanded(row: Import): boolean {
  return row.id != null && expandedRows.value[row.id] === true;
}

function toggleDetails(row: Import): void {
  if (row.id == null) return;
  const next = { ...expandedRows.value };
  if (next[row.id]) {
    delete next[row.id];
  } else {
    next[row.id] = true;
  }
  expandedRows.value = next;
}

onMounted(async () => {
  await getData();
});

async function getData() {
  try {
    const [custom, bank] = await Promise.all([
      dataStore.getImports("custom"),
      dataStore.getImports("bank"),
    ]);
    imports.value = [...custom, ...bank];
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
}

function refresh() {
  getData();
}

defineExpose({ refresh });

const activeColumns = computed<Column[]>(() => [
  { field: "name", header: "Name" },
  { field: "type", header: "Type", hideOnMobile: true },
  { field: "sub_type", header: "Sub type", hideOnMobile: true },
  { field: "status", header: "Status" },
]);

async function deleteConfirmation(id: number, name: string) {
  confirm.require({
    group: "delete-import",
    header: "Delete record?",
    message: `This will delete import: ${name}".`,
    icon: "pi pi-exclamation-triangle",
    acceptLabel: "Delete",
    rejectLabel: "Cancel",
    acceptClass: "p-button-danger",
    rejectClass: "p-button-text",
    accept: () => deleteRecord(id),
  });
}

async function deleteRecord(id: number) {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }

  try {
    let response = await sharedStore.deleteRecord("imports", id);
    toastStore.successResponseToast(response);
    await getData();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}
</script>

<template>
  <ConfirmDialog group="delete-import" class="rounded-dialog">
    <template #container="{ message, acceptCallback, rejectCallback }">
      <div class="flex flex-col gap-2 p-4 justify-center w-full">
        <div
          class="flex flex-col gap-4 p-8 justify-center items-center text-center"
        >
          <span class="text-lg">{{ message.message }}</span>
          <strong>This action is irreversible!</strong>
        </div>
        <div class="flex justify-end gap-2">
          <Button
            class="p-2 rounded-lg"
            :label="message.rejectProps?.label || 'Cancel'"
            variant="outlined"
            style="
              color: var(--text-primary);
              border-color: var(--text-primary);
            "
            @click="rejectCallback"
          />
          <Button
            class="p-2 rounded-lg"
            :label="message.acceptProps?.label || 'Confirm'"
            severity="danger"
            style="color: var(--text-primary)"
            @click="acceptCallback"
          />
        </div>
      </div>
    </template>
  </ConfirmDialog>

  <div class="flex flex-col w-full gap-4">
    <div
      class="flex flex-col w-full rounded-2xl"
      style="
        padding: 0.25rem 0.25rem 0 0.25rem;
        border: 1px solid var(--border-color);
      "
    >
      <DataTable
        v-model:expanded-rows="expandedRows"
        data-key="id"
        class="w-full enhanced-table"
        :loading="loading"
        :value="imports"
        scrollable
        column-resize-mode="fit"
        scroll-direction="both"
        paginator
        sort-field="created_at"
        :sort-order="-1"
        :rows="10"
        :rows-per-page-options="[10, 25, 50]"
      >
        <template #empty>
          <div style="padding: 10px">No records found.</div>
        </template>
        <template #loading>
          <LoadingSpinner />
        </template>
        <template #expansion="{ data }">
          <div class="flex flex-col gap-2 p-3">
            <div
              id="import-error-meta"
              class="flex flex-row gap-4 text-sm"
              style="color: var(--text-secondary)"
            >
              <span>Type: {{ data.type }}</span>
              <span>Sub type: {{ data.sub_type }}</span>
            </div>
            <div
              v-if="data.status === 'failed'"
              class="flex flex-row items-start gap-2"
              style="color: var(--p-red-300)"
            >
              <i class="pi pi-exclamation-circle mt-1 text-sm" />
              <span id="import-error-text">{{
                data.error || "No error detail was recorded."
              }}</span>
            </div>
          </div>
        </template>
        <Column header="Actions">
          <template #body="{ data }">
            <div class="flex flex-row items-center gap-2">
              <i
                v-if="hasPermission('manage_data')"
                class="pi pi-trash hover-icon text-sm"
                style="color: var(--p-red-300)"
                @click="deleteConfirmation(data?.id, data?.name)"
              />
              <i
                :id="
                  data.status === 'failed' ? undefined : 'import-expand-toggle'
                "
                v-tooltip="isExpanded(data) ? 'Hide details' : 'Show details'"
                :class="
                  isExpanded(data)
                    ? 'pi pi-chevron-down'
                    : 'pi pi-chevron-right'
                "
                class="hover-icon text-sm"
                @click="toggleDetails(data)"
              />
              <i
                v-if="data.investments_transferred"
                v-tooltip="'Investments transferred'"
                class="pi pi-database hover-icon text-sm"
              />
              <i
                v-if="data.savings_transferred"
                v-tooltip="'Savings transferred'"
                class="pi pi-credit-card hover-icon text-sm"
              />
              <i
                v-if="data.repayments_transferred"
                v-tooltip="'Repayments transferred'"
                class="pi pi-building-columns hover-icon text-sm"
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
              {{
                vueHelper.displayAsCurrency(
                  data.direction == "expense" ? data.amount * -1 : data.amount,
                )
              }}
            </template>
            <template
              v-else-if="
                col.field === 'started_at' || col.field === 'completed_at'
              "
            >
              {{ dateHelper.formatDate(data[col.field], true) }}
            </template>
            <template v-else-if="col.field === 'status'">
              <DisplayStatus :status="data.status" />
            </template>
            <template v-else-if="col.field === 'name'">
              <span v-tooltip.top="data[col.field]" class="truncate-text">
                {{ data[col.field] }}
              </span>
            </template>
            <template v-else>
              {{ data[col.field] }}
            </template>
          </template>
        </Column>
      </DataTable>
    </div>
  </div>
</template>

<style scoped>
#import-error-meta {
  display: none;
}

#import-expand-toggle {
  display: none;
}

@media (max-width: 768px) {
  #import-error-meta {
    display: flex;
  }

  #import-expand-toggle {
    display: inline-block;
  }

  #import-error-text {
    font-size: 0.75rem;
  }
}
</style>
