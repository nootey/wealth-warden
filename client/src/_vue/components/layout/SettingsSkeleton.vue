<script setup lang="ts"></script>

<template>
  <div
    class="w-full flex flex-column gap-2 p-2 border-round-lg"
    style="
      border: 1px solid var(--border-color);
      background-color: var(--background-secondary);
    "
  >
    <Transition appear name="settings-animation">
      <slot />
    </Transition>
  </div>
</template>

<style scoped>
.settings-animation-enter-active,
.settings-animation-leave-active {
  transition: all 0.6s cubic-bezier(0.4, 0, 0.2, 1);
}

.settings-animation-enter-from {
  opacity: 0;
  transform: translateY(30px) scale(0.95);
}

.settings-animation-leave-to {
  opacity: 0;
  transform: translateY(-30px) scale(0.95);
}
</style>
