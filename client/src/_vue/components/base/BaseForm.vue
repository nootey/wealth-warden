<script setup lang="ts">
const props = defineProps<{
  disabled?: boolean;
}>();

const emit = defineEmits<{
  (event: "submit"): void;
}>();

const richWidgetSelector =
  ".p-autocomplete, .p-datepicker, .p-select, .p-multiselect, .p-cascadeselect, .p-treeselect, .p-selectbutton";

function onKeydown(event: KeyboardEvent) {
  if (event.key !== "Enter" || event.isComposing) return;
  if (event.defaultPrevented || props.disabled) return;

  const target = event.target as HTMLElement | null;
  if (!target || target.tagName !== "INPUT") return;
  if (target.closest(richWidgetSelector)) return;

  event.preventDefault();
  emit("submit");
}
</script>

<template>
  <div @keydown="onKeydown">
    <slot />
  </div>
</template>
