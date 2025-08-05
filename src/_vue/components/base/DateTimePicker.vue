<script setup lang="ts">
import {computed, ref} from "vue";

const loading = ref(false)

const dateRange = ref([setMinTime(new Date), setMaxTime(new Date)]);
const timeRange = ref([setMinTime(new Date), setMaxTime(new Date)]);

const datetimeRange = computed(() => {
  let start = new Date(dateRange.value[0].getTime());
  start.setHours(timeRange.value[0].getHours());
  start.setMinutes(timeRange.value[0].getMinutes());

  let end = dateRange.value[1] !== null ? new Date(dateRange.value[1].getTime()) : new Date(dateRange.value[0].getTime());
  end.setHours(timeRange.value[1].getHours());
  end.setMinutes(timeRange.value[1].getMinutes());

  return [start, end];
});

function dateHide() {
  if (dateRange.value[1] === null) {
    dateRange.value = [dateRange.value[0], dateRange.value[0]];
  }
}

function thisWeek() {
  let today = new Date;
  let lastDay = new Date(today);
  let firstDay = new Date(today.setDate(today.getDate() - today.getDay() + 1));

  dateRange.value = [setMinTime(firstDay), setMaxTime(lastDay)];
  timeRange.value = [setMinTime(firstDay), setMaxTime(lastDay)];
}

function lastWeek() {
  let today = new Date;
  let lastDay = new Date(today.setDate(today.getDate() - today.getDay()));
  let firstDay = new Date(today.setDate(lastDay.getDate() - lastDay.getDay() - 6));
  dateRange.value = [setMinTime(firstDay), setMaxTime(lastDay)];
  timeRange.value = [setMinTime(firstDay), setMaxTime(lastDay)];
}

function lastTwoWeeks() {
  let today = new Date;
  let lastDay = new Date(today.setDate(today.getDate() - today.getDay()));
  let firstDay = new Date(today.setDate(lastDay.getDate() - lastDay.getDay() - 13));
  dateRange.value = [setMinTime(firstDay), setMaxTime(lastDay)];
  timeRange.value = [setMinTime(firstDay), setMaxTime(lastDay)];
}

function lastThreeWeeks() {
  let today = new Date;
  let lastDay = new Date(today.setDate(today.getDate() - today.getDay()));
  let firstDay = new Date(today.setDate(lastDay.getDate() - lastDay.getDay() - 20));
  dateRange.value = [setMinTime(firstDay), setMaxTime(lastDay)];
  timeRange.value = [setMinTime(firstDay), setMaxTime(lastDay)];
}

function thisMonth() {
  let today = new Date();
  let lastDay = new Date(today);
  let firstDay = new Date(today.getFullYear(), today.getMonth(), 1);
  dateRange.value = [setMinTime(firstDay), setMaxTime(lastDay)];
  timeRange.value = [setMinTime(firstDay), setMaxTime(lastDay)];
}

function lastTwoMonths() {
  let today = new Date();
  let endDate = new Date();
  let startDate = new Date(today.getFullYear(), today.getMonth() - 2, today.getDate());

  dateRange.value = [setMinTime(startDate), setMaxTime(endDate)];
  timeRange.value = [setMinTime(startDate), setMaxTime(endDate)];
}

function setMinTime(date: Date) {
  date.setHours(0);
  date.setMinutes(0);
  date.setSeconds(0);
  return date;
}

function setMaxTime(date: Date) {
  date.setHours(23);
  date.setMinutes(59);
  date.setSeconds(0);
  return date;
}

defineExpose({
  datetimeRange,
  thisWeek,
  lastWeek,
  lastTwoWeeks,
  lastThreeWeeks,
  thisMonth,
  lastTwoMonths
})
</script>

<template>
  <DatePicker v-model="dateRange" :disabled="loading" :inputStyle="{'text-align':'center', 'cursor':'pointer'}"
            :manualInput="false" :showIcon="false"
            dateFormat="yy-mm-dd" locale="en_GB"
            selectionMode="range" style="width:225px" @hide="dateHide">>
    <template #footer>
      <div class="flex justify-content-around w-full">
        <Button class="p-button small-button" label="This week"
                @click="thisWeek"/>
        <Button class="p-button small-button" label="Last week"
                @click="lastWeek"/>
        <Button class="p-button small-button" label="Last 2 weeks"
                @click="lastTwoWeeks"/>
        <Button class="p-button small-button" label="Last 3 weeks"
                @click="lastThreeWeeks"/>
        <Button class="p-button small-button" label="This month"
                @click="thisMonth"/>
      </div>
    </template>
  </DatePicker>
</template>

<style scoped>
.p-button {
  background: var(--accent-primary);
  border: 1px solid var(--accent-primary);
}
</style>