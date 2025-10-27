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

const checkingAccs = ref<Account[]>([]);
const selectedCheckingAcc = ref<Account | null>(null);
const filteredCheckingAccs = ref<Account[]>([]);

const lists: Record<string, Ref<Account[]>> = {
    checking: checkingAccs,
};

const filteredLists: Record<string, Ref<Account[]>> = {
    checking: filteredCheckingAccs,
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
        checkingAccs.value = await accStore.getAccountsBySubtype("checking");
        if (checkingAccs.value.length == 0) {
            toastStore.infoResponseToast(toastHelper.formatInfoToast("No accounts", "Please create at least one checking account"));
        }
    } catch (e) {
        toastStore.errorResponseToast(e);
    }
})

const importing = ref(false);
const uploadImportRef = ref<{ files: File[] } | null>(null);
const fileValidated = ref(false);
const validatedResponse = ref<CustomImportValidationResponse | null>(null);
const selectedFiles = ref<File[]>([]);

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

const onUpload = async (_nextStep?: any) => {

    if (!selectedFiles.value.length) return;
    importing.value = true;

    try {
        const fileText = await selectedFiles.value[0].text();
        const filePayload = JSON.parse(fileText);

        const categoryMappingsArray = Object.entries(categoryMappings.value).map(
            ([name, id]) => ({
                name,
                category_id: id,
            })
        );
        const payload = {
            ...filePayload,
            category_mappings: categoryMappingsArray,
        };

        // import cash
        const res = await dataStore.importFromJSON(payload, selectedCheckingAcc.value?.id!);
        toastStore.successResponseToast(res);

        resetWizard();
        emit("completeImport");

    } catch (error) {
        toastStore.errorResponseToast(error);
    } finally {
        importing.value = false;
    }
};

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


</script>

<template>
    <div class="flex flex-column w-full gap-2 p-2">

        <Tabs value="0">
            <TabList>
                <Tab value="0">Custom</Tab>
                <Tab value="1">Bank</Tab>
            </TabList>
            <TabPanels>
                <TabPanel value="0">

                    <div v-if="checkingAccs.length > 0" class="flex flex-column w-100 gap-3 p-1">
                        <span style="color: var(--text-secondary)">Custom imports are not complete. They require a specific import format, but are not really validated. Use at your own risk. </span>
                        <h3>Upload a JSON file</h3>
                        <span v-if="checkingAccs.length == 0" style="color: var(--text-secondary)">At least one checking account is required to proceed!</span>

                        <FileUpload v-if="!importing" ref="uploadImportRef" accept=".json, application/json"
                                    :maxFileSize="10485760" :multiple="false"
                                    customUpload
                                    :showUploadButton="false" :showCancelButton="false"
                                    @select="onSelect" @clear="onClear">

                            <template #header="{ chooseCallback }" class="w-full">
                                <div class="w-full flex flex-wrap justify-content-between gap-3">
                                    <Button class="main-button" @click="chooseCallback()"
                                            :disabled="checkingAccs.length == 0 || importing"
                                            label="Upload" />
                                </div>
                            </template>

                            <template #content>
                                <div v-if="selectedFiles.length > 0" class="flex flex-column gap-1 w-full">
                                    <h5>Pending</h5>
                                    <div class="flex flex-wrap gap-2 w-full">
                                        <div v-for="file in selectedFiles" :key="file.name + file.type + file.size"
                                             class="flex flex-row gap-2 p-1 w-full align-items-center">
                                            <span class="font-semibold text-ellipsis whitespace-nowrap overflow-hidden">{{ file.name }}</span>
                                            <Badge :value="fileValidated ? 'Validated' : 'Pending'" :severity="fileValidated ? 'info' : 'warn'" />
                                            <i class="pi pi-times hover-icon"
                                               @click="resetWizard"
                                               style="color: var(--p-red-300)" />
                                        </div>
                                    </div>
                                </div>
                            </template>

                        </FileUpload>
                        <ShowLoading v-else :numFields="3" />

                        <div class="flex flex-column p-1 gap-3">
                            <span v-if="!fileValidated" style="color: var(--text-secondary)">
                                Once you have uploaded a document, it needs to be validated.
                            </span>
                            <span v-if="fileValidated" style="color: var(--text-secondary)">
                                Start the import.
                            </span>
                            <div class="flex flex-row gap-2 align-items-center">
                                <Button v-if="!fileValidated" class="main-button w-2"
                                        @click="() => validateFile('cash')"
                                        :disabled="selectedFiles.length === 0 || checkingAccs.length == 0"
                                        label="Validate"
                                />
                                <Button v-if="fileValidated" class="main-button w-2"
                                        @click="onUpload"
                                        :disabled="selectedFiles.length === 0 || checkingAccs.length == 0 ||
                                        !selectedCheckingAcc ||
                                        importing"
                                        label="Import"
                                />
                            </div>
                        </div>

                        <div v-if="validatedResponse">
                            <div v-if="!importing" class="flex flex-column gap-3 w-full p-1">

                                <h3>Import account</h3>
                                <div class="flex flex-column w-6 gap-2 align-items-center">
                                    <span class="text-sm" style="color: var(--text-secondary)">Select an account which will receive the import transactions.</span>
                                    <div class="flex flex-column gap-1 w-full">
                                        <label>Checking account</label>
                                        <AutoComplete size="small" v-model="selectedCheckingAcc" :suggestions="filteredCheckingAccs"
                                                      @complete="searchAccount($event, 'checking')" optionLabel="name" forceSelection
                                                      placeholder="Select checking account" dropdown />
                                        <span class="text-sm" v-if="!selectedCheckingAcc" style="color: var(--text-secondary)">Please select an account.</span>
                                        <span class="text-sm" v-else style="color: var(--text-secondary)">Account's opening date is valid.</span>
                                    </div>
                                </div>

                                <h3>Validation response</h3>
                                <span class="text-sm" style="color: var(--text-secondary)">General information about your import.</span>
                                <div class="flex flex-row w-full gap-2">
                                    <span>Txn count: </span>
                                    <span>{{ validatedResponse.filtered_count }} </span>
                                </div>

                                <h3>Category mappings</h3>
                                <ImportCategoryMapping
                                        :importedCategories="validatedResponse.categories"
                                        :appCategories="filteredCategories"
                                        @save="onSaveMapping"
                                />
                            </div>
                            <ShowLoading v-else :numFields="5" />
                        </div>

                    </div>

                    <div v-else class="flex flex-column w-100 gap-2 justify-content-center align-items-center">
                        <i class="pi pi-inbox text-2xl mb-2" style="color: var(--text-secondary)"></i>
                        <span> No data yet - create a checking
                            <span class="hover-icon font-bold text-base" @click="router.push({name: 'accounts'})"> account </span>
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

</style>