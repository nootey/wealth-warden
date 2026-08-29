<script setup lang="ts">
import { ref } from "vue";
import JobsPanel from "./JobsPanel.vue";
import QueuesPanel from "./QueuesPanel.vue";
import PeriodicPanel from "./PeriodicPanel.vue";

const tabs = [
  { key: "jobs", label: "Jobs" },
  { key: "queues", label: "Queues" },
  { key: "periodic", label: "Scheduled" },
] as const;

const activeTab = ref<(typeof tabs)[number]["key"]>("jobs");
</script>

<template>
  <main class="flex flex-col w-full p-2 items-center">
    <div
      class="flex flex-col justify-center p-4 w-full gap-4 rounded-md"
      style="
        border: 1px solid var(--border-color);
        background: var(--background-secondary);
        max-width: 1000px;
      "
    >
      <div class="flex flex-row gap-4">
        <div
          v-for="tab in tabs"
          :key="tab.key"
          class="cursor-pointer pb-1 text-sm"
          style="color: var(--text-secondary)"
          :style="
            activeTab === tab.key
              ? 'color: var(--text-primary); border-bottom: 2px solid var(--text-primary)'
              : ''
          "
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
        </div>
      </div>

      <Transition name="fade" mode="out-in">
        <JobsPanel v-if="activeTab === 'jobs'" key="jobs" />
        <QueuesPanel v-else-if="activeTab === 'queues'" key="queues" />
        <PeriodicPanel v-else key="periodic" />
      </Transition>
    </div>
  </main>
</template>

<style scoped></style>
