<script setup lang="ts">
import { ref, computed, watch } from 'vue';

type OpVal = '=' | '>=' | '<=';
type Model = { min: number|null; max: number|null; single?: number|null; singleOp?: OpVal };

const model = defineModel<Model>({ required: true });
defineProps<{ label?: string }>();

const singleVal = ref<number|null>(model.value.single ?? null);

const operators = [
  { name: 'Equals to',     value: '=' as OpVal },
  { name: 'Greater than',  value: '>=' as OpVal },
  { name: 'Less than',     value: '<=' as OpVal },
];

const useRange = ref(false);

const singleDisabled = computed(() => useRange.value);
const rangeDisabled  = computed(() => !useRange.value);

const selectedOperator = ref(opObjFromValue(model.value.singleOp));
const filteredOperators = ref<typeof operators>([...operators]);

watch(() => model.value.single, v => { if (v !== singleVal.value) singleVal.value = v ?? null; });
watch(() => model.value.singleOp, v => { selectedOperator.value = opObjFromValue(v); });

function opObjFromValue(v: OpVal|undefined) {
    return operators.find(o => o.value === (v ?? '=')) ?? operators[0];
}

function onSingleInput(v: number|null) {
  singleVal.value = v;
  model.value.min = null;
  model.value.max = null;
  model.value.single = v;
  model.value.singleOp = selectedOperator.value.value;
}

function onOpSelect(e: { value: { name: string; value: OpVal } }) {
  selectedOperator.value = e.value;
  if (singleVal.value !== null) {
    model.value.single = singleVal.value;
    model.value.singleOp = selectedOperator.value.value;
    model.value.min = null;
    model.value.max = null;
  }
}

function onRangeMin(v: number|null) {
  singleVal.value = null;
  model.value.single = null;
  model.value.singleOp = '=';
  model.value.min = v;
}

function onRangeMax(v: number|null) {
  singleVal.value = null;
  model.value.single = null;
  model.value.singleOp = '=';
  model.value.max = v;
}

const searchOperator = (event: any) => {
  const q = (event.query ?? '').trim().toLowerCase();
  filteredOperators.value = q
      ? operators.filter(o => o.name.toLowerCase().startsWith(q))
      : [...operators];
};
</script>

<template>
    <div class="flex flex-column gap-2 w-full">

        <div v-if="!useRange" class="flex flex-row w-full">
            <AutoComplete size="small" class="w-full" :modelValue="selectedOperator"
                          :disabled="singleDisabled" :suggestions="filteredOperators"   optionLabel="name"
                          placeholder="Select operator" dropdown @complete="searchOperator" @item-select="onOpSelect"/>
        </div>

        <div v-if="!useRange" class="flex flex-row w-full">
            <IftaLabel class="w-full">
                <InputNumber size="small" class="w-full" inputId="single" :disabled="singleDisabled" :modelValue="singleVal" @update:modelValue="onSingleInput"
                             mode="currency" currency="EUR" locale="de-DE" placeholder="0,00 €"/>
                <label for="single">Single value</label>
            </IftaLabel>
        </div>

        <div v-if="useRange" class="flex flex-column gap-2 w-full">
            <IftaLabel>
                <InputNumber size="small" class="w-full" inputId="range_min" :disabled="rangeDisabled" :modelValue="model.min" @update:modelValue="onRangeMin"
                             mode="currency" currency="EUR" locale="de-DE" placeholder="0,00 €"/>
                <label for="range_min">Min</label>
            </IftaLabel>

            <IftaLabel>
                <InputNumber size="small" class="w-full" inputId="range_max" :disabled="rangeDisabled" :modelValue="model.max" @update:modelValue="onRangeMax"
                             mode="currency" currency="EUR" locale="de-DE" placeholder="0,00 €"/>
                <label for="range_max">Max</label>
            </IftaLabel>
        </div>

        <div class="flex align-items-center gap-2">
            <Checkbox v-model="useRange" inputId="useRange" binary />
            <label for="useRange">Use range</label>
        </div>

    </div>
</template>

<style scoped></style>
