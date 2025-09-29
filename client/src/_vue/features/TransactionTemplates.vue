<script setup lang="ts">
import {useSharedStore} from "../../services/stores/shared_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {usePermissions} from "../../utils/use_permissions.ts";
import {useConfirm} from "primevue/useconfirm";
import {computed, onMounted, ref} from "vue";
import type {TransactionTemplate} from "../../models/transaction_models.ts";
import filterHelper from "../../utils/filter_helper.ts";
import type {Column} from "../../services/filter_registry.ts";
import dateHelper from "../../utils/date_helper.ts";
import CustomPaginator from "../components/base/CustomPaginator.vue";
import LoadingSpinner from "../components/base/LoadingSpinner.vue";

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const { hasPermission } = usePermissions();

const confirm = useConfirm();

const apiPrefix = "transactions/templates";

onMounted(async () => {
    await getData();
})

const loadingRecords = ref(true);
const records = ref<TransactionTemplate[]>([]);

const params = computed(() => {
    return {
        rowsPerPage: paginator.value.rowsPerPage,
        sort: sort.value,
        filters: null,
    }
});
const rows = ref([5, 10, 25]);
const default_rows = ref(rows.value[0]);
const paginator = ref({
    total: 0,
    from: 0,
    to: 0,
    rowsPerPage: default_rows.value
});
const page = ref(1);
const sort = ref(filterHelper.initSort());

const activeColumns = computed<Column[]>(() => [
    { field: 'name', header: 'Name'},
    { field: 'account', header: 'Account'},
    { field: 'category', header: 'Category'},
    { field: 'amount', header: 'Amount'},
    { field: 'interval', header: 'Interval'},
]);

async function getData(new_page = null) {

    loadingRecords.value = true;
    if(new_page)
        page.value = new_page;

    try {
        let paginationResponse = await sharedStore.getRecordsPaginated(
            apiPrefix,
            { ...params.value },
            page.value
        );
        records.value = paginationResponse.data;
        paginator.value.total = paginationResponse.total_records;
        paginator.value.to = paginationResponse.to;
        paginator.value.from = paginationResponse.from;
        loadingRecords.value = false;
    } catch (error) {
        toastStore.errorResponseToast(error);
    }
}

async function onPage(event: any) {
    paginator.value.rowsPerPage = event.rows;
    page.value = (event.page+1)
    await getData();
}

async function deleteConfirmation(id: number, name: string) {
    confirm.require({
        header: 'Delete record?',
        message: `This will delete template: ${name}".`,
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
            apiPrefix,
            id,
        );
        toastStore.successResponseToast(response);
        await getData();
    } catch (error) {
        toastStore.errorResponseToast(error);
    }
}

function refresh() { getData(); }

defineExpose({ refresh });

</script>

<template>
    <div class="flex flex-column justify-content-center w-full gap-3"
         style="max-width: 1000px;">

        <div class="flex flex-row justify-content-between align-items-center text-center gap-2 w-full">
            <span style="color: var(--text-secondary)">Create and manage custom templates, for executing transactions.</span>
            <Button label="New template" icon="pi pi-plus" class="main-button ml-auto"
                     />
        </div>

        <div class="flex flex-row gap-2 w-full">
            <DataTable class="w-full enhanced-table" dataKey="id" :loading="loadingRecords" :value="records"
                       scrollable scroll-height="50vh">
                <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
                <template #loading> <LoadingSpinner></LoadingSpinner> </template>
                <template #footer>
                    <CustomPaginator :paginator="paginator" :rows="rows" @onPage="onPage"/>
                </template>

                <Column v-for="col of activeColumns" :key="col.field" :header="col.header" :field="col.field" style="width: 25%" >
                    <template #body="{ data, field }">
                        <template v-if="field === 'created_at'">
                            {{ dateHelper.formatDate(data?.created_at, true) }}
                        </template>
                        <template v-else>
                            {{ data[field] }}
                        </template>
                    </template>
                </Column>

                <Column header="Actions">
                    <template #body="{ data }">
                        <i v-if="hasPermission('manage_data')"
                           class="pi pi-trash hover-icon" style="font-size: 0.875rem; color: var(--p-red-300);"
                           @click="deleteConfirmation(data?.id, data?.name)"></i>
                        <i v-else class="pi pi-exclamation-circle" style="font-size: 0.875rem;"
                           v-tooltip="'No action available'"></i>
                    </template>
                </Column>

            </DataTable>
        </div>

    </div>
</template>

<style scoped>

</style>