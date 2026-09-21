<script setup lang="ts">
import { useSlots } from "vue";
import type { Slots } from "vue";

defineProps<{ pillsLast?: boolean }>();

const slots: Slots = useSlots();
</script>

<template>
  <div class="flex flex-row flex-wrap p-1 gap-2 items-center w-full">
    <div v-if="slots.dateTimePicker" class="flex flex-col p-1">
      <div class="flex flex-row gap-1 items-center">
        <slot name="dateTimePicker" />
      </div>
    </div>

    <div v-if="slots.allocation" class="flex flex-col p-1">
      Allocation:
      <slot name="allocation" />
    </div>

    <div
      v-if="slots.activeFilters"
      class="flex flex-col p-1 flex-1 min-w-0"
      :class="{ 'order-last': pillsLast }"
    >
      <slot name="activeFilters" />
    </div>

    <div
      v-if="slots.includeDeleted"
      class="flex flex-col p-1"
      :class="{ 'ml-auto': !pillsLast }"
    >
      <slot name="includeDeleted" />
    </div>

    <div
      v-if="slots.selectButton"
      class="flex flex-col p-1"
      :class="{
        'ml-auto': !pillsLast && !slots.includeDeleted && !slots.filterButton,
      }"
    >
      <slot name="selectButton" />
    </div>

    <div
      v-if="slots.filterButton"
      class="flex flex-col p-1"
      :class="{
        'ml-auto': !pillsLast && !slots.includeDeleted && !slots.selectButton,
      }"
    >
      <slot name="filterButton" />
    </div>
  </div>
</template>

<style scoped></style>
