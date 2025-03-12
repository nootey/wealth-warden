<script setup lang="ts">
import MonthlyBudget from "../Shared/MonthlyBudget.vue";
import {useAuthStore} from "../../services/stores/authStore.ts";

const authStore = useAuthStore();

</script>

<template>
  <main>
    <h1>Wealth Warden</h1>
    <div class="flex flex-column w-100 gap-3">

      <div v-if="authStore?.user?.secrets?.budget_initialized" class="flex flex-row w-100 gap-2">
        <div class="flex flex-column gap-1">
          <p> Selected year </p>
          <div> {{ "2025" }}</div>
        </div>
      </div>


      <div class="flex flex-row w-100 gap-3">
        <div class="flex flex-column gap-2">
          <label class="label"> Active budget </label>
          <div class="dashboard-item">
            <MonthlyBudget></MonthlyBudget>
          </div>
        </div>
      </div>



      <div v-if="authStore?.user?.secrets?.budget_initialized" class="flex flex-row w-100 gap-2">
        <hr>
        <div class="flex flex-column gap-1">
          <p> Stats </p>
          <span>These are the stats for the selected year, like: total inflows, effective inflows, take home, overflow ... Default is current year.</span>
        </div>
      </div>



      <div v-if="authStore?.user?.secrets?.budget_initialized" class="flex flex-row w-100 gap-2">
        <hr>
        <div class="flex flex-column gap-1">
          <p> Stats by category </p>
          <span>These are the stats for each category for the selected year ... Default is current year.</span>
        </div>
      </div>



      <div v-if="authStore?.user?.secrets?.budget_initialized" class="flex flex-row w-100 gap-2">
        <hr>
        <div class="flex flex-column gap-1">
          <p> Saving rates </p>
          <span>These are monthly rates for savings and investments for the selected year ... Default is current year.</span>
        </div>
      </div>

    </div>
  </main>
</template>

<style scoped>
.dashboard-item {
  border: 2px solid var(--border-color);
  border-radius: 15px;
  padding: 15px;
}
</style>