<script setup lang="ts">
type Opt = string | number | Record<string, any>;

const model = defineModel<Opt[] | null>();
const props = defineProps<{
  label?: string;
  // If you pass objects, also pass optionLabel/Value
  options: Opt[];
  optionLabel?: string;
  optionValue?: string;
  filter?: boolean;         // enable built-in search
  placeholder?: string;
}>();
</script>

<template>
  <div class="flex flex-column gap-2">
    <label class="text-sm">{{ label }}</label>
    <MultiSelect
        v-model="model"
        :options="props.options"
        :optionLabel="props.optionLabel"
        :optionValue="props.optionValue"
        :filter="props.filter ?? true"
        :placeholder="props.placeholder ?? 'Select…'"
        display="chip"
        class="w-full"
    />
  </div>
</template>

<style scoped>

</style>