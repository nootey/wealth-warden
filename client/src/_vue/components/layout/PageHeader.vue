<script setup lang="ts">
defineProps<{
  title: string;
  eyebrow?: string;
  description?: string;
}>();
</script>

<template>
  <header class="w-full flex flex-row items-end justify-between gap-4 pb-2">
    <div class="flex flex-col gap-1.5 min-w-0 flex-1">
      <span v-if="eyebrow" class="label">{{ eyebrow }}</span>
      <div class="flex items-center gap-2">
        <h1
          class="text-[2rem] leading-tight font-medium tracking-tight text-ink m-0"
        >
          {{ title }}
        </h1>
        <slot name="title-suffix" />
      </div>
      <p v-if="description" class="text-sm text-muted m-0 max-w-xl">
        {{ description }}
      </p>
    </div>
    <div
      v-if="$slots.actions"
      class="flex flex-row items-center gap-2 shrink-0"
    >
      <slot name="actions" />
    </div>
  </header>
</template>
