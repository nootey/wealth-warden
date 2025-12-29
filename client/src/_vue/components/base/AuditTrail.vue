<script setup lang="ts">
import type { ActivityLog } from "../../../models/logging_models.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import dateHelper from "../../../utils/date_helper.ts";
import { toUpperCase } from "uri-js/dist/esnext/util";
import { useLoggingStore } from "../../../services/stores/logging_store.ts";
import { onMounted, ref } from "vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";

const props = defineProps<{
  recordId: number | string;
  events: string[];
  category: string;
}>();

const loggingStore = useLoggingStore();
const toastStore = useToastStore();

const trail = ref<ActivityLog[]>([]);
const loading = ref(false);

onMounted(async () => {
  loading.value = true;
  try {
    const response = await loggingStore.getAuditTrail(
      props.recordId,
      props.events,
      props.category,
    );
    trail.value = response || [];
  } catch (error) {
    toastStore.errorResponseToast(error);
    trail.value = [];
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <Panel header="Audit trail" toggleable collapsed class="w-full">
    <div class="flex flex-column gap-2 w-full">
      <span class="text-sm" style="color: var(--text-secondary)"
        >Ordered from latest to oldest.</span
      >
      <div v-if="loading">
        <p>Loading audit trail ...</p>
      </div>
      <div v-else-if="trail.length === 0">
        <p>No audit trail found</p>
      </div>
      <div v-else class="flex flex-column w-full">
        <div v-for="(log, i) in trail" :key="i">
          <span
            class="ml-2 font-semibold text-sm w-full"
            style="color: var(--text-secondary)"
          >
            {{ "Log #" + (i + 1) + ":" }}
          </span>
          <div class="flex flex-column w-full p-1">
            <div class="text-sm">
              Timestamp: {{ dateHelper.formatDate(log.created_at, true) }}
            </div>
            <div class="text-sm">Event: {{ toUpperCase(log.event!) }}</div>
            <div class="text-sm">Payload:</div>
            <div
              v-if="log?.metadata"
              class="truncate-text ml-3"
              style="max-width: 50rem; color: var(--text-secondary)"
            >
              <div
                v-for="(item, index) in vueHelper.formatChanges(log?.metadata)"
                :key="index"
              >
                <label v-if="item?.prop !== 'id'" class="text-sm">{{
                  (item?.prop || "").toUpperCase() + ": "
                }}</label>
                <span
                  v-if="item?.prop !== 'id'"
                  v-tooltip="vueHelper.formatValue(item)"
                  class="text-sm"
                >
                  {{ vueHelper.formatValue(item) }}
                </span>
              </div>
            </div>
            <div v-else>
              {{ "Payload is empty" }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </Panel>
</template>

<style scoped></style>
