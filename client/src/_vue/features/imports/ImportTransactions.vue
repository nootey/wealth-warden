<script setup lang="ts">
import { computed, onMounted, type Ref, ref, watch } from "vue";
import { useDataStore } from "../../../services/stores/data_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import type {
  BankTxn,
  CustomImportValidationResponse,
} from "../../../models/dataio_models.ts";
import ShowLoading from "../../components/base/ShowLoading.vue";
import { useAccountStore } from "../../../services/stores/account_store.ts";
import type { Account } from "../../../models/account_models.ts";
import { useTransactionStore } from "../../../services/stores/transaction_store.ts";
import ImportCategoryMapping from "../../components/base/ImportCategoryMapping.vue";
import type { Category } from "../../../models/transaction_models.ts";
import { useRouter } from "vue-router";
import searchHelper from "../../../utils/search_helper.ts";
import Select from "primevue/select";
import RulesManager from "./RulesManager.vue";
import { useChartColors } from "../../../style/theme/chartColors.ts";

const emit = defineEmits<{
  (e: "completeImport"): void;
}>();

const dataStore = useDataStore();
const toastStore = useToastStore();
const accStore = useAccountStore();
const transactionStore = useTransactionStore();

const router = useRouter();

const { colors } = useChartColors();

const sourceAccounts = ref<Account[]>([]);
const selectedCheckingAcc = ref<Account | null>(null);
const filteredSourceAccounts = ref<Account[]>([]);

const lists: Record<string, Ref<Account[]>> = {
  source: sourceAccounts,
};

const filteredLists: Record<string, Ref<Account[]>> = {
  source: filteredSourceAccounts,
};

const allCategories = computed<Category[]>(() => transactionStore.categories);
const filteredCategories = computed(() =>
  allCategories.value.filter((cat) => {
    const isTopLevel = cat.parent_id == null;
    const isUncategorized = cat.name === "(uncategorized)";
    return !isTopLevel || isUncategorized;
  }),
);

const categoryMappings = ref<Record<string, number | null>>({});

const categoryOptions = computed(() =>
  filteredCategories.value.map((c) => ({
    label: c.display_name || c.name,
    value: c.id,
    classification: c.classification,
  })),
);

