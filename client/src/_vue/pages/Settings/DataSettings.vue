<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import ImportModule from "../../features/ImportModule.vue";

</script>

<template>
    <div class="flex flex-column w-full gap-3">
        <SettingsSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-2">
                <div class="w-full flex flex-column gap-2">
                    <h3>Data Import</h3>
                    <h5 style="color: var(--text-secondary)">Manage your imported data.</h5>
                </div>

                <ImportModule />


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