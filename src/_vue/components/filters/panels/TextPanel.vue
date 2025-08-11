<script setup lang="ts">
const model = defineModel<string | null>();
defineProps<{ label?: string }>();
</script>

<template>
  <div class="flex flex-column gap-2">
    <label class="text-sm">{{ label }}</label>
    <InputText v-model="model" placeholder="Type to search…" />
  </div>
</template>

<style scoped>

</style>