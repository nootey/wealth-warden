<script setup lang="ts">
import { computed, ref } from "vue";
import { useBackofficeStore } from "../../services/stores/backoffice_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { usePermissions } from "../../utils/use_permissions.ts";
import ActivityLogsPage from "./ActivityLogsPage.vue";

const backofficeStore = useBackofficeStore();
const toastStore = useToastStore();
const { hasPermission } = usePermissions();

const tabs = [
  { key: "logs", label: "Activity Logs", permission: "view_activity_logs" },
  { key: "admin", label: "Admin", permission: "access_backoffice" },
];

const visibleTabs = computed(() =>
  tabs.filter((t) => hasPermission(t.permission)),
);

const activeTab = ref(visibleTabs.value[0]?.key ?? null);

async function triggerAssetCashFlowSync() {
  try {
    const res = await backofficeStore.backFillAssetCashflow();
    toastStore.successResponseToast(res);
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
      class="flex flex-column justify-content-center w-full gap-3 border-round-md"
    >
      <div
        class="w-full flex flex-row justify-content-between p-1 gap-2 align-items-center"
      >
        <div class="w-full flex flex-column gap-2">
          <div style="font-weight: bold">Backoffice</div>
          <div>Watch your step - fragile grounds.</div>
        </div>
      </div>

      <div class="flex flex-row gap-3 p-2">
        <div
          v-for="tab in visibleTabs"
          :key="tab.key"
          class="cursor-pointer pb-1"
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
        <div v-if="activeTab === 'logs'" key="logs">
          <ActivityLogsPage />
        </div>
        <div
          v-else-if="activeTab === null"
          key="no-access"
          class="p-2"
          style="color: var(--text-secondary)"
        >
          You don't have access to any backoffice sections.
        </div>
        <div
          v-else-if="activeTab === 'admin'"
          key="admin"
          class="flex flex-column gap-3"
        >
          <div
            class="flex flex-column gap-1 p-3 border-1 border-round-md surface-border"
          >
            <div style="font-weight: bold">Asset Cash Flow Backfill</div>
            <div class="text-sm text-color-secondary">
              One-time migration job. Rewrites historical investment buy trades
              as cash outflows in the balance ledger, then rebuilds all account
              snapshots. Run this once after the investment cash flow refactor
              to fix historical net worth charts.
              <strong>Do not run again</strong> — it will double-count existing
              cash outflows.
            </div>
            <div class="mt-2">
              <Button
                label="Run backfill"
                severity="danger"
                @click="triggerAssetCashFlowSync"
              />
            </div>
          </div>
        </div>
      </Transition>
    </div>
  </main>
</template>

<style scoped></style>
