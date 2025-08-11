<script setup lang="ts">
import {inject} from "vue";
import vueHelper from "../../../utils/vueHelper.ts";

const props = defineProps(['header', 'field', 'sort']);

const switchSort = inject<(column: string) => void>('switchSort', () => {});

</script>

<template>
  <div class="flex flex-column w-100 gap-1">
    <div class="flex flex-row header_text gap-2 align-items-center">
      {{ header }}
      <i v-if="sort" class="pi hover_icon" :class="vueHelper.sortIcon(sort, field)" @click="switchSort(field)"></i>
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
</style>