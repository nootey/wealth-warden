<script setup lang="ts">
import vueHelper from "../../../utils/vue_helper.ts";
import dateHelper from "../../../utils/date_helper.ts";
import CustomPaginator from "../base/CustomPaginator.vue";
import ColumnHeader from "../base/ColumnHeader.vue";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import type {Transaction} from "../../../models/transaction_models.ts";
import type {Column} from "../../../services/filter_registry.ts";
import {computed, onMounted, ref, watch} from "vue";
import filterHelper from "../../../utils/filter_helper.ts";
import type { SortObj } from "../../../models/shared_models.ts";

const props = defineProps<{
    columns: Column[];
    sort?: SortObj;
    rows?: number[];
    filters?: any;
    include_deleted?: boolean;
    fetchPage: (args: {
        page: number;
        rows: number;
        sort?: SortObj;
        filters?: any;
        include_deleted?: boolean;
    }) => Promise<{ data: Transaction[]; total: number }>;
    readOnly: boolean;
}>();

const emits = defineEmits<{
    (e: "onPage", payload: { page: number; rows: number }): void;
    (e: "sortChange", column: string): void;
    (e: "rowClick", id: number): void;
}>();

const rowsOptions = computed(() => props.rows ?? [25, 50, 100]);
const pageLocal = ref(1);
const rowsPerPage = ref(rowsOptions.value[0]);
const total = ref(0);
const localSort = ref<SortObj>(props.sort ?? filterHelper.initSort());

const recordsLocal = ref<Transaction[]>([]);
const loading = ref(false);
const requestSeq = ref(0);

const derivedPaginator = computed(() => {
    const from = total.value === 0 ? 0 : (pageLocal.value - 1) * rowsPerPage.value + 1;
    const to = Math.min(pageLocal.value * rowsPerPage.value, total.value);
    return { total: total.value, from, to, rowsPerPage: rowsPerPage.value };
});

async function getData() {
    loading.value = true;
    const mySeq = ++requestSeq.value;

    try {
        const res = await props.fetchPage({
            page: pageLocal.value,
            rows: rowsPerPage.value,
            sort: localSort.value,
            filters: props.filters,
            include_deleted: props.include_deleted,
        });

        // Ignore stale responses
        if (mySeq !== requestSeq.value) return;

        recordsLocal.value = res.data;
        total.value = res.total ?? 0;
    } finally {
        if (mySeq === requestSeq.value) loading.value = false;
    }
}

onMounted(getData);

watch(
    () => [props.sort?.field, props.sort?.order, props.filters, props.include_deleted],
    () => { pageLocal.value = 1; getData(); }
);

function handlePage(e: { page: number; rows: number }) {
    pageLocal.value = e.page + 1;
    rowsPerPage.value = e.rows;
    emits("onPage", { page: pageLocal.value, rows: rowsPerPage.value });
    getData();
}

function triggerSort(col: string) {
    emits("sortChange", col);
}

function refresh() { getData(); }

defineExpose({ refresh });

</script>

<template>
    <DataTable class="w-full enhanced-table" dataKey="id"
               :loading="loading" :value="recordsLocal" scrollable scrollHeight="50vh"
               :rowClass="vueHelper.deletedRowClass" columnResizeMode="fit"
               scrollDirection="both">
        <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
        <template #loading> <LoadingSpinner></LoadingSpinner> </template>
        <template #footer>
            <CustomPaginator :paginator="derivedPaginator" :rows="rowsOptions" @onPage="handlePage"/>
        </template>

        <Column v-for="col of columns" :key="col.field" :field="col.field"
                :headerClass="col.hideOnMobile ? 'mobile-hide ' : ''"
                :bodyClass="col.hideOnMobile ? 'mobile-hide ' : ''">
            <template #header >
                <ColumnHeader :header="col.header" :field="col.field" :sort="localSort"
                              :sortable="!!sort"
                              @click="!sort && triggerSort(col.field as string)">
                </ColumnHeader>
            </template>
            <template #body="{ data, field }">
                <template v-if="field === 'amount'">
                    {{ vueHelper.displayAsCurrency(data.transaction_type == "expense" ? (data.amount*-1) : data.amount) }}
                </template>
                <template v-else-if="field === 'created_at'">
                    {{ dateHelper.formatDate(data?.created_at, true) }}
                </template>
                <template v-else-if="field === 'account'">
                    <div class="flex flex-row gap-2 align-items-center account-row">
                        <span class="hover" @click="$emit('rowClick', data.id)">
                            {{ data[field]["name"] }}
                        </span>
                        <i v-if="data[field]['deleted_at']" class="pi pi-ban popup-icon hover-icon" v-tooltip="'This account is closed.'"/>
                    </div>
                </template>
                <template v-else-if="field === 'category'">
                    {{ data[field]["display_name"] }}
                </template>
                <template v-else>
                    {{ data[field] }}
                </template>
            </template>
        </Column>
    </DataTable>
</template>

<style scoped>
.hover { font-weight: bold; }
.hover:hover { cursor: pointer; text-decoration: underline; }

.account-row .popup-icon {
    opacity: 0;
    transition: opacity .15s ease;
}
.account-row:hover .popup-icon {
    opacity: 1;
}

.account-row.advanced .popup-icon {
    opacity: 1;
}

</style>