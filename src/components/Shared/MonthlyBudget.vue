<script setup lang="ts">
import {useAuthStore} from "../../services/stores/authStore.ts";
import {useBudgetStore} from "../../services/stores/budgetStore.ts";
import {useToastStore} from "../../services/stores/toastStore.ts";
import {computed, onMounted, ref} from "vue";
import type {MonthlyBudget} from "./MonthlyBudget.vue";
import {useInflowStore} from "../../services/stores/inflowStore.ts";
import {useOutflowStore} from "../../services/stores/outflowStore.ts";
import InflowCategories from "../Inflows/InflowCategories.vue";
import DynamicCategories from "../Inflows/DynamicCategories.vue";
import OutflowCategories from "../Outflows/OutflowCategories.vue";
import vue from "@vitejs/plugin-vue";
import vueHelper from "../../utils/vueHelper.ts";

const authStore = useAuthStore();
const budgetStore = useBudgetStore();
const toastStore = useToastStore();
const inflowStore = useInflowStore();
const outflowStore = useOutflowStore();

const currentBudget = ref<MonthlyBudget>(null);
const createNewBudget = ref<MonthlyBudget>(initBudget());

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

async function createBudget() {
  if(!createNewBudget.value.dynamic_category) {
    toastStore.errorResponseToast(vueHelper.formatErrorToast("Dynamic category is required.", "Please assign a category to your budget."))
  }
  try {
    let response = await budgetStore.createNewBudget({
      id: null,
      dynamic_category_id: createNewBudget.value.dynamic_category.id,
      dynamic_category: createNewBudget.value.dynamic_category,
      month: 0,
      year: 0,
      total_inflow: 0,
      total_outflow: 0,
      effective_budget: 0,
      budget_snapshot: 0,
    });
    currentBudget.value = response.data;
    toastStore.successResponseToast(vueHelper.formatSuccessToast("Create success", "Budget has been created."));
    await authStore.getAuthUser()
  } catch (err) {
    toastStore.errorResponseToast(err)
  }
}

</script>

<template>
  <div v-if="!authStore?.user?.secrets?.budget_initialized" class="flex flex-column gap-3 w-full">
    <div> <b>{{ "Create form" }}</b></div>
    <div class="flex flex-row w-full">
      {{ "You haven't initialized your budget yet! Create one with the form below." }}
    </div>
    <div class="flex flex-row w-full">
      <b>{{ "Step 1: Inflow and outflow categories" }}</b>
    </div>
    <div class="flex flex-row w-full">
      {{ "Create at least one inflow and one outflow category. These categories will represent how your effective budget is calculated." }}
    </div>
    <div class="flex flex-row w-full">
      {{ "You can create as many as you wish, but at least one primary one is required for each type of flow, so that a budget can be calculated." }}
    </div>
    <div class="flex flex-row w-full">
      {{ "You will be able to manage them, once you created a budget." }}
    </div>
    <hr>
    <div class="flex flex-row w-full gap-2">
      <div class="flex-column w-6">
        <label><b>{{"Inflow categories"}}</b></label>
        <InflowCategories :restricted="true"></InflowCategories>
      </div>
      <div class="flex-column w-6">
        <label><b>{{"Outflow categories"}}</b></label>
        <OutflowCategories :restricted="true"></OutflowCategories>
      </div>
    </div>

    <div class="flex flex-row w-full">
      <b>{{ "Step 2: dynamic categories" }}</b>
    </div>
    <div class="flex flex-row w-full">
      {{ "Assign at least one inflow category as a primary link and at least one outflow category to the secondary link. You can reuse dynamic categories as primary links." }}
    </div>
    <div class="flex flex-row w-full">
      {{ "Assign one dynamic category to your budget. All values will be calculated based on it." }}
    </div>
    <div class="flex-column w-full">
      <label><b>{{"Dynamic categories"}}</b></label>
      <DynamicCategories :restricted="true"></DynamicCategories>
    </div>

    <div class="flex flex-row w-full">
      <b>{{ "Step 3: Assign category" }}</b>
    </div>
    <div class="flex flex-row w-full">
      {{ "Assign which category will be linked to your budget. All calculations will be based off of it." }}
    </div>
    <div class="flex flex-row gap-2 w-6">
      <div class="flex flex-column">
        <AutoComplete size="small" v-model="createNewBudget.dynamic_category" :suggestions="filteredDynamicCategories"
                      @complete="searchDynamicCategory" option-label="name" placeholder="Select category" dropdown></AutoComplete>
      </div>
    </div>
    <hr v-if="createNewBudget.dynamic_category !== null">
    <div v-if="createNewBudget.dynamic_category !== null" class="flex flex-row w-6">
      <div class="flex flex-column w-full gap-1">
        <span> <b>{{ "Primary links" }}</b></span>
        <span> {{ "These categories will be summed up to create your total inflows record." }}</span>
        <div v-for="mapping in createNewBudget.dynamic_category?.Mappings">
          <span v-if="mapping.related_type === 'inflow' || mapping.related_type === 'dynamic'">
            {{ "+ " + mergedCategories.filter(record => record.id === mapping.related_id)[0]["name"] }}
          </span>
        </div>

        <span> <b>{{ "Secondary links" }}</b></span>
        <span> {{ "These categories will be summed up to create your total outflows record. They will be deducted from your total inflows to form an effective budget." }}</span>
        <div v-for="mapping in createNewBudget.dynamic_category?.Mappings">
          <span v-if="mapping.related_type === 'outflow'">
            {{ "- " + outflowCategories.filter(record => record.id === mapping.related_id)[0]["name"] }}
          </span>
        </div>
      </div>
    </div>

    <div v-if="createNewBudget.dynamic_category !== null" class="flex flex-row gap-2 w-full">
      <div class="flex flex-column w-full gap-2">
        <label> <b>{{ "Step 4: Save your budget" }}</b></label>
        <span> {{ "Once you have created and assigned your desired dynamic category, you can create your budget. Once completed, you will get access to the rest of the app." }}</span>
        <Button class="w-2" @click="createBudget" icon="pi pi-calculator" label="Create"></Button>
      </div>
    </div>
  </div>
  <div v-else class="flex flex-column gap-3 w-full">
    <div> <b>{{ "Create form" }}</b></div>
    <div class="flex flex-row w-full">
      {{ currentBudget }}
    </div>
  </div>
</template>

<style scoped>

</style>