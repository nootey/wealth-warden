<script setup lang="ts">

import {computed, onMounted, ref, type Ref, watch} from "vue";
import type {Account} from "../../models/account_models.ts";
import {useDataStore} from "../../services/stores/data_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {useAccountStore} from "../../services/stores/account_store.ts";
import toastHelper from "../../utils/toast_helper.ts";
import type {CustomImportValidationResponse, Import} from "../../models/dataio_models.ts";
import ImportInvestmentMapping from "../components/base/ImportInvestmentMapping.vue";
import ShowLoading from "../components/base/ShowLoading.vue";

const emit = defineEmits<{
    (e: 'completeTransfer'): void;
}>();

const dataStore = useDataStore();
const toastStore = useToastStore();
const accStore = useAccountStore();

const transfering = ref(false);
const checkingAccs = ref<Account[]>([]);
const selectedCheckingAcc = ref<Account | null>(null);
const filteredCheckingAccs = ref<Account[]>([]);
const investmentAccs = ref<Account[]>([]);
const investmentMappings = ref<Record<string, number | null>>({});
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
        investmentMappings.value = {};
    }
});

onMounted(async () => {
    try {

        await getImports();

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

async function fetchValidationResponse(importId: number) {
    loadingValidation.value = true;
    try {
        validatedResponse.value = await dataStore.getCustomImportJSON(importId, "investments");
        // Reset mappings when a new import is selected
        investmentMappings.value = {};
    } catch (e) {
        toastStore.errorResponseToast(e);
        validatedResponse.value = null;
    } finally {
        loadingValidation.value = false;
    }
}

async function getImports() {
    try {
        imports.value = await dataStore.getImports("custom");
    } catch (e) {
        toastStore.errorResponseToast(e)
    }
}

function onSaveMapping(map: Record<string, number | null>) {
    investmentMappings.value = map
}

function searchAccount(event: { query: string }, accType: string) {
    const all = lists[accType].value ?? [];
    const q = event.query.trim().toLowerCase();

    filteredLists[accType].value = q
        ? all.filter(a => a.name.toLowerCase().includes(q))
        : [...all];
}

function resetWizard() {

    if(transfering.value) {
        toastStore.infoResponseToast({"Title": "Unavailable", "Message": "An operation is currently being executed!"})
    }
    // clear local state
    selectedCheckingAcc.value = null;
    transfering.value = false;
    validatedResponse.value = null;
}

async function transferInvestments() {

    if (!selectedImport.value?.id) {
        toastStore.errorResponseToast("Missing import ID");
        return;
    }

    if (Object.keys(investmentMappings.value).length === 0) {
        toastStore.errorResponseToast("Please set up your investment mappings first")
        return
    }

    transfering.value = true

    try {
        const payload = {
            import_id: selectedImport.value.id,
            checking_acc_id: selectedCheckingAcc.value?.id!,
            investment_mappings: Object.entries(investmentMappings.value).map(
                ([name, account_id]) => ({ name, account_id })
            ),
        }

        const res = await dataStore.transferInvestmentsFromImport(payload);
        toastStore.successResponseToast(res);

        resetWizard();
        emit("completeTransfer");
    } catch (error) {
        toastStore.errorResponseToast(error)
    } finally {
        transfering.value = false
    }
}

const isTransferDisabled = computed(() => {
    if (transfering.value) return true;
    if (!selectedCheckingAcc.value) return true;

    const mappings = Object.values(investmentMappings.value);
    const hasAtLeastOne = mappings.some(v => v !== null);
    return !hasAtLeastOne;
});

</script>

<template>
    <h3>Map investments from imported data</h3>

    <div class="flex flex-column w-full gap-3">
        <span>Select import</span>

        <Select size="small"
                style="width: 450px;"
                v-model="selectedImport"
                :options="imports"
                optionLabel="name"
                placeholder="Select import"
        />

        <div v-if="loadingValidation" class="flex flex-column w-full p-2">
            <ShowLoading :numFields="5" />
        </div>
        <div v-else-if="validatedResponse && validatedResponse.filtered_count == 0" class="flex flex-column w-full p-2">
            <span style="color: var(--text-secondary)">No investments were found in the provided import!</span>
        </div>
        <div v-else-if="validatedResponse">
            <div v-if="!transfering" class="flex flex-column gap-4 w-full">
            <span style="color: var(--text-secondary)">
                Start the transfer. The checking account and at least one investment mapping is required.
            </span>
                <Button class="main-button w-3" @click="transferInvestments" label="Transfer" :disabled="isTransferDisabled"/>

                <div v-if="!transfering" class="flex flex-column w-full gap-3">
                    <h3>Import account</h3>
                    <div class="flex flex-column w-6 gap-2 align-items-center">
                        <span class="text-sm w-full" style="color: var(--text-secondary)">Select an account which will receive the transfers.</span>
                        <div class="flex flex-column gap-1 w-full">
                            <label>Checking account</label>
                            <AutoComplete size="small"
                                          v-model="selectedCheckingAcc" :suggestions="filteredCheckingAccs"
                                          @complete="searchAccount($event, 'checking')" optionLabel="name" forceSelection
                                          placeholder="Select checking account" dropdown />
                            <span class="text-sm" v-if="!selectedCheckingAcc" style="color: var(--text-secondary)">Please select an account.</span>
                            <span class="text-sm" v-else style="color: var(--text-secondary)">Account's opening date is valid.</span>
                        </div>
                    </div>

                    <h3>Validation response</h3>
                    <span class="text-sm" style="color: var(--text-secondary)">General information about your import.</span>
                    <div class="flex flex-row w-full gap-2">
                        <span>Transfer count: </span>
                        <span>{{ validatedResponse.filtered_count }} </span>
                    </div>

                    <h4>Investment mappings</h4>
                    <div v-if="validatedResponse.filtered_count > 0" class="flex flex-row w-full gap-3 align-items-center">
                        <ImportInvestmentMapping  v-model:modelValue="investmentMappings"
                                                  :importedCategories="validatedResponse.categories"
                                                  :investmentAccounts="investmentAccs" @save="onSaveMapping"/>
                    </div>
                </div>
                <ShowLoading v-else :numFields="3" />

            </div>
            <ShowLoading v-else :numFields="5" />
        </div>
        <div v-else-if="!selectedImport" class="flex flex-column w-full p-2">
            <span style="color: var(--text-secondary)">Please select an import to begin.</span>
        </div>
    </div>
</template>

<style scoped>

</style>