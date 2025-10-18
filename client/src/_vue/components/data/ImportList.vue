<script setup lang="ts">

import type {Import} from "../../../models/dataio_models.ts";
import {computed, onMounted, ref} from "vue";
import {useDataStore} from "../../../services/stores/data_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import vueHelper from "../../../utils/vue_helper.ts";
import type {Column} from "../../../services/filter_registry.ts";
import dateHelper from "../../../utils/date_helper.ts";
import {usePermissions} from "../../../utils/use_permissions.ts";
import {useConfirm} from "primevue/useconfirm";
import {useSharedStore} from "../../../services/stores/shared_store.ts";

const emit = defineEmits<{
    (e: 'migrateInvestments', importId: string): void;
}>();

const dataStore = useDataStore();
const toastStore = useToastStore();
const sharedStore = useSharedStore();

const { hasPermission } = usePermissions();
const confirm = useConfirm();

const imports = ref<Import[]>([]);
const loading = ref(false);

onMounted(async () => {
        await getData()
})

async function getData() {
    try {
        imports.value = await dataStore.getImports("custom");
    } catch (e) {
        toastStore.errorResponseToast(e)
    }
}

function refresh() { getData(); }

defineExpose({ refresh });

const activeColumns = computed<Column[]>(() => [
    { field: 'name', header: 'Name'},
    { field: 'import_type', header: 'Type'},
    { field: 'status', header: 'Status'},
    { field: 'currency', header: 'Currency'},
    { field: 'step', header: 'Step'},
]);

async function deleteConfirmation(id: number, name: string) {
    confirm.require({
        header: 'Delete record?',
        message: `This will delete import: ${name}".`,
        rejectProps: { label: 'Cancel' },
        acceptProps: { label: 'Delete', severity: 'danger' },
        accept: () => deleteRecord(id),
    });
}

async function deleteRecord(id: number) {

    if(!hasPermission("manage_data")) {
        toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
        return;
    }

    try {
        let response = await sharedStore.deleteRecord(
            "imports",
            id,
        );
        toastStore.successResponseToast(response);
        await getData();
    } catch (error) {
        toastStore.errorResponseToast(error);
    }
}

</script>

<template>
    <div class="w-full flex flex-row gap-2 justify-content-center">
        <DataTable dataKey="id" class="w-full enhanced-table" :loading="loading" :value="imports"
                   scrollable scroll-height="50vh" columnResizeMode="fit"
                   scrollDirection="both">
            <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
            <template #loading> <LoadingSpinner></LoadingSpinner> </template>
            <Column header="Actions">
                <template #body="{ data }">
                    <div class="flex flex-row align-items-center gap-2">
                        <i v-if="data['step'] === 'investments'" class="pi pi-cart-plus hover-icon text-xs" v-tooltip="'Migrate investments'"
                           @click="emit('migrateInvestments', data.id)"/>
                        <i v-else>/</i>
                        <i v-if="hasPermission('manage_data')"
                           class="pi pi-trash hover-icon" style="font-size: 0.875rem; color: var(--p-red-300);"
                           @click="deleteConfirmation(data?.id, data?.name)"></i>
                    </div>
                </template>
            </Column>
            <Column v-for="col of activeColumns" :key="col.field" :header="col.header" :field="col.field"
                    :headerClass="col.hideOnMobile ? 'mobile-hide ' : ''"
                    :bodyClass="col.hideOnMobile ? 'mobile-hide ' : ''">
                <template #body="{ data, field }">
                    <template v-if="field === 'amount'">
                        {{ vueHelper.displayAsCurrency(data.transaction_type == "expense" ? (data.amount*-1) : data.amount) }}
                    </template>
                    <template v-else-if="field === 'started_at' || field === 'completed_at'">
                        {{ dateHelper.formatDate(data[field], true) }}
                    </template>
                    <template v-else>
                        {{ data[field] }}
                    </template>
                </template>
            </Column>
        </DataTable>
    </div>
</template>

<style scoped>

</style>