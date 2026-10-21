<script setup lang="ts">
import type {
  Category,
  CategoryGroup,
} from "../../../models/transaction_models.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import { computed, ref } from "vue";
import type { Column } from "../../../services/filter_registry.ts";
import { usePermissions } from "../../../utils/use_permissions.ts";
import CategoryGroupForm from "../forms/CategoryGroupForm.vue";

defineProps<{
  categoryGroups: CategoryGroup[];
  categories: Category[];
}>();

const emit = defineEmits<{
  (e: "completeOperation"): void;
  (e: "completeDelete"): void;
}>();

const { hasPermission } = usePermissions();

const updateModal = ref(false);
const selectedID = ref<number | null>(null);

const activeColumns = computed<Column[]>(() => [
  { field: "name", header: "Name" },
  { field: "classification", header: "Classification" },
]);

function openUpdate(id: number) {
  if (!hasPermission("manage_data")) return;
  selectedID.value = id;
  updateModal.value = true;
}

function completeOperation() {
  updateModal.value = false;
  selectedID.value = null;
  emit("completeOperation");
}

function completeDelete() {
  updateModal.value = false;
  selectedID.value = null;
  emit("completeDelete");
}
</script>

<template>
  <Dialog
    v-model:visible="updateModal"
    position="right"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Update group"
  >
    <CategoryGroupForm
      mode="update"
      :record-id="selectedID"
      :categories="categories"
      @complete-operation="completeOperation"
      @complete-delete="completeDelete"
    />
  </Dialog>

  <DataTable
    class="w-full enhanced-table"
    data-key="id"
    :value="categoryGroups"
    paginator
    :rows="10"
    :rows-per-page-options="[10, 25]"
    scrollable
    scroll-height="75vh"
    :row-class="vueHelper.deletedRowClass"
  >
    <template #empty>
      <div style="padding: 10px">No records found.</div>
    </template>
    <template #loading>
      <LoadingSpinner />
    </template>

    <Column
      v-for="col of activeColumns"
      :key="col.field"
      :field="col.field"
      :header="col.header"
    >
      <template #body="{ data }">
        <span
          v-if="col.field === 'name'"
          class="hover"
          @click="openUpdate(data.id!)"
        >
          {{ data.name }}
        </span>
        <template v-else>
          {{ data[col.field] }}
        </template>
      </template>
    </Column>

    <Column header="Categories">
      <template #body="{ data }">
        <div
          v-tooltip="
            'This group has ' + (data?.categories?.length ?? 0) + ' categories'
          "
          class="flex flex-row items-center gap-2"
        >
          <i class="pi pi-eye" />
          <span>{{ data?.categories?.length ?? 0 }}</span>
        </div>
      </template>
    </Column>
  </DataTable>
</template>

<style scoped>
.hover {
  font-weight: bold;
}
.hover:hover {
  cursor: pointer;
  text-decoration: underline;
}
</style>
