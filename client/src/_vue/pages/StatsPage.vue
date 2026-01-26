<script setup lang="ts">

import AccountBasicStats from "../features/AccountBasicStats.vue";
import SlotSkeleton from "../components/layout/SlotSkeleton.vue";
import YearlyCashFlowWidget from "../features/widgets/YearlyCashFlowWidget.vue";
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
        <SlotSkeleton bg="transparent">
          <div
            class="w-full flex flex-row justify-content-between p-1 gap-2 align-items-center"
          >
            <div class="w-full flex flex-column gap-2">
              <h3 style="font-weight: bold">Statistics view</h3>
              <div>Useful insignts about your finances.</div>
            </div>
          </div>
        </SlotSkeleton>

        <SlotSkeleton bg="secondary">
          <AccountBasicStats :pie-chart-size="200" />
        </SlotSkeleton>

        <Panel :collapsed="false" header="Cash-flow breakdown" toggleable>
          <div class="w-full flex flex-row justify-content-between p-1">
          <span style="color: var(--text-secondary)" class="text-sm">
            Yearly breakdown of your cash. Compare your current spending or investing habits, to any previous year.
            By default, last year will be used, if you've inputted your finances.
          </span>
          </div>

          <SlotSkeleton bg="transparent">
            TBD
          </SlotSkeleton>
        </Panel>

      </div>
    </main>
</template>

<style scoped>

</style>