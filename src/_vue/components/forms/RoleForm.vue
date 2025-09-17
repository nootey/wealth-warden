<script setup lang="ts">
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {nextTick, onMounted, ref} from "vue";
import {required} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import toastHelper from "../../../utils/toast_helper.ts";
import ValidationError from "../validation/ValidationError.vue";
import ShowLoading from "../base/ShowLoading.vue";
import type {Role} from "../../../models/user_models.ts";

const props = defineProps<{
    mode?: "create" | "update";
    recordId?: number | null;
}>();

const emit = defineEmits<{
    (event: 'completeOperation'): void;
}>();

const apiPrefix = "users/roles"

const sharedStore = useSharedStore();
const toastStore = useToastStore();

onMounted(async () => {
    if (props.mode === "update" && props.recordId) {
        await loadRecord(props.recordId);
    }
});

const readOnly = ref(false);
const loading = ref(false);

const record = ref<Role>(initData());


const rules = {
    record: {
        name: { required, $autoDirty: true },
        description: { $autoDirty: true },
    },
};

const v$ = useVuelidate(rules, { record });

function initData(): Role {

    return {
        id: 0,
        name: "",
        description: "",
        is_default: false,
    };
}

async function loadRecord(id: number) {
    try {
        loading.value = true;
        const data = await sharedStore.getRecordByID(apiPrefix, id, { deleted: true});

        record.value = {
            ...initData(),
            ...data,
        };

        await nextTick();
        loading.value = false;

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
        name: record.value.name,
        description: record.value.description,
        is_default: record.value.is_default,
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

</script>

<template>

    <div v-if="!loading" class="flex flex-column gap-3 p-1">
        <div v-if="readOnly">
            <h5 style="color: var(--text-secondary)">Read-only mode.</h5>
        </div>

        <div class="flex flex-column gap-3 p-1">
            <div class="flex flex-row w-full">
                <div class="flex flex-column w-full">
                    <ValidationError :isRequired="true" :message="v$.record.name.$errors[0]?.$message">
                        <label>Name</label>
                    </ValidationError>
                    <InputText :readonly="readOnly" :disabled="readOnly" size="small"
                               v-model="record.name"></InputText>
                </div>
            </div>

            <div class="flex flex-row w-full">
                <div class="flex flex-column gap-1 w-full">
                    <ValidationError :isRequired="false" :message="v$.record.description.$errors[0]?.$message">
                        <label>Description</label>
                    </ValidationError>
                    <InputText :readonly="readOnly" :disabled="readOnly" size="small"
                               v-model="record.description"></InputText>
                </div>
            </div>
        </div>

        <div v-if="mode === 'update' && record.is_default" class="flex flex-row w-full align-items-center gap-2">
            <i class="pi pi-info-circle"></i>
            <span class="text-sm" style="color: var(--text-secondary)">This role is a default. Permissions are not editable.</span>
        </div>

        <div class="flex flex-row gap-2 w-full">
            <div class="flex flex-column w-full">
                <Button v-if="!readOnly" class="main-button" :label="(mode == 'create' ? 'Add' : 'Update') +  ' role'"
                        @click="manageRecord" style="height: 42px;" />
            </div>
        </div>

    </div>
    <ShowLoading v-else :numFields="4" />

</template>

<style scoped>

</style>