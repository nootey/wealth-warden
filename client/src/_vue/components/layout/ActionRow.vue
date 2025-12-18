<script setup lang="ts">
import { useSlots } from 'vue';
import type { Slots } from 'vue';

const slots: Slots = useSlots();
</script>

<template>
  <div class="flex flex-row flex-wrap p-1 gap-2 align-items-center w-full">
    <div
      v-if="slots.dateTimePicker"
      class="flex flex-column p-1"
    >
      <div class="flex flex-row gap-1 align-items-center">
        <slot name="dateTimePicker" />
      </div>
    </div>

    <div
      v-if="slots.allocation"
      class="flex flex-column p-1"
    >
      Allocation:
      <slot name="allocation" />
    </div>

    <div
      v-if="slots.activeFilters"
      class="flex flex-column p-1"
    >
      <slot name="activeFilters" />
    </div>

    <div
      v-if="slots.includeDeleted"
      class="flex flex-column p-1 ml-auto"
    >
      <slot name="includeDeleted" />
    </div>

    <div
      v-if="slots.filterButton"
      class="flex flex-column p-1"
      :class="{ 'ml-auto': !slots.includeDeleted }"
    >
      <slot name="filterButton" />
    </div>
  </div>
</template>

<style scoped>

</style>