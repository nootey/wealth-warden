<script setup lang="ts">
import {computed, onMounted, type Ref, ref, watch} from "vue";
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
import dayjs from "dayjs";

const props = defineProps<{
    externalStep?: '1' | '2' | '3';
    externalImportId?: number | string | null;
}>();

const emit = defineEmits<{
    (e: 'completeImport'): void;
}>();

const dataStore = useDataStore();
const toastStore = useToastStore();
const accStore = useAccountStore();
const transactionStore = useTransactionStore();

const activeStep = ref<'1' | '2' | '3'>('1');

watch(
    () => props.externalStep,
    (val) => { if (val) activeStep.value = val; },
    { immediate: true }
);

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
const transfering = ref(false);
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

function removeLocalFile(index: number) {
    selectedFiles.value.splice(index, 1);
    fileValidated.value = false;
    validatedResponse.value = null;
}

const onUpload = async (nextStep: any) => {
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
        const res = await dataStore.importFromJSON(payload, selectedCheckingAcc.value?.id!);

        emit("completeImport");
        
        toastStore.successResponseToast(res);
        selectedFiles.value = [];
        fileValidated.value = false;
        validatedResponse.value = null;
        selectedCheckingAcc.value = null;

        nextStep();
    } catch (error) {
        toastStore.errorResponseToast(error);
    } finally {
        importing.value = false;
    }
};

