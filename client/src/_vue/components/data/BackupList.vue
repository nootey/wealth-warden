<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import type { Column } from "../../../services/filter_registry.ts";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import dateHelper from "../../../utils/date_helper.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { usePermissions } from "../../../utils/use_permissions.ts";
import { useConfirm } from "primevue/useconfirm";
import type { BackupInfo } from "../../../models/dataio_models.ts";
import { useAuthStore } from "../../../services/stores/auth_store.ts";

const settingsStore = useSettingsStore();
const toastStore = useToastStore();
const authStore = useAuthStore();

const { hasPermission } = usePermissions();
const confirm = useConfirm();

const backups = ref<BackupInfo[]>([]);
const loading = ref(false);

onMounted(async () => {
  await getData();
});

async function getData() {
  try {
    loading.value = true;
    const response = await settingsStore.getBackups();
    backups.value = response.data.backups || [];
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    loading.value = false;
  }
}

function refresh() {
  getData();
}

defineExpose({ refresh });

const activeColumns = computed<Column[]>(() => [
  { field: "name", header: "Name" },
  { field: "metadata.app_version", header: "App ver.", hideOnMobile: true },
  { field: "metadata.db_version", header: "DB ver.", hideOnMobile: true },
  { field: "metadata.backup_size", header: "Size" },
  { field: "metadata.created_at", header: "Created" },
]);

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 Bytes";
  const k = 1024;
  const sizes = ["Bytes", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + " " + sizes[i];
}

function getNestedValue(obj: any, path: string): any {
  return path.split(".").reduce((current, key) => current?.[key], obj);
}

async function restoreConfirmation(backupName: string) {
  confirm.require({
    header: "Restore database?",
    message: `This will restore the database from backup: "${backupName}". All current data will be replaced. This action cannot be undone.`,
    icon: "pi pi-exclamation-triangle",
    acceptLabel: "Restore",
    rejectLabel: "Cancel",
    acceptClass: "p-button-danger",
    rejectClass: "p-button-text",
    accept: () => restoreBackup(backupName),
  });
}

async function restoreBackup(backupName: string) {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }

  try {
    loading.value = true;
    await settingsStore.restoreFromDatabaseDump(backupName);
    toastStore.successResponseToast({
      title: "Success",
      message: "Database restored successfully",
    });
    authStore.logout();
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
}

async function downloadBackup(backupName: string) {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }

  // TODO: Implement download functionality
  toastStore.createInfoToast(
    "Not implemented",
    "Download functionality will be implemented soon.",
  );

  console.log("Download backup:", backupName);
}
</script>

<template>
  <div class="w-full flex flex-row gap-2 justify-content-center">
    <DataTable
      data-key="name"
      class="w-full enhanced-table"
      :loading="loading"
      :value="backups"
      scrollable
      scroll-height="50vh"
      column-resize-mode="fit"
      scroll-direction="both"
    >
      <template #empty>
        <div style="padding: 10px">No backups found.</div>
      </template>
      <template #loading>
        <LoadingSpinner />
      </template>
      <Column header="Actions">
        <template #body="{ data }">
          <div class="flex flex-row align-items-center gap-2">
            <i
              v-if="hasPermission('manage_data')"
              v-tooltip.top="'Download'"
              class="pi pi-download hover-icon"
              style="font-size: 0.875rem"
              @click="downloadBackup(data?.name)"
            />
            <i
              v-if="hasPermission('manage_data')"
              v-tooltip.top="'Restore'"
              class="pi pi-refresh hover-icon"
              style="font-size: 0.875rem; color: var(--p-orange-300)"
              @click="restoreConfirmation(data?.name)"
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
          <template v-if="col.field === 'metadata.created_at'">
            {{ dateHelper.formatDate(data.metadata.created_at, true) }}
          </template>
          <template v-else-if="col.field === 'metadata.backup_size'">
            {{ formatBytes(data.metadata.backup_size) }}
          </template>
          <template v-else>
            {{ getNestedValue(data, col.field) }}
          </template>
        </template>
      </Column>
    </DataTable>
  </div>
</template>

<style scoped></style>
