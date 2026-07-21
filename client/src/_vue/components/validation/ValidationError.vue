<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{
  message?: string;
  isRequired?: boolean;
}>();

const isDisplayed = computed(() => Boolean(props.message));
</script>

<template>
  <div
    class="flex flex-row items-center gap-1"
    :class="[isDisplayed ? 'invalid' : '']"
  >
    <div class="flex flex-col label items-center">
      <slot />
    </div>
    <small v-show="!isDisplayed && props.isRequired" class="invalid disclaimer">
      *
    </small>
    <Transition name="slide-fade">
      <span v-if="isDisplayed" class="text-xs">
        {{ props.message }}
      </span>
    </Transition>
  </div>
</template>

<style scoped>
.invalid {
  color: #dd0025;
}

.disclaimer {
  font-size: 0.8rem;
  font-style: italic;
}

.slide-fade-enter-active {
  transition:
    opacity 0.4s ease,
    transform 0.4s ease;
}
.slide-fade-enter-from {
  opacity: 0;
  transform: translateX(20px);
}
.slide-fade-enter-to {
  opacity: 1;
  transform: translateX(0);
}

.slide-fade-leave-active {
  transition:
    opacity 0.4s ease,
    transform 0.4s ease;
}
.slide-fade-leave-from {
  opacity: 1;
  transform: translateX(0);
}
.slide-fade-leave-to {
  opacity: 0;
  transform: translateX(20px);
}
</style>
