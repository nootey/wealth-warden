<template>
  <Drawer
    id="drawer"
    v-model:visible="open"
    header="Accounts"
    position="left"
    style="width: 100%; max-width: 468px"
  >
    <template #container="{ closeCallback }">
      <div class="flex flex-col h-full overflow-y-auto bg-canvas">
        <div
          class="sticky top-0 z-10 flex flex-row items-center justify-between gap-2 px-5 py-4 border-b border-line bg-canvas"
        >
          <div class="flex flex-col gap-0.5">
            <span class="label">Balances</span>
            <span class="text-lg font-medium tracking-tight text-ink">
              Accounts
            </span>
          </div>
          <div class="flex flex-row items-center gap-1">
            <button
              v-if="hasPermission('manage_data')"
              v-tooltip.bottom="'Go to accounts settings.'"
              type="button"
              class="flex items-center justify-center w-8 h-8 rounded-lg text-muted hover:text-ink hover:bg-sunken transition-colors cursor-pointer"
              aria-label="Go to accounts settings"
              @click="() => goToAccountSettings(closeCallback)"
            >
              <i class="pi pi-external-link text-sm" />
            </button>
            <button
              type="button"
              class="flex items-center justify-center w-8 h-8 rounded-lg text-muted hover:text-ink hover:bg-sunken transition-colors cursor-pointer"
              aria-label="Close"
              @click="closeCallback"
            >
              <i class="pi pi-times text-sm" />
            </button>
          </div>
        </div>

        <div class="flex flex-col p-4">
          <AccountsPanel
            ref="accountsPanelRef"
            :advanced="false"
            :allow-edit="true"
            :max-height="86"
          />
        </div>
      </div>
    </template>
  </Drawer>
</template>

<script setup lang="ts">
import { ref } from "vue";
import router from "./services/router/main.ts";
import { usePermissions } from "./utils/use_permissions.ts";
import AccountsPanel from "./_vue/features/AccountsPanel.vue";

const open = ref(false);
const { hasPermission } = usePermissions();

const toggle = () => (open.value = !open.value);

function goToAccountSettings(closeCallback: () => void) {
  closeCallback();
  router.push({ name: "settings.accounts" });
}

defineExpose({ open, toggle });
</script>

<style scoped lang="scss">
@media (max-width: 768px) {
  #drawer {
    max-width: 100% !important;
  }
  #inner-row {
    padding: 0.75rem !important;
    margin-bottom: -7px !important;
  }
}
</style>
