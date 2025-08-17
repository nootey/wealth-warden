<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue';

type Model = { date: Date|null; from: Date|null; to: Date|null };
const model = defineModel<Model>({ required: true });

const props = defineProps<{
  label?: string;
  defaultDate?: Date | string | [Date|string, Date|string];
}>();

const isRange = ref(false);

onMounted(() => {
  if (props.defaultDate === undefined) return;

  if (Array.isArray(props.defaultDate)) {
    const [a, b] = props.defaultDate;
    isRange.value = true;
    model.value.date = null;
    model.value.from = toDate(a);
    model.value.to   = toDate(b);
  } else {
    isRange.value = false;
    model.value.date = toDate(props.defaultDate);
    model.value.from = null;
    model.value.to   = null;
  }
});

const selectionMode = computed(() => (isRange.value ? 'range' : 'single'));

const dpValue = computed({
  get() {
    if (isRange.value) {
      if (model.value.from || model.value.to) {
        return [model.value.from, model.value.to] as [Date|null, Date|null];
      }
      return null;
    }
    return model.value.date;
  },
  set(v) {
    if (isRange.value) {
      const [start, end] = Array.isArray(v) ? v as [Date|null, Date|null] : [null, null];
      model.value.date = null;
      model.value.from = start ?? null;
      model.value.to   = end ?? null;
    } else {
      model.value.date = (v as Date) ?? null;
      model.value.from = null;
      model.value.to   = null;
    }
  }
});

watch(isRange, (nowRange) => {
  if (nowRange) {
    if (model.value.date) {
      model.value.from = model.value.date;
      model.value.to   = model.value.date;
      model.value.date = null;
    }
  } else {
    if (model.value.from) model.value.date = model.value.from;
    model.value.from = null;
    model.value.to   = null;
  }
});

function toDate(v: unknown): Date|null {
  if (!v) return null;
  return v instanceof Date ? v : new Date(String(v));
}
</script>

<template>
  <div class="flex flex-column gap-2 w-full">
    <div class="flex flex-row w-full">
      <IftaLabel class="w-full">
        <DatePicker inputID="date" v-model="dpValue" :selectionMode="selectionMode"
            date-format="dd/mm/yy" showIcon fluid iconDisplay="input" size="small"
            placeholder="Select date" :manualInput="false"/>
        <label for="date">{{ isRange ? 'Range' : 'Single' }}</label>
      </IftaLabel>
    </div>

    <div class="flex flex-row w-full gap-1 align-items-center">
      <Checkbox v-model="isRange" :binary="true" inputId="range-picker" />
      <label for="range-picker">Range</label>
    </div>
  </div>
</template>

<style scoped>
/* your styles */
</style>
