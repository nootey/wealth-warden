<script setup lang="ts">
import {inject} from "vue";
import vueHelper from "../../../utils/vueHelper.ts";

const props = defineProps(['header', 'field', 'sort', 'filter', 'filters', 'active_filter_col']);

const switchSort = inject<(column: string) => void>('switchSort', () => {});
const toggleFilterOverlay = inject<(event: Event, field: string) => void>('toggleFilterOverlay', () => {});

function isFilterActive(field: string): boolean {
  const activeFilters = new Set<string>();

  for (const key in props.filters) {
    const filter = props.filters[key];
    if (filter.parameter === field && !activeFilters.has(field)) {
      activeFilters.add(field);
    }
  }

  return activeFilters.has(field);
}


</script>

<template>
  <div class="flex flex-column w-100 gap-1">
    <div class="flex flex-row header_text gap-2 align-items-center">
      {{ header }}
      <i v-if="sort" class="pi hover_icon" :class="vueHelper.sortIcon(sort, field)" @click="switchSort(field)"></i>
      <i v-if="filter"
         class="hover_icon pi pi-filter" :class="{'active_filter': isFilterActive(field)}" @click="toggleFilterOverlay($event, field)"/>
      <slot name="logging_filter"></slot>
    </div>

  </div>
</template>

<style scoped>
.header_text{
  text-wrap: nowrap;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.active_filter {
  color: var(--accent-primary);
}
</style>