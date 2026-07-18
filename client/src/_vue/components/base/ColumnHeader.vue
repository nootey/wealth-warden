<script setup lang="ts">
import { inject } from "vue";
import type { SortObj } from "../../../models/shared_models";

defineProps<{
  header: string;
  field: string;
  sort?: SortObj;
  sortable?: boolean;
}>();

const switchSort = inject<(column: string) => void>("switchSort", () => {});
</script>

<template>
  <div class="flex flex-column">
    <div
      :class="[
        { highlight: sort && sort.field === field },
        'flex flex-row header_text align-items-center p-1',
      ]"
      :style="{ cursor: sortable ? 'pointer' : 'default' }"
      @click="sortable && switchSort(field)"
    >
      {{ header }}
    </div>
  </div>
</template>

<style scoped>
.header_text {
  position: relative;
  text-wrap: nowrap;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-weight: bold;
  cursor: pointer;
}

.header_text::after {
  content: "";
  position: absolute;
  bottom: 0;
  left: 50%;
  width: 0;
  height: 2.5px;
  background-color: var(--accent-primary);
  transition:
    width 0.3s ease,
    left 0.3s ease;
}

.highlight::after {
  width: 100%;
  left: 0;
}
</style>
