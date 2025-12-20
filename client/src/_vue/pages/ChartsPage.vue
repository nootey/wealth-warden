<script setup lang="ts">
import SlotSkeleton from "../components/layout/SlotSkeleton.vue";
import { onMounted } from "vue";
import MonthlyCashFlowWidget from "../features/widgets/MonthlyCashFlowWidget.vue";
import MonthlyCategoryBreakdownWidget from "../features/widgets/MonthlyCategoryBreakdownWidget.vue";
import { useTransactionStore } from "../../services/stores/transaction_store.ts";
import YearlyCashFlowWidget from "../features/widgets/YearlyCashFlowWidget.vue";

const transactionStore = useTransactionStore();

onMounted(async () => {
  await transactionStore.getCategories();
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

      <div class="w-full flex flex-row justify-content-between p-1">
        <h4>Yearly cash-flow breakdown</h4>
      </div>

      <div class="w-full flex flex-row justify-content-between p-1">
        <span style="color: var(--text-secondary)" class="text-sm">
          View a breakdown of where your cash flows over the year. This chart
          shows the flow of money for each month in the selected year, for your
          selected account. It displays how much money came in (inflows) and
          where it went—whether to expenses (outflows), investments, savings, or
          debt repayment. The remaining balance is shown as either take-home
          (positive) or overflow (negative) in the tooltip.
        </span>
      </div>

      <SlotSkeleton bg="secondary">
        <YearlyCashFlowWidget />
      </SlotSkeleton>

      <div class="w-full flex flex-row justify-content-between p-1">
        <h4>Monthly cash-flow breakdown</h4>
      </div>

      <div class="w-full flex flex-row justify-content-between p-1">
        <span style="color: var(--text-secondary)" class="text-sm">
          Track your monthly income and expenses throughout the year. This chart
          shows the flow of money in (green) and out (red) of your selected
          account, helping you identify spending patterns and seasonal trends.
        </span>
      </div>

      <SlotSkeleton bg="secondary">
        <MonthlyCashFlowWidget />
      </SlotSkeleton>

      <div class="w-full flex flex-row justify-content-between p-1">
        <h4>Monthly category display</h4>
      </div>

      <div class="w-full flex flex-row justify-content-between p-1">
        <span style="color: var(--text-secondary)" class="text-sm">
          Compare spending across categories and years. View how much you spend
          in each category by month, and see year-over-year trends to understand
          where your money goes over time.
        </span>
      </div>

      <SlotSkeleton bg="secondary">
        <MonthlyCategoryBreakdownWidget />
      </SlotSkeleton>
    </div>
  </main>
</template>

<style scoped></style>
