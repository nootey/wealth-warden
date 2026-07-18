<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import type { Account } from "../../../models/account_models.ts";
import { useDataStore } from "../../../services/stores/data_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useAccountStore } from "../../../services/stores/account_store.ts";
import type {
  CustomImportValidationResponse,
  Import,
} from "../../../models/dataio_models.ts";
import ShowLoading from "../../components/base/ShowLoading.vue";
import ImportTransferMapping from "../../components/base/ImportTransferMapping.vue";

const emit = defineEmits<{
  (e: "completeTransfer"): void;
}>();

const dataStore = useDataStore();
const toastStore = useToastStore();
const accStore = useAccountStore();

const transfering = ref(false);
const investmentAccs = ref<Account[]>([]);
const fileValidated = ref(false);

const investmentMappings = ref<Record<string, number | null>>({});
const validatedResponse = ref<CustomImportValidationResponse | null>(null);

const selectedFiles = ref<File[]>([]);
const selectedImport = ref<Import | null>(null);
const loadingValidation = ref(false);

watch(selectedImport, async (newImport) => {
  if (newImport && newImport.id) {
    await fetchValidationResponse(newImport.id);
  } else {
    validatedResponse.value = null;
    investmentMappings.value = {};
  }
});

onMounted(async () => {
  try {
    const [investments, crypto] = await Promise.all([
      accStore.getAccountsByType("investment"),
      accStore.getAccountsByType("crypto"),
    ]);

    // merge and remove duplicates
    const merged = [...investments, ...crypto];
    investmentAccs.value = merged.filter(
      (a, i, arr) => arr.findIndex((b) => b.id === a.id) === i,
    );

    if (investmentAccs.value.length === 0) {
      toastStore.infoResponseToast({
        title: "No accounts",
        message: "Please create at least one investment or crypto account",
      });
    }
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
});

async function fetchValidationResponse(importId: number) {
  loadingValidation.value = true;
  try {
    validatedResponse.value = await dataStore.getCustomImportJSON(
      importId,
      "investment_trades",
    );
  } catch (e) {
    toastStore.errorResponseToast(e);
    validatedResponse.value = null;
  } finally {
    loadingValidation.value = false;
  }
}

function onSaveMapping(map: Record<string, number | null>) {
  investmentMappings.value = map;
}

async function validateFile(type: string) {
  if (selectedFiles.value.length < 1) return;

  const file = selectedFiles.value[0];
  if (!file) return;

  try {
    const res = await dataStore.validateImport("custom", file, type);
    fileValidated.value = true;
    validatedResponse.value = res;
    toastStore.successResponseToast({
      title: "File validated",
      message: "Check details and proceed with import",
    });
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

function onSelect(e: { files: File[] }) {
  selectedFiles.value = e.files.slice(0, 1);
  fileValidated.value = false;
  validatedResponse.value = null;
}

function onClear() {
  selectedFiles.value = [];
  fileValidated.value = false;
  validatedResponse.value = null;
}

function resetWizard() {
  if (transfering.value) {
    toastStore.infoResponseToast({
      title: "Unavailable",
      message: "An operation is currently being executed!",
    });
  }
  // clear local state
  transfering.value = false;
  validatedResponse.value = null;
}

async function transferInvestmentTrades() {
  if (!selectedFiles.value.length) {
    toastStore.errorResponseToast({
      title: "Error",
      message: "No file selected",
    });
    return;
  }

  if (Object.keys(investmentMappings.value).length === 0) {
    toastStore.errorResponseToast({
      title: "Error",
      message: "Please set up your investment mappings first",
    });
    return;
  }

  const file = selectedFiles.value[0];
  if (!file) return;

  transfering.value = true;

  try {
    const formData = new FormData();
    formData.append("file", file);
    formData.append(
      "trade_mappings",
      JSON.stringify(
        Object.entries(investmentMappings.value).map(([name, account_id]) => ({
          name,
          account_id,
        })),
      ),
    );

    const res = await dataStore.transferInvestmentTradesFromImport(formData);
    toastStore.successResponseToast(res);

    emit("completeTransfer");
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    transfering.value = false;
    resetWizard();
  }
}

const isDisabled = computed(() => {
  if (transfering.value) return true;

  const mappings = Object.values(investmentMappings.value);
  const hasAtLeastOne = mappings.some((v) => v !== null);
  return !hasAtLeastOne;
});

defineExpose({ isDisabled, transferInvestmentTrades });
</script>

<template>
  <div
    class="flex flex-column w-full justify-content-center align-items-center text-center gap-3"
  >
    <h3>Import your investment trade data</h3>
    <span class="text-sm" style="color: var(--text-secondary)"
      >Upload your JSON file below. Please review the instructions before
      starting an import.</span
    >
    <span class="text-sm" style="color: var(--text-secondary)">
      NOTE: Assets will be created automatically. It is recommended to not have
      existing ones.
    </span>

    <FileUpload
      v-if="!transfering"
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
        <div class="flex flex-row w-full justify-content-center">
          <Button
            class="outline-button"
            :disabled="transfering"
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
              <span
                class="font-semibold text-ellipsis whitespace-nowrap overflow-hidden"
                >{{ file.name }}</span
              >
              <Badge value="Pending" severity="warn" />
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
    <ShowLoading v-else :num-fields="3" />

    <div
      v-if="!fileValidated"
      class="flex flex-column w-full justify-content-center align-items-center gap-3"
    >
      <span style="color: var(--text-secondary)">
        Once you have uploaded a document, it needs to be validated.
      </span>
      <div
        class="flex flex-row gap-2 align-items-center w-full justify-content-center gap-3"
      >
        <Button
          class="main-button w-3"
          :disabled="selectedFiles.length === 0 || investmentAccs.length == 0"
          label="Validate"
          @click="() => validateFile('investment_trades')"
        />
      </div>
    </div>

    <div v-if="validatedResponse">
      <div
        v-if="!transfering"
        class="flex flex-column w-full gap-3 justify-content-center align-items-center"
      >
        <span>---</span>

        <h4>Validation response</h4>
        <span class="text-sm" style="color: var(--text-secondary)"
          >General information about your import.</span
        >
        <div
          class="flex flex-row w-full gap-2 align-items-center justify-content-center"
        >
          <span>Trade count: </span>
          <span>{{ validatedResponse.filtered_count }} </span>
        </div>

        <span>---</span>

        <h4>Ticker mappings</h4>
        <div
          v-if="validatedResponse.filtered_count > 0"
          class="flex flex-row w-full gap-3 align-items-center"
        >
          <ImportTransferMapping
            v-model:model-value="investmentMappings"
            :imported-categories="validatedResponse.categories"
            :accounts="investmentAccs"
            @save="onSaveMapping"
          />
        </div>
      </div>
      <ShowLoading v-else :num-fields="5" />
    </div>
  </div>
</template>

<style scoped>
.p-fileupload {
  width: 80% !important;
}
</style>
