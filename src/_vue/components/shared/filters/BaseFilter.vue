<script setup lang="ts">
import {inject, ref} from "vue";
import dayjs from "dayjs";
import ActiveFilters from "./ActiveFilters.vue";

const props = defineProps(['filterType', 'activeColumn', 'filter', 'filters']);
const submitFilter = inject('submitFilter');

const filter_is_active = ref(false);
const operators = ref([
  {name: 'contains', value: 'like'},
  {name: 'equals', value: '='},
]);
const filteredOperators = ref([]);

props.filter.operator = props.filterType === "date" ? "equals" : "contains";
checkActiveFilter();

function checkActiveFilter() {

  props.filters.some((filter) => {
    if (filter.parameter === props.activeColumn) {
      filter_is_active.value = true;
    }
    return;
  })
}

function formatSearchText(){
  props.filter.value = props.filter.value.replace(/^\s+/, "");
}

function addActiveFilter() {
  submitFilter(props.activeColumn)
}

function selectDate(value: string | Date, enter: boolean | null = null): void {
  props.filter.value = dayjs(value).format('YYYY-MM-DD');
  if (enter) {
    submitFilter(props.activeColumn);
  }
}

function calendarDoubleClick(): void {
  const today = new Date();
  props.filter.value = dayjs(today).format('YYYY-MM-DD');
}

function checkDateValidity(event: Event): void {
  const testDate = dayjs(props.filter.value);
  if (testDate.isValid()) {
    props.filter.value = testDate.format('YYYY-MM-DD');
  }
}
const searchOperator = (event: any) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredOperators.value = [...operators.value];
    } else {
      filteredOperators.value = operators.value.filter((record) => {
        return record.name.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
}
</script>

<template>
  <div class="flex flex-column gap-2 w-100" style="background-color: var(--background-primary); border-radius: 9px; padding: 10px;">
    <AutoComplete v-if="filterType !== 'date'" v-model="filter.operator" @complete="searchOperator"
                  :suggestions="filteredOperators" optionLabel="name" size="small"
                  placeholder="Select operator" dropdown/>
    <InputText v-if="filterType === 'text'" v-model="filter.value" :placeholder="'Filter by ' + activeColumn"
               class="p-column-filter" @keydown.enter="addActiveFilter" @input="formatSearchText"/>
    <InputNumber v-if="filterType === 'number'" size="small" v-model="filter.value" mode="currency" currency="EUR"
                 locale="de-DE" placeholder="0,00 €"></InputNumber>
    <DatePicker v-if="filterType === 'date'" v-model="filter.value" :disabledDays="[0,6]" :showButtonBar="true"
              :showIcon="true" :showOnFocus="false" :showTime="false" autocomplete="off"
              class="calendar" @dblclick="calendarDoubleClick"
              @date-select="selectDate" :manualInput="true" :placeholder="'Filter by ' + activeColumn"
              dateFormat="dd.mm.yy" @keydown.enter="selectDate($event, 'enter')"
              @input="checkDateValidity($event)"/>
    <Button label='Apply' @click="addActiveFilter" class="save_button"></Button>
    <div class="flex flex-row w-100 align-items-center justify-content-center">
      <ActiveFilters :activeFilters="filters" :show-only-active="true" :active-filter="activeColumn"></ActiveFilters>
    </div>
  </div>
</template>

<style scoped>

</style>