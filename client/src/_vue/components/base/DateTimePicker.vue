<script setup lang="ts">
import { computed, ref } from "vue";

const loading = ref(false);

const dateRange = ref<[Date, Date]>([setMinTime(new Date()), setMaxTime(new Date())]);
const timeRange = ref<[Date, Date]>([setMinTime(new Date()), setMaxTime(new Date())]);

const datetimeRange = computed(() => {

  const startDate = dateRange.value[0];
  const endDate = dateRange.value[1];
  const startTime = timeRange.value[0];
  const endTime = timeRange.value[1];

  if (!startDate || !startTime || !endTime) return;

  let start = new Date(startDate.getTime());
  start.setHours(startTime.getHours());
  start.setMinutes(startTime.getMinutes());

  let end = endDate !== null
    ? new Date(endDate.getTime())
    : new Date(startDate.getTime());
  end.setHours(endTime.getHours());
  end.setMinutes(endTime.getMinutes());

  return [start, end];
});

function dateHide() {
  if (dateRange.value[1] === null) {
    dateRange.value = [dateRange.value[0], dateRange.value[0]];
  }
}

function thisWeek() {
  let today = new Date();
  let lastDay = new Date(today);
  let firstDay = new Date(today.setDate(today.getDate() - today.getDay() + 1));

  dateRange.value = [setMinTime(firstDay), setMaxTime(lastDay)];
  timeRange.value = [setMinTime(firstDay), setMaxTime(lastDay)];
}

function lastWeek() {
  let today = new Date();
  let lastDay = new Date(today.setDate(today.getDate() - today.getDay()));
  let firstDay = new Date(
    today.setDate(lastDay.getDate() - lastDay.getDay() - 6),
  );
  dateRange.value = [setMinTime(firstDay), setMaxTime(lastDay)];
  timeRange.value = [setMinTime(firstDay), setMaxTime(lastDay)];
}

function lastTwoWeeks() {
  let today = new Date();
  let lastDay = new Date(today.setDate(today.getDate() - today.getDay()));
  let firstDay = new Date(
    today.setDate(lastDay.getDate() - lastDay.getDay() - 13),
  );
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
  let startDate = new Date(
    today.getFullYear(),
    today.getMonth() - 2,
    today.getDate(),
  );

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
  thisMonth,
  lastTwoMonths,
});
</script>

<template>
  <DatePicker
    v-model="dateRange"
    :disabled="loading"
    :input-style="{ 'text-align': 'center', cursor: 'pointer' }"
    :manual-input="false"
    date-format="dd/mm/yy"
    selection-mode="range"
    show-icon
    fluid
    icon-display="input"
    size="small"
    @hide="dateHide"
  >
    <template #footer>
      <div class="flex justify-content-around w-full">
        <Button size="small" label="This week" @click="thisWeek" />
        <Button size="small" label="Last 2 weeks" @click="lastTwoWeeks" />
        <Button size="small" label="This month" @click="thisMonth" />
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
