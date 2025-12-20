<script setup lang="ts">
const model = defineModel<string | null>();
defineProps<{ label?: string }>();
const emit = defineEmits<{ (e: "commit"): void }>();
</script>

<template>
  <div class="flex flex-column gap-2 w-full">
    <label class="text-sm">{{ "Field search" }}</label>
    <InputText
      v-model="model"
      :placeholder="`Filter by ${label}`"
      style="border-radius: 8px"
      @keydown.enter.prevent="emit('commit')"
    />
  </div>
</template>

<style scoped></style>
