<script setup lang="ts">
import SlotSkeleton from "../components/layout/SlotSkeleton.vue";
import { onMounted, onUnmounted, ref } from "vue";
import MonthlyCashFlowWidget from "../features/widgets/MonthlyCashFlowWidget.vue";
import MonthlyCategoryBreakdownWidget from "../features/widgets/MonthlyCategoryBreakdownWidget.vue";
import { useTransactionStore } from "../../services/stores/transaction_store.ts";
import YearlyCashFlowWidget from "../features/widgets/YearlyCashFlowWidget.vue";
import YearlySankeyWidget from "../features/widgets/YearlySankeyWidget.vue";

const transactionStore = useTransactionStore();
const isMobile = ref(window.innerWidth <= 768);
const handleResize = () => {
  isMobile.value = window.innerWidth <= 768;
};

onMounted(async () => {
  await transactionStore.getCategories();
  window.addEventListener("resize", handleResize);
});

onUnmounted(() => {
  window.removeEventListener("resize", handleResize);
});
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
            <h3 style="font-weight: bold">Chart view</h3>
            <div>A better look at your cash-flow.</div>
          </div>
        </div>
      </SlotSkeleton>

      <Panel :collapsed="false" header="Cash-flow breakdown" toggleable>
        <div class="w-full flex flex-row justify-content-between p-1">
          <span style="color: var(--text-secondary)" class="text-sm">
            View a breakdown of where your cash flows over the year. This chart
            shows the flow of money for each month in the selected year, for
            your selected account. It displays how much money came in (inflows)
            and where it went—whether to expenses (outflows), investments,
            savings, or debt repayment. The remaining balance is shown as either
            take-home (positive) or overflow (negative) in the tooltip.
          </span>
        </div>

        <SlotSkeleton bg="transparent">
          <YearlyCashFlowWidget :is-mobile="isMobile" />
        </SlotSkeleton>
      </Panel>

      <Panel
        :collapsed="false"
        header="Comparative breakdown by category"
        toggleable
      >
        <div class="w-full flex flex-row justify-content-between p-1">
          <span style="color: var(--text-secondary)" class="text-sm">
            View and compare how your money moves through out different years
            and categories. You can compare up to 5 years at a time, with the
            option to filter by any income or expense category. Totals and
            average over time include ALL of your data.
          </span>
        </div>

        <SlotSkeleton bg="transparent">
          <MonthlyCategoryBreakdownWidget :is-mobile="isMobile" />
        </SlotSkeleton>
      </Panel>

      <Panel :collapsed="false" header="Sankey cash-flow breakdown" toggleable>
        <div class="w-full flex flex-row justify-content-between p-1">
          <span style="color: var(--text-secondary)" class="text-sm">
            Visualize how your income flows through each year. This Sankey
            diagram shows the complete journey of your money - from total income
            on the left, through major allocations like savings, investments,
            and debt repayments, to a detailed breakdown of your expense
            categories on the right.
          </span>
        </div>

        <SlotSkeleton bg="transparent">
          <YearlySankeyWidget :is-mobile="isMobile" />
        </SlotSkeleton>
      </Panel>

      <Panel :collapsed="true" header="Cash-flow pattern" toggleable>
        <div class="w-full flex flex-row justify-content-between p-1">
          <span style="color: var(--text-secondary)" class="text-sm">
            Track your monthly income and expenses throughout the year. This
            chart is the most basic representation of cash-flow. It shows the
            flow of money in (green) and out (red) of your selected account,
            helping you identify spending patterns and seasonal trends.
          </span>
        </div>

        <SlotSkeleton bg="transparent">
          <MonthlyCashFlowWidget :is-mobile="isMobile" />
        </SlotSkeleton>
      </Panel>
    </div>
  </main>
</template>

<style scoped></style>
