<script setup lang="ts">
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useTransactionStore } from "../../../services/stores/transaction_store.ts";
import { computed, nextTick, onMounted, ref } from "vue";
import { required, requiredIf } from "@regle/rules";
import { useRegle } from "@regle/core";
import ValidationError from "../validation/ValidationError.vue";
import ShowLoading from "../base/ShowLoading.vue";
import RuleConditionRow from "./RuleConditionRow.vue";
import { usePermissions } from "../../../utils/use_permissions.ts";
import { useConfirm } from "primevue/useconfirm";
import searchHelper from "../../../utils/search_helper.ts";
import type { Ref } from "vue";
import type {
  Rule,
  RuleCondition,
  RuleConditionReq,
} from "../../../models/rules_models.ts";
import type { Category } from "../../../models/transaction_models.ts";

const props = defineProps<{
  mode?: "create" | "update";
  recordId?: number | null;
}>();

const emit = defineEmits<{
  (event: "completeOperation"): void;
  (event: "completeDelete"): void;
}>();

const apiPrefix = "rules";

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const transactionStore = useTransactionStore();
const { hasPermission } = usePermissions();
const confirm = useConfirm();

onMounted(async () => {
  if (transactionStore.categories.length === 0) {
    await transactionStore.getCategories();
  }
  if (props.mode === "update" && props.recordId) {
    await loadRecord(props.recordId);
  }
});

const loading = ref(false);
const submitting = ref(false);

type RuleFormData = {
  name: string;
  is_active: boolean;
  match_type: string;
  conditions: RuleConditionReq[];
  category: Category | null;
};

const record = ref<RuleFormData>(initData());

const matchOptions = [
  { label: "All", value: "all" },
  { label: "Any", value: "any" },
];

const categories = computed<Category[]>(() =>
  transactionStore.categories.filter(
    (c) =>
      !c.name.startsWith("(") &&
      c.display_name !== "Expense" &&
      c.display_name !== "Income",
  ),
);
const filteredCategories = ref<Category[]>([]);

const conditionRules = {
  field: { required },
  operator: { required },
  value: { required },
};

const rules = {
  name: { required },
  match_type: { required },
  conditions: {
    $each: (item: Ref<RuleConditionReq>) => ({
      match_type: { required: requiredIf(() => item.value.is_group) },
      field: { required: requiredIf(() => !item.value.is_group) },
      operator: { required: requiredIf(() => !item.value.is_group) },
      value: { required: requiredIf(() => !item.value.is_group) },
      conditions: { $each: conditionRules },
    }),
  },
  category: {
    display_name: { required },
  },
};

const { r$ } = useRegle(record, rules);

function initData(): RuleFormData {
  return {
    name: "",
    is_active: true,
    match_type: "all",
    conditions: [newCondition()],
    category: null,
  };
}

function newCondition(): RuleConditionReq {
  return {
    is_group: false,
    match_type: "",
    field: "description",
    operator: "contains",
    value: "",
    conditions: [],
  };
}

function newGroup(): RuleConditionReq {
  return {
    is_group: true,
    match_type: "any",
    field: "",
    operator: "",
    value: "",
    conditions: [newCondition()],
  };
}

function removeAt(list: RuleConditionReq[], index: number): void {
  if (list.length > 1) list.splice(index, 1);
}

// The API returns a flat list with parent_id; rebuild one level of nesting from it.
function toConditionReqs(conditions: RuleCondition[]): RuleConditionReq[] {
  const children = (parentId: number | null): RuleCondition[] =>
    conditions
      .filter((c) => (c.parent_id ?? null) === parentId)
      .sort((a, b) => a.position - b.position);
  const plain = (c: RuleCondition): RuleConditionReq => ({
    ...newCondition(),
    field: c.field,
    operator: c.operator,
    value: c.value,
  });
  return children(null).map((c) =>
    c.is_group
      ? {
          ...newGroup(),
          match_type: c.match_type,
          conditions: children(c.id!).map(plain),
        }
      : plain(c),
  );
}

function toPayload(c: RuleConditionReq): object {
  if (c.is_group) {
    return {
      is_group: true,
      match_type: c.match_type,
      conditions: c.conditions.map(toPayload),
    };
  }
  return { field: c.field, operator: c.operator, value: c.value };
}

async function loadRecord(id: number) {
  try {
    loading.value = true;
    const data: Rule = await sharedStore.getRecordByID(apiPrefix, id);
    const action = data.actions?.[0];
    const conditions = toConditionReqs(data.conditions ?? []);

    record.value = {
      name: data.name,
      is_active: data.is_active,
      match_type: data.match_type,
      conditions: conditions.length > 0 ? conditions : [newCondition()],
      category:
        categories.value.find((c) => String(c.id) === action?.value) ?? null,
    };

    await nextTick();
    loading.value = false;
  } catch (err) {
    toastStore.errorResponseToast(err);
  }
}

async function isRecordValid() {
  const { valid: isValid } = await r$.$validate();
  if (!isValid) return false;
  return true;
}

async function manageRecord() {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }

  if (!(await isRecordValid())) return;

  const recordData = {
    name: record.value.name,
    ...(props.mode === "update" && { is_active: record.value.is_active }),
    match_type: record.value.match_type,
    conditions: record.value.conditions.map(toPayload),
    actions: [
      {
        action_type: "set_category",
        value: String(record.value.category!.id),
      },
    ],
  };

  submitting.value = true;
  try {
    let response = null;

    switch (props.mode) {
      case "create":
        response = await sharedStore.createRecord(apiPrefix, recordData);
        break;
      case "update":
        response = await sharedStore.updateRecord(
          apiPrefix,
          props.recordId!,
          recordData,
        );
        break;
      default:
        emit("completeOperation");
        break;
    }

    r$.$reset();
    toastStore.successResponseToast(response);
    emit("completeOperation");
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    submitting.value = false;
  }
}

