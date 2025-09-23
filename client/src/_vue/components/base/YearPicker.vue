<script setup lang="ts">
import { ref, watch, computed } from "vue";

const props = defineProps<{
  records: string;
  year: number;
  availableYears: number[] | null;
}>();

const emit = defineEmits(["update:year"]); // Allow updating year from parent
const localYear = ref(props.year); // Local copy of the selected year
const filteredYears = ref<number[]>([]);

// Watch for prop changes and update local state
watch(() => props.year, (newYear) => {
  localYear.value = newYear;
});

// Watch for availableYears update
const availableYearsComputed = computed(() => props.availableYears || []);

// Handle year selection
const searchDesiredYear = (event: any) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredYears.value = [...availableYearsComputed.value];
    } else {
      filteredYears.value = availableYearsComputed.value.filter((year) =>
          year.toString().startsWith(event.query) // Convert to string before filtering
      );
    }
  }, 250);
};

// Emit the new year when selected
watch(localYear, (newYear) => {
  emit("update:year", newYear);
});

</script>

<template>
  <AutoComplete
      size="small"
      v-model="localYear"
      :suggestions="filteredYears"
      placeholder="Select Year"
      dropdown
      @complete="searchDesiredYear"
  ></AutoComplete>
</template>
