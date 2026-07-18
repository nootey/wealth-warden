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

async function triggerCorrectFeeAccounting() {
  try {
    const res = await backofficeStore.correctFeeAccounting();
    toastStore.successResponseToast(res);
  } catch (err) {
    toastStore.errorResponseToast(err);
  }
}

async function runZeroCostMigration() {
  try {
    const res = await backofficeStore.migrateZeroCostTrades();
    toastStore.successResponseToast(res);
  } catch (err) {
    toastStore.errorResponseToast(err);
  }
}
</script>

<template>
  <main
    class="flex flex-col w-full items-center"
    style="padding: 0 0.5rem 0 0.5rem"
  >
    <div
      id="mobile-container"
      class="flex flex-col justify-center w-full gap-4 rounded-md"
    >
      <div class="w-full flex flex-row justify-between p-1 gap-2 items-center">
        <div class="w-full flex flex-col gap-2">
          <div style="font-weight: bold">Backoffice</div>
          <div>Watch your step - fragile grounds.</div>
        </div>
      </div>

      <div class="flex flex-row gap-4 p-2">
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
          class="flex flex-col gap-4"
        >
          <div class="flex flex-col gap-1 p-4 border rounded-md border-surface">
            <div style="font-weight: bold">Asset Cash Flow Backfill</div>
            <div class="text-sm text-muted-color">
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

          <div class="flex flex-col gap-1 p-4 border rounded-md border-surface">
            <div style="font-weight: bold">Correct Fee Accounting</div>
            <div class="text-sm text-muted-color">
              One-time correction for stock/ETF buy trades. Fixes
              <code>value_at_buy</code> from <code>qty*price-fee</code> to
              <code>qty*price+fee</code>, recalculates asset aggregates and
              average buy prices, then rebuilds all cash flows and snapshots.
              Run once after the fee accounting fix. Safe to re-run - idempotent
              on already-corrected trades.
            </div>
            <div class="mt-2">
              <Button
                label="Run correction"
                severity="danger"
                @click="triggerCorrectFeeAccounting"
              />
            </div>
          </div>

          <div class="flex flex-col gap-1 p-4 border rounded-md border-surface">
            <div style="font-weight: bold">Zero-Cost Trade Migration</div>
            <div class="text-sm text-muted-color">
              Migrates all buy trades with a zero price per unit to investment
              income. Crypto assets are classified as staking rewards;
              stocks/ETFs as dividends.
            </div>
            <div class="mt-2">
              <Button
                label="Run migration"
                severity="danger"
                @click="runZeroCostMigration"
              />
            </div>
          </div>
        </div>
      </Transition>
    </div>
  </main>
</template>

<style scoped></style>
