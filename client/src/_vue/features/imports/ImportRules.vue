<script setup lang="ts">
import { computed, ref } from "vue";
import ShowLoading from "../../components/base/ShowLoading.vue";
import UploadDropzone from "../../components/base/UploadDropzone.vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useDataStore } from "../../../services/stores/data_store.ts";

const emit = defineEmits<{
  (e: "completeImport"): void;
}>();

const toastStore = useToastStore();
const dataStore = useDataStore();

const importing = ref(false);

const selectedFiles = ref<File[]>([]);
const uploadImportRef = ref<{ files: File[] } | null>(null);

function onSelect(e: { files: File[] }) {
  selectedFiles.value = e.files.slice(0, 1);
}

function onClear() {
  selectedFiles.value = [];
}

function resetWizard() {
  if (importing.value) {
    toastStore.infoResponseToast({
      title: "Unavailable",
      message: "An operation is currently being executed!",
    });
  }
  // clear local state
  selectedFiles.value = [];
  importing.value = false;

  // clear FileUpload UI
  try {
    (uploadImportRef.value as any)?.clear?.();
  } catch {
    /* no-op */
  }
}

const isDisabled = computed(() => {
  if (importing.value) return true;
  if (!selectedFiles.value.length) return true;
  return false;
});

async function importRules() {
  if (selectedFiles.value.length < 1) return;

  const file = selectedFiles.value[0];
  if (!file) return;

  importing.value = true;

  try {
    const form = new FormData();
    form.append("file", file, "rules.json");

    const res = await dataStore.importRules(form);
    toastStore.successResponseToast(res);

    emit("completeImport");
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    importing.value = false;
    resetWizard();
  }
}

defineExpose({ isDisabled, importRules });
</script>

<template>
  <div class="flex flex-col w-full gap-4">
    <h3>Import your rule data</h3>
    <span class="text-sm" style="color: var(--text-secondary)"
      >Upload your JSON file below. Please review the instructions before
      starting an import.</span
    >

    <UploadDropzone
      v-if="!importing"
      ref="uploadImportRef"
      accept=".json, application/json"
      hint="Accepts .json"
      info="Only Wealth Warden JSON exports are supported. Other files may fail to import."
      :files="selectedFiles"
      @select="onSelect"
      @remove="resetWizard"
      @clear="onClear"
    />
    <ShowLoading v-else :num-fields="3" />
  </div>
</template>
