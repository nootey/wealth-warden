<script setup lang="ts"></script>

<template>
  <DataTable :value="[]" size="small">
    <Column field="type" header="Type" />
    <Column field="years" header="Years" />
    <Column field="generated_at" header="Generated" />
    <template #empty>
      <span style="color: var(--text-secondary)"
        >No reports generated yet.</span
      >
    </template>
  </DataTable>
</template>
