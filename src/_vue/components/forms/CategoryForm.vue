<script setup lang="ts">
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {nextTick, onMounted, ref} from "vue";
import type {Category} from "../../../models/transaction_models.ts";
import {required} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import toastHelper from "../../../utils/toast_helper.ts";
import ValidationError from "../validation/ValidationError.vue";

const props = defineProps<{
    mode?: "create" | "update";
    recordId?: number | null;
}>();

const emit = defineEmits<{
    (event: 'completeOperation'): void;
}>();

const apiPrefix = "transactions/categories"

const sharedStore = useSharedStore();
const toastStore = useToastStore();

onMounted(async () => {
    if (props.mode === "update" && props.recordId) {
        await loadRecord(props.recordId);
    }
});

const readOnly = ref(false);

const record = ref<Category>(initData());

const classifications = ref<string[]>(['income', 'expense']);
const filteredClassifications = ref<string[]>([]);

const rules = {
    record: {
        display_name: { required, $autoDirty: true },
        classification: { required, $autoDirty: true },
    },
};

const v$ = useVuelidate(rules, { record });

function initData(): Category {

    return {
        id: null,
        name: "",
        display_name: "",
        classification: "",
        parent_id: null,
        is_default: false,
        deleted_at: null,
    };
}

async function loadRecord(id: number) {
    try {
        const data = await sharedStore.getRecordByID(apiPrefix, id);

        readOnly.value = !!data?.deleted_at

        record.value = {
            ...initData(),
            ...data,
        };

        await nextTick();

    } catch (err) {
        toastStore.errorResponseToast(err);
    }
}

async function isRecordValid() {
    const isValid = await v$.value.record.$validate();
    if (!isValid) return false;
    return true;
}

async function manageRecord() {

    if (readOnly.value) {
        toastStore.infoResponseToast(toastHelper.formatInfoToast("Not allowed", "This record is read only!"))
        return;
    }

    if (!await isRecordValid()) return;

    const recordData: any = {
        display_name: record.value.display_name,
        classification: record.value.classification,
    }

    try {

        let response = null;

        switch (props.mode) {
            case "create":
                response = await sharedStore.createRecord(
                    apiPrefix,
                    recordData
                );
                break;
            case "update":
                response = await sharedStore.updateRecord(
                    apiPrefix,
                    record.value.id!,
                    recordData
                );
                break;
            default:
                emit("completeOperation")
                break;
        }

        v$.value.record.$reset();
        toastStore.successResponseToast(response);
        emit("completeOperation")

    } catch (error) {
        toastStore.errorResponseToast(error);
    }
}

const searchClassifications = (event: { query: string }) => {
    const q = event.query.trim().toLowerCase();
    const all = classifications.value;
    filteredClassifications.value = !q ? [...all] : all.filter(t => t.toLowerCase().startsWith(q));
};

</script>

<template>

    <div class="flex flex-column gap-3 p-1">
        <div v-if="readOnly">
            <h5 style="color: var(--text-secondary)">Read-only mode.</h5>
        </div>

        <div class="flex flex-row w-full">
            <div class="flex flex-column w-full">
                <ValidationError :isRequired="true" :message="v$.record.display_name.$errors[0]?.$message">
                    <label>Name</label>
                </ValidationError>
                <InputText :readonly="readOnly" :disabled="readOnly" size="small" v-model="record.display_name"></InputText>
            </div>
        </div>

        <div class="flex flex-row w-full">
            <div class="flex flex-column gap-1 w-full">
                <ValidationError :isRequired="true" :message="v$.record.classification.$errors[0]?.$message">
                    <label>Classification</label>
                </ValidationError>
                <AutoComplete :readonly="readOnly || record.is_default" :disabled="readOnly || record.is_default" size="small" v-model="record.classification"
                              :suggestions="filteredClassifications" @complete="searchClassifications"
                              placeholder="Select classification" dropdown>
                </AutoComplete>
            </div>
        </div>

        <div v-if="mode === 'update' && record.is_default" class="flex flex-row w-full align-items-center gap-2">
            <i class="pi pi-info-circle"></i>
            <span class="text-sm" style="color: var(--text-secondary)">This category is a default. Some parts are not editable.</span>
        </div>

        <div class="flex flex-row gap-2 w-full">
            <div class="flex flex-column w-full">
                <Button v-if="!readOnly" class="main-button" :label="(mode == 'create' ? 'Add' : 'Update') +  ' category'"
                        @click="manageRecord" style="height: 42px;" />
            </div>
        </div>

    </div>

</template>

<style scoped>

</style>