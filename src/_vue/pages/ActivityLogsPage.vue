<script setup lang="ts">
import LoadingSpinner from "../components/base/LoadingSpinner.vue";
import {computed, onMounted, provide, ref} from "vue";
import vueHelper from "../../utils/vue_helper.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {useLoggingStore} from "../../services/stores/logging_store.ts";
import ColumnHeader from "../components/base/ColumnHeader.vue";
import dateHelper from "../../utils/date_helper.ts";
import IconDisplay from "../components/base/IconDisplay.vue";
import ActionRow from "../components/layout/ActionRow.vue";
import type {ActivityLog, Causer} from "../../models/logging_models";
import filterHelper from "../../utils/filter_helper.ts";
import ActiveFilters from "../components/filters/ActiveFilters.vue";
import {useConfirm} from "primevue/useconfirm";
import type {Column} from "../../services/filter_registry.ts";
import type {FilterObj} from "../../models/shared_models.ts";
import FilterMenu from "../components/filters/FilterMenu.vue";
import {useSharedStore} from "../../services/stores/shared_store.ts";
import CustomPaginator from "../components/base/CustomPaginator.vue";
import {usePermissions} from "../../utils/use_permissions.ts";

const toastStore = useToastStore();
const loggingStore = useLoggingStore();
const sharedStore = useSharedStore();
const { hasPermission } = usePermissions();

const loadingRecords = ref(true);
const records = ref<ActivityLog[]>([]);

const confirm = useConfirm();

const apiPrefix = "logs";

const params = computed(() => {
    return {
        rowsPerPage: paginator.value.rowsPerPage,
        sort: sort.value,
        filters: filters.value,
    }
});

const rows = ref([10, 25, 50, 100]);
const default_rows = ref(rows.value[0]);
const paginator = ref({
    total: 0,
    from: 0,
    to: 0,
    rowsPerPage: default_rows.value
});
const page = ref(1);
const sort = ref(filterHelper.initSort());
const expandedRows = ref([]);
const filterStorageIndex = ref(apiPrefix+"-filters");
const filters = ref(JSON.parse(localStorage.getItem(filterStorageIndex.value) ?? "[]"));
const filterOverlayRef = ref<any>(null);

const loadingFilterData = ref(false);

const availableEvents = ref<string[]>([]);
const availableCategories = ref<string[]>([]);
const availableCausers = ref<Causer[]>([]);

const activeColumns = computed<Column[]>(() => [
    { field: 'created_at', header: 'Time', type: 'date'},
    { field: 'event', header: 'Event', type: 'enum', options: availableEvents.value, optionLabel: 'name'},
    { field: 'category', header: 'Category', type: 'enum', options: availableCategories.value, optionLabel: 'name'},
    { field: 'causer_id', header: 'Causer', type: 'enum', options: availableCausers.value, optionLabel: 'name' },
]);

onMounted(async () => {
    await init();
});

async function init() {
    await getData();
    await getFilterData();
}

async function getFilterData(){
    loadingFilterData.value = true;
    try {
        let response = await loggingStore.getFilterData("activity");
        availableEvents.value = response.data.events;
        availableCausers.value = response.data.causers;
        availableCategories.value = response.data.categories;
        loadingFilterData.value = false;
    } catch (error) {
        toastStore.errorResponseToast(error);
    }
}

