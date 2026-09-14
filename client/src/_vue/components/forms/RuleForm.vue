<script setup lang="ts">
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { useTransactionStore } from "../../../services/stores/transaction_store.ts";
import { computed, nextTick, onMounted, ref } from "vue";
import { required } from "@regle/rules";
import { useRegle } from "@regle/core";
import ValidationError from "../validation/ValidationError.vue";
import ShowLoading from "../base/ShowLoading.vue";
import { usePermissions } from "../../../utils/use_permissions.ts";
import { useConfirm } from "primevue/useconfirm";
import searchHelper from "../../../utils/search_helper.ts";
import currencyHelper from "../../../utils/currency_helper.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import type { Rule } from "../../../models/rules_models.ts";
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
const settingsStore = useSettingsStore();
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
  field: string;
  operator: string;
  value: string;
  category: Category | null;
};

const record = ref<RuleFormData>(initData());

const amountRef = computed({
  get: () => record.value.value,
  set: (v) => (record.value.value = v ?? ""),
});
const { number: amountNumber } = currencyHelper.useMoneyField(amountRef, 2);

const fieldOptions = [
  { label: "Description", value: "description" },
  { label: "Amount", value: "amount" },
];

const operatorOptions = computed(() => {
  if (record.value.field === "amount") {
    return [
      { label: "Equals", value: "equals" },
      { label: "Greater than", value: "gt" },
      { label: "Greater than or equal", value: "gte" },
      { label: "Less than", value: "lt" },
      { label: "Less than or equal", value: "lte" },
    ];
  }
  return [{ label: "Contains", value: "contains" }];
});

const categories = computed<Category[]>(() =>
  transactionStore.categories.filter(
    (c) =>
      !c.name.startsWith("(") &&
      c.display_name !== "Expense" &&
      c.display_name !== "Income",
  ),
);
const filteredCategories = ref<Category[]>([]);

const rules = {
  name: { required },
  field: { required },
  operator: { required },
  value: { required },
  category: {
    display_name: { required },
  },
};

const { r$ } = useRegle(record, rules);

function initData(): RuleFormData {
  return {
    name: "",
    is_active: true,
    field: "description",
    operator: "contains",
    value: "",
    category: null,
  };
}

function onFieldChange() {
  record.value.operator =
    record.value.field === "amount" ? "equals" : "contains";
  record.value.value = "";
}

async function loadRecord(id: number) {
  try {
    loading.value = true;
    const data: Rule = await sharedStore.getRecordByID(apiPrefix, id);
    const condition = data.conditions?.[0];
    const action = data.actions?.[0];

    record.value = {
      name: data.name,
      is_active: data.is_active,
      field: condition?.field ?? "description",
      operator: condition?.operator ?? "contains",
      value: condition?.value ?? "",
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
    conditions: [
      {
        field: record.value.field,
        operator: record.value.operator,
        value: record.value.value,
      },
    ],
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

      <div class="flex flex-row w-full gap-4">
        <div class="flex flex-col flex-1 gap-1">
          <ValidationError :is-required="true" :message="r$.field.$errors[0]">
            <label>Field</label>
          </ValidationError>
          <Select
            v-model="record.field"
            :options="fieldOptions"
            option-label="label"
            option-value="value"
            placeholder="Select field"
            size="small"
            @update:model-value="onFieldChange"
          />
        </div>
        <div class="flex flex-col flex-1 gap-1">
          <ValidationError
            :is-required="true"
            :message="r$.operator.$errors[0]"
          >
            <label>Operator</label>
          </ValidationError>
          <Select
            v-model="record.operator"
            :options="operatorOptions"
            option-label="label"
            option-value="value"
            placeholder="Select operator"
            size="small"
          />
        </div>
      </div>

      <div class="flex flex-row w-full">
        <div class="flex flex-col w-full gap-1">
          <ValidationError :is-required="true" :message="r$.value.$errors[0]">
            <label>Value</label>
          </ValidationError>
          <InputNumber
            v-if="record.field === 'amount'"
            v-model="amountNumber"
            size="small"
            mode="currency"
            :currency="settingsStore.defaultCurrency"
            :locale="vueHelper.getCurrencyLocale(settingsStore.defaultCurrency)"
            :placeholder="vueHelper.displayAsCurrency(0) ?? '0.00'"
          />
          <InputText v-else v-model="record.value" size="small" />
        </div>
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
          />
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
