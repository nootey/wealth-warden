<script setup lang="ts">
import SegmentedTabs from "../components/layout/SegmentedTabs.vue";
import AccountBasicStats from "../features/AccountBasicStats.vue";
import PageHeader from "../components/layout/PageHeader.vue";
import SlotSkeleton from "../components/layout/SlotSkeleton.vue";
import YearlyBreakdownStats from "../features/YearlyBreakdownStats.vue";
import NewReportModule from "../features/reports/NewReportModule.vue";
import ReportsPaginated from "../components/data/ReportsPaginated.vue";
import { ref } from "vue";

const newReportModal = ref(false);
const reportsPaginated = ref<InstanceType<typeof ReportsPaginated>>();

function onReportComplete() {
  newReportModal.value = false;
  reportsPaginated.value?.refresh();
}

const activeTab = ref("overview");
</script>

<template>
  <Dialog
    v-model:visible="newReportModal"
    class="rounded-dialog"
    :breakpoints="{ '751px': '90vw' }"
    :modal="true"
    :style="{ width: '750px' }"
    header="New Report"
  >
    <NewReportModule @complete="onReportComplete" />
  </Dialog>

  <main
    class="flex flex-col w-full items-center"
    style="padding: 0 0.5rem 0 0.5rem"
  >
    <div
      id="mobile-container"
      class="flex flex-col justify-center w-full gap-6"
    >
      <PageHeader
        eyebrow="Insights"
        title="Analytics"
        description="Comprehensive insights into your financial health."
      />

      <SegmentedTabs
        v-model="activeTab"
        :options="[
          { key: 'overview', label: 'Overview' },
          { key: 'reports', label: 'Reports' },
        ]"
      />

      <Transition name="fade" mode="out-in">
        <div
          v-if="activeTab === 'overview'"
          key="overview"
          class="flex flex-col justify-center w-full gap-4"
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
        <div v-else key="reports" class="w-full flex flex-col gap-4">
          <div class="flex flex-row justify-start">
            <Button class="main-button" @click="newReportModal = true">
              <div class="flex flex-row gap-1 items-center">
                <i class="pi pi-plus" />
                <span>New Report</span>
              </div>
            </Button>
          </div>
          <div
            class="flex flex-col w-full p-4 gap-4 rounded-2xl"
            style="
              background-color: var(--background-secondary);
              border: 1px solid var(--border-color);
            "
          >
            <span class="font-bold">Reports</span>
            <ReportsPaginated ref="reportsPaginated" />
          </div>
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
