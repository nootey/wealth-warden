<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {computed, onMounted, ref} from "vue";
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useConfirm} from "primevue/useconfirm";
import {useUserStore} from "../../../services/stores/user_store.ts";
import type {Role} from "../../../models/user_models.ts";
import type {Column} from "../../../services/filter_registry.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import LoadingSpinner from "../../components/base/LoadingSpinner.vue";
import RoleForm from "../../components/forms/RoleForm.vue";

const userStore = useUserStore();
const toastStore = useToastStore();
const sharedStore = useSharedStore();

onMounted(async () => {
    await getRoles();
});

const confirm = useConfirm();
const createModal = ref(false);
const updateModal = ref(false);
const selectedID = ref<number | null>(null);

const roles = computed<Role[]>(() => userStore.roles);

const columns = computed<Column[]>(() => [
    { field: 'name', header: 'Name'},
    { field: 'description', header: 'Description'},
]);

async function getRoles() {
    await userStore.getRoles(true);
}

async function handleEmit(type: string, data?: any) {
    switch (type) {
        case 'completeOperation': {
            createModal.value = false;
            updateModal.value = false;
            await getRoles();
            break;
        }
        case 'openCreate': {
            createModal.value = true;
            break;
        }
        case 'openUpdate': {
            updateModal.value = true;
            selectedID.value = data;
            break;
        }
        case 'deleteOperation': {
            await deleteConfirmation(data.id, data.name);
            break;
        }
        default: {
            break;
        }
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
        await getRoles();

    } catch (err) {
        toastStore.errorResponseToast(err)
    }
}

</script>

<template>

    <Dialog class="rounded-dialog" v-model:visible="createModal"
            :breakpoints="{ '501px': '90vw' }" :modal="true" :style="{ width: '500px' }" header="Create role">
        <RoleForm mode="create"
                      @completeOperation="handleEmit('completeOperation')"/>
    </Dialog>

    <Dialog position="right" class="rounded-dialog" v-model:visible="updateModal"
            :breakpoints="{ '501px': '90vw' }" :modal="true" :style="{ width: '500px' }" header="Update role">
        <RoleForm mode="update" :recordId="selectedID"
                  @completeOperation="handleEmit('completeOperation')"/>
    </Dialog>

    <div class="flex flex-column w-full gap-3">
        <SettingsSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-2">
                <div class="flex flex-row justify-content-between align-items-center gap-3">
                    <div class="w-full flex flex-column gap-2">
                        <h3>Roles</h3>
                        <h5 style="color: var(--text-secondary)">View and manage roles and permissions.</h5>
                    </div>
                    <Button class="main-button w-4" label="New role" icon="pi pi-plus" @click="handleEmit('openCreate')"/>
                </div>


                <div v-if="roles" class="w-full flex flex-column gap-2 w-full">
                    <DataTable class="w-full enhanced-table" dataKey="id" :value="roles"
                               paginator :rows="10" :rowsPerPageOptions="[10, 25]" scrollable scroll-height="75vh"
                               rowGroupMode="subheader" groupRowsBy="classification" :rowClass="vueHelper.deletedRowClass">
                        <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
                        <template #loading> <LoadingSpinner></LoadingSpinner> </template>

                        <Column v-for="col of columns" :key="col.field"
                                :field="col.field" :header="col.header"
                                sortable>
                        </Column>

                        <Column header="Actions">
                            <template #body="{ data }">
                                <div class="flex flex-row align-items-center gap-2">
                                    <i class="pi pi-pen-to-square hover-icon text-xs" v-tooltip="'Edit role'"
                                       @click="handleEmit('openUpdate', data.id!)"/>
                                    <i class="pi pi-trash hover-icon text-xs" v-tooltip="'Delete role'"
                                       style="color: var(--p-red-300);"
                                       @click="handleEmit('deleteOperation', data)"></i>
                                </div>
                            </template>
                        </Column>

                    </DataTable>
                </div>
            </div>
        </SettingsSkeleton>
    </div>
</template>

<style scoped>

</style>