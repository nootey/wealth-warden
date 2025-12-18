<script setup lang="ts">
import {computed, ref} from "vue";
import ShowLoading from "../components/base/ShowLoading.vue";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {useDataStore} from "../../services/stores/data_store.ts";

const emit = defineEmits<{
    (e: 'completeImport'): void;
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

    if(importing.value) {
        toastStore.infoResponseToast({"Title": "Unavailable", "Message": "An operation is currently being executed!"})
    }
    // clear local state
    selectedFiles.value = [];
    importing.value = false;
    useBalances.value = false;

    // clear FileUpload UI
    try {
        (uploadImportRef.value as any)?.clear?.();
    } catch { /* no-op */ }

}

const isDisabled = computed(() => {
    if (importing.value) return true;
    if (!selectedFiles.value.length) return true;
    return false
});

async function importAccounts() {

    if (!selectedFiles.value.length) return;
    importing.value = true;

    try {

        const form = new FormData();
        form.append("file", selectedFiles.value[0], "accounts.json");

        const res = await dataStore.importAccounts(form, useBalances.value);
        toastStore.successResponseToast(res);

        resetWizard();
        emit("completeImport");
    } catch (error) {
        toastStore.errorResponseToast(error);
    } finally {
        importing.value = false;
    }
}

defineExpose({isDisabled, importAccounts})

</script>

<template>
  <div class="flex flex-column w-full justify-content-center align-items-center text-center gap-3">
    <h3>Import your account data</h3>
    <span
      class="text-sm"
      style="color: var(--text-secondary)"
    >Upload your JSON file below. Please review the instructions before starting an import.</span>
    <span
      class="text-sm"
      style="color: var(--text-secondary)"
    >
      NOTE: You can also import the existing balances, but they will be set as starting balances. If you end up importing transactions after, note that those will count as additions to existing account balances.
    </span>
    <div class="flex align-items-center gap-1">
      <Checkbox
        v-model="useBalances"
        :binary="true"
        input-id="use-balances-pt"
      />
      <label
        for="use-balances-pt"
        style="color: var(--text-secondary)"
      >Use included balances</label>
    </div>
    <FileUpload
      v-if="!importing"
      ref="uploadImportRef"
      accept=".json, application/json"
      :max-file-size="10485760"
      :multiple="false"
      custom-upload
      :show-upload-button="false"
      :show-cancel-button="false"
      @select="onSelect"
      @clear="onClear"
    >
      <template #header="{ chooseCallback }">
        <div class="w-full flex flex-row justify-content-center">
          <Button
            class="outline-button w-3"
            :disabled="importing"
            label="Upload"
            @click="chooseCallback()"
          />
        </div>
      </template>

      <template #content>
        <div
          v-if="selectedFiles.length > 0"
          class="flex flex-column gap-1 w-full align-items-center"
        >
          <h5>Pending</h5>
          <div class="flex flex-wrap gap-2 w-full">
            <div
              v-for="file in selectedFiles"
              :key="file.name + file.type + file.size"
              class="flex flex-row gap-2 p-1 w-full justify-content-center align-items-center w-full"
            >
              <span class="font-semibold text-ellipsis whitespace-nowrap overflow-hidden">{{ file.name }}</span>
              <Badge
                value="Pending"
                severity="warn"
              />
              <i
                class="pi pi-times hover-icon"
                style="color: var(--p-red-300)"
                @click="resetWizard"
              />
            </div>
          </div>
        </div>
      </template>
    </FileUpload>
    <ShowLoading
      v-else
      :num-fields="3"
    />
  </div>
</template>

<style scoped>
.p-fileupload {
    width: 80% !important;
}
</style>