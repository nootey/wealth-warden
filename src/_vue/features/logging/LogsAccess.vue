<script setup lang="ts">
import LoadingSpinner from "../../components/base/LoadingSpinner.vue";
import {computed, onMounted, provide, ref} from "vue";
import vueHelper from "../../../utils/vue_helper.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useLoggingStore} from "../../../services/stores/logging_store.ts";
import ColumnHeader from "../../components/base/ColumnHeader.vue";
import dateHelper from "../../../utils/date_helper.ts";
import IconDisplay from "../../components/base/IconDisplay.vue";
import MultiSelectFilter from "../../components/filters/MultiSelectFilter.vue";
import dayjs from "dayjs";
import ActionRow from "../../components/layout/ActionRow.vue";
import DateTimePicker from "../../components/base/DateTimePicker.vue";
import type {Causer, FilterValue} from "../../../models/logging_models.ts";
import filterHelper from "../../../utils/filter_helper.ts";

const toastStore = useToastStore();
const loggingStore = useLoggingStore();

const loadingLogs = ref(true);
const accessLogs = ref([]);

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
const sort = ref(filterHelper.initSort());
const expandedRows = ref([]);

const filterOverlayRef = ref<any>(null);
const loadingFilterData = ref(false);

const availableValues = ref<FilterValue[]>([]);
const selectedValues = ref<FilterValue[]>([]);
const optionLabel = ref("");
const displayValueAsUppercase = ref(true);

const availableEvents = ref<string[]>([]);
const selectedEvents = ref<string[]>([]);
const availableCausers = ref<Causer[]>([]);
const selectedCausers = ref<Causer[]>([]);
const availableStates = ref<string[]>([]);
const selectedStates = ref<string[]>([]);

const datetimePickerRef = ref<any>(null);
const selectedDatetimeStart = computed(() => {
  return dayjs(datetimePickerRef.value?.datetimeRange[0]).format('YYYY-MM-DD HH:mm');
});
const selectedDatetimeEnd = computed(() => {
  return dayjs(datetimePickerRef.value?.datetimeRange[1]).format('YYYY-MM-DD HH:mm');
});

function toggleFilterOverlayPanel(event: any, field: string) {
  switch (field) {
    case "status":
      availableValues.value = availableStates.value;
      selectedValues.value = [...selectedStates.value];
      displayValueAsUppercase.value = true;
      optionLabel.value = ""
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
  filterOverlayRef.value.toggle(event)
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
    let response = await loggingStore.getFilterData("access");
    availableEvents.value = response.data.events;
    availableCausers.value = response.data.causers;
    availableStates.value = response.data.states;
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
      "states[]": selectedStates.value,
      "causers[]": causers,
      "events[]": selectedEvents.value,
      date_start: selectedDatetimeStart.value,
      date_stop: selectedDatetimeEnd.value
    };

    let paginationResponse = await loggingStore.getLogsPaginated(
        "access",
        payload,
        page.value
    );

    accessLogs.value = paginationResponse.data;
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
          <DataTable class="w-full enhanced-table" dataKey="id"
            :loading="loadingLogs" :value="accessLogs" size="small"
            v-model:expandedRows="expandedRows" :rowHover="true" :showGridlines="false">
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
                  style="border-radius: 8px;">
                  <template #end>
                    <div>
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
            <Column field="status" style="width: 100px;">
              <template #header>
                <ColumnHeader header="Status" field="status" :sort="sort" :filter="false">
                  <template #logging_filter>
                    <i v-if="!loadingFilterData" class="pi hover_icon" :class="'pi-filter'" @click="toggleFilterOverlayPanel($event, 'status')"></i>
                    <div v-else style="width: 13px;">
                      <ProgressSpinner animationDuration="1s" strokeWidth="8" style="width:12px;height:12px"/>
                    </div>
                  </template>
                </ColumnHeader>
              </template>
              <template #body="slotProps">
                <span class="status-badge" :class="'status-' + slotProps.data.status.toLowerCase()">
                  {{ slotProps.data.status.toUpperCase() }}
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
              <div>
                <div>
                  <b> {{ "Description: "  }}</b>
                  {{ slotProps.data.description ? slotProps.data?.description : "none provided" }}
                </div>
                <div v-if="slotProps.data?.metadata" class="truncate-text" style="max-width: 50rem;"
                     v-for="item in vueHelper.formatChanges(slotProps.data?.metadata)">
                  <label class="custom-label"> {{ item?.prop.toUpperCase() + ": " }}</label>
                  <span v-tooltip="vueHelper.formatValue(item)"> {{ vueHelper.formatValue(item) }} </span>
                </div>
                <div v-else>{{ "Payload is empty" }}</div>
              </div>
            </template>
          </DataTable>
        </div>
      </div>
    </div>

  </div>
</template>

<style scoped>

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

.status-badge {
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.status-success {
  background-color: rgba(34, 197, 94, 0.1);
  color: rgb(34, 197, 94);
}

.status-failed {
  background-color: rgba(239, 68, 68, 0.1);
  color: rgb(239, 68, 68);
}

.status-pending {
  background-color: rgba(245, 158, 11, 0.1);
  color: rgb(245, 158, 11);
}
</style>