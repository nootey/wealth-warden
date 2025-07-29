<script setup lang="ts">
import TransactionRecords from "../features/transactions/TransactionRecords.vue";

</script>

<template>
  <main>
    <div class="flex flex-column w-100 gap-3 justify-content-center align-items-center">

      <div class="main-item flex flex-row justify-content-between">
        <div style="font-weight: bold;">Transactions</div>
        <Button label="New transaction" icon="pi pi-plus"
                style="background-color: var(--text-primary); color: var(--background-primary);
                border: none; border-radius: 6px; font-size: 0.875rem; padding: 0.5rem 1rem;"></Button>
      </div>

      <div class="main-item flex flex-column justify-content-center">
        <div class="flex flex-row" style="width: 100%;">
          <InputText style="width: 100%; border-radius: 8px; font-size: 0.875rem; padding: 0.5rem 1rem;" placeholder="Search transactions ..." />
        </div>
        <div class="flex flex-row" style="width: 100%;">
          <TransactionRecords></TransactionRecords>
        </div>
        
      </div>

    </div>
  </main>
</template>

<style scoped>
.main-item {
  width: 100%;
  max-width: 1000px;
  align-items: center;
  padding: 1rem;
  border-radius: 8px; border: 1px solid var(--border-color); background-color: var(--background-secondary)
}
</style>