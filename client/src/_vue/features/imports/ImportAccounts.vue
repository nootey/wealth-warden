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
const useBalances = ref(false);

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
  useBalances.value = false;

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

async function importAccounts() {
  if (selectedFiles.value.length < 1) return;

  const file = selectedFiles.value[0];
  if (!file) return;

  importing.value = true;

  try {
    const form = new FormData();
    form.append("file", file, "accounts.json");

    const res = await dataStore.importAccounts(form, useBalances.value);
    toastStore.successResponseToast(res);

    emit("completeImport");
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    importing.value = false;
    resetWizard();
  }
}

defineExpose({ isDisabled, importAccounts });
</script>

<template>
  <div class="flex flex-col w-full gap-4">
    <h3>Import your account data</h3>
    <span class="text-sm" style="color: var(--text-secondary)"
      >Upload your JSON file below. Please review the instructions before
      starting an import.</span
    >
    <span class="text-sm" style="color: var(--text-secondary)">
      You can also import the existing balances, but they will be set as
      starting balances. If you end up importing transactions after, note that
      those will count as additions to existing account balances.
    </span>
    <div class="flex items-center gap-1">
      <Checkbox
        v-model="useBalances"
        :binary="true"
        input-id="use-balances-pt"
      />
      <label for="use-balances-pt" style="color: var(--text-secondary)"
        >Use included balances</label
      >
    </div>
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
