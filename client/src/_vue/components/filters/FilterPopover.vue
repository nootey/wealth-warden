<script setup lang="ts">
import { ref } from "vue";

const popoverRef = ref<any>(null);

function toggle(event: Event): void {
  popoverRef.value?.toggle(event);
}

function show(event: Event): void {
  popoverRef.value?.show(event);
}

function hide(): void {
  popoverRef.value?.hide();
}

defineExpose({ toggle, show, hide });
</script>

<template>
  <Popover
    ref="popoverRef"
    class="rounded-popover"
    :style="{ width: '450px' }"
    :breakpoints="{ '775px': '90vw' }"
  >
    <slot />
  </Popover>
</template>
