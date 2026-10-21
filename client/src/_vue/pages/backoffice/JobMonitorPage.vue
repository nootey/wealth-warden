<script setup lang="ts">
import SegmentedTabs from "../../components/layout/SegmentedTabs.vue";
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
  <main class="flex flex-col w-full items-center">
    <div
      class="flex flex-col justify-center p-4 w-full gap-4 rounded-2xl border border-line bg-card shadow-[var(--shadow-card)]"
    >
      <SegmentedTabs v-model="activeTab" :options="tabs" />

      <Transition name="fade" mode="out-in">
        <JobsPanel v-if="activeTab === 'jobs'" key="jobs" />
        <QueuesPanel v-else-if="activeTab === 'queues'" key="queues" />
        <PeriodicPanel v-else key="periodic" />
      </Transition>
    </div>
  </main>
</template>

<style scoped></style>
