<script setup lang="ts">
import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import { useWsStore } from "../../../services/stores/ws_store.ts";
import { usePermissions } from "../../../utils/use_permissions.ts";

const wsStore = useWsStore();
const { hasPermission } = usePermissions();

const isAdmin = hasPermission("access_backoffice");
const endpoint = wsStore.endpoint();
</script>

<template>
  <div class="flex flex-column w-full gap-3">
    <SettingsSkeleton class="w-full">
      <div class="w-full flex flex-column gap-3 p-2">
        <div class="w-full flex flex-column gap-2">
          <h3>Connection</h3>
          <h5 style="color: var(--text-secondary)">
            The live connection that delivers notifications and report updates
            without a page refresh.
          </h5>
        </div>

        <div class="w-full flex flex-column gap-2">
          <div class="flex flex-row align-items-center gap-2">
            <span class="text-sm w-10rem" style="color: var(--text-secondary)"
              >Status</span
            >
            <Tag
              :severity="wsStore.connected ? 'success' : 'danger'"
              :value="wsStore.connected ? 'Connected' : 'Disconnected'"
            />
          </div>

          <div class="flex flex-row align-items-center gap-2">
            <span class="text-sm w-10rem" style="color: var(--text-secondary)"
              >Reconnect attempts</span
            >
            <span class="text-sm">{{ wsStore.attempts }}</span>
          </div>

          <div v-if="isAdmin" class="flex flex-row align-items-center gap-2">
            <span class="text-sm w-10rem" style="color: var(--text-secondary)"
              >Endpoint</span
            >
            <code class="text-sm">{{ endpoint }}</code>
          </div>
        </div>

        <div class="text-sm" style="color: var(--text-secondary)">
          If updates stop arriving, disconnect and connect again to force a
          fresh connection. The app already retries on its own with a growing
          delay, so reach for this only when it has given up.
        </div>

        <div class="text-sm" style="color: var(--text-secondary)">
          Connecting and disconnecting here only lasts for as long as this page
          is open. Refreshing the page reopens the connection automatically, and
          logging out closes it.
        </div>

        <div class="flex flex-row gap-2">
          <Button
            class="main-button"
            label="Connect"
            :disabled="wsStore.connected"
            @click="wsStore.reconnect()"
          />
          <Button
            label="Disconnect"
            severity="danger"
            :disabled="!wsStore.connected"
            @click="wsStore.disconnect(true)"
          />
        </div>
      </div>
    </SettingsSkeleton>
  </div>
</template>
