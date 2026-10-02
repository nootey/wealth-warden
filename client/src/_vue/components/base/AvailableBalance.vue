<script setup lang="ts">
import { computed } from "vue";
import vueHelper from "../../../utils/vue_helper.ts";
import { useAvailableBalance } from "../../../utils/use_available_balance.ts";
import type { Account } from "../../../models/account_models.ts";

const props = defineProps<{
  account: Account | null | undefined;
}>();

const { available } = useAvailableBalance(() => props.account);

const amount = computed(() =>
  props.account?.account_type?.classification === "liability"
    ? null
    : (available.value?.available ?? null),
);
</script>

<template>
  <span
    v-if="amount !== null"
    class="text-sm"
    style="color: var(--text-secondary)"
  >
    Available:
    <span
      :style="{
        color: Number(amount) < 0 ? 'var(--negative)' : 'var(--text-primary)',
      }"
      >{{ vueHelper.displayAsCurrency(amount) }}</span
    >
  </span>
</template>
