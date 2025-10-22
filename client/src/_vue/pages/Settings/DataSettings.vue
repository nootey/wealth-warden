<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import ImportModule from "../../features/ImportModule.vue";
import ImportList from "../../components/data/ImportList.vue";
import {nextTick, ref} from "vue";
import {usePermissions} from "../../../utils/use_permissions.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";

const toastStore = useToastStore();
const { hasPermission } = usePermissions();

const importListRef = ref<InstanceType<typeof ImportList> | null>(null);
const externalStep = ref<'1' | '2' | '3'>('1');
const externalImportId = ref<string | null>(null);

const addImportModal = ref(false);

function updateList() {
    importListRef.value?.refresh();
}

function onMigrateInvestments(id: string) {
    externalImportId.value = null;
    nextTick(() => {
        externalImportId.value = id;
        externalStep.value = '3';
    });
}

function onResetExternal() {
    externalImportId.value = null;
    externalStep.value = '1';
}

function manipulateDialog(modal: string, value: any) {
    switch (modal) {
        case 'addImport': {
            if(!hasPermission("manage_data")) {
                toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
                return;
            }
            addImportModal.value = value;
            break;
        }
        default: {
            break;
        }
    }
}

</script>

<template>

    <Dialog class="rounded-dialog" v-model:visible="addImportModal" :breakpoints="{'801px': '90vw'}"
            :modal="true" :style="{width: '800px'}" header="New JSON Import">
            <ImportModule :externalStep="externalStep"
                          :externalImportId="externalImportId"
                          @completeImport="updateList"
                          @resetExternal="onResetExternal"
            />

    </Dialog>

    <div class="flex flex-column w-full gap-3">
        <SettingsSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-2">

                <div class="flex flex-row align-items-center gap-2 w-full">
                    <div class="w-full flex flex-column gap-2">
                        <h3>Data Import</h3>
                        <h5 style="color: var(--text-secondary)">Manage your imported data.</h5>
                    </div>
                    <Button class="main-button"
                            @click="manipulateDialog('addImport', true)">
                        <div class="flex flex-row gap-1 align-items-center">
                            <i class="pi pi-plus"></i>
                            <span> New </span>
                            <span class="mobile-hide"> Import </span>
                        </div>
                    </Button>
                </div>

                <h3>Imports</h3>
                <ImportList ref="importListRef" @migrateInvestments="onMigrateInvestments" />

            </div>
        </SettingsSkeleton>

        <SettingsSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-2">
                <div class="w-full flex flex-column gap-2">
                    <h3>Data Export</h3>
                    <h5 style="color: var(--text-secondary)">Manage your exported data.</h5>
                </div>

                <div class="w-full flex flex-row gap-2">
                    <Button class="main-button w-full" label="Export data" icon="pi pi-image"></Button>
                </div>

                <div class="w-full flex flex-row gap-2 justify-content-center">
                    <span style="color: var(--text-secondary)"> No exports yet </span>
                </div>
            </div>
        </SettingsSkeleton>
    </div>
</template>

<style scoped>

</style>