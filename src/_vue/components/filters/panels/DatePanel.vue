<script setup lang="ts">
const model = defineModel<{ from: string|null; to: string|null }>();
defineProps<{ label?: string }>();
</script>

<template>
  <div class="flex flex-column gap-2">
    <label class="text-sm">{{ label }}</label>
    <div class="flex gap-2">
      <DatePicker v-model="model.from" dateFormat="yy-mm-dd" showIcon placeholder="From" />
      <DatePicker v-model="model.to"   dateFormat="yy-mm-dd" showIcon placeholder="To" />
    </div>
  </div>
</template>

<style scoped>

</style>