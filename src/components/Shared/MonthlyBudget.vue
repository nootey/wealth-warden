<script setup lang="ts">
import {useAuthStore} from "../../services/stores/authStore.ts";
import {useBudgetStore} from "../../services/stores/budgetStore.ts";
import {useToastStore} from "../../services/stores/toastStore.ts";
import {computed, onMounted, ref} from "vue";
import type {MonthlyBudget} from "./MonthlyBudget.vue";
import {useInflowStore} from "../../services/stores/inflowStore.ts";
import dateHelper from "../../utils/dateHelper.ts";
import {useOutflowStore} from "../../services/stores/outflowStore.ts";

const authStore = useAuthStore();
const budgetStore = useBudgetStore();
const toastStore = useToastStore();
const inflowStore = useInflowStore();
const outflowStore = useOutflowStore();

const currentBudget = ref<MonthlyBudget>(null);
const createNewBudget = ref<MonthlyBudget>(initBudget(false));

const dynamicCategories = computed(() => inflowStore.dynamicCategories);
const inflowCategories = computed(() => inflowStore.inflowCategories);
const outflowCategories = computed(() => outflowStore.outflowCategories);
const mergedCategories = computed(() => {
  return [
    ...(dynamicCategories.value || []).map(category => ({
      ...category,
      category_type: 'dynamic'
    })),
    ...(inflowCategories.value || []).map(category => ({
      ...category,
      category_type: 'inflow'
    }))
  ];
});

const filteredDynamicCategories = ref([]);

onMounted(async () => {
  await getCurrentBudget();
  await inflowStore.getDynamicCategories();
  await inflowStore.getInflowCategories();
  await outflowStore.getOutflowCategories();
})

function initBudget(): MonthlyBudget<string, any> {
  return {
    dynamic_category: null,
    total_inflow: null,
    total_outflow: null,
    effective_budget: null,
    budget_snapshot: null,
  };
}

async function getCurrentBudget() {
  try {
    let response = await budgetStore.getCurrentBudget();
    currentBudget.value = response.data;
  } catch (err) {
    toastStore.errorResponseToast(err)
  }
}

const searchDynamicCategory = (event: any) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredDynamicCategories.value = [...dynamicCategories.value];
    } else {
      filteredDynamicCategories.value = dynamicCategories.value.filter((record) => {
        return record.name.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
}

async function previewBudget() {
  console.log("Preview budget");
}

</script>

<template>
  <div v-if="!authStore?.user?.secrets?.budget_initialized" class="flex flex-column gap-2 w-9">
    <div class="flex flex-row w-full">
      {{ "You haven't initialized your budget yet!" }}
    </div>
    <div class="flex flex-row w-full">
      {{ "Create one with the form below. Assign at least one inflow category as a primary link and at least one outflow category to the secondary link." }}
    </div>
    <div class="flex flex-row gap-2 w-6">
      <div class="flex flex-column w-full">
        <AutoComplete size="small" v-model="createNewBudget.dynamic_category" :suggestions="filteredDynamicCategories"
                      @complete="searchDynamicCategory" option-label="name" placeholder="Select category" dropdown></AutoComplete>
      </div>
    </div>

    <div v-if="createNewBudget.dynamic_category !== null" class="flex flex-row gap-2 w-6">
      <div class="flex flex-column w-full gap-2">
        <span> {{ "Primary links" }}</span>
        <div v-for="mapping in createNewBudget.dynamic_category?.Mappings">
          <span v-if="mapping.related_type === 'inflow' || mapping.related_type === 'dynamic'">
            {{ mergedCategories.filter(record => record.id === mapping.related_id)[0]["name"] }}
          </span>
        </div>

        <span> {{ "Secondary links" }}</span>
        <div v-for="mapping in createNewBudget.dynamic_category?.Mappings">
          <span v-if="mapping.related_type === 'outflow'">
            {{ outflowCategories.filter(record => record.id === mapping.related_id)[0]["name"] }}
          </span>
        </div>
      </div>
    </div>

    <div v-if="createNewBudget.dynamic_category !== null" class="flex flex-row gap-2 w-3">
      <div class="flex flex-column w-full gap-2">
        <label> {{ "Preview" }}</label>
        <Button @click="previewBudget" icon="pi pi-calculator"></Button>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>