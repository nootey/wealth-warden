<script setup lang="ts">
import { computed, ref } from "vue";
import ImportTransactions from "../../features/imports/ImportTransactions.vue";
import ImportInvestments from "../../features/imports/ImportInvestments.vue";
import ImportAccounts from "../../features/imports/ImportAccounts.vue";
import ImportCategories from "../../features/imports/ImportCategories.vue";
import ImportSavings from "../../features/imports/ImportSavings.vue";
import ImportRepayments from "../../features/imports/ImportRepayments.vue";

const emit = defineEmits<{
  (e: "refreshData", value: string): void;
}>();

const selectedRef = ref("");

const accRef = ref<InstanceType<typeof ImportAccounts> | null>(null);
const catRef = ref<InstanceType<typeof ImportCategories> | null>(null);
const txnRef = ref<InstanceType<typeof ImportTransactions> | null>(null);
const invRef = ref<InstanceType<typeof ImportInvestments> | null>(null);
const savRef = ref<InstanceType<typeof ImportSavings> | null>(null);
const repRef = ref<InstanceType<typeof ImportRepayments> | null>(null);

async function completeAction(val: string) {
  emit("refreshData", val);
  selectedRef.value = "";
}

async function startOperation() {
  switch (selectedRef.value) {
    case "transactions":
      txnRef.value?.importTransactions();
      break;
    case "investments":
      invRef.value?.transferInvestments();
      break;
    case "savings":
      savRef.value?.transferSavings();
      break;
    case "repayments":
      repRef.value?.transferRepayments();
      break;
    case "accounts":
      accRef.value?.importAccounts();
      break;
    case "categories":
      catRef.value?.importCategories();
      break;
    default:
      break;
  }
}

const isDisabled = computed(() => {
  switch (selectedRef.value) {
    case "transactions":
      return txnRef.value?.isDisabled ?? true;
    case "investments":
      return invRef.value?.isDisabled ?? true;
    case "savings":
      return savRef.value?.isDisabled ?? true;
    case "repayments":
      return repRef.value?.isDisabled ?? true;
    case "accounts":
      return accRef.value?.isDisabled ?? true;
    case "categories":
      return catRef.value?.isDisabled ?? true;
    default:
      return true;
  }
});

defineExpose({ isDisabled, startOperation });
</script>

<template>
  <div style="min-height: 350px">
    <div
      v-if="selectedRef !== ''"
      class="flex flex-row gap-2 p-3 mb-2 align-items-center cursor-pointer font-bold hoverable"
      style="color: var(--text-primary)"
    >
      <i class="pi pi-angle-left" />
      <span @click="selectedRef = ''">Back</span>
    </div>

    <Transition name="slide-down" mode="out-in">
      <div v-if="!selectedRef" class="flex flex-column w-full gap-2">
        <span>You can manually import various types of data via JSON.</span>
        <div
          class="flex flex-column w-full border-round-2xl p-2 gap-2"
          style="background: var(--background-secondary)"
        >
          <span>Sources</span>
          <div
            class="flex flex-column w-full border-round-2xl p-2 gap-2"
            style="background: var(--background-primary)"
          >
            <div
              class="flex flex-row gap-2 p-2 align-items-center hover-icon"
              @click="selectedRef = 'accounts'"
            >
              <i class="pi pi-building" style="color: #f05737" />
              <span>Import accounts</span>
              <i
                class="pi pi-chevron-right"
                style="margin-left: auto; color: var(--text-secondary)"
              />
            </div>
            <div style="border-bottom: 2px solid var(--border-color)" />
            <div
              class="flex flex-row gap-2 p-2 align-items-center hover-icon"
              @click="selectedRef = 'categories'"
            >
              <i class="pi pi-gift" style="color: #e39119" />
              <span>Import categories</span>
              <i
                class="pi pi-chevron-right"
                style="margin-left: auto; color: var(--text-secondary)"
              />
            </div>
            <div style="border-bottom: 2px solid var(--border-color)" />
            <div
              class="flex flex-row gap-2 p-2 align-items-center hover-icon"
              @click="selectedRef = 'transactions'"
            >
              <i class="pi pi-book" style="color: #486af0" />
              <span>Import transactions</span>
              <i
                class="pi pi-chevron-right"
                style="margin-left: auto; color: var(--text-secondary)"
              />
            </div>
            <div style="border-bottom: 2px solid var(--border-color)" />
            <div
              class="flex flex-row gap-2 p-2 align-items-center hover-icon"
              @click="selectedRef = 'investments'"
            >
              <i class="pi pi-chart-line" style="color: #9948f0" />
              <span>Transfer investments</span>
              <i
                class="pi pi-chevron-right"
                style="margin-left: auto; color: var(--text-secondary)"
              />
            </div>
            <div style="border-bottom: 2px solid var(--border-color)" />
            <div
              class="flex flex-row gap-2 p-2 align-items-center hover-icon"
              @click="selectedRef = 'savings'"
            >
              <i class="pi pi-building-columns" style="color: #c166f2" />
              <span>Transfer savings</span>
              <i
                class="pi pi-chevron-right"
                style="margin-left: auto; color: var(--text-secondary)"
              />
            </div>
            <div style="border-bottom: 2px solid var(--border-color)" />
            <div
              class="flex flex-row gap-2 p-2 align-items-center hover-icon"
              @click="selectedRef = 'repayments'"
            >
              <i class="pi pi-upload" style="color: #48f05c" />
              <span>Transfer repayments</span>
              <i
                class="pi pi-chevron-right"
                style="margin-left: auto; color: var(--text-secondary)"
              />
            </div>
          </div>
        </div>
      </div>
      <div v-else-if="selectedRef === 'custom'">
        Full custom import is not currently supported
      </div>
      <ImportAccounts
        v-else-if="selectedRef === 'accounts'"
        ref="accRef"
        @complete-import="completeAction('import')"
      />
      <ImportCategories
        v-else-if="selectedRef === 'categories'"
        ref="catRef"
        @complete-import="completeAction('import')"
      />
      <ImportTransactions
        v-else-if="selectedRef === 'transactions'"
        ref="txnRef"
        @complete-import="completeAction('import')"
      />
      <ImportInvestments
        v-else-if="selectedRef === 'investments'"
        ref="invRef"
        @complete-transfer="completeAction('import')"
      />
      <ImportSavings
        v-else-if="selectedRef === 'savings'"
        ref="savRef"
        @complete-transfer="completeAction('import')"
      />
      <ImportRepayments
        v-else-if="selectedRef === 'repayments'"
        ref="repRef"
        @complete-transfer="completeAction('import')"
      />
    </Transition>
  </div>
</template>

<style scoped>
.slide-down-enter-active,
.slide-down-leave-active {
  transition: all 0.3s ease;
}

.slide-down-enter-from {
  transform: translateY(-10px);
  opacity: 0;
}

.slide-down-leave-to {
  transform: translateY(-10px);
  opacity: 0;
}
</style>
