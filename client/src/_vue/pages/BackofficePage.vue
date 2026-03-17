<script setup lang="ts">
  import {useBackofficeStore} from "../../services/stores/backoffice_store.ts";
  import {useToastStore} from "../../services/stores/toast_store.ts";

  const backofficeStore = useBackofficeStore()
  const toastStore = useToastStore()

  async function triggerAssetCashFlowSync() {
    try {
      const res = await backofficeStore.backFillAssetCashflow();
      toastStore.successResponseToast(res)
    } catch (err) {
      toastStore.errorResponseToast(err)
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

      <div class="flex flex-column gap-3">
        <div class="flex flex-column gap-1 p-3 border-1 border-round-md surface-border">
          <div style="font-weight: bold">Asset Cash Flow Backfill</div>
          <div class="text-sm text-color-secondary">
            One-time migration job. Rewrites historical investment buy trades as
            cash outflows in the balance ledger, then rebuilds all account snapshots.
            Run this once after the investment cash flow refactor to fix historical
            net worth charts. <strong>Do not run again</strong> — it will double-count
            existing cash outflows.
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

    </div>
  </main>
</template>

<style scoped>

</style>