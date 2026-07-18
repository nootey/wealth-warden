<template>
  <Drawer
    id="drawer"
    v-model:visible="open"
    header="Accounts"
    position="left"
    style="width: 100%; max-width: 468px"
  >
    <template #container="{ closeCallback }">
      <div class="flex flex-column w-full p-2">
        <div class="flex flex-row justify-content-between p-2">
          <div class="flex flex-row align-items-center gap-2">
            <span>Accounts</span>
            <i
              v-if="hasPermission('manage_data')"
              v-tooltip="'Go to accounts settings.'"
              class="pi pi-external-link hover-icon mr-auto text-sm"
              @click="() => goToAccountSettings(closeCallback)"
            />
          </div>
          <i class="pi pi-times hover-icon" @click="closeCallback" />
        </div>

        <div class="flex flex-column w-full p-2">
          <AccountsPanel
            ref="accountsPanelRef"
            :advanced="false"
            :allow-edit="true"
            :max-height="94"
          />
        </div>
      </div>
    </template>
  </Drawer>
</template>

<script setup lang="ts">
import { ref, defineExpose } from "vue";
import router from "./services/router/main.ts";
import { usePermissions } from "./utils/use_permissions.ts";
import AccountsPanel from "./_vue/features/AccountsPanel.vue";

const open = ref(false);
const { hasPermission } = usePermissions();

const toggle = () => (open.value = !open.value);

function goToAccountSettings(closeCallback: () => void) {
  closeCallback();
  router.push("settings/accounts");
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
