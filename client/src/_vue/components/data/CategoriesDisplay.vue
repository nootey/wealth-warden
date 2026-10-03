<script setup lang="ts">
import type { Category } from "../../../models/transaction_models.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import { computed, ref } from "vue";
import type { Column } from "../../../services/filter_registry.ts";
import CategoryForm from "../forms/CategoryForm.vue";
import { usePermissions } from "../../../utils/use_permissions.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useTransactionStore } from "../../../services/stores/transaction_store.ts";

const props = defineProps<{
  categories: Category[];
}>();

const emit = defineEmits<{
  (e: "completeOperation"): void;
  (e: "completeDelete"): void;
}>();

const toastStore = useToastStore();
const transactionStore = useTransactionStore();

const { hasPermission } = usePermissions();

const seeding = ref(false);

async function seedDefaultCategories() {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }

  seeding.value = true;
  try {
    const res = await transactionStore.seedDefaultCategories();
    toastStore.successResponseToast(res);
    emit("completeOperation");
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    seeding.value = false;
  }
}

const localCategories = computed(() => {
  return props.categories.filter(
    (c) =>
      !c.name.startsWith("(") &&
      c.display_name !== "Expense" &&
      c.display_name !== "Income",
  );
});

const updateModal = ref(false);
const selectedID = ref<number | null>(null);

const categoryColumns = computed<Column[]>(() => [
  { field: "display_name", header: "Name" },
  { field: "is_default", header: "Type" },
  { field: "classification", header: "Classification" },
]);

function openUpdate(id: number) {
  if (!hasPermission("manage_data")) return;
  updateModal.value = true;
  selectedID.value = id;
}

function completeOperation() {
  updateModal.value = false;
  emit("completeOperation");
}

function completeDelete() {
  updateModal.value = false;
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
    header="Update category"
  >
    <CategoryForm
      mode="update"
      :record-id="selectedID"
      @complete-operation="completeOperation"
      @complete-delete="completeDelete"
    />
  </Dialog>

  <DataTable
    class="w-full enhanced-table"
    data-key="id"
    :value="localCategories"
    paginator
    :rows="10"
    :rows-per-page-options="[10, 25]"
    scrollable
    scroll-height="75vh"
    row-group-mode="subheader"
    group-rows-by="classification"
    :row-class="vueHelper.deletedRowClass"
  >
    <template #empty>
      <div class="w-full flex flex-col gap-3 items-center justify-center p-4">
        <span>No records found.</span>
        <Button
          class="main-button"
          :loading="seeding"
          @click="seedDefaultCategories"
        >
          <div class="flex flex-row gap-1 items-center">
            <i class="pi pi-sparkles" />
            <span> Seed default categories </span>
          </div>
        </Button>
      </div>
    </template>
    <template #loading>
      <LoadingSpinner />
    </template>

    <template #groupheader="slotProps">
      <div class="flex items-center gap-2">
        <span class="font-bold text-lg">{{
          vueHelper.capitalize(slotProps.data.classification)
        }}</span>
      </div>
    </template>

    <Column
      v-for="col of categoryColumns"
      :key="col.field"
      :field="col.field"
      :header="col.header"
      :sortable="col.field === 'is_default'"
    >
      <template #body="{ data }">
        <template v-if="col.field === 'display_name'">
          <span class="hover" @click="openUpdate(data.id!)">
            {{ data.display_name }}
          </span>
        </template>
        <template v-else-if="col.field === 'is_default'">
          {{ data.is_default ? "Default" : "Custom" }}
        </template>
        <template v-else>
          {{ data[col.field] }}
        </template>
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
