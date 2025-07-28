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

const filterOverlayRef = ref(null);
const loadingFilterData = ref(false);

const availableValues = ref([]);
const selectedValues = ref([]);
const optionLabel = ref("");
const displayValueAsUppercase = ref(true);

const availableEvents = ref([]);
const selectedEvents = ref([]);
const availableCausers = ref([]);
const selectedCausers = ref([]);
const availableCategories = ref([]);
const selectedCategories = ref([]);

const datetimePickerRef = ref(null);
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
      selectedValues.value = selectedCategories;
      displayValueAsUppercase.value = true;
      optionLabel.value = ""
      break;
    case "event":
      availableValues.value = availableEvents.value;
      selectedValues.value = selectedEvents;
      displayValueAsUppercase.value = true;
      optionLabel.value = ""
      break;
    case "causer":
      availableValues.value = availableCausers.value;
      selectedValues.value = selectedCausers;
      displayValueAsUppercase.value = false;
      optionLabel.value = "username"
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

function switchSort(column) {
  if (sort.value.field === column) {
    sort.value.order = vueHelper.toggleSort(sort.value.order);
  } else {
    sort.value.order = 1;
  }
  sort.value.field = column;
  getData();
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

    <div class="flex w-9 flex-column p-2 gap-3">

      <div class="flex flex-row p-1 w-full">
        <ActionRow>
          <template #dateTimePicker>
            <DateTimePicker ref="datetimePickerRef"></DateTimePicker>
            <Button class="p-button accent-button" style="border-radius: 20px;" icon="pi pi-search-plus" @click="getData(1)"></Button>
          </template>
        </ActionRow>
      </div>

      <div class="flex flex-row gap-2 w-full">
        <DataTable class="w-full" dataKey="id" :loading="loadingLogs"
                   :value="activityLogs" size="small" v-model:expandedRows="expandedRows">
          <template #empty> <div style="padding: 10px;"> No records found. </div> </template>
          <template #loading> <LoadingSpinner></LoadingSpinner> </template>
          <template #footer>
            <Paginator v-model:first="paginator.from"
                       v-model:rows="paginator.rowsPerPage"
                       :rowsPerPageOptions="rows"
                       :totalRecords="paginator.total"
                       @page="onPage($event)">
              <template #end>
                <div>
                  {{
                    "Showing " + paginator.from + " to " + paginator.to + " out of " + paginator.total + " " + "records"
                  }}
                </div>
              </template>
            </Paginator>
          </template>
          <Column header="Actions">
            <template #body="slotProps">
              <div class="flex flex-row align-items-center gap-2">
                <i class="pi pi-trash hover_icon" style="color: var(--accent-primary)"
                   @click="removeLog(slotProps.data?.id)"></i>
              </div>
            </template>
          </Column>
          <Column field="created_at">
            <template #header>
              <ColumnHeader header="Time" field="created_at" :sort="sort" :filter="false"></ColumnHeader>
            </template>
            <template #body="slotProps">
              <b>{{ dateHelper.formatDate(slotProps.data.created_at, true)}} </b>
            </template>
          </Column>
          <Column field="event" headerStyle="width:0">
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
          <Column field="category">
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
              {{ slotProps.data.category.toUpperCase() }}
            </template>
          </Column>
          <Column field="causer">
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
            <span v-if="slotProps.data?.causer_id">
              {{ vueHelper.displayCauserFromId(slotProps.data.causer_id, availableCausers) }}
            </span>
            </template>
          </Column>

          <Column :expander="true" header="Metadata"></Column>
          <template #expansion="slotProps">
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
          </template>
        </DataTable>
      </div>
    </div>

  </div>
</template>

<style scoped>

</style>