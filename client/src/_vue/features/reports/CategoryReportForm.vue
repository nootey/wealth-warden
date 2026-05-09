<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { requiredIf, helpers } from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import { useTransactionStore } from "../../../services/stores/transaction_store.ts";
import { useAnalyticsStore } from "../../../services/stores/analytics_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import ValidationError from "../../components/validation/ValidationError.vue";
import type { Category } from "../../../models/transaction_models.ts";
import vueHelper from "../../../utils/vue_helper.ts";

const emit = defineEmits<{ (e: "complete"): void }>();

const transactionStore = useTransactionStore();
const analyticsStore = useAnalyticsStore();
const toastStore = useToastStore();

const form = reactive({
  inflowCategories: [] as Category[],
  outflowCategories: [] as Category[],
  selectedYears: [new Date().getFullYear()] as number[],
  descriptionFilter: "",
  allTime: false,
});

const maxYears = 3;
const allYears = ref<number[]>([]);
const isLoadingCategories = ref(false);
const isLoadingYears = ref(false);
const isGenerating = ref(false);

const rules = {
  form: {
    selectedYears: {
      required: helpers.withMessage(
        "Select at least one year",
        requiredIf(() => !form.allTime),
      ),
      $autoDirty: true,
    },
    inflowCategories: {
      atLeastOne: helpers.withMessage(
        "Select at least one primary category",
        () => form.inflowCategories.length > 0,
      ),
    },
    outflowCategories: {
      atLeastOne: helpers.withMessage(
        "Must have a primary category selected",
        () => form.inflowCategories.length > 0,
      ),
    },
  },
};

const v$ = useVuelidate(rules, { form });

onMounted(async () => {
  const categoryLoad = async () => {
    if (!transactionStore.categories.length) {
      isLoadingCategories.value = true;
      try {
        await transactionStore.getCategories();
      } finally {
        isLoadingCategories.value = false;
      }
    }
  };

  const yearLoad = async () => {
    isLoadingYears.value = true;
    try {
      const result = await analyticsStore.getAvailableStatsYears(null);
      allYears.value = result.map((y) => y.year);
      const current = new Date().getFullYear();
      form.selectedYears = allYears.value.includes(current)
        ? [current]
        : allYears.value.slice(0, 1);
    } finally {
      isLoadingYears.value = false;
    }
  };

  try {
    await Promise.all([categoryLoad(), yearLoad()]);
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
});

const classOrder = [
  "income",
  "expense",
  "savings",
  "investments",
  "uncategorized",
];

const groupedCategories = computed(() => {
  const groups: Record<string, Category[]> = {};
  for (const cat of transactionStore.categories.filter(
    (c) => !c.deleted_at && c.parent_id !== null,
  )) {
    const key = cat.classification ?? "other";
    if (!groups[key]) groups[key] = [];
    groups[key].push(cat);
  }
  return classOrder
    .filter((k) => groups[k]?.length)
    .map((k) => ({
      label: vueHelper.formatString(k),
      items: groups[k]!.sort((a, b) =>
        (a.display_name ?? a.name).localeCompare(b.display_name ?? b.name),
      ),
    }));
});

async function generate() {
  const isValid = await v$.value.form.$validate();
  if (!isValid) return;

  isGenerating.value = true;
  try {
    await analyticsStore.getCategoryReport({
      inflowCategoryIDs: form.inflowCategories.map((c) => c.id as number),
      outflowCategoryIDs: form.outflowCategories.map((c) => c.id as number),
      years: form.selectedYears,
      description: form.descriptionFilter,
      allTime: form.allTime,
    });
    emit("complete");
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    isGenerating.value = false;
  }
}
</script>

<template>
  <div class="w-full flex flex-column gap-3 p-2">
    <div id="cat-row" class="flex flex-row gap-3 justify-content-center">
      <div class="flex flex-column gap-1" style="flex: 1; max-width: 280px">
        <ValidationError
          :is-required="true"
          :message="v$.form.inflowCategories.$errors[0]?.$message"
        >
          <label>Primary categories</label>
        </ValidationError>
        <MultiSelect
          v-model="form.inflowCategories"
          :options="groupedCategories"
          option-group-label="label"
          option-group-children="items"
          option-label="display_name"
          :loading="isLoadingCategories"
          :max-selected-labels="2"
          selected-items-label="{0} selected"
          placeholder="Select..."
          filter
          size="small"
        >
          <template #optiongroup="{ option }">
            <span
              class="text-xs font-semibold"
              style="color: var(--text-secondary)"
            >
              {{ option.label }}
            </span>
          </template>
        </MultiSelect>
      </div>

      <div class="flex flex-column gap-1" style="flex: 1; max-width: 280px">
        <ValidationError
          :is-required="false"
          :message="v$.form.outflowCategories.$errors[0]?.$message"
        >
          <label>Secondary categories</label>
        </ValidationError>
        <MultiSelect
          v-model="form.outflowCategories"
          :disabled="form.inflowCategories.length < 1"
          :options="groupedCategories"
          option-group-label="label"
          option-group-children="items"
          option-label="display_name"
          :loading="isLoadingCategories"
          :max-selected-labels="2"
          selected-items-label="{0} selected"
          placeholder="Select..."
          filter
          size="small"
        >
          <template #optiongroup="{ option }">
            <span
              class="text-xs font-semibold"
              style="color: var(--text-secondary)"
            >
              {{ option.label }}
            </span>
          </template>
        </MultiSelect>
      </div>
    </div>

    <div id="filter-row" class="flex flex-row gap-3 justify-content-center">
      <div class="flex flex-column gap-1" style="flex: 1; max-width: 280px">
        <ValidationError
          :is-required="!form.allTime"
          :message="v$.form.selectedYears.$errors[0]?.$message"
        >
          <label>Years (max {{ maxYears }})</label>
        </ValidationError>
        <MultiSelect
          v-model="form.selectedYears"
          :options="allYears"
          :selection-limit="maxYears"
          :max-selected-labels="3"
          :loading="isLoadingYears"
          :disabled="form.allTime"
          selected-items-label="{0} years"
          placeholder="Select years ..."
          size="small"
        />
      </div>

      <div class="flex flex-column gap-1" style="flex: 1; max-width: 280px">
        <ValidationError :is-required="false" :message="undefined">
          <label>Description</label>
        </ValidationError>
        <InputText
          v-model="form.descriptionFilter"
          size="small"
          placeholder="e.g. salary, rent ..."
        />
      </div>
    </div>

    <div class="flex w-full justify-content-center">
      <div class="flex flex-row gap-2 align-items-center">
        <Checkbox v-model="form.allTime" :binary="true" input-id="all-time" />
        <label for="all-time" class="text-sm cursor-pointer">
          All-time comparative
        </label>
      </div>
    </div>

    <div class="flex w-full justify-content-center">
      <Button
        class="main-button w-6"
        label="Generate"
        :loading="isGenerating"
        @click="generate"
      />
    </div>
  </div>
</template>

<style scoped>
@media (max-width: 768px) {
  #cat-row,
  #filter-row {
    flex-direction: column !important;
  }

  #cat-row > div,
  #filter-row > div {
    width: 100% !important;
    flex: unset !important;
  }
}
</style>
