<script setup lang="ts">
import { ref, watch } from 'vue';

const props = defineProps(["selectedValues", "availableValues", "optionLabel", "toUppercase"]);
const emit = defineEmits(["getData", "update:selectedValues"]);

const localSelectedValues = ref(props.selectedValues);

watch(localSelectedValues, (newVal) => {
  emit('update:selectedValues', newVal);
});
</script>

<template>
  <div class="flex flex-column p-2 gap-3 justify-content-center">
    <MultiSelect
        v-model="localSelectedValues"
        :optionLabel="optionLabel"
        :options="availableValues"
        placeholder="Select filter">
      <template #option="slotProps">
        {{ optionLabel !== "" ? slotProps.option[optionLabel] : slotProps.option}}
      </template>
    </MultiSelect>

    <Button
        label="Save"
        style="width: min-content; margin: 0 auto;"
        @click="emit('getData')"
    ></Button>
  </div>
</template>

<style scoped>

</style>