<script setup lang="ts">
import {onMounted, type Ref, ref} from "vue";
import {useDataStore} from "../../services/stores/data_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import toastHelper from "../../utils/toast_helper.ts";
import type { CustomImportValidationResponse } from "../../models/dataio_models"
import vueHelper from "../../utils/vue_helper.ts";
import ShowLoading from "../components/base/ShowLoading.vue";
import {useAccountStore} from "../../services/stores/account_store.ts";
import type {Account} from "../../models/account_models.ts";

const dataStore = useDataStore();
const toastStore = useToastStore();
const accStore = useAccountStore();

const checkingAccs = ref<Account[]>([]);
const investmentAccs = ref<Account[]>([]);
const selectedCheckingAcc = ref<Account | null>(null);
const selectedInvestmentAcc = ref<Account | null>(null);
const filteredCheckingAccs = ref<Account[]>([]);
const filteredInvestmentAccs = ref<Account[]>([]);

const lists: Record<string, Ref<Account[]>> = {
    checking: checkingAccs,
    investment: investmentAccs,
};

const filteredLists: Record<string, Ref<Account[]>> = {
    checking: filteredCheckingAccs,
    investment: filteredInvestmentAccs,
};

onMounted(async () => {
    try {
        checkingAccs.value = await accStore.getAccountsBySubtype("checking");
        investmentAccs.value = await accStore.getAccountsByType("investment");
        if (investmentAccs.value.length == 0) {
            toastStore.infoResponseToast(toastHelper.formatInfoToast("No accounts", "Please create at least one checking account"));
        }
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

function removeLocalFile(index: number) {
    selectedFiles.value.splice(index, 1);
    fileValidated.value = false;
    validatedResponse.value = null;
}

const onUpload = async () => {
    if (!selectedFiles.value.length) return;
    importing.value = true;
    try {
        const res = await dataStore.importFromJSON(
            selectedFiles.value[0],
            selectedCheckingAcc.value?.id!,
            selectedInvestmentAcc.value?.id!
        );
        toastStore.successResponseToast(res);
        selectedFiles.value = [];
        fileValidated.value = false;
        validatedResponse.value = null;
        selectedCheckingAcc.value = null;
        selectedInvestmentAcc.value = null;
    } catch (error) {
        toastStore.errorResponseToast(error);
    } finally {
        importing.value = false;
    }
};

async function validateFile() {
    if (!selectedFiles.value.length) return;
    try {
        const res = await dataStore.validateImport("custom", selectedFiles.value[0]);
        fileValidated.value = res.valid;
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

</script>

<template>
    <div class="flex flex-column w-full gap-3 p-1">
        <h3>Create a new import</h3>
        <span v-if="investmentAccs.length == 0 || checkingAccs.length == 0" style="color: var(--text-secondary)">At least one checking and investment account is required to proceed!</span>
        <FileUpload v-if="!importing" ref="uploadImportRef" accept=".json, application/json"
                    :maxFileSize="10485760" :multiple="false"
                    customUpload
                    :showUploadButton="false" :showCancelButton="false"
                    @select="onSelect" @clear="onClear"
        >
            <template #header="{ chooseCallback }" class="w-full">
                <div class="w-full flex flex-wrap justify-content-between gap-3">
                    <Button class="main-button" @click="chooseCallback()"
                            :disabled="investmentAccs.length == 0 || checkingAccs.length == 0"
                            label="Upload" />
                    <Button v-if="!fileValidated" class="main-button"
                            @click="validateFile"
                            :disabled="selectedFiles.length === 0 || (investmentAccs.length == 0 || checkingAccs.length == 0)"
                            label="Validate"
                    />
                    <Button v-if="fileValidated" class="main-button"
                            @click="onUpload"
                            :disabled="selectedFiles.length === 0 || (investmentAccs.length == 0 || checkingAccs.length == 0) || (!selectedCheckingAcc || !selectedInvestmentAcc)"
                            label="Import"
                    />

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

                    <div v-if="validatedResponse" class="flex flex-column gap-2 w-full">
                        <h4>Validation response</h4>
                        <div class="flex flex-row gap-1 align-items-center">
                            <span>Year: </span>
                            <span>{{ validatedResponse.year }} </span>
                        </div>
                        <div class="flex flex-row gap-1 align-items-center">
                            <span>Txn count: </span>
                            <span>{{ validatedResponse.count }} </span>
                        </div>

                        <h4>Sample transaction</h4>
                        <div class="flex flex-row gap-1 align-items-center">
                            <span>Txn. type: </span>
                            <span>{{ validatedResponse.sample.transaction_type }} </span>
                        </div>
                        <div class="flex flex-row gap-1 align-items-center">
                            <span>Txn. amount: </span>
                            <span>{{ vueHelper.displayAsCurrency(validatedResponse.sample.amount) }} </span>
                        </div>
                        <div class="flex flex-row gap-1 align-items-center">
                            <span>Txn. category: </span>
                            <span>{{ validatedResponse.sample.category }} </span>
                        </div>
                        <div class="flex flex-row gap-1 align-items-center">
                            <span>Txn. description: </span>
                            <span>{{ validatedResponse.sample.description }} </span>
                        </div>

                        <div class="flex flex-row w-full p-2 gap-2 align-items-center">
                            <span style="color: var(--text-secondary)">
                                Select which checking and investment accounts will receive the data from the import.
                            </span>
                            <div class="flex flex-column gap-1 w-6">
                                <label>Checking account</label>
                                <AutoComplete size="small"
                                              v-model="selectedCheckingAcc" :suggestions="filteredCheckingAccs"
                                              @complete="searchAccount($event, 'checking')" optionLabel="name" forceSelection
                                              placeholder="Select checking account" dropdown>
                                </AutoComplete>
                            </div>
                            <div class="flex flex-column gap-1 w-6">
                                <label>Investment account</label>
                                <AutoComplete size="small"
                                              v-model="selectedInvestmentAcc" :suggestions="filteredInvestmentAccs"
                                              @complete="searchAccount($event, 'investment')" optionLabel="name" forceSelection
                                              placeholder="Select investment account" dropdown>
                                </AutoComplete>
                            </div>
                        </div>

                    </div>

                </div>
            </template>
        </FileUpload>
        <ShowLoading v-else :numFields="7" />
    </div>
</template>

<style scoped>

</style>