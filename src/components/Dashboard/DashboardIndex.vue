<script setup lang="ts">
import MonthlyBudget from "../Shared/MonthlyBudget.vue";


</script>

<template>
  <main>
    <h1>Wealth Warden</h1>
    <div class="flex flex-column w-100 gap-3">

      <div class="flex flex-row w-100 gap-2">
        <div class="flex flex-column gap-1">
          <p> Selected year </p>
          <div> {{ "2025" }}</div>
        </div>
      </div>


      <div class="flex flex-row w-100 gap-2">
        <div class="flex flex-column gap-1">
          <p> Active budget </p>
          <MonthlyBudget></MonthlyBudget>
        </div>
      </div>

      <hr>

      <div class="flex flex-row w-100 gap-2">
        <div class="flex flex-column gap-1">
          <p> Stats </p>
          <span>These are the stats for the selected year, like: total inflows, effective inflows, take home, overflow ... Default is current year.</span>
        </div>
      </div>

      <hr>

      <div class="flex flex-row w-100 gap-2">
        <div class="flex flex-column gap-1">
          <p> Stats by category </p>
          <span>These are the stats for each category for the selected year ... Default is current year.</span>
        </div>
      </div>

      <hr>

      <div class="flex flex-row w-100 gap-2">
        <div class="flex flex-column gap-1">
          <p> Saving rates </p>
          <span>These are monthly rates for savings and investments for the selected year ... Default is current year.</span>
        </div>
      </div>

    </div>
  </main>
</template>

<style scoped>

</style>