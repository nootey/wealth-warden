<script setup lang="ts">
import { useSlots } from 'vue';
import type { Slots } from 'vue';

const slots: Slots = useSlots();
</script>

<template>
    <div class="flex flex-row p-1 gap-2 align-items-center w-full">

        <div class="flex flex-column p-1" v-if="slots.dateTimePicker">
            <div class="flex flex-row gap-1 align-items-center">
            <slot name="dateTimePicker"></slot>
            </div>
        </div>

        <div class="flex flex-column p-1" v-if="slots.allocation">
            Allocation:
            <slot name="allocation"></slot>
        </div>

        <div class="flex flex-column p-1" v-if="slots.activeFilters">
            <div>
            <slot name="activeFilters"></slot>
            </div>
        </div>

        <div style="margin-left: auto" class="flex flex-column p-1" v-if="slots.includeDeleted">
            <slot name="includeDeleted"></slot>
        </div>

        <div class="flex flex-column p-1" v-if="slots.filterButton">
            <slot name="filterButton"></slot>
        </div>

    </div>

</template>

<style scoped>

</style>