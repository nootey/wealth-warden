<script setup lang="ts">
import {useAuthStore} from "../../services/stores/auth_store.ts";

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
          <label class="label"> Test </label>
          <div class="dashboard-item">
            {{ "hello there" }}
          </div>
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