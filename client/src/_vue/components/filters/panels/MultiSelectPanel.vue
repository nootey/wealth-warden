<script setup lang="ts">
import {computed} from "vue";

type Opt = string | number | Record<string, any>;
const model = defineModel<Opt[] | null>();

const props = defineProps<{
  label?: string;
  options: Opt[];
  optionLabel?: string;
  optionValue?: string;
}>();

// Normalize options
const normalizedOptions = computed(() => {
  const opts = props.options ?? [];
  if (!opts.length) return [];

  if (typeof opts[0] !== 'object') {
    return (opts as (string|number)[]).map(v => ({ label: String(v), value: v, raw: v }));
  }

  const ol = props.optionLabel ?? 'label';
  const ov =
      props.optionValue ??
      (('id' in (opts[0] as any)) ? 'id' : ol);

  return (opts as Record<string, any>[]).map(o => ({
    label: String(o[ol]),
    value: o[ov],
    raw: o
  }));
});

// v-model proxy
const internal = computed<any[]>({
  get: () => (Array.isArray(model.value) ? model.value as any[] : []),
  set: v => { model.value = v; }
});

const labelMap = computed(() => new Map(normalizedOptions.value.map(o => [o.value, o.label])));
const selectedChips = computed(() => internal.value.map(v => ({ value: v, label: labelMap.value.get(v) ?? String(v) })));

function remove(value: any) {
  internal.value = internal.value.filter(v => v !== value);
}
</script>

<template>
  <div class="flex flex-column gap-2">
    <label class="text-sm">{{ label }}</label>
    <MultiSelect
        v-model="internal"
        :options="normalizedOptions"
        optionLabel="label"
        optionValue="value"
        :filter="true"
        :filterFields="['label']"
        :filterMatchMode="'contains'"
        :placeholder="`Select ${label}`"
        display="comma"
        :maxSelectedLabels="1"
        selectedItemsLabel="{0} selected"
        class="w-full"
    />

    <div v-if="selectedChips.length" class="flex flex-wrap gap-1" style="max-height: 125px; overflow-y: auto;">
      <Chip v-for="s in selectedChips" :key="String(s.value)" class="chip">
        <span class="chip-text">{{ s.label }}</span>
        <i class="pi pi-times chip-x" @click="remove(s.value)" />
      </Chip>
    </div>
  </div>
</template>

<style scoped>

.chip {
  background: transparent;
  border: 2px solid var(--border-color);
  padding: .4rem .6rem;
}
.chip-text {
  margin-right: .35rem;
}
.chip-x {
  font-size: .8rem;
  cursor: pointer;
  opacity: .7;
}
.chip-x:hover { opacity: 1; }
</style>