onMounted(async () => {
  try {
    await transactionStore.getCategories();
    await fetchSourceAccounts();
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
});

const activeTab = ref("0");
const importing = ref(false);
const uploadImportRef = ref<{ files: File[] } | null>(null);
const fileValidated = ref(false);
const validatedResponse = ref<CustomImportValidationResponse | null>(null);
const selectedFiles = ref<File[]>([]);

const useNonCheckingAccount = ref(false);

async function fetchSourceAccounts() {
  const accounts = useNonCheckingAccount.value
    ? await accStore.getAllAccounts(true)
    : await accStore.getAccountsBySubtype("checking");

  // Ensure we always have an array
  sourceAccounts.value = accounts ?? [];

  if (sourceAccounts.value.length === 0) {
    const accountType = useNonCheckingAccount.value ? "source" : "checking";
    toastStore.infoResponseToast({
      title: "No accounts",
      message: `Please create at least one ${accountType} account`,
    });
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

const bankUploadRef = ref<{ files: File[] } | null>(null);
const bankFiles = ref<File[]>([]);

function onBankSelect(e: { files: File[] }) {
  bankFiles.value = e.files;
  clearBankParse();
}

function onBankRemove(e: { files: File[] }) {
  bankFiles.value = e.files;
  clearBankParse();
}

const bankParsing = ref(false);
type BankRow = BankTxn & { row: number };
const bankTxns = ref<BankRow[]>([]);
const bankSelected = ref<BankRow[]>([]);
const bankRowCategories = ref<Record<number, number | null>>({});

// A row is categorized once it has a category (rule guess or manual choice), else uncategorized.
function rowCategorized(row: number): boolean {
  return bankRowCategories.value[row] != null;
}
const uncategorizedRows = computed(() =>
  bankTxns.value.filter((t) => !rowCategorized(t.row)),
);
const categorizedRows = computed(() =>
  bankTxns.value.filter((t) => rowCategorized(t.row)),
);
// Each panel drives its own checkboxes, but bankSelected stays the union sent on import.
const uncategorizedSelected = computed<BankRow[]>({
  get: () => bankSelected.value.filter((t) => !rowCategorized(t.row)),
  set: (rows) => {
    bankSelected.value = [
      ...bankSelected.value.filter((t) => rowCategorized(t.row)),
      ...rows,
    ];
  },
});
const categorizedSelected = computed<BankRow[]>({
  get: () => bankSelected.value.filter((t) => rowCategorized(t.row)),
  set: (rows) => {
    bankSelected.value = [
      ...bankSelected.value.filter((t) => !rowCategorized(t.row)),
      ...rows,
    ];
  },
});

const bankManualRows = ref<Set<number>>(new Set());

function setBankRowCategory(row: number, value: number | null) {
  bankRowCategories.value[row] = value;
  bankManualRows.value.add(row);
}

async function refreshBankGuesses() {
  if (bankTxns.value.length === 0) return;
  try {
    const res = await dataStore.applyBankRules(bankTxns.value);
    res.transactions.forEach((t, row) => {
      if (bankManualRows.value.has(row)) return;
      bankRowCategories.value[row] = t.category_id ?? null;
    });
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

function clearBankParse() {
  bankTxns.value = [];
  bankSelected.value = [];
  bankRowCategories.value = {};
  bankManualRows.value.clear();
}

function bankFormData(): FormData {
  const formData = new FormData();
  for (const file of bankFiles.value) {
    formData.append("files", file);
  }
  formData.append("bank", "auto");
  return formData;
}

async function parseBankStatement() {
  if (bankFiles.value.length === 0) return;

  bankParsing.value = true;
  try {
    const formData = bankFormData();
    const res = await dataStore.parseBankStatement(formData);
    bankTxns.value = res.transactions.map((t, row) => ({ ...t, row }));
    bankSelected.value = [...bankTxns.value];
    // Seed the per-row category with the rule-based guess from the parse. The user
    // then keeps, overrides, or clears it; the kept value is sent on import.
    const guesses: Record<number, number | null> = {};
    for (const t of bankTxns.value) {
      if (t.category_id != null) guesses[t.row] = t.category_id;
    }
    bankRowCategories.value = guesses;
    bankManualRows.value.clear();
    void checkBankDuplicates();
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    bankParsing.value = false;
  }
}

async function checkBankDuplicates() {
  const accId = selectedCheckingAcc.value?.id;
  if (accId == null || bankTxns.value.length === 0) return;
  try {
    const res = await dataStore.checkBankDuplicates(bankTxns.value, accId);
    const flags = res.transactions;
    bankTxns.value = bankTxns.value.map((t, i) => ({
      ...t,
      partial_match: flags[i]?.partial_match ?? null,
    }));
    const flaggedRows = new Set(
      bankTxns.value.filter((t) => t.partial_match).map((t) => t.row),
    );
    if (flaggedRows.size > 0) {
      bankSelected.value = bankSelected.value.filter(
        (t) => !flaggedRows.has(t.row),
      );
    }
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

// Re-check whenever the target account changes; matches are account-scoped.
watch(
  () => selectedCheckingAcc.value?.id,
  (id) => {
    if (id != null) void checkBankDuplicates();
  },
);

function partialMatchTooltip(row: BankRow): string {
  const pm = row.partial_match;
  if (!pm) return "";
  const desc = pm.existing_description?.trim() || "(no description)";
  const capped = desc.length > 80 ? `${desc.slice(0, 80)}…` : desc;
  const category = pm.existing_category?.trim() || "(uncategorized)";
  return `Existing transaction - Date: ${pm.existing_date}; Category: ${category}; Description: ${capped}`;
}

function onBankClear() {
  bankFiles.value = [];
  clearBankParse();
  try {
    (bankUploadRef.value as any)?.clear?.();
  } catch {
    /* no-op */
  }
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

function searchAccount(event: { query: string }, accType: string) {
  const listRef = lists[accType];
  const filteredListRef = filteredLists[accType];

  if (!listRef || !filteredListRef) return;

  const all = listRef.value ?? [];

  filteredListRef.value = searchHelper.filterByQuery(all, event.query, (a) => [
    a.name,
  ]);
}

function onSaveMapping(map: Record<string, number | null>) {
  categoryMappings.value = map;
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
  fileValidated.value = false;
  validatedResponse.value = null;
  selectedCheckingAcc.value = null;
  categoryMappings.value = {};
  importing.value = false;

  // clear FileUpload UI
  try {
    (uploadImportRef.value as any)?.clear?.();
  } catch {
    /* no-op */
  }
  onBankClear();
}

const isDisabled = computed(() => {
  if (importing.value) return true;
  if (activeTab.value === "0") {
    return bankSelected.value.length === 0 || !selectedCheckingAcc.value;
  }
  return !selectedCheckingAcc.value;
});

const importBankTransactions = async () => {
  if (bankFiles.value.length === 0 || !selectedCheckingAcc.value?.id) return;

  importing.value = true;
  try {
    const form = bankFormData();
    const rowCategories = Object.entries(bankRowCategories.value)
      .filter(([, id]) => id != null)
      .map(([row, id]) => ({ row: Number(row), category_id: id }));
    form.append("row_categories", JSON.stringify(rowCategories));
    const keep = new Set(bankSelected.value.map((t) => t.row));
    const skipRows = bankTxns.value
      .filter((t) => !keep.has(t.row))
      .map((t) => t.row);
    form.append("skip_rows", JSON.stringify(skipRows));
    const res = await dataStore.importBankTransactions(
      form,
      selectedCheckingAcc.value.id,
    );
    toastStore.successResponseToast(res);
    emit("completeImport");
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    importing.value = false;
    resetWizard();
  }
};

const importTransactions = async () => {
  if (activeTab.value === "0") {
    await importBankTransactions();
    return;
  }
  if (selectedFiles.value.length < 1) return;

  const file = selectedFiles.value[0];
  if (!file) return;

  importing.value = true;

  try {
    const form = new FormData();
    form.append("file", file, "transactions.json");

    const categoryMappingsArray = Object.entries(categoryMappings.value).map(
      ([name, id]) => ({
        name,
        category_id: id,
      }),
    );

    form.append("category_mappings", JSON.stringify(categoryMappingsArray));

    // import cash
    if (!selectedCheckingAcc.value?.id) {
      return;
    }
    const res = await dataStore.importTransactions(
      form,
      selectedCheckingAcc.value.id,
    );
    toastStore.successResponseToast(res);

    emit("completeImport");
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    importing.value = false;
    resetWizard();
  }
};

defineExpose({ isDisabled, importing, importTransactions });
</script>

<template>
  <div class="flex flex-col w-full gap-2 p-2">
    <div v-if="importing" class="flex flex-col items-center gap-4 py-6">
      <span class="font-semibold text-center"
        >Importing transactions... please do not close this window.</span
      >
      <ShowLoading :num-fields="5" />
    </div>
    <Tabs v-else v-model:value="activeTab">
      <TabList>
        <Tab value="0"> Bank </Tab>
        <Tab value="1"> Custom </Tab>
      </TabList>
      <TabPanels class="w-full">
        <TabPanel value="0">
          <div class="flex flex-col w-full gap-4">
            <h3>Import your bank statements</h3>
            <span class="text-sm w-full" style="color: var(--text-secondary)"
              >Import monthly statements from your bank. The transactions will
              be extracted automatically. Use one kind per import, since PDF
              rows cannot be deduplicated against CSV rows.</span
            >

            <div class="flex flex-col w-full gap-3">
              <h3>1. Select an account</h3>
              <span class="text-sm" style="color: var(--text-secondary)">
                Select an account which will receive the import transactions.
                The import will assume the transactions you're trying to import,
                are in the selected accounts currency.
              </span>
              <AutoComplete
                v-model="selectedCheckingAcc"
                size="small"
                :suggestions="filteredSourceAccounts"
                option-label="name"
                force-selection
                placeholder="Select checking account"
                dropdown
                class="w-full sm:w-1/4"
                @complete="searchAccount($event, 'source')"
              />
              <div
                class="flex items-center gap-1 text-sm"
                style="color: var(--text-secondary)"
              >
                <Checkbox
                  v-model="useNonCheckingAccount"
                  :binary="true"
                  input-id="use-non-check-bank"
                  @update:model-value="fetchSourceAccounts"
                />
                <label
                  for="use-non-check-bank"
                  style="color: var(--text-secondary)"
                  >Use non checking account</label
                >
              </div>
            </div>

            <div class="flex flex-col w-full gap-3">
              <h3>2. Upload your statements</h3>
              <span class="text-sm" style="color: var(--text-secondary)">
                Upload one or multiple bank statements, from which transactions
                will be parsed. If you have defined any rules, they will be used
                to match categories to the parsed transactions.
              </span>
              <FileUpload
                v-if="selectedCheckingAcc"
                ref="bankUploadRef"
                accept=".pdf, .csv, application/pdf, text/csv"
                :max-file-size="10485760"
                :multiple="true"
                custom-upload
                :show-upload-button="false"
                :show-cancel-button="false"
                @select="onBankSelect"
                @remove="onBankRemove"
                @clear="onBankClear"
              >
                <template #header="{ chooseCallback }">
                  <div class="flex flex-col gap-3 w-full">
                    <div
                      class="flex flex-row w-full gap-2 items-center p-3 rounded-lg text-xs"
                      style="
                        background: var(--background-secondary);
                        border: 1px solid var(--border-color);
                        color: var(--text-secondary);
                      "
                    >
                      <i class="pi pi-info-circle" style="flex-shrink: 0" />
                      <span>
                        Only NLB bank statements are tested. CSV import expects
                        these columns: amount, +/-, value date, description.
                        Other banks or formats may fail to import or import
                        incorrectly.
                      </span>
                    </div>

                    <div class="flex flex-col items-center gap-2 py-4">
                      <i
                        class="pi pi-cloud-upload text-3xl"
                        style="color: var(--text-secondary)"
                      />
                      <span class="text-sm" style="color: var(--text-secondary)"
                        >Drag files here, or</span
                      >
                      <Button
                        class="outline-button"
                        label="Upload"
                        @click="chooseCallback()"
                      />
                      <span class="text-xs" style="color: var(--text-secondary)"
                        >Accepts .pdf and .csv</span
                      >
                    </div>
                  </div>
                </template>

                <template #content="{ removeFileCallback }">
                  <div
                    v-if="bankFiles.length > 0"
                    class="flex flex-col gap-2 w-full"
                  >
                    <h5>Pending</h5>
                    <div class="flex flex-col gap-2 w-full">
                      <div
                        v-for="(file, index) in bankFiles"
                        :key="file.name + file.type + file.size"
                        class="flex flex-row gap-2 items-center justify-between p-2 rounded-lg"
                        style="border: 1px solid var(--border-color)"
                      >
                        <div class="flex flex-row gap-2 items-center min-w-0">
                          <i
                            class="pi pi-file"
                            style="color: var(--text-secondary); flex-shrink: 0"
                          />
                          <span
                            class="font-semibold text-ellipsis whitespace-nowrap overflow-hidden"
                            >{{ file.name }}</span
                          >
                        </div>
                        <div
                          class="flex flex-row gap-2 items-center"
                          style="flex-shrink: 0"
                        >
                          <Badge
                            :value="bankTxns.length > 0 ? 'Parsed' : 'Pending'"
                            :severity="bankTxns.length > 0 ? 'info' : 'warn'"
                          />
                          <i
                            class="pi pi-times hover-icon"
                            style="color: var(--p-red-300)"
                            @click="removeFileCallback(index)"
                          />
                        </div>
                      </div>
                    </div>
                    <Button
                      v-if="bankTxns.length === 0"
                      class="outline-button self-center mt-1"
                      label="Parse"
                      :loading="bankParsing"
                      @click="parseBankStatement"
                    />
                  </div>
                </template>
              </FileUpload>
            </div>

            <div v-if="bankTxns.length > 0" class="flex flex-col w-full gap-3">
              <h3>3. Categorize</h3>
              <span class="text-sm" style="color: var(--text-secondary)">
                Upload one or multiple bank statements, from which transactions
                will be parsed. Existing rules will be used to match categories
                to the parsed transactions, but you can create more on the fly.
              </span>

              <h3>Parsed transactions</h3>

              <div
                class="flex flex-col sm:flex-row w-full gap-2 sm:justify-between sm:items-start"
              >
                <span
                  class="text-sm sm:w-10/12"
                  style="color: var(--text-secondary)"
                >
                  Rows with a category sit on the right, uncategorized rows on
                  the left. Change or clear a row's category to move it between
                  the groups. Uncheck a row to leave it out of the import. Each
                  row receives an external transaction id, which is used for
                  de-duplication. Some monthly statements do not include it.
                </span>
                <RulesManager @changed="refreshBankGuesses" />
              </div>

              <div class="flex flex-col xl:flex-row w-full gap-4">
                <div class="flex flex-col w-full min-w-0 gap-2">
                  <div class="flex items-center gap-2">
                    <h5 class="m-0">Uncategorized</h5>
                    <Tag
                      :value="String(uncategorizedRows.length)"
                      severity="warn"
                    />
                  </div>
                  <DataTable
                    v-model:selection="uncategorizedSelected"
                    class="w-full enhanced-table bank-table"
                    :value="uncategorizedRows"
                    data-key="row"
                    size="small"
                    scrollable
                    scroll-height="40vh"
                  >
                    <Column
                      selection-mode="multiple"
                      header-style="width: 3rem"
                    />
                    <Column field="txn_date" header="Date">
                      <template #body="{ data }">
                        <span class="inline-flex items-center gap-1">
                          {{ data.txn_date.slice(0, 10) }}
                          <i
                            v-if="data.partial_match"
                            v-tooltip.top="partialMatchTooltip(data)"
                            class="pi pi-exclamation-triangle text-yellow-500"
                          />
                        </span>
                      </template>
                    </Column>
                    <Column field="transaction_type" header="Direction">
                      <template #body="{ data }">
                        <span
                          class="capitalize"
                          :style="{
                            color:
                              data.transaction_type === 'expense'
                                ? colors.neg
                                : colors.pos,
                          }"
                        >
                          {{ data.transaction_type }}
                        </span>
                      </template>
                    </Column>
                    <Column field="amount" header="Amount" />
                    <Column field="description" header="Description">
                      <template #body="{ data }">
                        <span
                          v-tooltip="data.description"
                          class="truncate-text"
                          style="max-width: 200px"
                        >
                          {{ data.description }}
                        </span>
                      </template>
                    </Column>
                    <Column header="Category">
                      <template #body="{ data }">
                        <Select
                          class="w-full"
                          size="small"
                          :model-value="bankRowCategories[data.row] ?? null"
                          :options="categoryOptions"
                          option-label="label"
                          option-value="value"
                          show-clear
                          filter
                          placeholder="Auto (rules)"
                          @update:model-value="
                            setBankRowCategory(data.row, $event)
                          "
                        >
                          <template #option="{ option }">
                            <div class="flex justify-between w-full gap-2">
                              <span>{{ option.label }}</span>
                              <small class="text-muted-color">
                                {{ option.classification }}
                              </small>
                            </div>
                          </template>
                        </Select>
                      </template>
                    </Column>
                  </DataTable>
                </div>

                <div class="flex flex-col w-full min-w-0 gap-2">
                  <div class="flex items-center gap-2">
                    <h5 class="m-0">Categorized</h5>
                    <Tag
                      :value="String(categorizedRows.length)"
                      severity="success"
                    />
                  </div>
                  <DataTable
                    v-model:selection="categorizedSelected"
                    class="w-full enhanced-table bank-table"
                    :value="categorizedRows"
                    data-key="row"
                    size="small"
                    scrollable
                    scroll-height="40vh"
                  >
                    <Column
                      selection-mode="multiple"
                      header-style="width: 3rem"
                    />
                    <Column field="txn_date" header="Date">
                      <template #body="{ data }">
                        <span class="inline-flex items-center gap-1">
                          {{ data.txn_date.slice(0, 10) }}
                          <i
                            v-if="data.partial_match"
                            v-tooltip.top="partialMatchTooltip(data)"
                            class="pi pi-exclamation-triangle text-yellow-500"
                          />
                        </span>
                      </template>
                    </Column>
                    <Column field="transaction_type" header="Direction">
                      <template #body="{ data }">
                        <span
                          class="capitalize"
                          :style="{
                            color:
                              data.transaction_type === 'expense'
                                ? colors.neg
                                : colors.pos,
                          }"
                        >
                          {{ data.transaction_type }}
                        </span>
                      </template>
                    </Column>
                    <Column field="amount" header="Amount" />
                    <Column field="description" header="Description">
                      <template #body="{ data }">
                        <span
                          v-tooltip="data.description"
                          class="truncate-text"
                          style="max-width: 200px"
                        >
                          {{ data.description }}
                        </span>
                      </template>
                    </Column>
                    <Column header="Category">
                      <template #body="{ data }">
                        <Select
                          class="w-full"
                          size="small"
                          :model-value="bankRowCategories[data.row] ?? null"
                          :options="categoryOptions"
                          option-label="label"
                          option-value="value"
                          show-clear
                          filter
                          placeholder="Auto (rules)"
                          @update:model-value="
                            setBankRowCategory(data.row, $event)
                          "
                        >
                          <template #option="{ option }">
                            <div class="flex justify-between w-full gap-2">
                              <span>{{ option.label }}</span>
                              <small class="text-muted-color">
                                {{ option.classification }}
                              </small>
                            </div>
                          </template>
                        </Select>
                      </template>
                    </Column>
                  </DataTable>
                </div>
              </div>
            </div>
          </div>
        </TabPanel>
        <TabPanel value="1">
          <div
            v-if="sourceAccounts.length > 0"
            class="flex flex-col w-full justify-center items-center gap-4"
          >
            <h3>Import your transaction data</h3>
            <span class="text-sm" style="color: var(--text-secondary)"
              >Upload your JSON file below. Please review the instructions
              before starting an import.</span
            >
            <span
              v-if="sourceAccounts.length == 0"
              style="color: var(--text-secondary)"
              >At least one checking account is required to proceed!</span
            >

            <FileUpload
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
                <div class="w-full flex flex-row justify-center">
                  <Button
                    v-if="!fileValidated"
                    class="outline-button w-3/12"
                    :disabled="sourceAccounts.length == 0"
                    label="Upload"
                    @click="chooseCallback()"
                  />
                </div>
              </template>

              <template #content>
                <div
                  v-if="selectedFiles.length > 0"
                  class="flex flex-col gap-1 w-full items-center"
                >
                  <h5>Pending</h5>
                  <div class="flex flex-wrap gap-2 w-full">
                    <div
                      v-for="file in selectedFiles"
                      :key="file.name + file.type + file.size"
                      class="flex flex-row gap-2 p-1 w-full justify-center items-center w-full"
                    >
                      <span
                        class="font-semibold text-ellipsis whitespace-nowrap overflow-hidden"
                        >{{ file.name }}</span
                      >
                      <Badge
                        :value="fileValidated ? 'Validated' : 'Pending'"
                        :severity="fileValidated ? 'info' : 'warn'"
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

            <div
              v-if="!fileValidated"
              class="flex flex-col w-full justify-center items-center gap-4"
            >
              <span style="color: var(--text-secondary)">
                Once you have uploaded a document, it needs to be validated.
              </span>
              <div
                class="flex flex-row gap-2 items-center w-full justify-center gap-4"
              >
                <Button
                  class="main-button w-3/12"
                  :disabled="
                    selectedFiles.length === 0 || sourceAccounts.length == 0
                  "
                  label="Validate"
                  @click="() => validateFile('cash')"
                />
              </div>
            </div>

            <div
              v-if="validatedResponse"
              class="flex flex-col w-full justify-center items-center gap-4"
            >
              <div
                class="flex flex-col w-full gap-2 items-center justify-center"
              >
                <div class="text-sm" style="color: var(--text-secondary)">
                  Select an account which will receive the import transactions.
                  <div class="flex items-center gap-1">
                    <Checkbox
                      v-model="useNonCheckingAccount"
                      :binary="true"
                      input-id="use-non-check-pt"
                      @update:model-value="fetchSourceAccounts"
                    />
                    <label
                      for="use-non-check-pt"
                      style="color: var(--text-secondary)"
                      >Use non checking account</label
                    >
                  </div>
                </div>
                <AutoComplete
                  v-model="selectedCheckingAcc"
                  size="small"
                  :suggestions="filteredSourceAccounts"
                  option-label="name"
                  force-selection
                  placeholder="Select checking account"
                  dropdown
                  @complete="searchAccount($event, 'source')"
                />
                <span
                  v-if="!selectedCheckingAcc"
                  class="text-sm"
                  style="color: var(--text-secondary)"
                  >Please select an account.</span
                >
                <span
                  v-else
                  class="text-sm"
                  style="color: var(--text-secondary)"
                  >Account's opening date is valid.</span
                >
              </div>

              <span>---</span>

              <h4>Validation response</h4>
              <span class="text-sm" style="color: var(--text-secondary)"
                >General information about your import.</span
              >
              <div
                class="flex flex-row w-full gap-2 items-center justify-center"
              >
                <span>Txn count: </span>
                <span>{{ validatedResponse.filtered_count }} </span>
              </div>

              <span>---</span>

              <h4>Category mappings</h4>
              <ImportCategoryMapping
                :imported-categories="validatedResponse.categories"
                :app-categories="filteredCategories"
                @save="onSaveMapping"
              />
            </div>
          </div>

          <div
            v-else
            class="flex flex-col w-100 gap-2 justify-center items-center"
          >
            <i
              class="pi pi-inbox text-2xl mb-2"
              style="color: var(--text-secondary)"
            />
            <span>
              No data yet - create a checking
              <span
                class="hover-icon font-bold text-base"
                @click="router.push({ name: 'accounts' })"
              >
                account
              </span>
              <span> to start importing. </span>
            </span>
          </div>
        </TabPanel>
      </TabPanels>
    </Tabs>
  </div>
</template>

<style scoped>
.bank-table :deep(td),
.bank-table :deep(th) {
  padding: 0.25rem 0.5rem;
  font-size: 0.8rem;
}

.bank-table :deep(.p-select) {
  font-size: 0.8rem;
}
</style>
