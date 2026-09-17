<template>
  <Chip :label="displayLabel" :class="chipClass" class="status-chip-small" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import Chip from "primevue/chip";

interface Props {
  status: string;
}

const props = defineProps<Props>();

const statusConfig: Record<string, { label: string; class: string }> = {
  pending: { label: "Pending", class: "status-pending" },
  processing: { label: "Processing", class: "status-processing" },
  completed: { label: "Completed", class: "status-success" },
  success: { label: "Success", class: "status-success" },
  failed: { label: "Failed", class: "status-fail" },
};

const displayLabel = computed(() => {
  return statusConfig[props.status]?.label || props.status;
});

const chipClass = computed(() => {
  return statusConfig[props.status]?.class || "status-default";
});
</script>

<style scoped>
/* PrimeVue tokens. Light: soft tint (step 100) + strong text (step 700).
   Dark: deep tint (step 900) + light text (step 300), via the ancestor. */
.status-chip-small {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  height: auto;
}

.status-pending {
  background-color: var(--p-amber-100);
  color: var(--p-amber-700);
}

.status-processing {
  background-color: var(--p-blue-100);
  color: var(--p-blue-700);
}

.status-success {
  background-color: var(--p-green-100);
  color: var(--p-green-700);
}

.status-fail {
  background-color: var(--p-red-100);
  color: var(--p-red-700);
}

.status-default {
  background-color: var(--p-surface-200);
  color: var(--p-surface-700);
}

:global(.my-app-dark) .status-pending {
  background-color: var(--p-amber-900);
  color: var(--p-amber-300);
}

:global(.my-app-dark) .status-processing {
  background-color: var(--p-blue-900);
  color: var(--p-blue-300);
}

:global(.my-app-dark) .status-success {
  background-color: var(--p-green-900);
  color: var(--p-green-300);
}

:global(.my-app-dark) .status-fail {
  background-color: var(--p-red-900);
  color: var(--p-red-300);
}

:global(.my-app-dark) .status-default {
  background-color: var(--p-surface-800);
  color: var(--p-surface-200);
}
</style>
