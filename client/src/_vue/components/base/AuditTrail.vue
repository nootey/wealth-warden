<script setup lang="ts">
import vueHelper from "../../../utils/vue_helper.ts";
import dateHelper from "../../../utils/date_helper.ts";
import { useLoggingStore } from "../../../services/stores/logging_store.ts";
import { onMounted, ref } from "vue";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import SimplePaginator from "./SimplePaginator.vue";
import type { ActivityLog } from "../../../models/logging_models.ts";
import type { PaginatorState } from "../../../models/shared_models.ts";

const props = defineProps<{
  recordId: number | string;
  events: string[];
  categories: string[];
}>();

const loggingStore = useLoggingStore();
const toastStore = useToastStore();

const trail = ref<ActivityLog[]>([]);
const loading = ref(false);
const page = ref(1);
const paginator = ref<PaginatorState>({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: 3,
});

function formatAuditValue(item: { newVal: unknown; oldVal: unknown }): string {
  const fmt = (v: unknown): string => {
    if (typeof v === "string" && /^\d{4}-\d{2}-\d{2}T/.test(v)) {
      return dateHelper.formatDate(v, true);
    }
    return String(v ?? "");
  };

  const hasNew =
    item.newVal !== undefined && item.newVal !== null && item.newVal !== "";
  const hasOld =
    item.oldVal !== undefined && item.oldVal !== null && item.oldVal !== "";

  if (hasNew && hasOld && item.newVal !== item.oldVal) {
    return `${fmt(item.oldVal)} => ${fmt(item.newVal)}`;
  }
  return fmt(hasNew ? item.newVal : hasOld ? item.oldVal : "");
}

async function loadTrail(page_num: number) {
  if (loading.value) return;

  loading.value = true;
  try {
    const response = await loggingStore.getAuditTrail(
      props.recordId,
      props.events,
      props.categories,
      page_num,
      paginator.value.rowsPerPage,
    );

    trail.value = response.data || [];
    paginator.value.total = response.total_records;
    paginator.value.from = response.from;
    paginator.value.to = response.to;
    page.value = page_num;
  } catch (error) {
    toastStore.errorResponseToast(error);
    trail.value = [];
  } finally {
    loading.value = false;
  }
}

onMounted(async () => {
  await loadTrail(1);
});
</script>

<template>
  <Panel header="Audit trail" toggleable collapsed class="w-full">
    <div class="flex flex-col gap-2 w-full">
      <span class="text-sm" style="color: var(--text-secondary)"
        >Ordered from latest to oldest.</span
      >
      <div v-if="loading && trail.length === 0">
        <p>Loading audit trail ...</p>
      </div>
      <div v-else-if="trail.length === 0">
        <p>No audit trail found</p>
      </div>
      <div v-else class="relative w-full">
        <div
          v-if="loading"
          class="absolute inset-0 z-10 flex items-center justify-center"
        >
          <i class="pi pi-spin pi-spinner text-xl" />
        </div>
        <div
          class="flex flex-col w-full transition-opacity duration-200"
          :class="loading ? 'opacity-40' : 'opacity-100'"
        >
          <div v-for="(log, i) in trail" :key="i">
            <span
              class="ml-2 font-semibold text-sm w-full"
              style="color: var(--text-secondary)"
            >
              {{ "Log #" + (paginator.from + i) + ":" }}
            </span>
            <div class="flex flex-col w-full p-1">
              <div class="text-sm">
                Timestamp: {{ dateHelper.formatDate(log.created_at, true) }}
              </div>
              <div class="text-sm">Event: {{ log.event!.toUpperCase() }}</div>
              <div class="text-sm">Payload:</div>
              <div
                v-if="log?.metadata"
                class="truncate-text ml-4"
                style="max-width: 50rem; color: var(--text-secondary)"
              >
                <div
                  v-for="(item, index) in vueHelper.formatChanges(
                    log?.metadata,
                  )"
                  :key="index"
                >
                  <label v-if="item?.prop !== 'id'" class="text-sm">{{
                    (item?.prop || "").toUpperCase() + ": "
                  }}</label>
                  <span
                    v-if="item?.prop !== 'id'"
                    v-tooltip="formatAuditValue(item)"
                    class="text-sm"
                  >
                    {{ formatAuditValue(item) }}
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
      <SimplePaginator
        :current-page="page"
        :total-records="paginator.total"
        :rows-per-page="paginator.rowsPerPage"
        @page-change="loadTrail"
      />
    </div>
  </Panel>
</template>

<style scoped></style>
