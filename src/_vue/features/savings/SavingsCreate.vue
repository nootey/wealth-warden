<script setup lang="ts">
import {useSavingsStore} from "../../../services/stores/savingsStore.ts";
import {useToastStore} from "../../../services/stores/toastStore.ts";
import {computed, ref} from "vue";

const savingsStore = useSavingsStore();
const toastStore = useToastStore();
const newSavingsAllocaton = ref(initSavingsAllocation(false));

const savingsCategories = computed(() => savingsStore.inflowCategories);
const filteredSavingsCategories = ref([]);
</script>

<template>

</template>

<style scoped>

</style>