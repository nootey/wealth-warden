<script setup lang="ts">
import { computed } from "vue";
import ValidationError from "../validation/ValidationError.vue";
import currencyHelper from "../../../utils/currency_helper.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import type { RuleConditionReq } from "../../../models/rules_models.ts";

const model = defineModel<RuleConditionReq>({ required: true });

defineProps<{
  errors: { field?: string; operator?: string; value?: string };
  removable: boolean;
}>();

const emit = defineEmits<{
  (event: "remove"): void;
}>();

const settingsStore = useSettingsStore();

const amountRef = computed({
  get: () => model.value.value,
  set: (v) => (model.value.value = v ?? ""),
});
const { number: amountNumber } = currencyHelper.useMoneyField(amountRef, 2);

const fieldOptions = [
  { label: "Description", value: "description" },
  { label: "Amount", value: "amount" },
];

const operatorOptions = computed(() => {
  if (model.value.field === "amount") {
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

function onFieldChange(): void {
  model.value.operator = model.value.field === "amount" ? "equals" : "contains";
  model.value.value = "";
}
</script>

<template>
  <div class="flex flex-row w-full gap-2 items-end">
    <div class="flex flex-col flex-1 gap-1 min-w-0">
      <ValidationError :is-required="true" :message="errors.field">
        <label>Field</label>
      </ValidationError>
      <Select
        v-model="model.field"
        :options="fieldOptions"
        option-label="label"
        option-value="value"
        placeholder="Select field"
        size="small"
        @update:model-value="onFieldChange"
      />
    </div>
    <div class="flex flex-col flex-1 gap-1 min-w-0">
      <ValidationError :is-required="true" :message="errors.operator">
        <label>Operator</label>
      </ValidationError>
      <Select
        v-model="model.operator"
        :options="operatorOptions"
        option-label="label"
        option-value="value"
        placeholder="Select operator"
        size="small"
      />
    </div>
    <div class="flex flex-col flex-1 gap-1 min-w-0">
      <ValidationError :is-required="true" :message="errors.value">
        <label>Value</label>
      </ValidationError>
      <InputNumber
        v-if="model.field === 'amount'"
        v-model="amountNumber"
        size="small"
        fluid
        mode="currency"
        :currency="settingsStore.defaultCurrency"
        :locale="vueHelper.getCurrencyLocale(settingsStore.defaultCurrency)"
        :placeholder="vueHelper.displayAsCurrency(0) ?? '0.00'"
      />
      <InputText v-else v-model="model.value" size="small" />
    </div>
    <Button
      icon="pi pi-times"
      severity="secondary"
      text
      size="small"
      :disabled="!removable"
      aria-label="Remove condition"
      @click="emit('remove')"
    />
  </div>
</template>

<style scoped></style>
