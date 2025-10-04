<script setup lang="ts">
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {computed, nextTick, onMounted, ref} from "vue";
import {required} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import toastHelper from "../../../utils/toast_helper.ts";
import ValidationError from "../validation/ValidationError.vue";
import ShowLoading from "../base/ShowLoading.vue";
import type {Permission, Role} from "../../../models/user_models.ts";
import {useUserStore} from "../../../services/stores/user_store.ts";
import {useConfirm} from "primevue/useconfirm";

const props = defineProps<{
    mode?: "create" | "update";
    recordId?: number | null;
}>();

const emit = defineEmits<{
    (event: 'completeOperation'): void;
    (event: 'completeRoleDelete'): void;
}>();

const apiPrefix = "users/roles"

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const userStore = useUserStore();

const confirm = useConfirm();

onMounted(async () => {
    try {
        await userStore.getPermissions();
        if (props.mode === "update" && props.recordId) {
            await loadRecord(props.recordId);
        }
    } catch (err) {
        toastStore.errorResponseToast(err);
    }
});

const readOnly = ref(false);
const loading = ref(false);

const record = ref<Role>(initData());
const permissions = computed<Permission[]>(() => userStore.permissions ?? []);
const selectedPermissions = ref<Permission[]>([]);

const rules = {
    record: {
        name: { required, $autoDirty: true },
        description: { $autoDirty: true },
    },
    selectedPermissions: {
        required,
        $autoDirty: true,
    },
};

const v$ = useVuelidate(rules, { record, selectedPermissions });

function initData(): Role {

    return {
        id: 0,
        name: "",
        description: "",
        is_default: false,
        permissions: undefined,
    };
}

async function loadRecord(id: number) {
    try {
        loading.value = true;
        const data = await sharedStore.getRecordByID(apiPrefix, id, { with_permissions: true});

        record.value = {
            ...initData(),
            ...data,
        };

        selectedPermissions.value = (data.permissions ?? []) as Permission[];

        await nextTick();

    } catch (err) {
        toastStore.errorResponseToast(err);
    } finally {
        loading.value = false;
    }
}

async function isRecordValid() {
    return await v$.value.$validate();
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
        permissions: selectedPermissions.value,
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

async function deleteConfirmation(id: number, name: string) {
    confirm.require({
        header: 'Confirm operation',
        message: `You are about to delete role: "${name}". This action is irreversible!`,
        rejectProps: { label: 'Cancel' },
        acceptProps: { label: 'Continue', severity: 'danger' },
        accept: () => deleteRecord(id),
    });
}

async function deleteRecord(id: number) {
    try {
        let response = await sharedStore.deleteRecord(
            "users/roles",
            id,
        );
        toastStore.successResponseToast(response);
        emit("completeRoleDelete");

    } catch (err) {
        toastStore.errorResponseToast(err)
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

        <div v-if="!record.is_default" class="flex flex-row gap-2 w-full">
            <div class="flex flex-column w-full">
                <ValidationError :isRequired="true" :message="v$.selectedPermissions.$errors[0]?.$message">
                    <label>Permissions</label>
                </ValidationError>
                <MultiSelect v-model="selectedPermissions" :options="permissions" optionLabel="name"
                    display="comma" filter dataKey="id"
                    :disabled="readOnly || (mode === 'update' && record.is_default)"
                    placeholder="Select permissions" class="w-full"
                />
            </div>
        </div>

        <div class="flex flex-row gap-3 w-full">
            <div v-if="selectedPermissions.length"
                 class="flex flex-column gap-2 w-full p-1 w-full"
                 style="max-height: 220px; overflow-y: auto;">

                <div v-for="perm in selectedPermissions" :key="perm.id" style="width: 99%;"
                     class="flex flex-column p-2 border-round-lg border-1 surface-border gap-1">
                    <div><strong>Name:</strong> {{ perm.name }}</div>
                    <div><strong>Description:</strong> {{ perm.description }}</div>
                </div>

            </div>
        </div>

        <div class="flex flex-row gap-2 w-full">
            <div class="flex flex-column w-full gap-2">
                <Button v-if="!readOnly" class="main-button" :label="(mode == 'create' ? 'Add' : 'Update') +  ' role'"
                        @click="manageRecord" style="height: 42px;" />
                <Button v-if="!readOnly && mode == 'update'"
                        label="Delete role" class="delete-button"
                        @click="deleteConfirmation(record.id!, record.name)" style="height: 42px;" />
            </div>
        </div>

    </div>
    <ShowLoading v-else :numFields="4" />

</template>

<style scoped>

</style>