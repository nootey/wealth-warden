<script setup lang="ts">
import {inject, ref, watch} from "vue";
import type {FilterObj} from "../../../models/shared_models.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import currencyHelper from "../../../utils/currency_helper.ts";
import dateHelper from "../../../utils/date_helper.ts";

const props = defineProps<{
  activeFilters: FilterObj[];
  showOnlyActive: boolean;
  activeFilter: string;
}>();

const removeFilter = inject<(originalIndex: number) => void>("removeFilter", () => {});
type FilterWithIndex = FilterObj & { originalIndex: number };

const filters = ref<FilterWithIndex[]>([]);

watch(
    () => [props.activeFilters, props.showOnlyActive, props.activeFilter],
    initFilters,
    { immediate: true, deep: true }
);

function initFilters() {
  const withIndex = props.activeFilters.map((f, i) => ({ ...f, originalIndex: i }));

  filters.value = props.showOnlyActive
      ? withIndex.filter(f => f.field === props.activeFilter)
      : withIndex;
}

function clearFilter(originalIndex: number): void {
  removeFilter && removeFilter(originalIndex);
  initFilters();
}

const icons: Record<string, string> = {
  'account': 'pi pi-wallet',
  'category': 'pi pi-book',
};

function iconClass(field: string | null): string | null {
  if (!field) return null;
  const key = field.toLowerCase();
  return icons[key] ?? null;
}

</script>

<template>
  <div v-if="filters.length > 0" class="flex flex-wrap gap-1 w-full" style="line-height: 1; max-height: 135px; overflow-y: auto;">
    <Chip v-for="filter in filters" :key="filter.originalIndex"
          style="background-color: transparent; border: 3px solid var(--border-color); padding: 0.65rem;">
      <div  class="flex flex-row align-items-center gap-2">

        <div v-if="iconClass(filter.field)"
             v-tooltip="filter.field"
             style="width: 16px; height: 16px; border-radius: 50%; display:flex;
                    align-items:center; justify-content:center; font-size:0.75rem;
                    color:white; border:2px solid var(--border-color);">
          <i :class="iconClass(filter.field)" style="font-size: 0.75rem;"></i>
        </div>

        <template v-else>
          <div>{{ vueHelper.capitalize(filter.field) }}</div>
          <div>{{ filter.operator }}</div>
        </template>

        <div>{{ currencyHelper.mightBeBalance(filter.field) ? vueHelper.displayAsCurrency(filter.value) :
            (dateHelper.mightBeDate(filter.field) ? dateHelper.formatDate(filter.value) : (filter.display ? filter.display : filter.value)) }}</div>
        <div
            @click="clearFilter(filter.originalIndex)"
            class="flex align-items-center justify-content-center">
          <i class="pi pi-times hover_icon" style="color: grey; font-size: 0.75rem;" ></i>
        </div>
      </div>

    </Chip>
  </div>
  <div v-else>
    <span> {{ "No filters active"}}</span>
  </div>
</template>

<style scoped>

</style>