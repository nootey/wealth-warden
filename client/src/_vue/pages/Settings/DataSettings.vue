<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import ImportModule from "../../features/ImportModule.vue";
import ImportList from "../../components/data/ImportList.vue";
import {nextTick, ref} from "vue";

const importListRef = ref<InstanceType<typeof ImportList> | null>(null);
const externalStep = ref<'1' | '2' | '3'>('1');
const externalImportId = ref<string | null>(null);

async function updateList() {
    importListRef.value?.refresh();
}

function onMigrateInvestments(id: string) {
    externalImportId.value = null;
    nextTick(() => {
        externalImportId.value = id;
        externalStep.value = '3';
    });
}

</script>

<template>
    <div class="flex flex-column w-full gap-3">
        <SettingsSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-2">
                <div class="w-full flex flex-column gap-2">
                    <h3>Data Import</h3>
                    <h5 style="color: var(--text-secondary)">Manage your imported data.</h5>
                </div>

                <ImportModule :externalStep="externalStep"
                              :externalImportId="externalImportId"
                              @completeImport="updateList"/>

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