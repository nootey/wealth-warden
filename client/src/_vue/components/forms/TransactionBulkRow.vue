<script setup lang="ts">
import { computed, ref } from "vue";
import ValidationError from "../validation/ValidationError.vue";
import currencyHelper from "../../../utils/currency_helper.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import searchHelper from "../../../utils/search_helper.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import type {
  BulkTransactionRow,
  Category,
} from "../../../models/transaction_models.ts";

const model = defineModel<BulkTransactionRow>({ required: true });

const props = defineProps<{
  directions: Category[];
  categories: Category[];
  errors: {
    direction?: string;
    amount?: string;
    category?: string;
    description?: string;
  };
  showDirection: boolean;
  showDescription: boolean;
  removable: boolean;
}>();

const emit = defineEmits<{
  (event: "remove"): void;
}>();

const settingsStore = useSettingsStore();

const amountRef = computed({
  get: () => model.value.amount,
  set: (v) => (model.value.amount = v),
});
const { number: amountNumber } = currencyHelper.useMoneyField(amountRef, 2);

const filteredCategories = ref<Category[]>([]);

const searchCategory = (event: { query: string }) => {
  filteredCategories.value = searchHelper.filterByQuery(
    props.categories,
    event.query,
    (c) => [c.display_name, c.name],
    { sort: true },
  );
};
</script>

<template>
  <div class="flex flex-col sm:flex-row w-full gap-2 sm:items-end">
    <div v-if="showDirection" class="flex flex-col gap-1">
      <ValidationError :is-required="true" :message="errors.direction">
        <label>Direction</label>
      </ValidationError>
      <SelectButton
        v-model="model.direction"
        style="font-size: 0.875rem"
        size="small"
        :options="directions"
        option-label="display_name"
        option-value="classification"
        :allow-empty="false"
        @update:model-value="model.category = null"
      />
    </div>
    <div class="flex flex-col gap-1 sm:w-40">
      <ValidationError :is-required="true" :message="errors.amount">
        <label>Amount</label>
      </ValidationError>
      <InputNumber
        v-model="amountNumber"
        size="small"
        fluid
        mode="currency"
        :currency="settingsStore.defaultCurrency"
        :locale="vueHelper.getCurrencyLocale(settingsStore.defaultCurrency)"
        :placeholder="vueHelper.displayAsCurrency(0) ?? '0.00'"
      />
    </div>
    <div class="flex flex-col flex-1 gap-1 min-w-0">
      <ValidationError :is-required="false" :message="errors.category">
        <label>Category</label>
      </ValidationError>
      <AutoComplete
        v-model="model.category"
        size="small"
        :suggestions="filteredCategories"
        option-label="display_name"
        overlay-class="w-0"
        placeholder="Select category"
        dropdown
        @complete="searchCategory"
      />
    </div>
    <div v-if="showDescription" class="flex flex-col flex-1 gap-1 min-w-0">
      <ValidationError :is-required="false" :message="errors.description">
        <label>Description</label>
      </ValidationError>
      <InputText
        v-model="model.description"
        size="small"
        placeholder="Describe transaction"
      />
    </div>
    <Button
      icon="pi pi-times"
      severity="secondary"
      text
      size="small"
      class="self-end"
      :disabled="!removable"
      aria-label="Remove transaction"
      @click="emit('remove')"
    />
  </div>
</template>

<style scoped></style>
