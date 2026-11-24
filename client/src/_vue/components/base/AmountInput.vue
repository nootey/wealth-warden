<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import Decimal from "decimal.js";
import currencyHelper from "../../../utils/currency_helper.ts";
import type { InputNumberInputEvent } from "primevue/inputnumber";

defineOptions({ inheritAttrs: false });

const props = withDefaults(
  defineProps<{
    mode?: "decimal" | "currency";
    currency?: string;
    locale?: string;
    min?: number;
    max?: number;
    minFractionDigits?: number;
    maxFractionDigits?: number;
    size?: "small" | "large";
    fluid?: boolean;
    readonly?: boolean;
    disabled?: boolean;
  }>(),
  {
    mode: "decimal",
    currency: undefined,
    locale: undefined,
    min: undefined,
    max: undefined,
    minFractionDigits: undefined,
    maxFractionDigits: undefined,
    size: undefined,
    fluid: undefined,
  },
);

const model = defineModel<number | null>();

const OPERATOR_KEYS = ["+", "-", "*", "/", "%", "="];

const root = ref<HTMLElement | null>(null);
const expression = ref<string | null>(null);

// InputNumber only writes its model on blur, so track what is typed.
const typedValue = ref<number | null>(model.value ?? null);
watch(model, (value) => (typedValue.value = value ?? null));

const decimalMark = computed(
  () =>
    new Intl.NumberFormat(props.locale)
      .formatToParts(1.5)
      .find((part) => part.type === "decimal")?.value ?? ".",
);

// Clamps the same way InputNumber does on blur.
const result = computed(() =>
  expression.value === null
    ? null
    : (currencyHelper
        .evaluateExpression(expression.value)
        ?.clampedTo(props.min ?? -Infinity, props.max ?? Infinity) ?? null),
);

const preview = computed(() =>
  result.value
    ? new Intl.NumberFormat(props.locale, {
        style: props.mode,
        currency: props.currency,
        minimumFractionDigits: props.minFractionDigits,
        maximumFractionDigits: props.maxFractionDigits,
      }).format(result.value.toNumber())
    : null,
);

function onTyped(event: InputNumberInputEvent) {
  typedValue.value = typeof event.value === "number" ? event.value : null;
}

function focusInput() {
  nextTick(() => {
    const input = root.value?.querySelector("input");
    if (!input) return;
    input.focus();
    // In the number field, stop after the last digit, before a currency suffix.
    const end =
      expression.value === null
        ? input.value.search(/\d\D*$/) + 1
        : input.value.length;
    input.setSelectionRange(end, end);
  });
}

function onKeydownCapture(event: KeyboardEvent) {
  if (expression.value !== null || props.readonly || props.disabled) return;
  if (event.ctrlKey || event.metaKey || event.altKey || event.isComposing) {
    return;
  }
  if (!OPERATOR_KEYS.includes(event.key)) return;

  const current = typedValue.value;
  if (current === null && event.key !== "=") return;
  // A minus at the start still makes the number negative.
  const input = event.target as HTMLInputElement;
  if (event.key === "-" && input.selectionStart === 0) return;

  event.preventDefault();
  event.stopPropagation();

  model.value = current;
  const start =
    current === null
      ? ""
      : new Decimal(current).toFixed().replace(".", decimalMark.value);
  expression.value = event.key === "=" ? start : start + event.key;
  focusInput();
}

function applyExpression(): boolean {
  if (expression.value?.trim() === "") {
    model.value = null;
    return true;
  }
  if (!result.value) return false;
  model.value = result.value.toNumber();
  return true;
}

function closeExpression(refocus: boolean) {
  expression.value = null;
  if (refocus) focusInput();
}

function onExpressionKeydown(event: KeyboardEvent) {
  if (event.key === "Enter") {
    // Stops BaseForm from submitting on the Enter that applies the calculation.
    event.preventDefault();
    if (applyExpression()) closeExpression(true);
  } else if (event.key === "Escape") {
    // Stops the dialog from closing.
    event.stopPropagation();
    closeExpression(true);
  }
}

function onExpressionBlur() {
  if (expression.value === null) return;
  applyExpression();
  closeExpression(false);
}
</script>

<template>
  <div
    ref="root"
    class="flex flex-col gap-1"
    @keydown.capture="onKeydownCapture"
  >
    <InputText
      v-if="expression !== null"
      v-model="expression"
      :size="size"
      :fluid="fluid"
      @keydown="onExpressionKeydown"
      @blur="onExpressionBlur"
    />
    <InputNumber
      v-else
      v-model="model"
      v-bind="$attrs"
      :mode="mode"
      :currency="currency"
      :locale="locale"
      :min="min"
      :max="max"
      :min-fraction-digits="minFractionDigits"
      :max-fraction-digits="maxFractionDigits"
      :size="size"
      :fluid="fluid"
      :readonly="readonly"
      :disabled="disabled"
      @input="onTyped"
    />
    <span
      v-if="expression"
      class="text-sm"
      style="color: var(--text-secondary)"
    >
      {{ preview ? `= ${preview}` : "Not a valid calculation" }}
    </span>
  </div>
</template>
