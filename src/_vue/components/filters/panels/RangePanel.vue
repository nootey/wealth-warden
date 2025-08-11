<script setup lang="ts">
const model = defineModel<{ min: number|null; max: number|null }>();
defineProps<{ label?: string }>();
</script>

<template>
  <div class="flex flex-column gap-2">
    <label class="text-sm">{{ label }}</label>
    <div class="flex gap-2">
      <InputNumber v-model="model.min" mode="decimal" placeholder="Min" :useGrouping="false" />
      <InputNumber v-model="model.max" mode="decimal" placeholder="Max" :useGrouping="false" />
    </div>
  </div>
</template>

<style scoped>

</style>