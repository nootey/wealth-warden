<script setup lang="ts">
import { computed, onMounted, ref, type Ref, watch } from "vue";
import type { Account } from "../../models/account_models.ts";
import { useDataStore } from "../../services/stores/data_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { useAccountStore } from "../../services/stores/account_store.ts";
import type {
  CustomImportValidationResponse,
  Import,
} from "../../models/dataio_models.ts";
import ImportTransferMapping from "../components/base/ImportTransferMapping.vue";
import ShowLoading from "../components/base/ShowLoading.vue";

const emit = defineEmits<{
  (e: "completeTransfer"): void;
}>();

const dataStore = useDataStore();
const toastStore = useToastStore();
const accStore = useAccountStore();

const transfering = ref(false);
const checkingAccs = ref<Account[]>([]);
const selectedCheckingAcc = ref<Account | null>(null);
const filteredCheckingAccs = ref<Account[]>([]);
const savingsAccs = ref<Account[]>([]);
const savingsMappings = ref<Record<string, number | null>>({});
const validatedResponse = ref<CustomImportValidationResponse | null>(null);

const imports = ref<Import[]>([]);
const selectedImport = ref<Import | null>(null);
const loadingValidation = ref(false);

const lists: Record<string, Ref<Account[]>> = {
  checking: checkingAccs,
};

const filteredLists: Record<string, Ref<Account[]>> = {
  checking: filteredCheckingAccs,
};

watch(selectedImport, async (newImport) => {
  if (newImport && newImport.id) {
    await fetchValidationResponse(newImport.id);
  } else {
    validatedResponse.value = null;
    savingsMappings.value = {};
  }
});

onMounted(async () => {
  try {
    await getImports();

    checkingAccs.value = await accStore.getAccountsBySubtype("checking");
    if (checkingAccs.value.length == 0) {
      toastStore.infoResponseToast({
        title: "No accounts",
        message: "Please create at least one checking account",
      });
    }
    savingsAccs.value = await accStore.getAccountsBySubtype("savings");

    if (savingsAccs.value.length === 0) {
      toastStore.infoResponseToast({
        title: "No accounts",
        message: "Please create at least one savings account",
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
      "savings",
    );
    // Reset mappings when a new import is selected
    savingsMappings.value = {};
  } catch (e) {
    toastStore.errorResponseToast(e);
    validatedResponse.value = null;
  } finally {
    loadingValidation.value = false;
  }
}

async function getImports() {
  try {
    const allImports = await dataStore.getImports("custom");

    imports.value = allImports.filter(
      (importItem: any) =>
        !importItem.savings_transferred &&
        !importItem.name.toLowerCase().includes("account") &&
        !importItem.name.toLowerCase().includes("categories"),
    );
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
}

function onSaveMapping(map: Record<string, number | null>) {
  savingsMappings.value = map;
}

function searchAccount(event: { query: string }, accType: string) {
  const all = lists[accType].value ?? [];
  const q = event.query.trim().toLowerCase();

  filteredLists[accType].value = q
    ? all.filter((a) => a.name.toLowerCase().includes(q))
    : [...all];
}

function resetWizard() {
  if (transfering.value) {
    toastStore.infoResponseToast({
      title: "Unavailable",
      message: "An operation is currently being executed!",
    });
  }
  // clear local state
  selectedCheckingAcc.value = null;
  transfering.value = false;
  validatedResponse.value = null;
}

async function transferSavings() {
  if (!selectedImport.value?.id) {
    toastStore.errorResponseToast({
      title: "Error",
      message: "Missing import ID",
    });
    return;
  }

  if (Object.keys(savingsMappings.value).length === 0) {
    toastStore.errorResponseToast({
      title: "Error",
      message: "Please set up your savings mappings first",
    });
    return;
  }

  transfering.value = true;

  if (!selectedCheckingAcc.value?.id) {
    toastStore.errorResponseToast({
      title: "Error",
      message: "No checking account",
    });
    return;
  }

  try {
    const payload = {
      import_id: selectedImport.value.id,
      checking_acc_id: selectedCheckingAcc.value.id,
      savings_mappings: Object.entries(savingsMappings.value).map(
        ([name, account_id]) => ({ name, account_id }),
      ),
    };

    const res = await dataStore.transferSavingsFromImport(payload);
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
  if (!selectedCheckingAcc.value) return true;

  const mappings = Object.values(savingsMappings.value);
  const hasAtLeastOne = mappings.some((v) => v !== null);
  return !hasAtLeastOne;
});

defineExpose({ isDisabled, transferSavings });
</script>

<template>
  <div
    v-if="!transfering"
    class="flex flex-column w-full justify-content-center align-items-center gap-3"
  >
    <h3>Map savings from imported data</h3>
    <span>Select import</span>

    <Select
      v-model="selectedImport"
      size="small"
      style="width: 450px"
      :options="imports"
      option-label="name"
      placeholder="Select import"
    />
    <div v-if="loadingValidation" class="flex flex-column w-full p-2">
      <ShowLoading :num-fields="5" />
    </div>
    <div
      v-else-if="validatedResponse && validatedResponse.filtered_count == 0"
      class="flex flex-column w-full p-2 align-items-center"
    >
      <span style="color: var(--text-secondary)"
        >No savings were found in the provided import!</span
      >
    </div>
    <div v-else-if="validatedResponse">
      <div
        class="flex flex-column w-full gap-3 justify-content-center align-items-center"
      >
        <div
          class="flex flex-column w-full gap-2 align-items-center justify-content-center"
        >
          <span class="text-sm" style="color: var(--text-secondary)"
            >Select an account which will receive the import transactions.</span
          >
          <AutoComplete
            v-model="selectedCheckingAcc"
            size="small"
            :suggestions="filteredCheckingAccs"
            option-label="name"
            force-selection
            placeholder="Select checking account"
            dropdown
            @complete="searchAccount($event, 'checking')"
          />
          <span
            v-if="!selectedCheckingAcc"
            class="text-sm"
            style="color: var(--text-secondary)"
            >Please select an account.</span
          >
          <span v-else class="text-sm" style="color: var(--text-secondary)"
            >Account's opening date is valid.</span
          >
        </div>

        <span>---</span>

        <h4>Validation response</h4>
        <span class="text-sm" style="color: var(--text-secondary)"
          >General information about your import.</span
        >
        <div
          class="flex flex-row w-full gap-2 align-items-center justify-content-center"
        >
          <span>Transfer count: </span>
          <span>{{ validatedResponse.filtered_count }} </span>
        </div>

        <span>---</span>

        <h4>Savings mappings</h4>
        <div
          v-if="validatedResponse.filtered_count > 0"
          class="flex flex-row w-full gap-3 align-items-center"
        >
          <ImportTransferMapping
            v-model:model-value="savingsMappings"
            :imported-categories="validatedResponse.categories"
            :accounts="savingsAccs"
            @save="onSaveMapping"
          />
        </div>
      </div>
    </div>
    <div
      v-else-if="!selectedImport"
      class="flex flex-column w-full p-2 w-full align-items-center"
    >
      <span style="color: var(--text-secondary)"
        >Please select an import to begin.</span
      >
    </div>
  </div>
  <ShowLoading v-else :num-fields="5" />
</template>

<style scoped></style>
