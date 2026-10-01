<script setup lang="ts"></script>

<template>
  <div class="relative min-h-screen w-full flex bg-canvas text-ink">
    <div
      class="absolute inset-0 opacity-[0.06] pointer-events-none"
      style="
        background-image:
          linear-gradient(currentColor 1px, transparent 1px),
          linear-gradient(90deg, currentColor 1px, transparent 1px);
        background-size: 32px 32px;
        mask-image: radial-gradient(
          ellipse at center,
          #000 30%,
          transparent 75%
        );
      "
    />
    <Transition appear name="auth-animation">
      <div
        class="relative flex-1 flex flex-col justify-center items-center px-4 sm:px-6 py-12"
      >
        <div class="w-full" style="max-width: 464px">
          <div
            class="rounded-2xl border border-line bg-card overflow-hidden shadow-[var(--shadow-card)]"
          >
            <div
              class="flex flex-col items-center gap-2 px-6 py-5 text-neutral-900"
              style="
                background: linear-gradient(135deg, #ffffff 0%, #c4c4c4 100%);
              "
            >
              <img
                src="../../../assets/images/logo.png"
                alt="Wealth Warden"
                class="w-12 brightness-0"
              />
              <span class="text-xl font-bold tracking-tight">
                WealthWarden
              </span>
            </div>
            <div class="p-6 sm:p-8">
              <slot />
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.auth-animation-enter-active {
  transition:
    opacity 0.4s ease,
    transform 0.4s ease;
}

.auth-animation-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
</style>
