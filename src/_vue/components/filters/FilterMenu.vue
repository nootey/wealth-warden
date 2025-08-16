<script setup lang="ts">

import {reactive, computed, ref} from 'vue';
import type { FilterObj } from '../../../models/shared_models';
import {resolveFor} from "../../../services/filter_registry.ts";

const props = defineProps<{
  columns: Array<{ field: string; header: string }>
  apiSource?: string
}>();

const items = computed(() => props.columns.map(c => {
  const { def, icon } = resolveFor(c);
  return { col: c, def, icon, label: c.header, key: c.field };
}));

const selectedKey = ref<string | null>(null);
const models = reactive<Record<string, any>>({});
items.value.forEach(i => { models[i.key] = i.def.makeModel(); });

const activeItem = computed(() => items.value.find(i => i.key === selectedKey.value) || null);

const emit = defineEmits<{
  (e:'apply', payload: FilterObj[]): void;
  (e:'clear'): void;
  (e:'cancel'): void;
}>();

function apply() {
  const list: FilterObj[] = items.value.flatMap(i =>
      i.def.toFilters(models[i.key], { field: i.col.field, source: props.apiSource ?? "" })
  );
  emit('apply', list);
}
function clear() {
  for (const i of items.value) models[i.key] = i.def.makeModel();
  emit('clear');
}

</script>

<template>
  <div class="flex flex-row w-100 gap-2 p-3">
    <div class="flex flex-column w-25 gap-2 p-1">
      <button v-for="i in items" :key="i.key"
          class="flex align-items-center gap-2 p-2 w-full menu-button" :class="{ active: i.key === selectedKey }"
          @click="selectedKey = i.key" style="background-color: transparent; border: 2px solid var(--border-color); border-radius: 10px;">
        <i :class="i.icon" />
        <span>{{ i.label }}</span>
      </button>
    </div>

    <div class="flex flex-column w-100">
      <keep-alive>
        <component v-if="activeItem" v-model="models[activeItem.key]"
            :is="activeItem.def.component" :field="activeItem.col.field" :label="activeItem.col.header">
        </component>
      </keep-alive>
    </div>
  </div>
  <div class="flex flex-row w-full justify-content-end align-items-center gap-3 p-1">
    <div class="hover_icon" style="margin-right: auto;" @click="clear">Clear filters</div>
    <div class="hover_icon" @click="$emit('cancel')">Cancel</div>
    <Button size="small" label="Apply" class="main-button" @click="apply"></Button>
  </div>
</template>

<style scoped>
.menu-button:hover{
  cursor: pointer;
  background-color: var(--background-secondary) !important;
}
</style>