<script setup lang="ts">
import LoadingSpinner from "../../components/base/LoadingSpinner.vue";
import {computed, onMounted, provide, ref} from "vue";
import vueHelper from "../../../utils/vueHelper.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useLoggingStore} from "../../../services/stores/logging_store.ts";
import ColumnHeader from "../../components/base/ColumnHeader.vue";
import dateHelper from "../../../utils/dateHelper.ts";
import IconDisplay from "../../components/base/IconDisplay.vue";
import MultiSelectFilter from "../../components/filters/MultiSelectFilter.vue";
import dayjs from "dayjs";
import ActionRow from "../../components/layout/ActionRow.vue";
import DateTimePicker from "../../components/base/DateTimePicker.vue";
import type { Causer, FilterValue } from "../../../models/logging_models";

const toastStore = useToastStore();
const loggingStore = useLoggingStore();

const loadingLogs = ref(true);
const activityLogs = ref([]);

const params = computed(() => {
  return {
    rowsPerPage: paginator.value.rowsPerPage,
    sort: sort.value,
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
const sort = ref(vueHelper.initSort());
const expandedRows = ref([]);

const filterOverlayRef = ref<any>(null);
const loadingFilterData = ref(false);

const availableValues = ref<FilterValue[]>([]);
const selectedValues = ref<FilterValue[]>([]);
const optionLabel = ref("");
const displayValueAsUppercase = ref(true);

const availableEvents = ref<string[]>([]);
const selectedEvents = ref<string[]>([]);
const availableCategories = ref<string[]>([]);
const selectedCategories = ref<string[]>([]);
const availableCausers = ref<Causer[]>([]);
const selectedCausers = ref<Causer[]>([]);

const datetimePickerRef = ref<any>(null);
const selectedDatetimeStart = computed(() => {
  return dayjs(datetimePickerRef.value?.datetimeRange[0]).format('YYYY-MM-DD HH:mm');
});
const selectedDatetimeEnd = computed(() => {
  return dayjs(datetimePickerRef.value?.datetimeRange[1]).format('YYYY-MM-DD HH:mm');
});

function toggleFilterOverlayPanel(event: any, field: string) {
  switch (field) {
    case "category":
      availableValues.value = availableCategories.value;
      selectedValues.value = [...selectedCategories.value];
      displayValueAsUppercase.value = true;
      optionLabel.value = "";
      break;
    case "event":
      availableValues.value = availableEvents.value;
      selectedValues.value = [...selectedEvents.value];
      displayValueAsUppercase.value = true;
      optionLabel.value = "";
      break;
    case "causer":
      availableValues.value = availableCausers.value;
      selectedValues.value = [...selectedCausers.value];
      displayValueAsUppercase.value = false;
      optionLabel.value = "username";
      break;
  }
  filterOverlayRef.value.toggle(event);
}


onMounted(async () => {
  datetimePickerRef.value?.lastTwoMonths();
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

  loadingLogs.value = true;
  if(new_page)
    page.value = new_page;

  try {

    let causers = selectedCausers.value
        .map(causer => causer.id)
        .filter(id => id !== null && id !== undefined);

    let payload = {
      ...params.value,
      "categories[]": selectedCategories.value,
      "causers[]": causers,
      "events[]": selectedEvents.value,
      date_start: selectedDatetimeStart.value,
      date_stop: selectedDatetimeEnd.value
    };

    let paginationResponse = await loggingStore.getLogsPaginated(
        "activity",
        payload,
        page.value
    );

    activityLogs.value = paginationResponse.data;
    paginator.value.total = paginationResponse.total_records;
    paginator.value.to = paginationResponse.to;
    paginator.value.from = paginationResponse.from;
    loadingLogs.value = false;
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function onPage(event: any) {
  paginator.value.rowsPerPage = event.rows;
  page.value = (event.page+1)
  await getData();
}

function switchSort(column: string) {
  if (sort.value.field === column) {
    sort.value.order = vueHelper.toggleSort(sort.value.order);
  } else {
    sort.value.order = 1;
  }
  sort.value.field = column;
  getData();
}

async function removeLog(id: number) {
  try {
    // TODO: Implement log removal when backend endpoint is available
    console.log("Remove log with ID:", id);
    toastStore.successResponseToast({ data: { title: "Info", message: "Log removal not yet implemented" } });
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

provide("switchSort", switchSort);
provide("toggleFilterOverlay", null);

</script>

<template>

  <div class="flex w-full p-2">

    <Popover ref="filterOverlayRef">
      <MultiSelectFilter :availableValues="availableValues" :selectedValues="selectedValues"
                         :optionLabel="optionLabel" :toUppercase="displayValueAsUppercase ?? false"
                         @getData="getData" />
    </Popover>

    <div class="flex w-full flex-column p-2 gap-3">

      <div class="flex flex-row p-1 w-full">
        <ActionRow>
          <template #dateTimePicker>
            <DateTimePicker ref="datetimePickerRef"></DateTimePicker>
            <Button class="p-button accent-button" style="border-radius: 20px;" icon="pi pi-search-plus" @click="getData(1)"></Button>
          </template>
        </ActionRow>
      </div>

      <div class="flex flex-row gap-2 w-full">
        <div class="w-full table-container">
          <DataTable 
            class="w-full enhanced-table" 
            dataKey="id" 
            :loading="loadingLogs"
            :value="activityLogs" 
            size="small" 
            v-model:expandedRows="expandedRows"
            :rowHover="true"
            :showGridlines="false"
            style="border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);"
          >
            <template #empty> 
              <div style="padding: 20px; text-align: center; color: var(--text-secondary);"> 
                No records found. 
              </div> 
            </template> 
            <template #loading> 
              <LoadingSpinner></LoadingSpinner> 
            </template> 
            <template #footer>
              <div class="table-footer">
                <Paginator 
                  v-model:first="paginator.from"
                  v-model:rows="paginator.rowsPerPage"
                  :rowsPerPageOptions="rows"
                  :totalRecords="paginator.total"
                  @page="onPage($event)"
                  style="border-radius: 8px;"
                >
                  <template #end>
                    <div class="pagination-info">
                      {{
                        "Showing " + paginator.from + " to " + paginator.to + " out of " + paginator.total + " " + "records"
                      }}
                    </div>
                  </template>
                </Paginator>
              </div>
            </template>
            <Column header="Actions" style="width: 80px;">
              <template #body="slotProps">
                <div class="flex flex-row align-items-center gap-2">
                  <i class="pi pi-trash hover_icon action-icon" 
                     @click="removeLog(slotProps.data?.id)"></i>
                </div>
              </template>
            </Column>
            <Column field="created_at" style="width: 180px;">
              <template #header>
                <ColumnHeader header="Time" field="created_at" :sort="sort" :filter="false"></ColumnHeader>
              </template>
              <template #body="slotProps">
                <b>{{ dateHelper.formatDate(slotProps.data.created_at, true)}} </b>
              </template>
            </Column>
            <Column field="event" headerStyle="width:0" style="width: 120px;">
              <template #header>
                <ColumnHeader header="Event" field="event" :sort="sort" :filter="false">
                  <template #logging_filter>
                    <i v-if="!loadingFilterData" class="pi hover_icon" :class="'pi-filter'" @click="toggleFilterOverlayPanel($event, 'event')"></i>
                    <div v-else style="width: 13px;">
                      <ProgressSpinner animationDuration="1s" strokeWidth="8" style="width:12px;height:12px"/>
                    </div>
                  </template>
                </ColumnHeader>
              </template>
              <template #body="slotProps">
                <IconDisplay :event="slotProps.data.event"></IconDisplay>
              </template>
            </Column>
            <Column field="category" style="width: 120px;">
              <template #header>
                <ColumnHeader header="Category" field="category" :sort="sort" :filter="false">
                  <template #logging_filter>
                    <i v-if="!loadingFilterData" class="pi hover_icon" :class="'pi-filter'" @click="toggleFilterOverlayPanel($event, 'category')"></i>
                    <div v-else style="width: 13px;">
                      <ProgressSpinner animationDuration="1s" strokeWidth="8" style="width:12px;height:12px"/>
                    </div>
                  </template>
                </ColumnHeader>
              </template>
              <template #body="slotProps">
                <span class="category-badge">
                  {{ slotProps.data.category.toUpperCase() }}
                </span>
              </template>
            </Column>
            <Column field="causer" style="width: 150px;">
              <template #header>
                <ColumnHeader header="Causer" field="causer" :sort="sort" :filter="false">
                  <template #logging_filter>
                    <i v-if="!loadingFilterData" class="pi hover_icon" :class="'pi-filter'" @click="toggleFilterOverlayPanel($event, 'causer')"></i>
                    <div v-else style="width: 13px;">
                      <ProgressSpinner animationDuration="1s" strokeWidth="8" style="width:12px;height:12px"/>
                    </div>
                  </template>
                </ColumnHeader>
              </template>
              <template #body="slotProps">
                <span v-if="slotProps.data?.causer_id" class="causer-name">
                  {{ vueHelper.displayCauserFromId(slotProps.data.causer_id, availableCausers) }}
                </span>
                <span v-else class="no-causer">-</span>
              </template>
            </Column>

            <Column :expander="true" header="Metadata" style="width: 80px;"></Column>
            <template #expansion="slotProps">
              <div class="expansion-content">
                <div class="expansion-item">
                  <b> {{ "Description: "  }}</b>
                  {{ slotProps.data.description ? slotProps.data?.description : "none provided" }}
                </div>
                <div v-if="slotProps.data?.metadata" class="truncate-text" style="max-width: 50rem;"
                     v-for="item in vueHelper.formatChanges(slotProps.data?.metadata)">
                  <label class="custom-label"> {{ item?.prop.toUpperCase() + ": " }}</label>
                  <span v-tooltip="vueHelper.formatValue(item)"> {{ vueHelper.formatValue(item) }} </span>
                </div>
                <div v-else class="empty-payload">{{ "Payload is empty" }}</div>
              </div>
            </template>
          </DataTable>
        </div>
      </div>
    </div>

  </div>
</template>

<style scoped>
.table-container {
  background: var(--background-primary);
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
}

.enhanced-table {
  border-radius: 12px;
  overflow: hidden;
}

.enhanced-table :deep(.p-datatable-wrapper) {
  border-radius: 12px;
  overflow: hidden;
}

.enhanced-table :deep(.p-datatable-table) {
  border-radius: 12px;
  overflow: hidden;
}

.enhanced-table :deep(.p-datatable-thead > tr > th) {
  background: var(--background-secondary);
  border-bottom: 2px solid var(--border-color);
  padding: 16px 12px;
  font-weight: 600;
  color: var(--text-primary);
  border-radius: 0;
}

.enhanced-table :deep(.p-datatable-tbody > tr) {
  transition: background-color 0.2s ease;
  border-bottom: 1px solid var(--border-color);
}

.enhanced-table :deep(.p-datatable-tbody > tr:hover) {
  background-color: var(--background-secondary);
}

.enhanced-table :deep(.p-datatable-tbody > tr > td) {
  padding: 16px 12px;
  border: none;
  color: var(--text-primary);
}

.enhanced-table :deep(.p-datatable-tbody > tr:nth-child(even)) {
  background-color: rgba(0, 0, 0, 0.02);
}

.table-footer {
  padding: 16px;
  background: var(--background-secondary);
  border-top: 1px solid var(--border-color);
  border-radius: 0 0 12px 12px;
}

.pagination-info {
  color: var(--text-secondary);
  font-size: 0.875rem;
  padding: 8px 0;
}

.action-icon {
  color: var(--accent-primary);
  font-size: 1rem;
  padding: 8px;
  border-radius: 6px;
  transition: all 0.2s ease;
}

.action-icon:hover {
  background-color: var(--background-secondary);
  transform: scale(1.1);
}

.category-badge {
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  background-color: rgba(99, 102, 241, 0.1);
  color: rgb(99, 102, 241);
}

.causer-name {
  font-weight: 500;
  color: var(--text-primary);
}

.no-causer {
  color: var(--text-secondary);
  font-style: italic;
}

.expansion-content {
  padding: 16px;
  background: var(--background-secondary);
  border-top: 1px solid var(--border-color);
}

.expansion-item {
  margin-bottom: 12px;
  line-height: 1.5;
}

.expansion-item:last-child {
  margin-bottom: 0;
}

.empty-payload {
  color: var(--text-secondary);
  font-style: italic;
}

.custom-label {
  font-weight: 600;
  color: var(--text-primary);
  margin-right: 8px;
}

.truncate-text {
  margin-bottom: 8px;
  word-break: break-word;
}

/* Responsive adjustments */
@media (max-width: 768px) {
  .enhanced-table :deep(.p-datatable-thead > tr > th),
  .enhanced-table :deep(.p-datatable-tbody > tr > td) {
    padding: 12px 8px;
  }
  
  .table-footer {
    padding: 12px;
  }
}
</style>