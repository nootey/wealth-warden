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
.status-chip-small {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  height: auto;
}

.status-pending {
  background-color: #fef3c7;
  color: #92400e;
}

.status-success {
  background-color: #d1fae5;
  color: #065f46;
}

.status-fail {
  background-color: #fee2e2;
  color: #991b1b;
}

.status-default {
  background-color: #e5e7eb;
  color: #374151;
}
</style>
