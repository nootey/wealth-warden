<script setup lang="ts">

import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {nextTick, onMounted, ref} from "vue";
import type {Role, User} from "../../../models/user_models.ts";
import type {Transaction} from "../../../models/transaction_models.ts";
import {email, required} from "@vuelidate/validators";
import {decimalMax, decimalMin, decimalValid} from "../../../validators/currency.ts";
import useVuelidate from "@vuelidate/core";
import dayjs from "dayjs";
import type {Account} from "../../../models/account_models.ts";
import {useUserStore} from "../../../services/stores/user_store.ts";
import toastHelper from "../../../utils/toast_helper.ts";
import ShowLoading from "../base/ShowLoading.vue";
import ValidationError from "../validation/ValidationError.vue";

const props = defineProps<{
    mode?: "create" | "update";
    recordId?: number | null;
    roles: Role[];
}>();
const emit = defineEmits<{
    (event: 'completeOperation'): void;
}>();

const sharedStore = useSharedStore();
const userStore = useUserStore();
const toastStore = useToastStore();

onMounted(async () => {
    if (props.mode === "update" && props.recordId) {
        await loadRecord(props.recordId);
    }
});

const readOnly = ref(false);
const loading = ref(false);

const record = ref<User>(initData());
const filteredRoles = ref<Role[]>([]);

const rules = {
    record: {
        role: {
            name: {
                required,
                $autoDirty: true
            }
        },
        display_name: {
            $autoDirty: true
        },
        email: {
            required,
            email,
            $autoDirty: true
        },
    },
};

const v$ = useVuelidate(rules, { record });

function initData(): User {

    return {
        role: {
            name: "",
        },
        display_name: "",
        email: "",
        deleted_at: null,
    };
}

const searchRole = (event: { query: string }) => {
    setTimeout(() => {
        if (!event.query.trim().length) {
            filteredRoles.value = [...props.roles];
        } else {
            filteredRoles.value = props.roles.filter((record) => {
                return record.name.toLowerCase().startsWith(event.query.toLowerCase());
            });
        }
    }, 250);
}

async function loadRecord(id: number) {
    try {
        loading.value = true;
        const data = await sharedStore.getRecordByID("users", id);

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

    const recordData = {
        email: record.value.email,
        role_id: record.value?.role?.id,
    }

    try {
        let response = null;

        switch (props.mode) {
            case "create":
                response = await sharedStore.createRecord(
                    userStore.apiPrefix + "/invitations",
                    recordData
                );
                break;
            case "update":
                response = await sharedStore.updateRecord(
                    userStore.apiPrefix,
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

        <div class="flex flex-column gap-3">

            <div class="flex flex-row w-full">
                <div class="flex flex-column gap-1 w-full">
                    <ValidationError :isRequired="true" :message="v$.record.email.$errors[0]?.$message">
                        <label>Email</label>
                    </ValidationError>
                    <InputText :readonly="readOnly || mode == 'update'" :disabled="readOnly" size="small" v-model="record.email"
                               placeholder="Input email"></InputText>
                </div>
            </div>

            <div class="flex flex-row w-full">
                <div class="flex flex-column gap-1 w-full">
                    <ValidationError :isRequired="true" :message="v$.record.role.name.$errors[0]?.$message">
                        <label>Role</label>
                    </ValidationError>
                    <AutoComplete :readonly="readOnly" :disabled="readOnly" size="small"
                                  v-model="record.role" :suggestions="filteredRoles"
                                  @complete="searchRole" optionLabel="name" forceSelection
                                  placeholder="Select role" dropdown>
                    </AutoComplete>
                </div>
            </div>

            <div class="flex flex-row gap-2 w-full" >
                <div class="flex flex-column w-full">
                    <Button v-if="!readOnly" class="main-button"
                            :label="(mode == 'create' ? 'Invite' : 'Update') +  ' user'"
                            @click="manageRecord" style="height: 42px;" />
                </div>
            </div>

        </div>
    </div>
    <ShowLoading v-else :numFields="7" />
</template>

<style scoped>

</style>