<script setup lang="ts">
import {computed} from 'vue';

const props = defineProps(['message']);

const isDisplayed = computed(() => {
  return props.message !== undefined && props.message !== null;
});
</script>

<template>
  <div class="flex flex-row align-items-center text-align-center gap-1">
    <div v-if="isDisplayed" class="p-error shake-form flex flex-column align-items-center text-align-center" v-tooltip="message">
      <i class="pi pi-info-circle"></i>
    </div>
    <div class="flex flex-column w-full text-align-center" >
      <slot></slot>
    </div>
  </div>
</template>

<style scoped>

</style>