<script setup lang="ts">
import { useSharedStore } from "../../services/stores/shared_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { computed, onMounted, provide, ref } from "vue";
import { useUserStore } from "../../services/stores/user_store.ts";
import type { Role, User } from "../../models/user_models.ts";
import filterHelper from "../../utils/filter_helper.ts";
import type { Column } from "../../services/filter_registry.ts";
import type {FilterObj, PaginatorState} from "../../models/shared_models.ts";
import FilterMenu from "../components/filters/FilterMenu.vue";
import ActiveFilters from "../components/filters/ActiveFilters.vue";
import ActionRow from "../components/layout/ActionRow.vue";
import dateHelper from "../../utils/date_helper.ts";
import LoadingSpinner from "../components/base/LoadingSpinner.vue";
import ColumnHeader from "../components/base/ColumnHeader.vue";
import CustomPaginator from "../components/base/CustomPaginator.vue";
import UserForm from "../components/forms/UserForm.vue";
import InvitationsPaginated from "../components/data/InvitationsPaginated.vue";
import { useRouter } from "vue-router";
import { usePermissions } from "../../utils/use_permissions.ts";

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const userStore = useUserStore();
const { hasPermission } = usePermissions();

onMounted(async () => {
  await userStore.getRoles();
});

const router = useRouter();
const apiPrefix = userStore.apiPrefix;

const createModal = ref(false);
const updateModal = ref(false);
const updateUserID = ref(null);

const loading = ref(false);
const records = ref<User[]>([]);
const roles = computed<Role[]>(() => userStore.roles);

const params = computed(() => {
  return {
    rowsPerPage: paginator.value.rowsPerPage,
    sort: sort.value,
    filters: filters.value,
  };
});

const rows = ref([10, 25, 50, 100]);
const default_rows = ref(rows.value[0]);
const paginator = ref<PaginatorState>({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: default_rows.value!,
});
const page = ref(1);
const sort = ref(filterHelper.initSort());
const filterStorageIndex = ref(apiPrefix + "-filters");
const filters = ref(
  JSON.parse(localStorage.getItem(filterStorageIndex.value) ?? "[]"),
);
const filterOverlayRef = ref<any>(null);

const activeColumns = computed<Column[]>(() => [
  { field: "display_name", header: "Name", type: "text" },
  { field: "email", header: "Email", type: "text" },
  {
    field: "role",
    header: "Role",
    type: "enum",
    options: roles.value,
    optionLabel: "name",
  },
  { field: "email_confirmed", header: "Verified", type: "date" },
]);

const invRef = ref<InstanceType<typeof InvitationsPaginated> | null>(null);

onMounted(async () => {
  await init();
});

async function init() {
  await getData();
}

