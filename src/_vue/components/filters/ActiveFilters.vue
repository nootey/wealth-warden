<script setup lang="ts">
import {inject, ref, watch} from "vue";
import type {FilterObj} from "../../../models/shared_models.ts";

const props = defineProps<{
  activeFilters: FilterObj[];
  showOnlyActive: boolean;
  activeFilter: string;
}>();
const removeFilter = inject<(index: number) => void>("removeFilter", () => {});
const filters = ref<FilterObj[]>([]);

watch(
    () => props.activeFilters,
    () => {
      initFilters();
    },
    { immediate: true, deep: true }
);

function initFilters() {
  // Start by setting the filters to active filters as per props
  filters.value = [...props.activeFilters];

  if (props.showOnlyActive) {
    // Filter the array to only include the one that matches the activeFilter
    filters.value = filters.value
        .map((filter, index) => ({ ...filter, index })) // Add index to each filter
        .filter(filter => filter.parameter === props.activeFilter);
  }
}

function clearFilter(index: number): void {
  if (props.showOnlyActive) {
    removeFilter && removeFilter(index);
  } else {
    removeFilter && removeFilter(index);
  }
  initFilters();
}

function calcMaxWidth(type: any){
  let width = 10;
  switch(type){
    case "parameter": {
      width = props.showOnlyActive ? 4 : 10;
      break;
    }
    case "operator": {
      width = props.showOnlyActive ? 2 : 3;
      break;
    }
    case "value": {
      width = props.showOnlyActive ? 2 : 10;
      break;
    }
    default: break;
  }

  return `${width}rem`;
}
</script>

<template>
  <div v-if="filters.length > 0" class="flex flex-row gap-2 w-full" style="background-color: var(--background-primary); border-radius: 9px; padding: 10px;">
    <div class="flex flex-column w-full">
      <div v-for="(filter, index) in filters" class="flex flex-row gap-2 align-items-center w-full">
        <div class="flex flex-row align-items-center gap-5 w-full">
          <span class="truncate-text" v-tooltip="filter.parameter" :style="{ maxWidth: calcMaxWidth('parameter') }">{{ filter.parameter }}</span>
          <small class="truncate-text" v-tooltip="filter.operator" :style="{ maxWidth: calcMaxWidth('operator') }">{{ filter.operator}}</small>
          <span class="truncate-text" v-tooltip="filter.value" :style="{ maxWidth: calcMaxWidth('value') }">{{ filter.value}}</span>
          <i class="pi pi-times hover_icon" @click="clearFilter(index)" style="color: red;"></i>
        </div>
      </div>
    </div>
  </div>
  <div v-else>
    <span> {{ "No filters active"}}</span>
  </div>
</template>

<style scoped>

</style>