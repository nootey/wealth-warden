<script setup lang="ts">
import AccountBasicStats from "../features/AccountBasicStats.vue";
import SlotSkeleton from "../components/layout/SlotSkeleton.vue";
import YearlyBreakdownStats from "../features/YearlyBreakdownStats.vue";
import { ref } from "vue";

const activeTab = ref("overview");
</script>

<template>
  <main
    class="flex flex-column w-full align-items-center"
    style="padding: 0 0.5rem 0 0.5rem"
  >
    <div
      id="mobile-container"
      class="flex flex-column justify-content-center w-full gap-3 border-round-md"
    >
      <div
        class="w-full flex flex-row justify-content-between p-1 gap-2 align-items-center"
      >
        <div class="w-full flex flex-column gap-2">
          <div style="font-weight: bold">Analytics</div>
          <div>Comprehensive insights into your financial health.</div>
        </div>
      </div>

      <div class="flex flex-row gap-3 p-2">
        <div
          class="cursor-pointer pb-1"
          style="color: var(--text-secondary)"
          :style="
            activeTab === 'overview'
              ? 'color: var(--text-primary); border-bottom: 2px solid var(--text-primary)'
              : ''
          "
          @click="activeTab = 'overview'"
        >
          Overview
        </div>
        <div
          class="cursor-pointer pb-1"
          style="color: var(--text-secondary)"
          :style="
            activeTab === 'reports'
              ? 'color: var(--text-primary); border-bottom: 2px solid var(--text-primary)'
              : ''
          "
          @click="activeTab = 'reports'"
        >
          Reports
        </div>
      </div>

      <Transition name="fade" mode="out-in">
        <div
          v-if="activeTab === 'overview'"
          key="overview"
          class="flex flex-column justify-content-center w-full gap-3"
        >
          <Panel :collapsed="false" header="Basic" toggleable>
            <SlotSkeleton bg="transparent">
              <AccountBasicStats :pie-chart-size="200" />
            </SlotSkeleton>
          </Panel>
          <Panel :collapsed="false" header="Compare" toggleable>
            <SlotSkeleton bg="transparent">
              <YearlyBreakdownStats />
            </SlotSkeleton>
          </Panel>
        </div>
        <div v-else key="reports" class="w-full">
          <Panel :collapsed="false" header="Reports" toggleable>
            <SlotSkeleton bg="transparent">
              <span>Coming soon ...</span>
            </SlotSkeleton>
          </Panel>
        </div>
      </Transition>
    </div>
  </main>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
