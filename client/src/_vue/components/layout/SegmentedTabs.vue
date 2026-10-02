<script setup lang="ts" generic="T extends string | number">
defineProps<{ options: readonly { key: T; label: string }[] }>();

const model = defineModel<T | null>({ required: true });
</script>

<template>
  <div
    class="inline-flex self-start max-w-full overflow-x-auto gap-1 rounded-xl border border-line bg-card p-1"
    role="tablist"
  >
    <button
      v-for="tab in options"
      :key="tab.key"
      type="button"
      role="tab"
      :aria-selected="model === tab.key"
      class="h-8 px-4 shrink-0 rounded-lg text-sm font-medium whitespace-nowrap transition-colors cursor-pointer"
      :class="
        model === tab.key ? 'bg-ink text-canvas' : 'text-muted hover:text-ink'
      "
      @click="model = tab.key"
    >
      {{ tab.label }}
    </button>
  </div>
</template>
