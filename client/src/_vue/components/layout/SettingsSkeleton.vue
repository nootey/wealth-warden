<script setup lang="ts"></script>

<template>
  <div
    class="w-full flex flex-col gap-2 p-4 rounded-2xl border border-line bg-card shadow-[var(--shadow-card)]"
  >
    <Transition appear name="settings-animation">
      <slot />
    </Transition>
  </div>
</template>

<style scoped>
.settings-animation-enter-active {
  transition:
    opacity 0.25s ease,
    transform 0.25s ease;
}

.settings-animation-enter-from {
  opacity: 0;
  transform: translateY(6px);
}
</style>
