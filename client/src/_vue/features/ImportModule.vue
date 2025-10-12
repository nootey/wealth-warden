<script setup lang="ts">
import {ref} from "vue";
import {useDataStore} from "../../services/stores/data_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import toastHelper from "../../utils/toast_helper.ts";
import type { CustomImportValidationResponse } from "../../models/dataio_models"
import vueHelper from "../../utils/vue_helper.ts";
import ShowLoading from "../components/base/ShowLoading.vue";

const dataStore = useDataStore();
const toastStore = useToastStore();

const uploadImportRef = ref<{ files: File[] } | null>(null);
const fileValidated = ref(false);
const validatedResponse = ref<CustomImportValidationResponse | null>(null);
const importing = ref(false);

const onUpload = async () => {
    const files = uploadImportRef.value?.files ?? [];
    if (!files.length) return;

    importing.value = true;
    try {
        const res = await dataStore.importFromJSON(files[0]);
        toastStore.successResponseToast(res)
        fileValidated.value = false;
        validatedResponse.value = null;
    } catch (error) {
        toastStore.errorResponseToast(error);
    } finally {
        importing.value = false;
    }
};

function clearFile(index: number, removeFileCallback: any) {
    fileValidated.value = false;
    validatedResponse.value = null;
    removeFileCallback(index);
}

async function validateFile(files: any) {

    // Currently, only single file is supported!
    let file = files[0];

    try {
        const res = await dataStore.validateImport("custom", file);
        fileValidated.value = res.valid;
        validatedResponse.value = res;
        toastHelper.formatSuccessToast("File validated", "Check details and proceed with import");

    } catch (error) {
        toastStore.errorResponseToast(error)
    }
}

</script>

<template>
    <div class="flex flex-column w-full gap-3 p-1">
        <h3>Create a new import</h3>
        <FileUpload v-if="!importing" ref="uploadImportRef" accept=".json, application/json" :maxFileSize="10485760"
                    customUpload>
            <template #header="{ chooseCallback, files }" class="w-full">
                <div class="w-full flex flex-wrap justify-content-between gap-3">
                    <Button class="main-button" @click="chooseCallback()" label="Upload"></Button>
                    <Button v-if="!fileValidated" class="main-button" @click="validateFile(files)" :disabled="!files || files.length === 0" label="Validate"></Button>
                    <Button v-if="fileValidated" class="main-button" @click="onUpload" :disabled="!files || files.length === 0" label="Import"></Button>
                </div>
            </template>

            <template #content="{ files, uploadedFiles, removeUploadedFileCallback, removeFileCallback }">

                <div class="flex flex-column gap-2 pt-1 w-full">

                    <div v-if="files.length > 0">
                        <h5>Pending</h5>
                        <div class="flex flex-wrap gap-2 w-full">
                            <div v-for="(file, index) of files" :key="file.name + file.type + file.size"
                                 class="flex flex-row gap-3 p-2 w-full align-items-center">
                                <span class="font-semibold text-ellipsis whitespace-nowrap overflow-hidden">{{ file.name }}</span>
                                <Badge value="Pending" severity="warn" />
                                <i class="pi pi-times hover-icon" @click="clearFile(index, removeFileCallback)" style="color: var(--p-red-300)" />
                            </div>
                        </div>
                    </div>

                    <div v-if="uploadedFiles.length > 0">
                        <h5>Completed</h5>
                        <div class="flex flex-wrap gap-2 w-full">
                            <div v-for="(file, index) of uploadedFiles" :key="file.name + file.type + file.size"
                                 class="flex flex-row gap-3 p-2 w-full align-items-center">
                                <span class="font-semibold text-ellipsis max-w-60 whitespace-nowrap overflow-hidden">{{ file.name }}</span>
                                <Badge value="Completed" severity="success" />
                                <i class="pi pi-times hover-icon" @click="removeUploadedFileCallback(index)" style="color: var(--p-red-300)" />
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
                    </div>
                </div>
            </template>

            <template #empty>
                <div class="flex flex-column align-items-center justify-content-center p-2">
                    <i class="pi pi-cloud-upload text-4xl" />
                    <p>Drag and drop files to here to upload.</p>
                </div>
            </template>
        </FileUpload>
        <ShowLoading v-else :numFields="7" />

        <hr>

        <h3>Imports</h3>
        <div class="w-full flex flex-row gap-2 justify-content-center">
            <span style="color: var(--text-secondary)"> No imports yet </span>
        </div>
    </div>
</template>

<style scoped>

</style>