async function getData(new_page: number | null = null) {
  loading.value = true;
  if (new_page) page.value = new_page;

  try {
    let payload = {
      ...params.value,
    };

    let paginationResponse = await sharedStore.getRecordsPaginated(
      apiPrefix,
      payload,
      page.value,
    );

    records.value = paginationResponse.data;
    paginator.value.total = paginationResponse.total_records;
    paginator.value.to = paginationResponse.to;
    paginator.value.from = paginationResponse.from;
    loading.value = false;
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

function manipulateDialog(modal: string, value: any) {
  switch (modal) {
    case "inviteUser": {
      createModal.value = value;
      break;
    }
    case "updateUser": {
      updateModal.value = true;
      updateUserID.value = value;
      break;
    }
    default: {
      break;
    }
  }
}

async function handleEmit(emitType: any) {
  switch (emitType) {
    case "completeOperation": {
      createModal.value = false;
      updateModal.value = false;
      await getData();
      invRef.value?.refresh();
      break;
    }
    case "deleteUser": {
      createModal.value = false;
      updateModal.value = false;
      await getData();
      break;
    }
    default: {
      break;
    }
  }
}

async function onPage(event: any) {
  paginator.value.rowsPerPage = event.rows;
  page.value = event.page + 1;
  await getData();
}

function applyFilters(list: FilterObj[]) {
  filters.value = filterHelper.mergeFilters(filters.value, list);
  localStorage.setItem(filterStorageIndex.value, JSON.stringify(filters.value));
  getData();
  filterOverlayRef.value.hide();
}

function clearFilters() {
  filters.value = [];
  localStorage.removeItem(filterStorageIndex.value);
  cancelFilters();
  getData();
}

function cancelFilters() {
  filterOverlayRef.value.hide();
}

function removeFilter(index: number) {
  if (index < 0 || index >= filters.value.length) return;

  const next = filters.value.slice();
  next.splice(index, 1);
  filters.value = next;

  if (filters.value.length > 0) {
    localStorage.setItem(
      filterStorageIndex.value,
      JSON.stringify(filters.value),
    );
  } else {
    localStorage.removeItem(filterStorageIndex.value);
  }

  getData();
}

function switchSort(column: string) {
  if (sort.value.field === column) {
    sort.value.order = filterHelper.toggleSort(sort.value.order);
  } else {
    sort.value.order = 1;
  }
  sort.value.field = column;
  getData();
}

function toggleFilterOverlay(event: any) {
  filterOverlayRef.value.toggle(event);
}

provide("switchSort", switchSort);
provide("removeFilter", removeFilter);
</script>

<template>
  <Popover
    ref="filterOverlayRef"
    class="rounded-popover"
    :style="{ width: '420px' }"
    :breakpoints="{ '775px': '90vw' }"
  >
    <FilterMenu
      v-model:value="filters"
      :columns="activeColumns"
      :api-source="apiPrefix"
      @apply="(list) => applyFilters(list)"
      @clear="clearFilters"
      @cancel="cancelFilters"
    />
  </Popover>

  <Dialog
    v-model:visible="createModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Invite user"
  >
    <UserForm
      mode="create"
      :roles="roles"
      @complete-operation="handleEmit('completeOperation')"
    />
  </Dialog>

  <Dialog
    v-model:visible="updateModal"
    position="right"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="User details"
  >
    <UserForm
      mode="update"
      :roles="roles"
      :record-id="updateUserID"
      @complete-operation="handleEmit('completeOperation')"
      @complete-user-delete="handleEmit('deleteUser')"
    />
  </Dialog>

  <main class="flex flex-column w-full p-2 align-items-center">
    <div
      id="mobile-container"
      class="flex flex-column justify-content-center p-3 w-full gap-3 border-round-md"
      style="
        border: 1px solid var(--border-color);
        background: var(--background-secondary);
      "
    >
      <div
        class="flex flex-row justify-content-between align-items-center text-center gap-2 w-full"
      >
        <div style="font-weight: bold">Users</div>

        <i
          v-if="hasPermission('manage_roles')"
          v-tooltip="'Go to roles settings.'"
          class="pi pi-external-link hover-icon mr-auto text-sm"
          @click="router.push('settings/roles')"
        />
        <Button
          class="main-button"
          @click="manipulateDialog('inviteUser', true)"
        >
          <div class="flex flex-row gap-1 align-items-center">
            <i class="pi pi-plus" />
            <span> New </span>
            <span class="mobile-hide"> User </span>
          </div>
        </Button>
      </div>

      <div
        class="flex flex-row justify-content-between align-items-center p-1 gap-3 w-full border-round-md"
        style="
          border: 1px solid var(--border-color);
          background: var(--background-secondary);
        "
      >
        <ActionRow>
          <template #activeFilters>
            <ActiveFilters
              :active-filters="filters"
              :show-only-active="false"
              active-filter=""
            />
          </template>
          <template #filterButton>
            <div
              class="hover-icon flex flex-row align-items-center gap-2"
              style="
                padding: 0.5rem 1rem;
                border-radius: 8px;
                border: 1px solid var(--border-color);
              "
              @click="toggleFilterOverlay($event)"
            >
              <i class="pi pi-filter" style="font-size: 0.845rem" />
              <div>Filter</div>
            </div>
          </template>
        </ActionRow>
      </div>

      <div id="mobile-row" class="flex flex-row gap-2 w-full">
        <DataTable
          class="w-full enhanced-table"
          data-key="id"
          :loading="loading"
          :value="records"
          :row-hover="true"
          :show-gridlines="false"
          scrollable
          column-resize-mode="fit"
        >
          <template #empty>
            <div style="padding: 10px">No records found.</div>
          </template>
          <template #loading>
            <LoadingSpinner />
          </template>
          <template #footer>
            <CustomPaginator
              :paginator="paginator"
              :rows="rows"
              @on-page="onPage"
            />
          </template>

          <Column
            v-for="col of activeColumns"
            :key="col.field"
            :field="col.field"
          >
            <template #header>
              <ColumnHeader
                :header="col.header"
                :field="col.field"
                :sort="sort"
              />
            </template>
            <template #body="{ data }">
              <template v-if="col.field === 'email_confirmed'">
                {{ dateHelper.formatDate(data?.email_confirmed, true) }}
              </template>
              <template v-else-if="col.field === 'display_name'">
                <span
                  class="hover-icon font-bold"
                  @click="manipulateDialog('updateUser', data.id)"
                >
                  {{ data[col.field] }}
                </span>
              </template>
              <template v-else-if="col.field === 'role'">
                <span>
                  {{ data[col.field]["name"] }}
                </span>
              </template>
              <template v-else>
                {{ data[col.field] }}
              </template>
            </template>
          </Column>
        </DataTable>
      </div>

      <label>Invitations</label>
      <div class="flex flex-row gap-2 w-full">
        <InvitationsPaginated ref="invRef" />
      </div>
    </div>
  </main>
</template>

<style scoped></style>
