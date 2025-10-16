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
import {useRouter} from "vue-router";
import ImportInvestmentMapping from "../components/base/ImportInvestmentMapping.vue";

const props = defineProps<{
    externalStep?: '1' | '2' | '3';
    externalImportId?: number | string | null;
}>();

const emit = defineEmits<{
    (e: 'completeImport'): void;
    (e: 'resetExternal'): void;
}>();

const dataStore = useDataStore();
const toastStore = useToastStore();
const accStore = useAccountStore();
const transactionStore = useTransactionStore();

const router = useRouter();
const activeStep = ref<'1' | '2' | '3'>('1');

watch(
    () => props.externalStep,
    async (val) => {
        if (val) {
            activeStep.value = val;

            if (val === '3' && props.externalImportId) {
                await loadExistingImport(props.externalImportId);
            }
        }
    },
    { immediate: true }
);

watch(
    () => props.externalImportId,
    async (id, _old) => {
        if (!id) return;

        selectedFiles.value = [];
        fileValidated.value = false;
        validatedResponse.value = null;
        selectedCheckingAcc.value = null;
        categoryMappings.value = {};

        await loadExistingImport(id);
        activeStep.value = '3';
    }
);

const checkingAccs = ref<Account[]>([]);
const investmentAccs = ref<Account[]>([]);
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
const investmentMappings = ref<Record<string, number | null>>({})

onMounted(async () => {
    try {
        await transactionStore.getCategories();
        checkingAccs.value = await accStore.getAccountsBySubtype("checking");
        if (checkingAccs.value.length == 0) {
            toastStore.infoResponseToast(toastHelper.formatInfoToast("No accounts", "Please create at least one checking account"));
        }
        const [investments, crypto] = await Promise.all([
            accStore.getAccountsByType("investment"),
            accStore.getAccountsByType("crypto")
        ])

        // merge and remove duplicates
        const merged = [...investments, ...crypto]
        investmentAccs.value = merged.filter(
            (a, i, arr) => arr.findIndex(b => b.id === a.id) === i
        )

        if (investmentAccs.value.length === 0) {
            toastStore.infoResponseToast(
                toastHelper.formatInfoToast("No accounts", "Please create at least one investment or crypto account")
            )
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

const onUpload = async (_nextStep?: any) => {

    if (!selectedFiles.value.length) return;
    importing.value = true;

    try {
        const file = selectedFiles.value[0];
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
        emit("completeImport");
        toastStore.successResponseToast(res);

        // validate investments using the same file
        const invRes = await dataStore.validateImport("custom", file, "investment");
        validatedResponse.value = invRes;
        fileValidated.value = invRes.valid;

        // advance to step 3
        activeStep.value = '3';

        // clear local selections
        selectedFiles.value = [];
        selectedCheckingAcc.value = null;

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

        if (res.valid) activeStep.value = '2';
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

function onSaveInvestmentMapping(map: Record<string, number | null>) {
    investmentMappings.value = map
}

function checkCheckingAccDateValidity(): boolean {

    const openedAtYear = dayjs(selectedCheckingAcc.value?.opened_at).year()
    const responseYear = validatedResponse.value?.year!

    return openedAtYear >= responseYear;

}

async function loadExistingImport(importId: number | string) {
    try {
        transfering.value = true;
        const res = await dataStore.getCustomImportJSON(importId, 'investments');

        validatedResponse.value = res;
        fileValidated.value = res.valid;
    } catch (e) {
        toastStore.errorResponseToast(e);
    } finally {
        transfering.value = false;
    }
}

function resetWizard() {
    // clear local state
    selectedFiles.value = [];
    fileValidated.value = false;
    validatedResponse.value = null;
    selectedCheckingAcc.value = null;
    categoryMappings.value = {};
    importing.value = false;
    transfering.value = false;
    activeStep.value = '1';

    // clear FileUpload UI
    try {
        (uploadImportRef.value as any)?.clear?.();
    } catch { /* no-op */ }

    // tell parent to clear externalStep/externalImportId
    emit('resetExternal');
}

async function transferInvestments() {

    if (!props.externalImportId) {
        toastStore.errorResponseToast("Missing import ID")
        return
    }

    if (Object.keys(investmentMappings.value).length === 0) {
        toastStore.errorResponseToast("Please set up your investment mappings first")
        return
    }

    transfering.value = true

    try {
        const payload = {
            import_id: props.externalImportId,
            investment_mappings: Object.entries(investmentMappings.value).map(
                ([name, account_id]) => ({ name, account_id })
            ),
        }

        const res = await dataStore.transferInvestmentsFromImport(payload)

        toastStore.successResponseToast(res)
        emit("completeImport")
    } catch (error) {
        toastStore.errorResponseToast(error)
    } finally {
        transfering.value = false
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

                    <div v-if="checkingAccs.length > 0" class="flex flex-column w-100 gap-2">
                        <h3>About</h3>
                        <span style="color: var(--text-secondary)">Custom imports are not complete. They require a specific import format, but are not really validated. Use at your own risk. </span>

                        <h4>Create a new import</h4>
                        <span v-if="checkingAccs.length == 0" style="color: var(--text-secondary)">At least one checking account is required to proceed!</span>

                        <div v-if="activeStep !== '3'">
                            <FileUpload v-if="!importing && !transfering" ref="uploadImportRef" accept=".json, application/json"
                                        :maxFileSize="10485760" :multiple="false"
                                        customUpload
                                        :showUploadButton="false" :showCancelButton="false"
                                        @select="onSelect" @clear="onClear">
                                <template #header="{ chooseCallback }" class="w-full">
                                    <div class="w-full flex flex-wrap justify-content-between gap-3">
                                        <Button v-if="activeStep === '1'" class="main-button" @click="chooseCallback()"
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
                        </div>
                        <Button
                                class="outline-button w-2"
                                icon="pi pi-eraser"
                                label="Clear"
                                @click="resetWizard"
                                :disabled="importing || transfering"
                        />
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

                                        <div class="flex flex-row gap-1 align-items-center">
                                            <span v-if="validatedResponse.filtered_count == 0" style="color: var(--text-secondary)">No investments were found in the provided data!</span>
                                            <Button v-else class="main-button w-3"
                                                    @click="transferInvestments"
                                                    label="Transfer"
                                            />
                                        </div>

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

                                            </div>
                                        </div>

                                        <h4>Investment mappings</h4>
                                        <div v-if="validatedResponse.filtered_count > 0" class="flex flex-row w-full p-2 gap-2 align-items-center">
                                            <ImportInvestmentMapping
                                                    :importedCategories="validatedResponse.categories"
                                                    :investmentAccounts="investmentAccs" @save="onSaveInvestmentMapping"
                                            />
                                        </div>
                                    </div>

                                    <ShowLoading v-else :numFields="5" />
                                </StepPanel>
                            </StepPanels>
                        </Stepper>
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
                        NLB imports are currently unsupported!
                    </span>
                </TabPanel>
            </TabPanels>
        </Tabs>

    </div>
</template>

<style scoped>

</style>