<template>
  <Drawer
    id="notif-drawer"
    v-model:visible="open"
    position="right"
    style="width: 100%; max-width: 468px; overflow-y: auto"
  >
    <template #container="{ closeCallback }">
      <div class="flex flex-column w-full p-3 gap-3">
        <div
          class="flex flex-row justify-content-between align-items-center p-2"
        >
          <h3>Notifications</h3>
          <i class="pi pi-times hover-icon" @click="closeCallback" />
        </div>

        <div class="flex flex-row justify-content-between align-items-center">
          <div
            class="flex flex-row align-items-center gap-2 text-sm"
            style="cursor: pointer; color: var(--text-secondary)"
            @click="toggleUnreadFilter"
          >
            <i :class="['pi', onlyUnread ? 'pi-filter-fill' : 'pi-filter']" />
            <span>{{ onlyUnread ? "Unread only" : "All" }}</span>
          </div>
          <span
            v-if="notifications.some((n) => !n.read_at)"
            class="text-xs"
            style="cursor: pointer; color: var(--text-secondary)"
            @click="markAllAsRead"
          >
            Mark all as read
          </span>
        </div>

        <SimplePaginator
          :current-page="page"
          :total-records="paginator.total"
          :rows-per-page="paginator.rowsPerPage"
          @page-change="loadNotifications"
        />

        <div
          v-for="n in notifications"
          :key="n.id"
          class="p-3 border-round-xl"
          :style="{
            backgroundColor: n.read_at
              ? 'var(--background-secondary)'
              : 'var(--background-primary)',
            border: '1px solid var(--border-color)',
          }"
        >
          <div class="flex flex-column gap-2">
            <div class="flex flex-row align-items-center gap-2">
              <i
                :class="['pi', typeIcon(n.type)]"
                :style="{ color: typeColor(n.type), fontSize: '0.9rem' }"
              />
              <span
                class="font-semibold text-sm"
                :style="{
                  color: n.read_at
                    ? 'var(--text-secondary)'
                    : 'var(--text-primary)',
                }"
              >
                {{ n.title }}
              </span>
            </div>

            <p
              class="text-sm m-0"
              style="
                color: var(--text-secondary);
                line-height: 1.4;
                white-space: pre-line;
              "
            >
              {{ n.message }}
            </p>

            <div
              class="flex flex-row justify-content-between align-items-center text-xs"
              style="color: var(--text-secondary)"
            >
              <span>{{ dateHelper.formatDate(n.created_at) }}</span>
              <i
                v-if="!n.read_at"
                v-tooltip="'Mark as read'"
                class="pi pi-check-square text-sm hover-icon"
                @click="markAsRead(n.id)"
              />
            </div>
          </div>
        </div>

        <div
          v-if="notifications.length === 0"
          class="text-center p-4"
          style="color: var(--text-secondary)"
        >
          {{ onlyUnread ? "No unread notifications" : "No notifications yet" }}
        </div>
      </div>
    </template>
  </Drawer>
</template>

<script setup lang="ts">
import { ref, defineExpose, onMounted } from "vue";
import { useNotificationStore } from "../../services/stores/notification_store.ts";
import { useSharedStore } from "../../services/stores/shared_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import type {
  Notification,
  NotificationType,
} from "../../models/notification_models.ts";
import type { PaginatorState } from "../../models/shared_models.ts";
import dateHelper from "../../utils/date_helper.ts";
import SimplePaginator from "../components/base/SimplePaginator.vue";

const notificationStore = useNotificationStore();
const sharedStore = useSharedStore();
const toastStore = useToastStore();

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

const toggleUnreadFilter = async () => {
  onlyUnread.value = !onlyUnread.value;
  await loadNotifications(1);
};

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

onMounted(() => notificationStore.checkUnread());

defineExpose({ open, toggle });
</script>

<style scoped lang="scss">
@media (max-width: 768px) {
  #notif-drawer {
    max-width: 100% !important;
  }
}
</style>
