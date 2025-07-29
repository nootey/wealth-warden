<script setup lang="ts">

</script>

<template>
  <main>
    <div class="flex flex-column w-100 gap-3 justify-content-center align-items-center">

      <div class="main-item flex flex-row justify-content-between">
        <div style="font-weight: bold;">Accounts</div>
        <Button label="New Account" icon="pi pi-plus"
                style="background-color: var(--text-primary); color: var(--background-primary);
                border: none; border-radius: 6px; font-size: 0.875rem; padding: 0.5rem 1rem;"></Button>
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