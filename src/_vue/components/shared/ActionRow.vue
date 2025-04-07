<script setup lang="ts">
import {useSlots} from "vue";

const slots = useSlots()
</script>

<template>
  <div class="flex flex-row p-1 gap-2 align-items-center">

    <div class="flex flex-column p-1" v-if="slots.yearPicker">
      Select year:
      <slot name="yearPicker"></slot>
    </div>

    <div class="flex flex-column p-1" v-if="slots.allocation">
      Allocation:
      <slot name="allocation"></slot>
    </div>

    <div class="flex flex-column p-1" v-if="slots.activeFilters">
      Active filters:
      <div>
        <slot name="activeFilters"></slot>
      </div>
    </div>

  </div>

</template>

<style scoped>

</style>