async function validateFile(type: string, nextStep?: unknown, ) {
    if (!selectedFiles.value.length) return;

    try {
        const res = await dataStore.validateImport("custom", selectedFiles.value[0], type);
        fileValidated.value = res.valid;
        validatedResponse.value = res;
        toastHelper.formatSuccessToast("File validated", "Check details and proceed with import");

        if (typeof nextStep === 'function') {
            (nextStep as () => void)();
        }
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

function checkCheckingAccDateValidity(): boolean {

    const openedAtYear = dayjs(selectedCheckingAcc.value?.opened_at).year()
    const responseYear = validatedResponse.value?.year!

    return openedAtYear >= responseYear;

}

async function transferInvestments() {

    return;

    if (!selectedFiles.value.length) return;
    transfering.value = true;

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
        const res = await dataStore.importFromJSON(payload, selectedCheckingAcc.value?.id!);

        emit("completeImport");

        toastStore.successResponseToast(res);
        selectedFiles.value = [];
        fileValidated.value = false;
        validatedResponse.value = null;
        selectedCheckingAcc.value = null;

    } catch (error) {
        toastStore.errorResponseToast(error);
    } finally {
        transfering.value = false;
    }
}


</script>

<template>
    <div class="flex flex-column w-full gap-3 p-1">

        <Tabs value="0">
            <TabList>
                <Tab value="0">Custom</Tab>
                <Tab value="1">NLB</Tab>
            </TabList>
            <TabPanels>
                <TabPanel value="0">

                    <div class="flex flex-column w-100 gap-2">
                        <h3>About</h3>
                        <span style="color: var(--text-secondary)">Custom imports are not complete. They require a specific import format, but are not really validated. Use at your own risk. </span>

                        <h3>Create a new custom import</h3>
                        <span v-if="checkingAccs.length == 0" style="color: var(--text-secondary)">At least one checking account is required to proceed!</span>

                        <FileUpload v-if="!importing && !transfering" ref="uploadImportRef" accept=".json, application/json"
                                    :maxFileSize="10485760" :multiple="false"
                                    customUpload
                                    :showUploadButton="false" :showCancelButton="false"
                                    @select="onSelect" @clear="onClear">
                            <template #header="{ chooseCallback }" class="w-full">
                                <div class="w-full flex flex-wrap justify-content-between gap-3">
                                    <Button class="main-button" @click="chooseCallback()"
                                            :disabled="checkingAccs.length == 0 || importing || transfering"
                                            label="Upload" />
                                </div>
                            </template>

                            <template #content>

                                <div class="flex flex-column gap-2 pt-1 w-full">

                                    <div v-if="selectedFiles.length > 0">
                                        <h5>Pending</h5>
                                        <div class="flex flex-wrap gap-2 w-full">
                                            <div v-for="(file, index) in selectedFiles"
                                                 :key="file.name + file.type + file.size"
                                                 class="flex flex-row gap-3 p-2 w-full align-items-center">
                                                <span class="font-semibold text-ellipsis whitespace-nowrap overflow-hidden">{{ file.name }}</span>
                                                <Badge :value="fileValidated ? 'Validated' : 'Pending'" :severity="fileValidated ? 'info' : 'warn'" />
                                                <i class="pi pi-times hover-icon"
                                                   @click="removeLocalFile(index)"
                                                   style="color: var(--p-red-300)" />
                                            </div>
                                        </div>
                                    </div>

                                </div>
                            </template>
                        </FileUpload>
                        <ShowLoading v-else :numFields="7" />

                        <Stepper :value="activeStep">
                            <StepList>
                                <Step value="1">Validate</Step>
                                <Step value="2">Cash</Step>
                                <Step value="3">Investments</Step>
                            </StepList>
                            <StepPanels>
                                <StepPanel v-slot="{ activateCallback }" value="1">
                                    <div class="flex flex-column p-3 gap-2">
                                        <span style="color: var(--text-secondary)">
                                            Once you have uploaded a document, it needs to be validated.
                                        </span>
                                        <Button v-if="!fileValidated" class="main-button w-2"
                                                @click="() => validateFile('cash', () => activateCallback('2'))"
                                                :disabled="selectedFiles.length === 0 || checkingAccs.length == 0"
                                                label="Validate"
                                        />
                                    </div>
                                </StepPanel>
                                <StepPanel v-slot="{ activateCallback }" value="2">
                                    <div v-if="validatedResponse && !importing" class="flex flex-column gap-2 w-full p-2">

                                        <Button v-if="fileValidated" class="main-button w-2"
                                                @click="onUpload(() => activateCallback('3'))"
                                                :disabled="selectedFiles.length === 0 || checkingAccs.length == 0 || !selectedCheckingAcc || checkCheckingAccDateValidity()"
                                                label="Import"
                                        />

                                        <h4>Validation response</h4>
                                        <div class="flex flex-row w-full align-items-center gap-2">
                                            <div class="flex flex-column w-6 p-2 gap-2 align-items-center">
                                                <div class="flex flex-row gap-1 align-items-center">
                                                    <span>Year: </span>
                                                    <span>{{ validatedResponse.year }} </span>
                                                </div>
                                                <div class="flex flex-row gap-1 align-items-center">
                                                    <span>Txn count: </span>
                                                    <span>{{ validatedResponse.filtered_count }} </span>
                                                </div>
                                            </div>

                                            <div class="flex flex-column w-6 p-2 gap-2 align-items-center">
                                                <div class="flex flex-column gap-1 w-full">
                                                    <label>Checking account</label>
                                                    <AutoComplete size="small"
                                                                  v-model="selectedCheckingAcc" :suggestions="filteredCheckingAccs"
                                                                  @complete="searchAccount($event, 'checking')" optionLabel="name" forceSelection
                                                                  placeholder="Select checking account" dropdown>
                                                    </AutoComplete>
                                                </div>
                                            </div>

                                            <div class="flex flex-column w-6 p-2 gap-2 align-items-center">
                                                <div class="flex flex-column gap-1 w-full">
                                                    <label>Account status</label>
                                                    <span v-if="!selectedCheckingAcc" style="color: var(--text-secondary)">Please select an account.</span>
                                                    <span v-else-if="checkCheckingAccDateValidity()" style="color: var(--text-secondary)">Account was opened after the year of this import!</span>
                                                    <span v-else style="color: var(--text-secondary)">Account's opening date is valid.</span>
                                                </div>
                                            </div>
                                        </div>

                                        <h4>Category mappings</h4>
                                        <div class="flex flex-row w-full p-2 gap-2 align-items-center">
                                            <ImportCategoryMapping
                                                    :importedCategories="validatedResponse.categories"
                                                    :appCategories="filteredCategories"
                                                    @save="onSaveMapping"
                                            />
                                        </div>

                                    </div>
                                    <ShowLoading v-else :numFields="5" />
                                </StepPanel>
                                <StepPanel value="3">
                                    <div v-if="validatedResponse && !transfering" class="flex flex-column gap-2 w-full p-2">
                                        <h4>Validation response</h4>
                                        <div class="flex flex-row w-full align-items-center gap-2">
                                            <div class="flex flex-column w-6 p-2 gap-2">
                                                <div class="flex flex-row gap-1 align-items-center">
                                                    <span>Year: </span>
                                                    <span>{{ validatedResponse.year }} </span>
                                                </div>
                                                <div class="flex flex-row gap-1 align-items-center">
                                                    <span>Investments count: </span>
                                                    <span>{{ validatedResponse.filtered_count }} </span>
                                                </div>
                                                <div class="flex flex-row gap-1 align-items-center">
                                                    <span v-if="validatedResponse.filtered_count == 0" style="color: var(--text-secondary)">No investments were found in the provided data!</span>
                                                    <Button v-else class="main-button w-3"
                                                            @click="transferInvestments"
                                                            label="Transfer"
                                                    />
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                    <div v-else-if="!transfering" class="flex flex-column gap-2 w-full p-2">
                                        <span style="color: var(--text-secondary)">Please upload and re-validate the json file, to proceed with investment migration.</span>
                                        <Button v-if="!fileValidated" class="main-button w-2"
                                                @click="() => validateFile('investment')"
                                                :disabled="selectedFiles.length === 0 || checkingAccs.length == 0"
                                                label="Validate"
                                        />
                                    </div>
                                    <ShowLoading v-else :numFields="5" />
                                </StepPanel>
                            </StepPanels>
                        </Stepper>
                    </div>

                </TabPanel>
                <TabPanel value="1">
                    <span style="color: var(--text-secondary)">
                        NLB imports are currently unsupported!
                    </span>
                </TabPanel>
            </TabPanels>
        </Tabs>

    </div>
</template>

<style scoped>

</style>