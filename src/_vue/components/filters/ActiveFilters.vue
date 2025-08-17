<script setup lang="ts">
import {inject, ref, watch} from "vue";
import type {FilterObj} from "../../../models/shared_models.ts";

const props = defineProps<{
  activeFilters: FilterObj[];
  showOnlyActive: boolean;
  activeFilter: string;
}>();

const removeFilter = inject<(originalIndex: number) => void>("removeFilter", () => {});
type FilterWithIndex = FilterObj & { originalIndex: number };

const filters = ref<FilterWithIndex[]>([]);

watch(
    () => [props.activeFilters, props.showOnlyActive, props.activeFilter],
    initFilters,
    { immediate: true, deep: true }
);

function initFilters() {
  const withIndex = props.activeFilters.map((f, i) => ({ ...f, originalIndex: i }));

  filters.value = props.showOnlyActive
      ? withIndex.filter(f => f.field === props.activeFilter)
      : withIndex;
}

function clearFilter(originalIndex: number): void {
  removeFilter && removeFilter(originalIndex);
  initFilters();
}

</script>

<template>
  <div v-if="filters.length > 0" class="flex flex-wrap gap-1 w-full" style="line-height: 1; max-height: 135px; overflow-y: auto;">
    <Chip v-for="filter in filters" :key="filter.originalIndex"
          style="background-color: transparent; border: 3px solid var(--border-color); padding: 0.65rem;">
      <div  class="flex flex-row align-items-center gap-2">
        <div v-tooltip="filter.field"
             style="width: 16px; height: 16px; border-radius: 50%; display: flex;
                  align-items: center; justify-content: center; font-size: 0.55rem;
                  font-weight: bold; color: white; border: 2px solid var(--border-color);">
          {{filter.field?.split(' ').map(n => n[0]).join('').toUpperCase() }}
        </div>
        <div>{{ filter.operator }}</div>
        <div>{{ filter.display ? filter.display : filter.value }}</div>
        <div
            @click="clearFilter(filter.originalIndex)"
            class="flex align-items-center justify-content-center">
          <i class="pi pi-times hover_icon" style="color: grey; font-size: 0.75rem;" ></i>
        </div>
      </div>

    </Chip>
  </div>
  <div v-else>
    <span> {{ "No filters active"}}</span>
  </div>
</template>

<style scoped>

</style>