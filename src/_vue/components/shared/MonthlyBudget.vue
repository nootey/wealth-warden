<script setup lang="ts">
import {useAuthStore} from "../../../services/stores/authStore.ts";
import {useBudgetStore} from "../../../services/stores/budgetStore.ts";
import {useToastStore} from "../../../services/stores/toastStore.ts";
import {computed, onMounted, ref} from "vue";
import type {MonthlyBudget} from "./MonthlyBudget.vue";
import {useInflowStore} from "../../../services/stores/inflowStore.ts";
import {useOutflowStore} from "../../../services/stores/outflowStore.ts";
import InflowCategories from "../../features/inflows/InflowCategories.vue";
import DynamicCategories from "../../features/inflows/DynamicCategories.vue";
import OutflowCategories from "../../features/outflows/OutflowCategories.vue";
import vueHelper from "../../../utils/vueHelper.ts";
import dateHelper from "../../../utils/dateHelper.ts";
import {useConfirm} from "primevue";
import ValidationError from "../validation/ValidationError.vue";
import {numeric, required} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";

const authStore = useAuthStore();
const budgetStore = useBudgetStore();
const toastStore = useToastStore();
const inflowStore = useInflowStore();
const outflowStore = useOutflowStore();

const currentBudget = ref<MonthlyBudget>(null);
const budgetChanged = ref(false);
const currentBudgetOriginalCategory = ref(null);

const createNewBudget = ref<MonthlyBudget>(initBudget());
const createNewAllocation = ref(initBudgetAllocation());

const rules = {
  createNewAllocation: {
    category: {
      name: {
        required,
        $autoDirty: true
      },
    },
    allocation: {
      required,
      numeric,
      minValue: 0,
      maxValue: 1000000000,
      $autoDirty: true
    },
  },
};

const v$ = useVuelidate(rules, { createNewAllocation });

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
const budgetAllocations = ref([
    {"name": "savings"},
    {"name": "investments"},
    {"name": "other"},
])

const filteredDynamicCategories = ref([]);
const filteredBudgetAllocations = ref([]);

const availableBudgetAllocation = ref();

const confirm = useConfirm();

onMounted(async () => {
  try {
    await Promise.all([
      getCurrentBudget(),
      inflowStore.getDynamicCategories(),
      inflowStore.getInflowCategories(),
      outflowStore.getOutflowCategories()
    ]);
  } catch (err) {
    toastStore.errorResponseToast(err);
  }
});


function initBudget(): MonthlyBudget<string, any> {
  return {
    dynamic_category: null,
    total_inflow: null,
    total_outflow: null,
    effective_budget: null,
    budget_snapshot: null,
  };
}

function initBudgetAllocation() {
  return {
    category: {name: ""},
    allocation: null,
  };
}

async function getCurrentBudget() {
  try {
    let response = await budgetStore.getCurrentBudget();
    if (response?.data) {
      currentBudget.value = response.data;
      currentBudgetOriginalCategory.value = currentBudget.value.dynamic_category;
    } else {
      currentBudget.value = null;
      currentBudgetOriginalCategory.value = null;
    }
    calculateAvailableBudgetAllocation(currentBudget.value);
  } catch (err) {
    toastStore.errorResponseToast(err)
  }
}