function deleteConfirmation() {
  confirm.require({
    header: "Delete record?",
    message: `This will delete rule: "${record.value.name}".`,
    rejectProps: { label: "Cancel" },
    acceptProps: { label: "Delete", severity: "danger" },
    accept: () => deleteRecord(),
  });
}

async function deleteRecord() {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }

  try {
    const response = await sharedStore.deleteRecord(apiPrefix, props.recordId!);
    toastStore.successResponseToast(response);
    emit("completeDelete");
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}

const searchCategory = (event: { query: string }) => {
  filteredCategories.value = searchHelper.filterByQuery(
    categories.value,
    event.query,
    (c) => [c.display_name, c.name],
  );
};
</script>

<template>
  <div v-if="!loading" class="flex flex-col gap-4 p-1">
    <div class="flex flex-col gap-4 p-1">
      <div class="flex flex-row w-full gap-4">
        <div class="flex flex-col flex-1 gap-1">
          <ValidationError :is-required="true" :message="r$.name.$errors[0]">
            <label>Name</label>
          </ValidationError>
          <InputText v-model="record.name" size="small" />
        </div>
        <div v-if="mode === 'update'" class="flex flex-col gap-1">
          <label>Active</label>
          <div class="flex items-center" style="height: 100%">
            <ToggleSwitch v-model="record.is_active" />
          </div>
        </div>
      </div>

      <div class="flex flex-row w-full items-center gap-2">
        <span>Match</span>
        <Select
          v-model="record.match_type"
          :options="matchOptions"
          option-label="label"
          option-value="value"
          size="small"
          class="w-24"
        />
        <span>of the following</span>
      </div>

      <span class="text-sm" style="color: var(--text-secondary)">
        All requires every condition to match. Any requires just one. Add a
        group to nest a separate All/Any block inside the rule.
      </span>

      <template v-for="(item, i) in record.conditions" :key="i">
        <div
          v-if="item.is_group"
          class="flex flex-col p-3 rounded-lg border border-surface gap-3"
        >
          <div class="flex flex-row w-full items-center gap-2">
            <span>Match</span>
            <Select
              v-model="item.match_type"
              :options="matchOptions"
              option-label="label"
              option-value="value"
              size="small"
              class="w-24"
            />
            <span>of the following</span>
            <Button
              icon="pi pi-times"
              severity="secondary"
              text
              size="small"
              class="ml-auto"
              :disabled="record.conditions.length <= 1"
              aria-label="Remove group"
              @click="removeAt(record.conditions, i)"
            />
          </div>
          <RuleConditionRow
            v-for="(_, j) in item.conditions"
            :key="j"
            v-model="item.conditions[j]"
            :errors="{
              field:
                r$.conditions.$each[i]?.conditions.$each[j]?.field.$errors[0],
              operator:
                r$.conditions.$each[i]?.conditions.$each[j]?.operator
                  .$errors[0],
              value:
                r$.conditions.$each[i]?.conditions.$each[j]?.value.$errors[0],
            }"
            :removable="item.conditions.length > 1"
            @remove="removeAt(item.conditions, j)"
          />
          <Button
            label="Add condition"
            icon="pi pi-plus"
            severity="secondary"
            text
            size="small"
            class="self-start"
            @click="item.conditions.push(newCondition())"
          />
        </div>
        <RuleConditionRow
          v-else
          v-model="record.conditions[i]"
          :errors="{
            field: r$.conditions.$each[i]?.field.$errors[0],
            operator: r$.conditions.$each[i]?.operator.$errors[0],
            value: r$.conditions.$each[i]?.value.$errors[0],
          }"
          :removable="record.conditions.length > 1"
          @remove="removeAt(record.conditions, i)"
        />
      </template>

      <div class="flex flex-row gap-2">
        <Button
          label="Add condition"
          icon="pi pi-plus"
          severity="secondary"
          text
          size="small"
          @click="record.conditions.push(newCondition())"
        />
        <Button
          label="Add group"
          icon="pi pi-plus"
          severity="secondary"
          text
          size="small"
          @click="record.conditions.push(newGroup())"
        />
      </div>

      <div class="flex flex-row w-full">
        <div class="flex flex-col w-full gap-1">
          <ValidationError
            :is-required="true"
            :message="r$.category.display_name.$errors[0]"
          >
            <label>Set category</label>
          </ValidationError>
          <AutoComplete
            v-model="record.category"
            size="small"
            :suggestions="filteredCategories"
            option-label="display_name"
            placeholder="Select category"
            dropdown
            @complete="searchCategory"
          >
            <template #option="{ option }">
              <div class="flex justify-between w-full">
                <span>{{ option.display_name }}</span>
                <small class="text-muted-color">
                  {{ option.classification }}
                </small>
              </div>
            </template>
          </AutoComplete>
        </div>
      </div>
    </div>

    <div class="flex flex-row gap-2 w-full">
      <div class="flex flex-col w-full gap-2">
        <Button
          class="main-button"
          :label="(mode == 'create' ? 'Add' : 'Update') + ' rule'"
          :disabled="submitting"
          :loading="submitting"
          style="height: 42px"
          @click="manageRecord"
        />
        <Button
          v-if="mode == 'update'"
          label="Delete rule"
          class="delete-button"
          style="height: 42px"
          @click="deleteConfirmation"
        />
      </div>
    </div>
  </div>
  <ShowLoading v-else :num-fields="5" />
</template>

<style scoped></style>
