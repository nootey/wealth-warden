<script setup lang="ts">
import vueHelper from "../../../utils/vue_helper.ts";
const props = defineProps(['validationObject']);
</script>

<template>
  <div :class="vueHelper.getValidationClass(props.validationObject, 'p-invalid')" class="flex flex-row align-items-center text-align-center gap-1">
    <slot></slot>
  </div>
</template>

<style scoped>

</style>