<script setup lang="ts">
import {useAuthStore} from "../../services/stores/authStore.ts";
import {useBudgetStore} from "../../services/stores/budgetStore.ts";
import {useToastStore} from "../../services/stores/toastStore.ts";
import {onMounted, ref} from "vue";

const authStore = useAuthStore();
const budgetStore = useBudgetStore();
const toastStore = useToastStore();

const currentBudget = ref(null);

onMounted(() => {
  getCurrentBudget();
})

async function getCurrentBudget() {
  try {
    let response = await budgetStore.getCurrentBudget();
    currentBudget.value = response.data;
  } catch (err) {
    toastStore.errorResponseToast(err)
  }
}
</script>

<template>
  <div v-if="!authStore?.user?.secrets?.budget_initialized" class="flex flex-row gap-2">
    <div class="flex flex-row">
    {{ "You haven't initialized your budget yet!" }}
    </div>
    <div class="flex flex-row">
      {{ "Create one with the form below. Assign at least one inflow category as a primary link and at least one outflow category to the secondary link." }}
    </div>
  </div>
</template>

<style scoped>

</style>