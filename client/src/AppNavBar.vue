<script setup lang="ts">
import { useAuthStore } from "./services/stores/auth_store.ts";
import { storeToRefs } from "pinia";
import { computed, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { usePermissions } from "./utils/use_permissions.ts";
import { useConfirm } from "primevue/useconfirm";
import NotesSideBar from "./_vue/features/NotesSideBar.vue";

defineProps<{ hasUnread: boolean }>();

const emit = defineEmits<{
  openAccounts: [];
  openStats: [];
  openNotifications: [];
}>();

const collapsed = defineModel<boolean>("collapsed", { default: false });

const authStore = useAuthStore();
const { user } = storeToRefs(authStore);
const { hasPermission } = usePermissions();

const route = useRoute();
const router = useRouter();
const confirm = useConfirm();

const notesRef = ref<InstanceType<typeof NotesSideBar> | null>(null);
const profileMenuRef = ref<any>(null);

interface NavItem {
  to: string;
  icon: string;
  text: string;
  short: string;
}

const navItems: NavItem[] = [
  { to: "/", icon: "pi-th-large", text: "Dashboard", short: "Home" },
  {
    to: "/transactions",
    icon: "pi-arrow-right-arrow-left",
    text: "Transactions",
    short: "Activity",
  },
  { to: "/goals", icon: "pi-flag", text: "Goals", short: "Goals" },
  {
    to: "/investments",
    icon: "pi-chart-line",
    text: "Investments",
    short: "Invest",
  },
  { to: "/analytics", icon: "pi-chart-bar", text: "Analytics", short: "Stats" },
];

interface ActionItem {
  icon: string;
  text: string;
  action: () => void;
  permission?: string;
  danger?: boolean;
}

const quickViews = computed<ActionItem[]>(() => [
  { icon: "pi-wallet", text: "Accounts", action: () => emit("openAccounts") },
  {
    icon: "pi-calendar",
    text: "Monthly overview",
    action: () => emit("openStats"),
  },
  {
    icon: "pi-bookmark",
    text: "Notes",
    action: () => notesRef.value?.toggle(),
  },
]);

const settingsItem: ActionItem = {
  icon: "pi-cog",
  text: "Settings",
  action: () => router.push("/settings"),
};

const backofficeItem: ActionItem = {
  icon: "pi-briefcase",
  text: "Backoffice",
  permission: "access_backoffice",
  action: () => router.push("/backoffice"),
};

const signOut: ActionItem = {
  icon: "pi-sign-out",
  text: "Sign out",
  danger: true,
  action: () =>
    confirm.require({
      message: "Are you sure you want to sign out?",
      header: "Sign out",
      rejectLabel: "Cancel",
      acceptLabel: "Sign out",
      accept: () => authStore.logoutUser(),
    }),
};

const visibleSystemItems = computed(() =>
  [backofficeItem, settingsItem].filter(
    (item) => !item.permission || hasPermission(item.permission),
  ),
);

// Quick views live in the sidebar on desktop, so the popover shows them on mobile only.
const menuGroups = computed(() => [
  { id: undefined, items: visibleSystemItems.value },
  { id: "profile-quick-views", items: quickViews.value },
  { id: undefined, items: [signOut] },
]);

const initials = computed(() =>
  (user.value?.display_name ?? "")
    .split(" ")
    .filter(Boolean)
    .map((n) => n[0])
    .slice(0, 2)
    .join("")
    .toUpperCase(),
);

function isActive(to: string) {
  return to === "/" ? route.path === "/" : route.path.startsWith(to);
}

function isSystemActive(item: ActionItem) {
  if (item.text === "Settings") return route.path.startsWith("/settings");
  if (item.text === "Backoffice") return route.path.startsWith("/backoffice");
  return false;
}

function toggleProfileMenu(event: Event) {
  profileMenuRef.value?.toggle(event);
}

function handleMenuClick(item: ActionItem) {
  profileMenuRef.value?.hide();
  item.action();
}
</script>

<template>
  <NotesSideBar ref="notesRef" />

  <template v-if="authStore.authenticated && authStore.isValidated">
    <aside
      id="app-sidebar"
      class="fixed inset-y-0 left-0 z-30 flex flex-col border-r border-line bg-canvas transition-[width] duration-200 ease-out"
      :style="{ width: collapsed ? '76px' : '248px' }"
    >
      <div
        class="flex items-center h-16 shrink-0 gap-3"
        :class="collapsed ? 'justify-center px-0' : 'px-5'"
      >
        <router-link
          to="/"
          class="size-9 shrink-0 rounded-xl grid place-items-center shadow-sm logo-mark"
        >
          <img
            src="./assets/images/logo.png"
            alt="Wealth Warden"
            class="w-5 brightness-0 opacity-85"
          />
        </router-link>
        <div v-if="!collapsed" class="flex flex-col leading-tight min-w-0">
          <span class="text-[1.05rem] font-semibold tracking-tight text-ink">
            Wealth Warden
          </span>
          <span class="text-[0.7rem] text-muted">Personal ledger</span>
        </div>
      </div>

      <nav class="flex flex-col gap-0.5 mt-3 px-3 overflow-y-auto">
        <router-link
          v-for="item in navItems"
          :key="item.to"
          v-tooltip.right="collapsed ? item.text : null"
          :to="item.to"
          class="flex items-center gap-3 h-10 rounded-lg text-sm font-medium no-underline transition-colors"
          :class="[
            collapsed ? 'justify-center' : 'px-3',
            isActive(item.to)
              ? 'bg-card text-ink shadow-sm ring-1 ring-line'
              : 'text-muted hover:bg-sunken hover:text-ink',
          ]"
        >
          <i
            class="pi text-[0.95rem]"
            :class="[item.icon, isActive(item.to) ? 'text-accent' : '']"
          />
          <span v-if="!collapsed">{{ item.text }}</span>
        </router-link>

        <span
          v-if="!collapsed"
          class="label mt-6 mb-2 px-3"
          style="color: var(--text-faint)"
        >
          Quick views
        </span>
        <div v-else class="h-px bg-line my-4 mx-2" />

        <button
          v-for="item in quickViews"
          :key="item.text"
          v-tooltip.right="collapsed ? item.text : null"
          type="button"
          class="flex items-center gap-3 h-10 rounded-lg text-sm font-medium text-muted hover:bg-sunken hover:text-ink transition-colors cursor-pointer"
          :class="collapsed ? 'justify-center' : 'px-3'"
          @click="item.action"
        >
          <i class="pi text-[0.95rem]" :class="item.icon" />
          <span v-if="!collapsed">{{ item.text }}</span>
        </button>

        <button
          v-tooltip.right="collapsed ? 'Notifications' : null"
          type="button"
          class="flex items-center gap-3 h-10 rounded-lg text-sm font-medium text-muted hover:bg-sunken hover:text-ink transition-colors cursor-pointer"
          :class="collapsed ? 'justify-center' : 'px-3'"
          @click="emit('openNotifications')"
        >
          <span class="relative inline-flex">
            <i class="pi pi-bell text-[0.95rem]" />
            <span
              v-if="hasUnread"
              class="absolute -top-0.5 -right-0.5 size-2 rounded-full ring-2 ring-canvas"
              style="background: var(--negative)"
            />
          </span>
          <span v-if="!collapsed">Notifications</span>
        </button>
      </nav>

      <div class="mt-auto flex flex-col gap-0.5 px-3 pb-3">
        <button
          v-for="item in visibleSystemItems"
          :key="item.text"
          v-tooltip.right="collapsed ? item.text : null"
          type="button"
          class="flex items-center gap-3 h-10 rounded-lg text-sm font-medium transition-colors cursor-pointer"
          :class="[
            collapsed ? 'justify-center' : 'px-3',
            isSystemActive(item)
              ? 'bg-card text-ink shadow-sm ring-1 ring-line'
              : 'text-muted hover:bg-sunken hover:text-ink',
          ]"
          @click="item.action"
        >
          <i
            class="pi text-[0.95rem]"
            :class="[item.icon, isSystemActive(item) ? 'text-accent' : '']"
          />
          <span v-if="!collapsed">{{ item.text }}</span>
        </button>

        <button
          v-tooltip.right="collapsed ? 'Expand' : null"
          type="button"
          class="flex items-center gap-3 h-10 rounded-lg text-sm text-muted hover:bg-sunken hover:text-ink transition-colors cursor-pointer"
          :class="collapsed ? 'justify-center' : 'px-3'"
          @click="collapsed = !collapsed"
        >
          <i
            class="pi text-[0.95rem]"
            :class="
              collapsed ? 'pi-angle-double-right' : 'pi-angle-double-left'
            "
          />
          <span v-if="!collapsed">Collapse</span>
        </button>

        <div class="h-px bg-line my-2" />

        <button
          type="button"
          class="flex items-center gap-3 rounded-xl p-2 hover:bg-sunken transition-colors cursor-pointer text-left"
          :class="collapsed ? 'justify-center' : ''"
          @click="toggleProfileMenu($event)"
        >
          <span
            class="size-8 shrink-0 rounded-full grid place-items-center text-xs font-semibold bg-ink text-canvas"
          >
            {{ initials }}
          </span>
          <span v-if="!collapsed" class="flex flex-col min-w-0 flex-1">
            <span class="text-sm font-medium text-ink truncate">
              {{ user?.display_name }}
            </span>
            <span class="text-xs text-muted truncate">{{ user?.email }}</span>
          </span>
          <i v-if="!collapsed" class="pi pi-ellipsis-v text-xs text-muted" />
        </button>
      </div>
    </aside>

    <header
      id="mobile-topbar"
      class="fixed top-0 inset-x-0 z-30 h-14 items-center justify-between px-4 border-b border-line bg-canvas/90 backdrop-blur"
    >
      <router-link to="/" class="flex items-center gap-2 no-underline">
        <span class="size-8 rounded-lg grid place-items-center logo-mark">
          <img
            src="./assets/images/logo.png"
            alt="Wealth Warden"
            class="w-4 brightness-0 opacity-85"
          />
        </span>
        <span class="font-semibold tracking-tight text-ink">
          Wealth Warden
        </span>
      </router-link>
      <div class="flex items-center gap-1">
        <button
          type="button"
          class="size-9 grid place-items-center rounded-lg text-muted hover:bg-sunken cursor-pointer"
          aria-label="Accounts"
          @click="emit('openAccounts')"
        >
          <i class="pi pi-wallet" />
        </button>
        <button
          type="button"
          class="relative size-9 grid place-items-center rounded-lg text-muted hover:bg-sunken cursor-pointer"
          aria-label="Notifications"
          @click="emit('openNotifications')"
        >
          <i class="pi pi-bell" />
          <span
            v-if="hasUnread"
            class="absolute top-2 right-2 size-2 rounded-full"
            style="background: var(--negative)"
          />
        </button>
      </div>
    </header>

    <nav
      id="mobile-tabbar"
      class="fixed bottom-0 inset-x-0 z-30 h-16 items-stretch justify-around border-t border-line bg-canvas/95 backdrop-blur"
      style="padding-bottom: env(safe-area-inset-bottom)"
    >
      <router-link
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        class="flex flex-1 flex-col items-center justify-center gap-1 no-underline text-[0.68rem] font-medium"
        :class="isActive(item.to) ? 'text-ink' : 'text-muted'"
      >
        <span
          class="grid place-items-center h-7 w-12 rounded-full transition-colors"
          :class="isActive(item.to) ? 'bg-accent-soft text-accent' : ''"
        >
          <i class="pi text-[0.95rem]" :class="item.icon" />
        </span>
        {{ item.short }}
      </router-link>
      <button
        type="button"
        class="flex flex-1 flex-col items-center justify-center gap-1 text-[0.68rem] font-medium text-muted cursor-pointer"
        @click="toggleProfileMenu($event)"
      >
        <span
          class="size-7 rounded-full grid place-items-center text-[0.6rem] font-semibold bg-ink text-canvas"
        >
          {{ initials }}
        </span>
        You
      </button>
    </nav>

    <Popover ref="profileMenuRef" class="rounded-popover">
      <div class="flex flex-col w-64">
        <div class="flex items-center gap-3 p-2 pb-3 border-b border-line">
          <span
            class="size-10 shrink-0 rounded-full grid place-items-center text-sm font-semibold bg-ink text-canvas"
          >
            {{ initials }}
          </span>
          <div class="flex flex-col min-w-0">
            <span class="font-medium text-ink truncate">
              {{ user?.display_name }}
            </span>
            <span class="text-xs text-muted truncate">{{ user?.email }}</span>
          </div>
        </div>

        <div class="flex flex-col gap-0.5 pt-2">
          <div
            v-for="(group, i) in menuGroups"
            :id="group.id"
            :key="i"
            class="flex flex-col gap-0.5"
          >
            <button
              v-for="item in group.items"
              :key="item.text"
              type="button"
              class="flex items-center gap-3 h-9 px-2 rounded-lg text-sm hover:bg-sunken transition-colors cursor-pointer text-left"
              :style="{
                color: item.danger ? 'var(--negative)' : 'var(--text-primary)',
              }"
              @click="handleMenuClick(item)"
            >
              <i class="pi text-sm" :class="item.icon" />
              <span>{{ item.text }}</span>
            </button>
          </div>
        </div>
      </div>
    </Popover>
  </template>
</template>

<style scoped>
#mobile-topbar,
#mobile-tabbar,
#profile-quick-views {
  display: none;
}

@media (max-width: 768px) {
  #app-sidebar {
    display: none;
  }
  #mobile-topbar,
  #mobile-tabbar,
  #profile-quick-views {
    display: flex;
  }
}
</style>
