<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import {computed, onMounted, ref} from "vue";
import {useUserStore} from "../../../services/stores/user_store.ts";
import type {Role} from "../../../models/user_models.ts";
import type {Column} from "../../../services/filter_registry.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import LoadingSpinner from "../../components/base/LoadingSpinner.vue";
import RoleForm from "../../components/forms/RoleForm.vue";

const userStore = useUserStore();

onMounted(async () => {
    await getRoles();
});


const createModal = ref(false);
const updateModal = ref(false);
const selectedID = ref<number | null>(null);

const roles = computed<Role[]>(() => userStore.roles);

const columns = computed<Column[]>(() => [
    { field: 'name', header: 'Name'},
    { field: 'description', header: 'Description', hideOnMobile: true },
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
        case 'deleteRole': {
            createModal.value = false;
            updateModal.value = false;
            await getRoles();
            break;
        }
        default: {
            break;
        }
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
                  @completeOperation="handleEmit('completeOperation')"
                  @completeRoleDelete="handleEmit('deleteRole')"/>
    </Dialog>

    <div class="flex flex-column w-full gap-3">
        <SettingsSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-2">
                <div class="flex flex-row justify-content-between align-items-center gap-3">
                    <div class="w-full flex flex-column gap-2">
                        <h3>Roles</h3>
                        <h5 style="color: var(--text-secondary)">View and manage roles and permissions.</h5>
                    </div>
                    <Button class="main-button" @click="handleEmit('openCreate')">
                        <div class="flex flex-row gap-1 align-items-center">
                            <i class="pi pi-plus"></i>
                            <span> New </span>
                            <span class="mobile-hide"> Role </span>
                        </div>
                    </Button>
                </div>


                <div v-if="roles" class="w-full flex flex-column gap-2 w-full">
                    <DataTable class="w-full enhanced-table" dataKey="id" :value="roles"
                               paginator :rows="10" :rowsPerPageOptions="[10, 25]" scrollable scroll-height="75vh"
                               rowGroupMode="subheader" groupRowsBy="classification" :rowClass="vueHelper.deletedRowClass">
                        <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
                        <template #loading> <LoadingSpinner></LoadingSpinner> </template>

                        <Column v-for="col of columns" :key="col.field"
                                :field="col.field" :header="col.header"
                                sortable :headerClass="col.hideOnMobile ? 'mobile-hide ' : ''"
                                :bodyClass="col.hideOnMobile ? 'mobile-hide ' : ''">
                            <template #body="{ data }">
                               <span class="hover" @click="handleEmit('openUpdate', data.id!)">
                                    {{ data[col.field] }}
                                </span>
                            </template>
                        </Column>

                        <Column header="Permissions">
                            <template #body="{ data }">
                                <div class="flex flex-row align-items-center gap-2"
                                     v-tooltip="'This role has ' + (data?.permissions?.length ?? 0) + ' permissions'">
                                    <i class="pi pi-eye"></i>
                                    <span>{{ data?.permissions?.length ?? 0 }}</span>
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
    .hover { font-weight: bold; }
    .hover:hover { cursor: pointer; text-decoration: underline; }
</style>