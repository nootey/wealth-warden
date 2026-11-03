<template>
  <Drawer
    id="drawer"
    v-model:visible="open"
    header="Monthly overview"
    position="left"
    style="width: 100%; max-width: 468px"
  >
    <template #container="{ closeCallback }">
      <div class="flex flex-col h-full overflow-y-auto bg-canvas">
        <div
          class="sticky top-0 z-10 flex flex-row items-center justify-between gap-2 px-3 py-3 md:px-5 md:py-4 border-b border-line bg-canvas"
        >
          <div class="flex flex-col gap-0.5">
            <span class="label">Activity</span>
            <span class="text-lg font-medium tracking-tight text-ink">
              Monthly overview
            </span>
          </div>
          <button
            type="button"
            class="flex items-center justify-center w-8 h-8 rounded-lg text-muted hover:text-ink hover:bg-sunken transition-colors cursor-pointer"
            aria-label="Close"
            @click="closeCallback"
          >
            <i class="pi pi-times text-sm" />
          </button>
        </div>

        <div class="flex flex-col gap-3 p-3">
          <DailyStats />
          <MonthlyStats />
          <MonthlyProjection />
        </div>
      </div>
    </template>
  </Drawer>
</template>

<script setup lang="ts">
import { ref } from "vue";
import MonthlyStats from "../components/data/MonthlyStats.vue";
import DailyStats from "../components/data/DailyStats.vue";
import MonthlyProjection from "../components/data/MonthlyProjection.vue";

const open = ref(false);

const toggle = () => (open.value = !open.value);

defineExpose({ open, toggle });
</script>

<style scoped lang="scss">
@media (max-width: 768px) {
  #drawer {
    max-width: 100% !important;
  }
}
</style>
