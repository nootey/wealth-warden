<script setup lang="ts">
import type { Rule } from "../../../models/rules_models.ts";
import LoadingSpinner from "../base/LoadingSpinner.vue";
import dateHelper from "../../../utils/date_helper.ts";
import { computed, ref } from "vue";
import type { Column } from "../../../services/filter_registry.ts";
import RuleForm from "../forms/RuleForm.vue";
import { usePermissions } from "../../../utils/use_permissions.ts";

defineProps<{
  rules: Rule[];
}>();

const emit = defineEmits<{
  (e: "completeOperation"): void;
  (e: "completeDelete"): void;
}>();

const { hasPermission } = usePermissions();

const updateModal = ref(false);
const selectedID = ref<number | null>(null);

const ruleColumns = computed<Column[]>(() => [
  { field: "name", header: "Name" },
  { field: "is_active", header: "Active" },
  { field: "effective_date", header: "Effective date" },
]);

function conditionCount(rule: Rule): number {
  return (rule.conditions ?? []).filter((c) => !c.is_group).length;
}

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
    header="Update rule"
  >
    <RuleForm
      mode="update"
      :record-id="selectedID"
      @complete-operation="completeOperation"
      @complete-delete="completeDelete"
    />
  </Dialog>

  <DataTable
    class="w-full enhanced-table"
    data-key="id"
    :value="rules"
    paginator
    :rows="10"
    :rows-per-page-options="[10, 25]"
    scrollable
    scroll-height="75vh"
  >
    <template #empty>
      <div style="padding: 10px">No records found.</div>
    </template>
    <template #loading>
      <LoadingSpinner />
    </template>

    <Column
      v-for="col of ruleColumns"
      :key="col.field"
      :field="col.field"
      :header="col.header"
      :sortable="col.field !== 'effective_date'"
    >
      <template #body="{ data }">
        <template v-if="col.field === 'name'">
          <span class="hover" @click="openUpdate(data.id!)">
            {{ data.name }}
          </span>
        </template>
        <template v-else-if="col.field === 'is_active'">
          {{ data.is_active ? "Yes" : "No" }}
        </template>
        <template v-else-if="col.field === 'effective_date'">
          {{
            data.effective_date
              ? dateHelper.formatDate(data.effective_date)
              : "-"
          }}
        </template>
        <template v-else>
          {{ data[col.field] }}
        </template>
      </template>
    </Column>

    <Column header="Conditions">
      <template #body="{ data }">
        <div
          v-tooltip="'This rule has ' + conditionCount(data) + ' conditions'"
          class="flex flex-row items-center gap-2"
        >
          <i class="pi pi-eye" />
          <span>{{ conditionCount(data) }}</span>
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
