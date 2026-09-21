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
import UploadDropzone from "../../components/base/UploadDropzone.vue";
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
    class="flex flex-col w-full justify-center items-center text-center gap-4"
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

    <UploadDropzone
      v-if="!transfering"
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

    <div
      v-if="!fileValidated"
      class="flex flex-col w-full justify-center items-center gap-4"
    >
      <span style="color: var(--text-secondary)">
        Once you have uploaded a document, it needs to be validated.
      </span>
      <div class="flex flex-row gap-2 items-center w-full justify-center gap-4">
        <Button
          class="main-button w-3/12"
          :disabled="selectedFiles.length === 0 || investmentAccs.length == 0"
          label="Validate"
          @click="() => validateFile('investment_trades')"
        />
      </div>
    </div>

    <div v-if="validatedResponse">
      <div
        v-if="!transfering"
        class="flex flex-col w-full gap-4 justify-center items-center"
      >
        <span>---</span>

        <h4>Validation response</h4>
        <span class="text-sm" style="color: var(--text-secondary)"
          >General information about your import.</span
        >
        <div class="flex flex-row w-full gap-2 items-center justify-center">
          <span>Trade count: </span>
          <span>{{ validatedResponse.filtered_count }} </span>
        </div>

        <span>---</span>

        <h4>Ticker mappings</h4>
        <div
          v-if="validatedResponse.filtered_count > 0"
          class="flex flex-row w-full gap-4 items-center"
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
