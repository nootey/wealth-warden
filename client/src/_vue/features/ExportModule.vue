<script setup lang="ts">
import { useDataStore } from "../../services/stores/data_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { ref } from "vue";

const emit = defineEmits<{
  (e: "completeExport"): void;
}>();

const dataStore = useDataStore();
const toastStore = useToastStore();

const exporting = ref(false);

async function exportData() {
  exporting.value = true;
  try {
    await dataStore.exportData();
    emit("completeExport");
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    exporting.value = false;
  }
}
</script>

<template>
  <div class="flex flex-column gap-3">
    <h3>Export your data</h3>
    <span
      >This will create a downloadable snapshot of your accounts, transactions,
      transfers and categories.</span
    >
    <span>A zip file will be created for the seperate exported modules.</span>
    <Button
      class="main-button w-3"
      label="Export"
      :disabled="exporting"
      :icon="exporting ? 'pi pi-spinner pi-spin' : ''"
      @click="exportData"
    />
  </div>
</template>

<style scoped></style>
