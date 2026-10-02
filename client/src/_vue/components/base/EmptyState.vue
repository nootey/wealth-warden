<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string;
    description?: string;
    icon?: string;
  }>(),
  {
    description: undefined,
    icon: "pi pi-inbox",
  },
);
</script>

<template>
  <div
    class="flex flex-col items-center justify-center gap-2 w-full rounded-xl border border-dashed border-line p-6 text-center"
  >
    <i :class="icon" class="text-2xl text-muted" />
    <span class="text-sm text-muted">{{ title }}</span>
    <span v-if="description" class="text-xs text-faint">
      {{ description }}
    </span>
    <slot />
  </div>
</template>
