<script setup lang="ts">
import { useAuthStore } from "../../services/stores/auth_store.ts";
import { useAccountStore } from "../../services/stores/account_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import SlotSkeleton from "../components/layout/SlotSkeleton.vue";
import NetworthWidget from "../features/widgets/NetworthWidget.vue";
import { ref } from "vue";
import AccountAllocations from "../features/AccountAllocations.vue";
import AccountBasicStats from "../features/AccountBasicStats.vue";

const authStore = useAuthStore();
const accountStore = useAccountStore();
const toastStore = useToastStore();

const nWidgetRef = ref<InstanceType<typeof NetworthWidget> | null>(null);

async function backfillBalances() {
  try {
    const response = await accountStore.backfillBalances();
    toastStore.successResponseToast(response.data);
    nWidgetRef.value?.refresh();
  } catch (err) {
    toastStore.errorResponseToast(err);
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
      class="flex flex-column justify-content-center w-full gap-2 border-round-md"
    >
      <SlotSkeleton bg="transparent">
        <div
          class="w-full flex flex-row justify-content-between p-1 gap-2 align-items-center"
        >
          <div class="w-full flex flex-column gap-2">
            <div style="font-weight: bold">
              Welcome back {{ authStore?.user?.display_name }}
            </div>
            <div>{{ "Here's what's happening with your finances." }}</div>
          </div>
          <Button
            label="Refresh"
            icon="pi pi-refresh"
            style="height: 42px"
            class="main-button"
            @click="backfillBalances"
          />
        </div>
      </SlotSkeleton>

      <SlotSkeleton bg="secondary">
        <NetworthWidget ref="nWidgetRef" :chart-height="400" />
      </SlotSkeleton>

      <div class="w-full flex flex-row justify-content-between p-2 gap-2">
        <h3>Account allocation by type</h3>
      </div>

      <SlotSkeleton bg="secondary">
        <AccountAllocations title="Assets" classification="asset" />
      </SlotSkeleton>

      <SlotSkeleton bg="secondary">
        <AccountAllocations title="Liabilities" classification="liability" />
      </SlotSkeleton>

      <div class="w-full flex flex-row justify-content-between p-2 gap-2">
        <h3>Stats</h3>
      </div>

      <SlotSkeleton bg="secondary">
        <AccountBasicStats :pie-chart-size="300" />
      </SlotSkeleton>
    </div>
  </main>
</template>

<style scoped></style>