function calculateAvailableBudgetAllocation(budget: MonthlyBudget|null){
  if (budget == null ){
    availableBudgetAllocation.value = 0;
    return;
  }
  let sum = 0;
  budget.allocations.forEach((allocation:any) => {
    sum += allocation.total_allocated_value;
  })

  availableBudgetAllocation.value = budget.budget_snapshot - sum;
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

const searchBudgetAllocation = (event: any) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredBudgetAllocations.value = [...budgetAllocations.value];
    } else {
      filteredBudgetAllocations.value = budgetAllocations.value.filter((record) => {
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

async function createNewBudgetAllocation() {
  const isValidAllocation = await v$.value.createNewAllocation.$validate();
  if (!isValidAllocation) return true;

  if(!currentBudget.value) {
    return;
  }

  try {

    let response = await budgetStore.createNewBudgetAllocation({
      id: null,
      monthly_budget_id: currentBudget.value.id,
      category: createNewAllocation.value.category.name,
      allocation: createNewAllocation.value.allocation,
    });

    toastStore.successResponseToast(response);
    createNewAllocation.value = initBudgetAllocation();
    v$.value.createNewAllocation.$reset();
    await getCurrentBudget();

  } catch (err) {
    toastStore.errorResponseToast(err)
  }

}

async function updateBudgetSnapshot() {

  if(!currentBudget.value) {
    return;
  }

  try {

    let response = await budgetStore.updateBudgetSnapshot(currentBudget.value.id);
    await getCurrentBudget();
    toastStore.successResponseToast(response);

    v$.value.createNewAllocation.$reset();
  } catch (err) {
    toastStore.errorResponseToast(err)
  }

}

function checkCategoryStatus() {
  if (!currentBudget.value) {
    return;
  }

  if(currentBudget.value.dynamic_category != currentBudgetOriginalCategory.value) {
    budgetChanged.value = true;
  } else {
    budgetChanged.value = false;
  }
}

const confirmSnapshotUpdate = (event: any) => {
  confirm.require({
    target: event.currentTarget,
    message: 'You are about to update your budget snapshot. Are you sure you want to proceed?',
    icon: 'pi pi-exclamation-triangle',
    rejectProps: {
      label: 'Cancel',
      severity: 'secondary',
      outlined: true
    },
    acceptProps: {
      label: 'Update'
    },
    accept: async () => {
      await updateBudgetSnapshot();
    },
    reject: () => {
      toastStore.infoResponseToast(vueHelper.formatInfoToast("Update declined", "Nothing has been updated."));
    }
  });
};

const confirmBudgetUpdate = (event: any) => {
  confirm.require({
    target: event.currentTarget,
    message: 'You are about to change this months budget category. \n All allocations for this month will be reset! \n Are you sure you want to proceed?',
    icon: 'pi pi-exclamation-triangle',
    rejectProps: {
      label: 'Cancel',
      severity: 'secondary',
      outlined: true
    },
    acceptProps: {
      label: 'Change'
    },
    accept: () => {
      toastStore.successResponseToast(vueHelper.formatSuccessToast("Update success", "Budget has been updated."));
    },
    reject: () => {
      toastStore.infoResponseToast(vueHelper.formatInfoToast("Update declined", "Nothing has been updated."));
    }
  });
};

</script>

<template>
  <div v-if="authStore?.user && !authStore?.user?.secrets?.budget_initialized" class="flex flex-column gap-3 w-full">
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
  <div v-else-if="currentBudget && currentBudget?.dynamic_category" class="flex flex-column gap-3 w-full">
    <div> <b>{{ "Budget" }}</b></div>
    <div class="flex flex-row gap-3 align-items-center">
      <div class="flex flex-column gap-1">
        <span> <b>{{ "Year" }}</b></span>
        <div> {{ currentBudget.year}}</div>
      </div>
      <div class="flex flex-column gap-1">
        <span> <b>{{ "Month" }}</b></span>
        <div> {{ currentBudget.month}}</div>
      </div>
      <div class="flex flex-column gap-1">
        <span> <b>{{ "Updated" }}</b></span>
        <div> {{ dateHelper.formatDate(currentBudget.updated_at, true)}}</div>
      </div>
      <div class="flex flex-row">
        <div class="flex flex-column gap-1">
          <span> <b>{{ "Actions" }}</b></span>
          <div class="flex flex-row gap-2">
            <div class="flex flex-column">
              <Button size="small" label="Update snapshot" @click="confirmSnapshotUpdate($event)"></Button>
            </div>
            <div v-if="budgetChanged" class="flex flex-column">
              <Button size="small" label="Update budget" @click="confirmBudgetUpdate($event)"></Button>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div class="flex flex-row w-full gap-2">
      <div class="flex flex-column w-3 gap-1">
        <span> <b>{{ "Total inflows" }}</b></span>
        <InputNumber disabled size="small" v-model="currentBudget.total_inflow" mode="currency" currency="EUR" locale="de-DE" autofocus fluid></InputNumber>
      </div>
      <div class="flex flex-column w-3 gap-1">
        <span> <b>{{ "Total outflows" }}</b></span>
        <InputNumber disabled size="small" v-model="currentBudget.total_outflow" mode="currency" currency="EUR" locale="de-DE" autofocus fluid></InputNumber>
      </div>
      <div class="flex flex-column w-3 gap-1">
        <span> <b>{{ "Effective budget" }}</b></span>
        <InputNumber disabled size="small" v-model="currentBudget.effective_budget" mode="currency" currency="EUR" locale="de-DE" autofocus fluid></InputNumber>
      </div>
      <div class="flex flex-column w-3 gap-1">
        <span> <b>{{ "Budget snapshot" }}</b></span>
        <InputNumber disabled size="small" v-model="currentBudget.budget_snapshot" mode="currency" currency="EUR" locale="de-DE" autofocus fluid></InputNumber>
      </div>
    </div>
    <div class="flex flex-row gap-2 w-9">
      <div class="flex flex-column">
        <label>Linked dynamic category</label>
        <AutoComplete class="w-full" size="small" v-model="currentBudget.dynamic_category" :suggestions="filteredDynamicCategories"
                      @complete="searchDynamicCategory" @change="checkCategoryStatus" option-label="name" placeholder="Select category" dropdown></AutoComplete>
      </div>
    </div>
    <div v-if="currentBudget.dynamic_category" class="flex flex-row w-full">
      <div class="flex flex-column w-6 gap-1">
        <span> <b>{{ "Inflows" }}</b></span>
        <div v-for="mapping in currentBudget.dynamic_category?.Mappings">
          <span v-if="mapping.related_type === 'inflow' || mapping.related_type === 'dynamic'">
            {{ "+ " + mergedCategories.filter(record => record.id === mapping.related_id)[0]["name"] }}
          </span>
        </div>
      </div>
      <div class="flex flex-column w-6 gap-1">
        <span> <b>{{ "Outflows" }}</b></span>
        <div v-for="mapping in currentBudget.dynamic_category?.Mappings">
          <span v-if="mapping.related_type === 'outflow'">
            {{ "- " + outflowCategories.filter(record => record.id === mapping.related_id)[0]["name"] }}
          </span>
        </div>
      </div>
    </div>

    <div> <b>{{ "Allocations" }}</b></div>
    <div class="flex flex-row gap-2 align-items-center">
      <div class="flex flex-column gap-1">
            {{ "Define and view your budget allocations. The total value must be lower than the calculated effective budget."}}
      </div>
    </div>
    <div class="flex flex-row gap-2 align-items-center">
      <div class="flex flex-column gap-1">
        {{ "Available to allocate: "}}
      </div>

      <div class="flex flex-column gap-1">
          <b>{{ vueHelper.displayAsCurrency(availableBudgetAllocation) }}</b>
      </div>
    </div>


    <div class="flex flex-row">
      <label class="label"> New allocation </label>
    </div>
    <div class="flex flex-row gap-2 w-9">
      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.createNewAllocation.category.name.$errors[0]?.$message">
          <label>Name</label>
        </ValidationError>
        <AutoComplete class="w-full" size="small" v-model="createNewAllocation.category" :suggestions="filteredBudgetAllocations"
                      @complete="searchBudgetAllocation" option-label="name" placeholder="Select allocation" dropdown></AutoComplete>
      </div>
      <div class="flex flex-column">
        <ValidationError :isRequired="true" :message="v$.createNewAllocation.allocation.$errors[0]?.$message">
          <label>Allocation</label>
        </ValidationError>
        <InputNumber size="small" v-model="createNewAllocation.allocation" mode="currency" currency="EUR"
                     locale="de-DE" placeholder="0,00"></InputNumber>
      </div>
      <div class="flex flex-column">
        <ValidationError :isRequired="false" message="">
          <label>Actions</label>
        </ValidationError>
         <Button size="small" icon="pi pi-cart-plus" @click="createNewBudgetAllocation" />
      </div>
    </div>

    <div class="flex flex-row">
      <label class="label"> Existing allocations </label>
    </div>
    <div class="flex flex-row gap-2 w-9">
      <div v-if="currentBudget.allocations && Object.keys(currentBudget.allocations).length > 0" class="flex flex-column">
        <div v-for="allocation in currentBudget.allocations">
            <div class="flex flex-row gap-2 align-items-center">
              <div class="flex flex-column">
                {{ "-" }}
              </div>
              <div class="flex flex-column">
                {{ allocation.category }}
              </div>
              <div class="flex flex-column">
                {{ vueHelper.displayAsCurrency(allocation.total_allocated_value) }}
              </div>
            </div>
        </div>
      </div>
      <div v-else class="flex flex-column">
        {{ "You haven't allocated any budget yet." }}
      </div>
    </div>

  </div>
  <ProgressSpinner v-else animationDuration="1s" strokeWidth="8" style="width:50px;height:50px"/>

</template>

<style scoped>

</style>