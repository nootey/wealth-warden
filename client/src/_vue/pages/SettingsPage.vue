<script setup lang="ts">
import { computed } from "vue";
import { useRoute, RouterLink } from "vue-router";
import vueHelper from "../../utils/vue_helper.ts";
import { usePermissions } from "../../utils/use_permissions.ts";
import PageHeader from "../components/layout/PageHeader.vue";

const route = useRoute();
const { hasPermission } = usePermissions();

type SettingsMenuItem = {
  name: string;
  label: string;
  icon?: string;
  block?: boolean;
  separator?: boolean;
};

const items: SettingsMenuItem[] = [
  {
    name: "settings.general",
    label: "General",
    icon: "pi-cog",
    block: !hasPermission("root_access"),
  },
  { name: "settings.profile", label: "Profile", icon: "pi-user" },
  { name: "settings.security", label: "Security", icon: "pi-shield" },
  { name: "settings.data", label: "Data", icon: "pi-eject" },
  {
    name: "",
    label: "Transactions",
    separator: true,
    block: !hasPermission("manage_data"),
  },
  {
    name: "settings.accounts",
    label: "Accounts",
    icon: "pi-book",
    block: !hasPermission("manage_data"),
  },
  {
    name: "settings.categories",
    label: "Categories",
    icon: "pi-box",
    block: !hasPermission("manage_data"),
  },
  {
    name: "settings.rules",
    label: "Rules",
    icon: "pi-filter",
    block: !hasPermission("manage_data"),
  },
  {
    name: "",
    label: "Roles",
    separator: true,
    block: !hasPermission("manage_roles"),
  },
  {
    name: "settings.roles",
    label: "Roles",
    icon: "pi-unlock",
    block: !hasPermission("manage_roles"),
  },
];

const visibleItems = computed(() => items.filter((item) => !item.block));

const isActive = (name: SettingsMenuItem["name"]) => route.name === name;

const pageTitle = computed(() => {
  if (!route.name) return "Settings";
  const parts = String(route.name).split(".");
  const last = parts[parts.length - 1];
  return vueHelper.capitalize(last);
});
</script>

<template>
  <main class="flex flex-col w-full items-center">
    <div id="mobile-container" class="flex flex-col w-full gap-6">
      <PageHeader
        eyebrow="Settings"
        :title="pageTitle"
        description="Manage your profile, data and how Wealth Warden works for you."
      />

      <div id="settings-layout" class="grid gap-8">
        <nav id="settings-nav" class="flex flex-col gap-0.5 self-start">
          <span class="label px-3 mb-2" style="color: var(--text-faint)">
            General
          </span>

          <template v-for="item in visibleItems" :key="item.name ?? item.label">
            <span
              v-if="item.separator"
              class="label px-3 mt-5 mb-2"
              style="color: var(--text-faint)"
            >
              {{ item.label }}
            </span>

            <RouterLink
              v-else
              :to="{ name: item.name }"
              class="flex items-center gap-3 h-9 px-3 shrink-0 rounded-lg text-sm font-medium no-underline transition-colors"
              :class="
                isActive(item.name!)
                  ? 'bg-card text-ink shadow-sm ring-1 ring-line'
                  : 'text-muted hover:bg-sunken hover:text-ink'
              "
            >
              <i
                class="pi text-sm"
                :class="[item.icon, isActive(item.name!) ? 'text-accent' : '']"
              />
              <span>{{ item.label }}</span>
            </RouterLink>
          </template>
        </nav>

        <section class="min-w-0 w-full" style="max-width: 880px">
          <router-view />
        </section>
      </div>
    </div>
  </main>
</template>

<style scoped>
#settings-layout {
  grid-template-columns: 200px minmax(0, 1fr);
}

#settings-nav {
  position: sticky;
  top: 2rem;
}

@media (max-width: 1000px) {
  #settings-layout {
    grid-template-columns: minmax(0, 1fr);
    gap: 1.25rem;
  }
  #settings-nav {
    position: static;
    flex-direction: row;
    overflow-x: auto;
    padding-bottom: 0.25rem;
  }
  #settings-nav > span {
    display: none;
  }
}
</style>
