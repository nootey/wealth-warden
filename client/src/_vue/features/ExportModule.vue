<script setup lang="ts">
import {useDataStore} from "../../services/stores/data_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {onMounted, ref} from "vue";
import type {CustomExportValidationResponse} from "../../models/dataio_models.ts";

const emit = defineEmits<{
    (e: 'completeExport'): void;
}>();

const dataStore = useDataStore();
const toastStore = useToastStore();

const exporting = ref(false);
const validatedResponse = ref<CustomExportValidationResponse | null>(null);

onMounted(async () => {
    try {
        console.log("tbd")
    } catch (e) {
        toastStore.errorResponseToast(e);
    }
})

</script>

<template>
    <div class="flex flex-column gap-2">
        <h3>Export your data</h3>
        <span>This will create a downloadable snapshot of your accounts, transactions, transfers and categories.</span>

    </div>
</template>

<style scoped>

</style>