async function getData(new_page: number|null = null) {

    loadingRecords.value = true;
    if(new_page)
        page.value = new_page;

    try {

        let payload = {
            ...params.value,
        };

        let paginationResponse = await loggingStore.getLogsPaginated(
            payload,
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

function applyFilters(list: FilterObj[]){
    filters.value = filterHelper.mergeFilters(filters.value, list);
    localStorage.setItem(filterStorageIndex.value, JSON.stringify(filters.value));
    getData();
    filterOverlayRef.value.hide();
}

function clearFilters(){
    filters.value = [];
    localStorage.removeItem(filterStorageIndex.value);
    cancelFilters();
    getData();
}

function cancelFilters(){
    filterOverlayRef.value.hide();
}

function removeFilter(index: number) {
    if (index < 0 || index >= filters.value.length) return;

    const next = filters.value.slice();
    next.splice(index, 1);
    filters.value = next;

    if (filters.value.length > 0) {
        localStorage.setItem(filterStorageIndex.value, JSON.stringify(filters.value));
    } else {
        localStorage.removeItem(filterStorageIndex.value);
    }

    getData();
}

function switchSort(column:string) {
    if (sort.value.field === column) {
        sort.value.order = filterHelper.toggleSort(sort.value.order);
    } else {
        sort.value.order = 1;
    }
    sort.value.field = column;
    getData();
}

function toggleFilterOverlay(event: any) {
    filterOverlayRef.value.toggle(event);
}

async function deleteConfirmation(id: number) {
    confirm.require({
        header: 'Delete record?',
        message: `This will delete record: "${id}".`,
        rejectProps: { label: 'Cancel' },
        acceptProps: { label: 'Delete', severity: 'danger' },
        accept: () => deleteRecord(id),
    });
}

async function deleteRecord(id: number) {

    if(!hasPermission("delete_activity_logs")) {
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

provide("switchSort", switchSort);
provide("removeFilter", removeFilter);

</script>

<template>

    <Popover ref="filterOverlayRef" class="rounded-popover">
        <div class="flex flex-column gap-2" style="width: 400px">
            <FilterMenu
                    v-model:value="filters"
                    :columns="activeColumns"
                    :apiSource="apiPrefix"
                    @apply="(list) => applyFilters(list)"
                    @clear="clearFilters"
                    @cancel="cancelFilters"
            />
        </div>
    </Popover>

    <main class="flex flex-column w-full p-2 align-items-center" style="height: 100vh;">

        <div class="flex flex-column justify-content-center p-3 w-full gap-3 border-round-md"
             style="border: 1px solid var(--border-color); background: var(--background-secondary); max-width: 1000px;">

            <div style="font-weight: bold;">Activity logs</div>

            <div class="flex flex-row justify-content-between align-items-center p-1 gap-3 w-full border-round-md"
                 style="border: 1px solid var(--border-color);background: var(--background-secondary);">

                <ActionRow>
                    <template #activeFilters>
                        <ActiveFilters :activeFilters="filters" :showOnlyActive="false" activeFilter="" />
                    </template>
                    <template #filterButton>
                        <div class="hover-icon flex flex-row align-items-center gap-2" @click="toggleFilterOverlay($event)"
                             style="padding: 0.5rem 1rem; border-radius: 8px; border: 1px solid var(--border-color)">
                            <i class="pi pi-filter" style="font-size: 0.845rem"></i>
                            <div>Filter</div>
                        </div>
                    </template>
                </ActionRow>
            </div>

            <div class="flex flex-row gap-2 w-full">
                <div class="w-full">
                    <DataTable class="w-full enhanced-table" dataKey="id" :loading="loadingRecords" :value="records"
                               v-model:expandedRows="expandedRows" :rowHover="true" :showGridlines="false">
                        <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
                        <template #loading> <LoadingSpinner></LoadingSpinner> </template>
                        <template #footer>
                            <CustomPaginator :paginator="paginator" :rows="rows" @onPage="onPage"/>
                        </template>

                        <Column header="Actions">
                            <template #body="slotProps">
                                <i v-if="hasPermission('delete_activity_logs')" class="pi pi-trash hover-icon" style="font-size: 0.875rem; color: var(--p-red-300);"
                                   @click="deleteConfirmation(slotProps.data?.id)"></i>
                                <i v-else class="pi pi-ban hover-icon" v-tooltip="'No action currently available.'"></i>
                            </template>
                        </Column>

                        <Column v-for="col of activeColumns" :key="col.field" :field="col.field" style="width: 25%">
                            <template #header >
                                <ColumnHeader  :header="col.header" :field="col.field" :sort="sort"></ColumnHeader>
                            </template>
                            <template #body="{ data, field }">
                                <template v-if="field === 'created_at'">
                                    {{ dateHelper.formatDate(data?.created_at, true) }}
                                </template>
                                <template v-else-if="field === 'causer_id'">
                                    {{ vueHelper.displayCauserFromId(data.causer_id, availableCausers) }}
                                </template>
                                <template v-else-if="field === 'event'">
                                    <IconDisplay :event="data.event"></IconDisplay>
                                </template>
                                <template v-else-if="field === 'category'">
                                    <span class="formal">
                                        {{ data[field] }}
                                    </span>
                                </template>
                                <template v-else>
                                    {{ data[field] }}
                                </template>
                            </template>
                        </Column>

                        <Column :expander="true" header="Metadata" style="width: 80px;"></Column>
                        <template #expansion="slotProps">
                            <div>
                                <div>
                                    <b> {{ "Description: "  }}</b>
                                    {{ slotProps.data.description ? slotProps.data?.description : "none provided" }}
                                </div>
                                <div v-if="slotProps.data?.metadata" class="truncate-text" style="max-width: 50rem;">
                                    <div v-for="item in vueHelper.formatChanges(slotProps.data?.metadata)">
                                        <label>{{ (item?.prop || '').toUpperCase() + ': ' }}</label>
                                        <span v-tooltip="vueHelper.formatValue(item)">
                                          {{ vueHelper.formatValue(item) }}
                                        </span>
                                    </div>

                                </div>
                                <div v-else>{{ "Payload is empty" }}</div>
                            </div>
                        </template>
                    </DataTable>
                </div>
            </div>
        </div>


    </main>

</template>

<style scoped>

.formal {
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    line-height: 1;
}

.truncate-text {
    margin-bottom: 8px;
    word-break: break-word;
}

</style>