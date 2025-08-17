<script setup lang="ts">

import {reactive, computed, ref, watch} from 'vue';
import type { FilterObj } from '../../../models/shared_models';
import {type Column, resolveFor} from "../../../services/filter_registry.ts";

const props = defineProps<{
  columns: Column[];
  apiSource?: string
  value?: FilterObj[]
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
  (e:'update:value', payload: FilterObj[]): void; // <--
  (e:'apply', payload: FilterObj[]): void;
  (e:'clear'): void;
  (e:'cancel'): void;
}>();

// initialize
hydrateFromFilters(props.value);

// keep in sync if parent changes saved filters
watch(() => props.value, (v) => hydrateFromFilters(v), { deep: true });

// call reset when columns change
watch(items, () => hydrateFromFilters(props.value));

function apply() {
  const list: FilterObj[] = items.value.flatMap(i =>
      i.def.toFilters(models[i.key], { field: i.col.field, source: props.apiSource ?? "" })
  );
  emit('update:value', list);
  emit('apply', list);
}
function clear() {
  resetModels();
  emit('update:value', []);
  emit('clear');
}

function resetModels() {
  items.value.forEach(i => { models[i.key] = i.def.makeModel(); });
}

function hydrateFromFilters(list: FilterObj[]|undefined) {
  resetModels();
  if (!list?.length) return;

  for (const i of items.value) {
    const rel = list.filter(f => f.field === i.col.field);

    if (i.col.type === 'date') {
      const m = i.def.makeModel();
      for (const f of rel) {
        if (f.operator === '>=') m.from = f.value ?? null;
        if (f.operator === '<=') m.to   = f.value ?? null;
      }
      models[i.key] = m;
    }

    else if (i.col.type === 'number' || /^amount$|^balance$/.test(i.col.field)) {
      const m = i.def.makeModel();
      for (const f of rel) {
        if (f.operator === '>=') m.min = f.value ?? null;
        if (f.operator === '<=') m.max = f.value ?? null;
      }
      models[i.key] = m;
    }

    else if (i.col.type === 'enum') {
      const eqs = rel.filter(f => f.operator === '=' || f.operator === 'equals');
      const selected = eqs.map(f => f.value);
      models[i.key] = selected.length ? selected : null;
    }

    else {
      const like = rel.find(f => f.operator === 'like');
      models[i.key] = like?.value ?? null;
    }
  }
}

function onCommit() {
  if (activeItem.value?.col.type !== 'enum') {
    apply();
  }
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

    <div class="flex flex-column w-full">
      <component v-if="activeItem" v-model="models[activeItem.key]"
                 :is="activeItem.def.component"
                 :field="activeItem.col.field"
                 :label="activeItem.col.header"
                 v-bind="activeItem.def.passProps"
                 @commit="onCommit"
      >
      </component>
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