<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{
  currentPage: number;
  totalRecords: number;
  rowsPerPage: number;
}>();

const emit = defineEmits<{
  pageChange: [page: number];
}>();

const totalPages = computed(() =>
  Math.ceil(props.totalRecords / props.rowsPerPage),
);

const canGoPrev = computed(() => props.currentPage > 1);
const canGoNext = computed(() => props.currentPage < totalPages.value);

const goToPage = (page: number) => {
  if (page >= 1 && page <= totalPages.value) {
    emit("pageChange", page);
  }
};
</script>

<template>
  <div
    v-if="totalPages > 1"
    class="flex flex-row align-items-center justify-content-center gap-3 p-2"
  >
    <i
      class="pi pi-chevron-left text-sm"
      :class="canGoPrev ? 'hover-icon' : 'opacity-30'"
      @click="canGoPrev && goToPage(currentPage - 1)"
    />
    <span class="text-sm">({{ currentPage }} of {{ totalPages }})</span>
    <i
      class="pi pi-chevron-right text-sm"
      :class="canGoNext ? 'hover-icon' : 'opacity-30'"
      @click="canGoNext && goToPage(currentPage + 1)"
    />
  </div>
</template>
