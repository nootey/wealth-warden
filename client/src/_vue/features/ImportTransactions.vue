<script setup lang="ts">
import {computed, onMounted, type Ref, ref} from "vue";
import {useDataStore} from "../../services/stores/data_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import toastHelper from "../../utils/toast_helper.ts";
import type { CustomImportValidationResponse } from "../../models/dataio_models"
import ShowLoading from "../components/base/ShowLoading.vue";
import {useAccountStore} from "../../services/stores/account_store.ts";
import type {Account} from "../../models/account_models.ts";
import {useTransactionStore} from "../../services/stores/transaction_store.ts";
import ImportCategoryMapping from "../components/base/ImportCategoryMapping.vue";
import type {Category} from "../../models/transaction_models.ts";
import {useRouter} from "vue-router";

const emit = defineEmits<{
    (e: 'completeImport'): void;
}>();

const dataStore = useDataStore();
const toastStore = useToastStore();
const accStore = useAccountStore();
const transactionStore = useTransactionStore();

const router = useRouter();

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
    allCategories.value.filter(cat => {
        const isTopLevel = cat.parent_id == null;
        const isUncategorized = cat.name === '(uncategorized)';
        return !isTopLevel || isUncategorized;
    })
);

const categoryMappings = ref<Record<string, number | null>>({})

onMounted(async () => {
    try {
        await transactionStore.getCategories();
        await fetchSourceAccounts();
    } catch (e) {
        toastStore.errorResponseToast(e);
    }
})

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
        toastStore.infoResponseToast(
            toastHelper.formatInfoToast(
                "No accounts",
                `Please create at least one ${accountType} account`
            )
        );
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

async function validateFile(type: string) {
    if (!selectedFiles.value.length) return;

    try {
        const res = await dataStore.validateImport("custom", selectedFiles.value[0], type);
        fileValidated.value = true;
        validatedResponse.value = res;
        toastHelper.formatSuccessToast("File validated", "Check details and proceed with import");

    } catch (error) {
        toastStore.errorResponseToast(error);
    }
}

function searchAccount(event: { query: string }, accType: string) {
    const all = lists[accType].value ?? [];
    const q = event.query.trim().toLowerCase();

    filteredLists[accType].value = q
        ? all.filter(a => a.name.toLowerCase().includes(q))
        : [...all];
}

function onSaveMapping(map: Record<string, number | null>) {
    categoryMappings.value = map
}

function resetWizard() {

    if(importing.value) {
        toastStore.infoResponseToast({"Title": "Unavailable", "Message": "An operation is currently being executed!"})
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
    } catch { /* no-op */ }

}

const isDisabled = computed(() => {
    if (importing.value) return true;
    return !selectedCheckingAcc.value;
});

const importTransactions = async () => {
    if (!selectedFiles.value.length) return;
    importing.value = true;

    try {
        const form = new FormData();
        form.append("file", selectedFiles.value[0], "transactions.json");

        const categoryMappingsArray = Object.entries(categoryMappings.value).map(
            ([name, id]) => ({
                name,
                category_id: id,
            })
        );

        form.append("category_mappings", JSON.stringify(categoryMappingsArray));

        // import cash
        if (!selectedCheckingAcc.value?.id) {
            return;
        }
        const res = await dataStore.importTransactions(form, selectedCheckingAcc.value.id);
        toastStore.successResponseToast(res);

        resetWizard();
        emit("completeImport");

    } catch (error) {
        toastStore.errorResponseToast(error);
    } finally {
        importing.value = false;
    }
};

defineExpose({isDisabled, importTransactions})

</script>

<template>
  <div class="flex flex-column w-full gap-2 p-2">
    <Tabs value="0">
      <TabList>
        <Tab value="0">
          Custom
        </Tab>
        <Tab value="1">
          Bank
        </Tab>
      </TabList>
      <TabPanels>
        <TabPanel value="0">
          <div
            v-if="sourceAccounts.length > 0"
            class="flex flex-column w-full justify-content-center align-items-center gap-3"
          >
            <h3>Import your transaction data</h3>
            <span
              class="text-sm"
              style="color: var(--text-secondary)"
            >Upload your JSON file below. Please review the instructions before starting an import.</span>
            <span
              v-if="sourceAccounts.length == 0"
              style="color: var(--text-secondary)"
            >At least one checking account is required to proceed!</span>

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
                    v-if="!fileValidated"
                    class="outline-button w-3"
                    :disabled="sourceAccounts.length == 0 || importing"
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
            <ShowLoading
              v-else
              :num-fields="3"
            />

            <div
              v-if="!fileValidated"
              class="flex flex-column w-full justify-content-center align-items-center gap-3"
            >
              <span style="color: var(--text-secondary)">
                Once you have uploaded a document, it needs to be validated.
              </span>
              <div class="flex flex-row gap-2 align-items-center w-full justify-content-center gap-3">
                <Button
                  class="main-button w-3"
                  :disabled="selectedFiles.length === 0 || sourceAccounts.length == 0"
                  label="Validate"
                  @click="() => validateFile('cash')"
                />
              </div>
            </div>

            <div v-if="validatedResponse">
              <div
                v-if="!importing"
                class="flex flex-column w-full justify-content-center align-items-center gap-3"
              >
                <div class="flex flex-column w-full gap-2 align-items-center justify-content-center">
                  <div
                    class="text-sm"
                    style="color: var(--text-secondary)"
                  >
                    Select an account which will receive the import transactions.
                    <div class="flex align-items-center gap-1">
                      <Checkbox
                        v-model="useNonCheckingAccount"
                        :binary="true"
                        input-id="use-non-check-pt"
                        @update:model-value="fetchSourceAccounts"
                      />
                      <label
                        for="use-non-check-pt"
                        style="color: var(--text-secondary)"
                      >Use non checking account</label>
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
                  >Please select an account.</span>
                  <span
                    v-else
                    class="text-sm"
                    style="color: var(--text-secondary)"
                  >Account's opening date is valid.</span>
                </div>

                <span>---</span>

                <h4>Validation response</h4>
                <span
                  class="text-sm"
                  style="color: var(--text-secondary)"
                >General information about your import.</span>
                <div class="flex flex-row w-full gap-2 align-items-center justify-content-center">
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
              <ShowLoading
                v-else
                :num-fields="5"
              />
            </div>
          </div>

          <div
            v-else
            class="flex flex-column w-100 gap-2 justify-content-center align-items-center"
          >
            <i
              class="pi pi-inbox text-2xl mb-2"
              style="color: var(--text-secondary)"
            />
            <span> No data yet - create a checking
              <span
                class="hover-icon font-bold text-base"
                @click="router.push({name: 'accounts'})"
              > account </span>
              <span> to start importing. </span>
            </span>
          </div>
        </TabPanel>
        <TabPanel value="1">
          <span style="color: var(--text-secondary)">
            Bank imports are currently unsupported!
          </span>
        </TabPanel>
      </TabPanels>
    </Tabs>
  </div>
</template>

<style scoped>
.p-fileupload {
    width: 80% !important;
}
</style>