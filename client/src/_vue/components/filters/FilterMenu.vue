<script setup lang="ts">
import { reactive, computed, ref, watch } from "vue";
import type { FilterObj } from "../../../models/shared_models";
import { type Column, resolveFor } from "../../../services/filter_registry.ts";

const props = defineProps<{
  columns: Column[];
  apiSource?: string;
  value?: FilterObj[];
}>();

const items = computed(() =>
  props.columns
    .filter((c) => !c.hideFromFilter)
    .map((c) => {
      const { def, icon } = resolveFor(c);
      return { col: c, def, icon, label: c.header, key: c.field };
    }),
);

const selectedKey = ref<string | null>(null);
const models = reactive<Record<string, any>>({});
items.value.forEach((i) => {
  models[i.key] = i.def.makeModel();
});

const activeItem = computed(
  () => items.value.find((i) => i.key === selectedKey.value) || null,
);

const emit = defineEmits<{
  (e: "update:value", payload: FilterObj[]): void; // <--
  (e: "apply", payload: FilterObj[]): void;
  (e: "clear"): void;
  (e: "cancel"): void;
}>();

// initialize
hydrateFromFilters(props.value);

// keep in sync if parent changes saved filters
watch(
  () => props.value,
  (v) => hydrateFromFilters(v),
  { deep: true },
);

// call reset when columns change
watch(items, () => hydrateFromFilters(props.value));

function apply() {
  const list: FilterObj[] = items.value.flatMap((i) =>
    i.def.toFilters(models[i.key], {
      field: i.col.field,
      source: props.apiSource ?? "",
    }),
  );
  emit("update:value", list);
  emit("apply", list);
}
function clear() {
  resetModels();
  emit("update:value", []);
  emit("clear");
}

function resetModels() {
  items.value.forEach((i) => {
    models[i.key] = i.def.makeModel();
  });
}

function hydrateFromFilters(list: FilterObj[] | undefined) {
  resetModels();
  if (!list?.length) return;

  for (const i of items.value) {
    const rel = list.filter((f) => f.field === i.col.field);

    if (i.col.type === "date") {
      const m = i.def.makeModel();
      for (const f of rel) {
        if (f.operator === ">=") m.from = f.value ?? null;
        if (f.operator === "<=") m.to = f.value ?? null;
      }
      models[i.key] = m;
    } else if (i.col.type === "int") {
      const eq = rel.find((f) => f.operator === "=");
      models[i.key] = eq?.value != null ? Number(eq.value) : null;
    } else if (
      i.col.type === "number" ||
      /^amount$|^balance$/.test(i.col.field)
    ) {
      const m = i.def.makeModel();
      for (const f of rel) {
        if (f.operator === "=" || f.operator === ">=" || f.operator === "<=") {
          if (f.operator === "=") {
            m.single = parseFloat(String(f.value));
            m.singleOp = "=";
          }
          if (f.operator === ">=") m.min = parseFloat(String(f.value));
          if (f.operator === "<=") m.max = parseFloat(String(f.value));
        }
      }
      if (m.single != null) {
        m.min = null;
        m.max = null;
      }
      models[i.key] = m;
    } else if (i.col.type === "enum") {
      const eqs = rel.filter(
        (f) => f.operator === "=" || f.operator === "equals",
      );
      const selected = eqs.map((f) => f.value);
      models[i.key] = selected.length ? selected : null;
    } else {
      const like = rel.find((f) => f.operator === "like");
      models[i.key] = like?.value ?? null;
    }
  }
}

function onCommit() {
  if (activeItem.value?.col.type !== "enum") {
    apply();
  }
}
</script>

<template>
  <div
    id="mobile-row"
    class="flex flex-row w-full gap-3 p-3"
    style="height: 280px"
  >
    <div
      id="filter-fields"
      class="flex flex-col w-4/12 gap-1"
      style="overflow-y: auto"
    >
      <div
        v-for="i in items"
        :key="i.key"
        class="flex items-center gap-1 w-full align-center hover-icon"
        :class="{ active: i.key === selectedKey }"
        style="
          padding: 5px;
          background-color: transparent;
          border: 2px solid var(--border-color);
          border-radius: 10px;
        "
        @click="selectedKey = i.key"
      >
        <i :class="i.icon" class="text-sm" />
        <span
          style="
            display: inline-block;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
          "
        >
          {{ i.label }}
        </span>
      </div>
    </div>
    <div class="flex flex-col w-8/12">
      <component
        :is="activeItem.def.component"
        v-if="activeItem"
        v-model="models[activeItem.key]"
        :field="activeItem.col.field"
        :label="activeItem.col.header"
        v-bind="activeItem.def.passProps"
        @commit="onCommit"
      />
    </div>
  </div>

  <div
    id="mobile-row"
    class="flex flex-row w-full justify-end items-center gap-4 p-2"
  >
    <Button
      text
      severity="danger"
      size="small"
      icon="pi pi-filter-slash"
      label="Clear filters"
      class="mr-auto"
      @click="clear"
    />
    <div class="hover-icon" @click="$emit('cancel')">Cancel</div>
    <Button size="small" label="Apply" class="main-button" @click="apply" />
  </div>
</template>

<style scoped>
@media (max-width: 768px) {
  #mobile-row i {
    display: none;
  }
  #filter-fields > div {
    justify-content: center;
  }
}
</style>
