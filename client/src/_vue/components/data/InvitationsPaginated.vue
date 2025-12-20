<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import dateHelper from "../../../utils/date_helper.ts";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import filterHelper from "../../../utils/filter_helper.ts";
import type { Column } from "../../../services/filter_registry.ts";
import { useConfirm } from "primevue/useconfirm";
import CustomPaginator from "../base/CustomPaginator.vue";
import type { Invitation } from "../../../models/user_models.ts";
import { useUserStore } from "../../../services/stores/user_store.ts";
import { usePermissions } from "../../../utils/use_permissions.ts";

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const userStore = useUserStore();
const { hasPermission } = usePermissions();

const confirm = useConfirm();

const apiPrefix = "users/invitations";

onMounted(async () => {
  await getData();
});

const loading = ref(true);
const records = ref<Invitation[]>([]);

const params = computed(() => {
  return {
    rowsPerPage: paginator.value.rowsPerPage,
    sort: sort.value,
    filters: null,
  };
});
const rows = ref([5, 10, 25]);
const default_rows = ref(rows.value[0]);
const paginator = ref({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: default_rows.value,
});
const page = ref(1);
const sort = ref(filterHelper.initSort());

const activeColumns = computed((): Column[] => [
  { field: "email", header: "Email", type: "text" },
  { field: "role", header: "Role", type: "enum" },
  { field: "created_at", header: "Created", type: "date" },
]);

async function getData(new_page = null) {
  loading.value = true;
  if (new_page) page.value = new_page;

  try {
    let paginationResponse = await sharedStore.getRecordsPaginated(
      apiPrefix,
      { ...params.value },
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

async function onPage(event: any) {
  paginator.value.rowsPerPage = event.rows;
  page.value = event.page + 1;
  await getData();
}

async function deleteConfirmation(id: number) {
  confirm.require({
    header: "Delete record?",
    message: `This will delete the following invitation.`,
    rejectProps: { label: "Cancel" },
    acceptProps: { label: "Delete", severity: "danger" },
    accept: () => deleteRecord(id),
  });
}

async function resendConfirmation(id: number) {
  confirm.require({
    header: "Resend invitation via email?",
    message: `This will invalidate the previous invitation, and resend a new on via email.`,
    rejectProps: { label: "Cancel" },
    acceptProps: { label: "Confirm" },
    accept: () => resendInvitation(id),
  });
}

async function deleteRecord(id: number) {
  if (!hasPermission("delete_users")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }

  try {
    let response = await sharedStore.deleteRecord(apiPrefix, id);
    toastStore.successResponseToast(response);
    await getData();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

async function resendInvitation(id: number) {
  try {
    loading.value = true;
    let response = await userStore.resendInvitation(id);
    toastStore.successResponseToast(response);
    await getData();
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
}

function refresh() {
  getData();
}

defineExpose({ refresh });
</script>

<template>
  <div
    class="flex flex-column justify-content-center w-full gap-3"
    style="background: var(--background-secondary); max-width: 1000px"
  >
    <div class="flex flex-row gap-2 w-full">
      <DataTable
        class="w-full enhanced-table"
        data-key="id"
        :loading="loading"
        :value="records"
        scrollable
        scroll-height="50vh"
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
          :header="col.header"
          :field="col.field"
          style="width: 30%"
        >
          <template #body="{ data }">
            <template v-if="col.field === 'created_at'">
              {{ dateHelper.formatDate(data?.created_at, true) }}
            </template>
            <template v-else-if="col.field === 'role'">
              {{ data[col.field]["name"] }}
            </template>
            <template v-else>
              {{ data[col.field] }}
            </template>
          </template>
        </Column>

        <Column header="Actions">
          <template #body="{ data }">
            <div class="flex flex-row gap-2 align-items-center">
              <i
                v-tooltip="'Resend email'"
                class="pi pi-refresh hover-icon text-sm"
                @click="resendConfirmation(data?.id)"
              />
              <i
                v-if="hasPermission('delete_users')"
                v-tooltip="'Delete invitation'"
                class="pi pi-trash hover-icon text-sm"
                style="color: var(--p-red-300)"
                @click="deleteConfirmation(data?.id)"
              />
            </div>
          </template>
        </Column>
      </DataTable>
    </div>
  </div>
</template>

<style scoped></style>
