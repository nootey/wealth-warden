<template>
  <Drawer
    id="notif-drawer"
    v-model:visible="open"
    position="left"
    style="width: 100%; max-width: 468px"
  >
    <template #container="{ closeCallback }">
      <div class="flex flex-col h-full overflow-y-auto bg-canvas">
        <div
          class="sticky top-0 z-10 flex flex-row items-center justify-between gap-2 px-5 py-4 border-b border-line bg-canvas"
        >
          <div class="flex flex-col gap-0.5">
            <span class="label">Inbox</span>
            <span class="text-lg font-medium tracking-tight text-ink">
              Notifications
            </span>
          </div>
          <button
            type="button"
            class="flex items-center justify-center w-8 h-8 rounded-lg text-muted hover:text-ink hover:bg-sunken transition-colors cursor-pointer"
            aria-label="Close"
            @click="closeCallback"
          >
            <i class="pi pi-times text-sm" />
          </button>
        </div>

        <div class="flex flex-col gap-4 p-4">
          <div class="flex flex-row items-center justify-between gap-2">
            <SegmentedTabs v-model="filter" :options="filterOptions" />
            <button
              v-if="notifications.some((n) => !n.read_at)"
              type="button"
              class="flex items-center gap-1.5 h-8 px-3 rounded-lg text-xs font-medium text-muted hover:text-ink hover:bg-sunken transition-colors cursor-pointer"
              @click="markAllAsRead"
            >
              <i class="pi pi-check text-xs" />
              <span>Mark all as read</span>
            </button>
          </div>

          <div
            v-for="n in notifications"
            :key="n.id"
            class="flex flex-row gap-3 rounded-2xl border p-4"
            :class="
              n.read_at
                ? 'border-transparent bg-sunken'
                : 'border-line bg-card shadow-[var(--shadow-card)]'
            "
          >
            <div
              class="flex items-center justify-center shrink-0 w-8 h-8 rounded-full"
              :style="{
                backgroundColor: `color-mix(in srgb, ${typeColor(n.type)}, transparent 85%)`,
                opacity: n.read_at ? 0.6 : 1,
              }"
            >
              <i
                :class="['pi', typeIcon(n.type), 'text-sm']"
                :style="{ color: typeColor(n.type) }"
              />
            </div>

            <div class="flex flex-col gap-1.5 min-w-0 flex-1">
              <div class="flex flex-row items-start justify-between gap-2">
                <span
                  class="text-sm font-medium"
                  :class="n.read_at ? 'text-muted' : 'text-ink'"
                >
                  {{ n.title }}
                </span>
                <span
                  v-if="!n.read_at"
                  class="shrink-0 mt-1.5 w-2 h-2 rounded-full bg-accent"
                />
              </div>

              <p
                class="text-sm text-muted m-0 leading-snug whitespace-pre-line"
              >
                {{ n.message }}
              </p>

              <div class="flex flex-row items-center justify-between gap-2">
                <span class="text-xs text-faint">
                  {{ dateHelper.formatDate(n.created_at) }}
                </span>
                <button
                  v-if="!n.read_at"
                  v-tooltip.top="'Mark as read'"
                  type="button"
                  class="flex items-center justify-center w-7 h-7 rounded-lg text-muted hover:text-ink hover:bg-sunken transition-colors cursor-pointer"
                  aria-label="Mark as read"
                  @click="markAsRead(n.id)"
                >
                  <i class="pi pi-check text-xs" />
                </button>
              </div>
            </div>
          </div>

          <EmptyState
            v-if="notifications.length === 0"
            icon="pi pi-bell"
            :title="
              onlyUnread ? 'No unread notifications.' : 'No notifications yet.'
            "
          />

          <SimplePaginator
            :current-page="page"
            :total-records="paginator.total"
            :rows-per-page="paginator.rowsPerPage"
            @page-change="loadNotifications"
          />
        </div>
      </div>
    </template>
  </Drawer>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from "vue";
import { useNotificationStore } from "../../services/stores/notification_store.ts";
import { useSharedStore } from "../../services/stores/shared_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { useWsStore } from "../../services/stores/ws_store.ts";
import dateHelper from "../../utils/date_helper.ts";
import SimplePaginator from "../components/base/SimplePaginator.vue";
import EmptyState from "../components/base/EmptyState.vue";
import SegmentedTabs from "../components/layout/SegmentedTabs.vue";
import type {
  Notification,
  NotificationType,
} from "../../models/notification_models.ts";
import type { PaginatorState } from "../../models/shared_models.ts";

const notificationStore = useNotificationStore();
const sharedStore = useSharedStore();
const toastStore = useToastStore();
const wsStore = useWsStore();

const open = ref(false);
const notifications = ref<Notification[]>([]);
const onlyUnread = ref(false);
const page = ref(1);
const paginator = ref<PaginatorState>({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: 5,
});

const toggle = async () => {
  open.value = !open.value;
  if (open.value) {
    await loadNotifications();
  }
};

const loadNotifications = async (page_num = 1) => {
  try {
    const response = await sharedStore.getRecordsPaginated(
      notificationStore.apiPrefix,
      {
        rowsPerPage: paginator.value.rowsPerPage,
        unread: onlyUnread.value || undefined,
      },
      page_num,
    );
    notifications.value = response.data || [];
    paginator.value.total = response.total_records ?? 0;
    paginator.value.from = response.from ?? 0;
    paginator.value.to = response.to ?? 0;
    page.value = page_num;
    notificationStore.hasUnread = notifications.value.some((n) => !n.read_at);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
};

const filterOptions = [
  { key: "all", label: "All" },
  { key: "unread", label: "Unread" },
] as const;

const filter = computed({
  get: () => (onlyUnread.value ? "unread" : "all"),
  set: (value) => {
    onlyUnread.value = value === "unread";
    loadNotifications(1);
  },
});

const markAsRead = async (id: number) => {
  try {
    await notificationStore.markAsRead(id);
    await loadNotifications(page.value);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
};

const markAllAsRead = async () => {
  try {
    await notificationStore.markAllAsRead();
    await loadNotifications(page.value);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
};

const typeIcon = (type: NotificationType) => {
  const icons: Record<NotificationType, string> = {
    info: "pi-info-circle",
    success: "pi-check-circle",
    warning: "pi-exclamation-triangle",
    error: "pi-times-circle",
  };
  return icons[type] ?? "pi-bell";
};

const typeColor = (type: NotificationType) => {
  const colors: Record<NotificationType, string> = {
    info: "var(--p-blue-400)",
    success: "var(--p-green-400)",
    warning: "var(--p-yellow-400)",
    error: "var(--p-red-400)",
  };
  return colors[type] ?? "var(--text-secondary)";
};

let unsubscribe: (() => void) | null = null;

onMounted(() => {
  notificationStore.checkUnread();
  unsubscribe = wsStore.on("notification.created", () => {
    notificationStore.checkUnread();
    if (open.value) loadNotifications(page.value);
  });
});

onUnmounted(() => unsubscribe?.());

defineExpose({ open, toggle });
</script>

<style scoped lang="scss">
@media (max-width: 768px) {
  #notif-drawer {
    max-width: 100% !important;
  }
}
</style>
