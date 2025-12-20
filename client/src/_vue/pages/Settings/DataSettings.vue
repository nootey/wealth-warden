<script setup lang="ts">
import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import ImportList from "../../components/data/ImportList.vue";
import { ref } from "vue";
import { usePermissions } from "../../../utils/use_permissions.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import ExportModule from "../../features/imports/ExportModule.vue";
import ExportList from "../../components/data/ExportList.vue";
import ImportModule from "../../components/data/ImportModule.vue";

const toastStore = useToastStore();
const { hasPermission } = usePermissions();

const importListRef = ref<InstanceType<typeof ImportList> | null>(null);
const exportListRef = ref<InstanceType<typeof ExportList> | null>(null);
const importModuleRef = ref<InstanceType<typeof ImportModule> | null>(null);

const addImportModal = ref(false);
const addExportModal = ref(false);
const transferModal = ref(false);

function refreshData(module: string) {
  switch (module) {
    case "import": {
      importListRef.value?.refresh();
      addImportModal.value = false;
      transferModal.value = false;
      break;
    }
    case "export": {
      addExportModal.value = false;
      exportListRef.value?.refresh();
      break;
    }
    default: {
      break;
    }
  }
}

function manipulateDialog(modal: string, value: any) {
  switch (modal) {
    case "addImport": {
      if (!hasPermission("manage_data")) {
        toastStore.createInfoToast(
          "Access denied",
          "You don't have permission to perform this action.",
        );
        return;
      }
      addImportModal.value = value;
      break;
    }
    case "addExport": {
      if (!hasPermission("manage_data")) {
        toastStore.createInfoToast(
          "Access denied",
          "You don't have permission to perform this action.",
        );
        return;
      }
      addExportModal.value = value;
      break;
    }
    default: {
      break;
    }
  }
}
</script>

<template>
  <Dialog
    v-model:visible="addImportModal"
    class="rounded-dialog"
    :breakpoints="{ '751px': '90vw' }"
    :modal="true"
    :style="{ width: '750px' }"
    header="New Import"
  >
    <ImportModule ref="importModuleRef" @refresh-data="(e) => refreshData(e)" />
    <template #footer>
      <Button
        label="Start"
        class="main-button w-4"
        :disabled="importModuleRef?.isDisabled"
        @click="importModuleRef?.startOperation"
      />
    </template>
  </Dialog>

  <Dialog
    v-model:visible="addExportModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="New Export"
  >
    <ExportModule @complete-export="refreshData('export')" />
  </Dialog>

  <div class="flex flex-column w-full gap-3">
    <SettingsSkeleton class="w-full">
      <div class="w-full flex flex-column gap-3 p-2">
        <div class="flex flex-row align-items-center gap-2 w-full">
          <div class="w-full flex flex-column gap-2">
            <h3>Data Import</h3>
            <h5 style="color: var(--text-secondary)">
              Manage your imported data.
            </h5>
          </div>
          <Button
            class="main-button"
            @click="manipulateDialog('addImport', true)"
          >
            <div class="flex flex-row gap-1 align-items-center">
              <i class="pi pi-plus" />
              <span> New </span>
              <span class="mobile-hide"> Import </span>
            </div>
          </Button>
        </div>

        <h3>Imports</h3>
        <ImportList ref="importListRef" />
      </div>
    </SettingsSkeleton>

    <SettingsSkeleton class="w-full">
      <div class="w-full flex flex-column gap-3 p-2">
        <div class="flex flex-row align-items-center gap-2 w-full">
          <div class="w-full flex flex-column gap-2">
            <h3>Data Export</h3>
            <h5 style="color: var(--text-secondary)">Export your data.</h5>
          </div>
          <Button
            class="main-button"
            @click="manipulateDialog('addExport', true)"
          >
            <div class="flex flex-row gap-1 align-items-center">
              <i class="pi pi-plus" />
              <span> New </span>
              <span class="mobile-hide"> Export </span>
            </div>
          </Button>
        </div>

        <h3>Exports</h3>
        <ExportList ref="exportListRef" />
      </div>
    </SettingsSkeleton>
  </div>
</template>

<style scoped></style>
