<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";
import dayjs from "dayjs";

type Model = { date: Date | null; from: Date | null; to: Date | null };
const model = defineModel<Model>({ required: true });

const presets = [
  { key: "week", label: "Last week" },
  { key: "month", label: "Last month" },
  { key: "6months", label: "Last 6 months" },
  { key: "year", label: "Last year" },
];
const activePreset = ref<string | null>(null);

function applyPreset(key: string | null) {
  if (!key) return;
  const to = dayjs();
  let from = to;
  if (key === "week") from = to.subtract(1, "week");
  else if (key === "month") from = to.subtract(1, "month");
  else if (key === "6months") from = to.subtract(6, "month");
  else if (key === "year") from = to.subtract(1, "year");
  isRange.value = true;
  model.value.date = null;
  model.value.from = from.toDate();
  model.value.to = to.toDate();
}

watch(activePreset, (key) => applyPreset(key));

const props = defineProps<{
  label?: string;
  defaultDate?: Date | string | [Date | string, Date | string];
}>();

const isRange = ref(false);

onMounted(() => {
  if (props.defaultDate === undefined) return;

  if (Array.isArray(props.defaultDate)) {
    const [a, b] = props.defaultDate;
    isRange.value = true;
    model.value.date = null;
    model.value.from = toDate(a);
    model.value.to = toDate(b);
  } else {
    isRange.value = false;
    model.value.date = toDate(props.defaultDate);
    model.value.from = null;
    model.value.to = null;
  }
});

const selectionMode = computed(() => (isRange.value ? "range" : "single"));

const dpValue = computed({
  get() {
    if (isRange.value) {
      if (model.value.from || model.value.to) {
        return [model.value.from, model.value.to] as [Date | null, Date | null];
      }
      return null;
    }
    return model.value.date;
  },
  set(v) {
    if (isRange.value) {
      const [start, end] = Array.isArray(v)
        ? (v as [Date | null, Date | null])
        : [null, null];
      activePreset.value = null;
      model.value.date = null;
      model.value.from = start ?? null;
      model.value.to = end ?? null;
    } else {
      model.value.date = (v as Date) ?? null;
      model.value.from = null;
      model.value.to = null;
    }
  },
});

watch(isRange, (nowRange) => {
  if (nowRange) {
    if (model.value.date) {
      model.value.from = model.value.date;
      model.value.to = model.value.date;
      model.value.date = null;
    }
  } else {
    activePreset.value = null;
    if (model.value.from) model.value.date = model.value.from;
    model.value.from = null;
    model.value.to = null;
  }
});

function toDate(v: unknown): Date | null {
  if (!v) return null;
  return v instanceof Date ? v : new Date(String(v));
}
</script>

<template>
  <div class="flex flex-col gap-2 w-full">
    <label class="text-sm">{{ label }}</label>

    <div class="grid grid-cols-2 gap-1 w-full">
      <Button
        v-for="p in presets"
        :key="p.key"
        :label="p.label"
        fluid
        class="outline-button"
        style="max-height: 35px; font-size: small; color: var(--text-secondary)"
        @click="activePreset = p.key"
      />
    </div>

    <div class="flex flex-row w-full">
      <IftaLabel class="w-full">
        <DatePicker
          v-model="dpValue"
          input-i-d="date"
          :selection-mode="selectionMode"
          date-format="dd/mm/yy"
          show-icon
          fluid
          icon-display="input"
          size="small"
          placeholder="Select date"
          :manual-input="false"
        />
        <label for="date">{{ isRange ? "Range" : "Single" }}</label>
      </IftaLabel>
    </div>

    <div class="flex flex-row w-full gap-1 items-center">
      <Checkbox v-model="isRange" :binary="true" input-id="range-picker" />
      <label for="range-picker">Range</label>
    </div>
  </div>
</template>

<style scoped>
/* your styles */
</style>
