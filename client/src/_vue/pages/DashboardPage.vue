<script setup lang="ts">
import { useAuthStore } from "../../services/stores/auth_store.ts";
import { useAccountStore } from "../../services/stores/account_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import SlotSkeleton from "../components/layout/SlotSkeleton.vue";
import NetworthWidget from "../features/widgets/NetworthWidget.vue";
import { onMounted, onUnmounted, ref } from "vue";
import AccountAllocations from "../features/AccountAllocations.vue";
import { useTransactionStore } from "../../services/stores/transaction_store.ts";
import YearlyCashFlowWidget from "../features/widgets/YearlyCashFlowWidget.vue";
import MonthlyCategoryBreakdownWidget from "../features/widgets/MonthlyCategoryBreakdownWidget.vue";
import YearlySankeyWidget from "../features/widgets/YearlySankeyWidget.vue";

const authStore = useAuthStore();
const accountStore = useAccountStore();
const toastStore = useToastStore();
const transactionStore = useTransactionStore();

const nWidgetRef = ref<InstanceType<typeof NetworthWidget> | null>(null);
const backfilling = ref(false);
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

async function backfillBalances() {
  backfilling.value = true;
  try {
    const response = await accountStore.backfillBalances();
    toastStore.successResponseToast(response.data);
    nWidgetRef.value?.refresh();
  } catch (err) {
    toastStore.errorResponseToast(err);
  } finally {
    backfilling.value = false;
  }
}
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
            <div style="font-weight: bold">
              Welcome back {{ authStore?.user?.display_name }}
            </div>
            <div>Here's what's happening with your finances.</div>
          </div>
          <Button
            label="Refresh"
            icon="pi pi-refresh"
            style="height: 42px"
            class="main-button"
            :disabled="backfilling"
            @click="backfillBalances"
          />
        </div>
      </SlotSkeleton>

      <Panel :collapsed="false" header="Net worth">
        <SlotSkeleton bg="transparent">
          <NetworthWidget
            ref="nWidgetRef"
            :chart-height="400"
            :is-refreshing="backfilling"
          />
        </SlotSkeleton>
      </Panel>

      <Panel :collapsed="false" header="Yearly overview" toggleable>
        <SlotSkeleton bg="transparent">
          <YearlyCashFlowWidget :is-mobile="isMobile" />
        </SlotSkeleton>
      </Panel>

      <Panel :collapsed="false" header="Cash flow" toggleable>
        <SlotSkeleton bg="transparent">
          <YearlySankeyWidget :is-mobile="isMobile" />
        </SlotSkeleton>
      </Panel>

      <Panel :collapsed="false" header="Overview by category" toggleable>
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

      <Panel :collapsed="false" header="Balance sheet" toggleable>
        <SlotSkeleton bg="transparent">
          <AccountAllocations title="Assets" classification="asset" />
        </SlotSkeleton>

        <SlotSkeleton bg="transparent">
          <AccountAllocations title="Liabilities" classification="liability" />
        </SlotSkeleton>
      </Panel>
    </div>
  </main>
</template>

<style scoped></style>
