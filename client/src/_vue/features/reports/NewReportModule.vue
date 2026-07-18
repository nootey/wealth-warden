<script setup lang="ts">
import { ref } from "vue";
import CategoryReportForm from "./CategoryReportForm.vue";

const emit = defineEmits<{ (e: "complete"): void }>();

const selectedType = ref("");

const reportTypes = [
  {
    key: "category",
    label: "By category",
    description:
      "Combine primary and secondary categories into an effective view",
    icon: "pi-chart-bar",
    color: "#486af0",
  },
];
</script>

<template>
  <div style="min-height: 300px">
    <div
      v-if="selectedType"
      class="flex flex-row gap-2 p-2 mb-4 items-center cursor-pointer font-bold"
      style="color: var(--text-primary)"
      @click="selectedType = ''"
    >
      <i class="pi pi-angle-left" />
      <span>Back</span>
    </div>

    <Transition name="slide-down" mode="out-in">
      <div v-if="!selectedType" class="flex flex-col w-full gap-2">
        <span class="text-sm" style="color: var(--text-secondary)">
          Select a report type to generate.
        </span>
        <div
          class="flex flex-col w-full rounded-2xl p-2 gap-2"
          style="background: var(--background-secondary)"
        >
          <div
            class="flex flex-col w-full rounded-2xl p-4 gap-4"
            style="background: var(--background-primary)"
          >
            <template v-for="(type, index) in reportTypes" :key="type.key">
              <div
                v-if="index > 0"
                style="border-bottom: 2px solid var(--border-color)"
              />
              <div
                class="flex flex-row gap-2 p-2 items-center hover-icon"
                @click="selectedType = type.key"
              >
                <i
                  class="pi"
                  :class="type.icon"
                  :style="{ color: type.color }"
                />
                <div class="flex flex-col gap-1">
                  <span>{{ type.label }}</span>
                  <span class="text-xs" style="color: var(--text-secondary)">
                    {{ type.description }}
                  </span>
                </div>
                <i
                  class="pi pi-chevron-right"
                  style="margin-left: auto; color: var(--text-secondary)"
                />
              </div>
            </template>
          </div>
        </div>
      </div>

      <CategoryReportForm
        v-else-if="selectedType === 'category'"
        @complete="emit('complete')"
      />
    </Transition>
  </div>
</template>

<style scoped>
.slide-down-enter-active,
.slide-down-leave-active {
  transition: all 0.3s ease;
}

.slide-down-enter-from {
  transform: translateY(-10px);
  opacity: 0;
}

.slide-down-leave-to {
  transform: translateY(-10px);
  opacity: 0;
}
</style>
