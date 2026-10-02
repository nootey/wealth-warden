<script setup lang="ts">
import { useAuthStore } from "../../services/stores/auth_store.ts";
import { useAccountStore } from "../../services/stores/account_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import SlotSkeleton from "../components/layout/SlotSkeleton.vue";
import NetworthWidget from "../features/widgets/NetworthWidget.vue";
import { computed, onMounted, onUnmounted, ref } from "vue";
import dayjs from "dayjs";
import PageHeader from "../components/layout/PageHeader.vue";
import AccountAllocations from "../features/AccountAllocations.vue";
import { useTransactionStore } from "../../services/stores/transaction_store.ts";
import YearlyCashFlowWidget from "../features/widgets/YearlyCashFlowWidget.vue";
import MonthlyCategoryBreakdownWidget from "../features/widgets/MonthlyCategoryBreakdownWidget.vue";
import YearlySankeyWidget from "../features/widgets/YearlySankeyWidget.vue";
import GettingStartedCard from "../features/widgets/GettingStartedCard.vue";

const authStore = useAuthStore();
const accountStore = useAccountStore();
const toastStore = useToastStore();
const transactionStore = useTransactionStore();

const nWidgetRef = ref<InstanceType<typeof NetworthWidget> | null>(null);
const backfilling = ref(false);
const onboarded = ref<boolean | null>(null);
const isMobile = ref(window.innerWidth <= 768);

const DAY_PARTS = [
  { until: 12, label: "morning" },
  { until: 18, label: "afternoon" },
  { until: 24, label: "evening" },
];

const today = dayjs().format("dddd, D MMMM YYYY");
const greeting = computed(() => {
  const hour = dayjs().hour();
  const part = DAY_PARTS.find((p) => hour < p.until)!.label;
  const first = authStore.user?.display_name?.split(" ")[0];
  return first ? `Good ${part}, ${first}` : `Good ${part}`;
});

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
  <main class="flex flex-col w-full items-center">
    <div id="mobile-container" class="flex flex-col w-full gap-6">
      <PageHeader
        :eyebrow="today"
        :title="greeting"
        description="Here's where your money stands today."
      >
        <template v-if="onboarded === true" #actions>
          <Button
            class="main-button"
            :disabled="backfilling"
            @click="backfillBalances"
          >
            <div class="flex flex-row gap-2 items-center">
              <i class="pi pi-refresh" :class="{ 'pi-spin': backfilling }" />
              <span>Refresh</span>
            </div>
          </Button>
        </template>
      </PageHeader>

      <GettingStartedCard @ready="onboarded = $event" />

      <template v-if="onboarded === true">
        <div id="dash-hero" class="grid gap-6">
          <section
            class="min-w-0 rounded-2xl border border-line bg-card p-6 shadow-[var(--shadow-card)]"
          >
            <NetworthWidget
              ref="nWidgetRef"
              :chart-height="320"
              :is-refreshing="backfilling"
            />
          </section>

          <section
            class="min-w-0 flex flex-col gap-2 rounded-2xl border border-line bg-card p-6 shadow-[var(--shadow-card)]"
          >
            <div class="flex flex-col gap-1">
              <span class="text-lg font-medium tracking-tight">
                Balance sheet
              </span>
              <span class="text-sm text-muted">
                What you own against what you owe.
              </span>
            </div>
            <AccountAllocations title="Assets" classification="asset" />
            <div class="h-px bg-line" />
            <AccountAllocations
              title="Liabilities"
              classification="liability"
            />
          </section>
        </div>

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
          <p class="text-sm text-muted m-0 pb-2 max-w-3xl">
            Compare how your money moves across years and categories. Compare up
            to 5 years at a time, and filter by any income or expense category.
            Totals and averages over time include all of your data.
          </p>

          <SlotSkeleton bg="transparent">
            <MonthlyCategoryBreakdownWidget :is-mobile="isMobile" />
          </SlotSkeleton>
        </Panel>
      </template>
    </div>
  </main>
</template>

<style scoped>
#dash-hero {
  grid-template-columns: minmax(0, 2fr) minmax(0, 1fr);
}

@media (max-width: 1100px) {
  #dash-hero {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 768px) {
  #dash-hero > section {
    padding: 1.25rem;
  }
}
</style>
