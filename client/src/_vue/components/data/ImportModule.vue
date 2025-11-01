<script setup lang="ts">
import {computed, ref} from "vue";
import ImportTransactions from "../../features/ImportTransactions.vue";
import ImportInvestments from "../../features/ImportInvestments.vue";
import ImportAccounts from "../../features/ImportAccounts.vue";
import ImportCategories from "../../features/ImportCategories.vue";

const emit = defineEmits<{
    (e: 'refreshData', value: string): void;
}>();

const selectedRef = ref("");

const accRef = ref<InstanceType<typeof ImportAccounts> | null>(null);
const catRef = ref<InstanceType<typeof ImportCategories> | null>(null);
const txnRef = ref<InstanceType<typeof ImportTransactions> | null>(null);
const invRef = ref<InstanceType<typeof ImportInvestments> | null>(null);

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
        case "accounts":
            return accRef.value?.isDisabled ?? true;
        case "categories":
            return catRef.value?.isDisabled ?? true;
        default:
            return true
    }
});

defineExpose({isDisabled, startOperation})

</script>

<template>
    <div style="min-height: 350px;">
        <div v-if="selectedRef !== ''" class="flex flex-row gap-2 p-3 mb-2 align-items-center cursor-pointer font-bold hoverable"
             style="color: var(--text-primary)">
            <i class="pi pi-angle-left"></i>
            <span @click="selectedRef = ''">Back</span>
        </div>

        <Transition name="slide-down" mode="out-in">
            <div class="flex flex-column w-full gap-2" v-if="!selectedRef">
                <span>You can manually import various types of data via JSON.</span>
                <div class="flex flex-column w-full border-round-2xl p-2 gap-2" style="background: var(--background-secondary)">
                    <span>Sources</span>
                    <div class="flex flex-column w-full border-round-2xl p-2 gap-2" style="background: var(--background-primary)">
                        <div class="flex flex-row gap-2 p-2 align-items-center hover-icon" @click="selectedRef = 'custom'">
                            <i class="pi pi-upload" style="color: #48F05C"></i>
                            <span>Import from zip</span>
                            <span class="text-xs" style="color: var(--text-secondary)">From exported data</span>
                            <i class="pi pi-chevron-right" style="margin-left: auto; color: var(--text-secondary)"></i>
                        </div>
                        <div style="border-bottom: 2px solid var(--border-color)"></div>
                        <div class="flex flex-row gap-2 p-2 align-items-center hover-icon" @click="selectedRef = 'accounts'">
                            <i class="pi pi-building" style="color: #F05737"></i>
                            <span>Import accounts</span>
                            <i class="pi pi-chevron-right" style="margin-left: auto; color: var(--text-secondary)"></i>
                        </div>
                        <div style="border-bottom: 2px solid var(--border-color)"></div>
                        <div class="flex flex-row gap-2 p-2 align-items-center hover-icon" @click="selectedRef = 'categories'">
                            <i class="pi pi-gift" style="color: #E39119"></i>
                            <span>Import categories</span>
                            <i class="pi pi-chevron-right" style="margin-left: auto; color: var(--text-secondary)"></i>
                        </div>
                        <div style="border-bottom: 2px solid var(--border-color)"></div>
                        <div class="flex flex-row gap-2 p-2 align-items-center hover-icon" @click="selectedRef = 'transactions'">
                            <i class="pi pi-book" style="color: #486AF0"></i>
                            <span>Import transactions</span>
                            <i class="pi pi-chevron-right" style="margin-left: auto; color: var(--text-secondary)"></i>
                        </div>
                        <div style="border-bottom: 2px solid var(--border-color)"></div>
                        <div class="flex flex-row gap-2 p-2 align-items-center hover-icon" @click="selectedRef = 'investments'">
                            <i class="pi pi-building-columns" style="color: #9948F0"></i>
                            <span>Transfer investments</span>
                            <i class="pi pi-chevron-right" style="margin-left: auto; color: var(--text-secondary)"></i>
                        </div>
                    </div>
                </div>
            </div>
            <div v-else-if="selectedRef === 'custom'">Full custom import is not currently supported</div>
            <ImportAccounts ref="accRef" v-else-if="selectedRef === 'accounts'"  @completeImport="completeAction( 'import')"/>
            <ImportCategories ref="catRef" v-else-if="selectedRef === 'categories'"  @completeImport="completeAction( 'import')"/>
            <ImportTransactions ref="txnRef" v-else-if="selectedRef === 'transactions'" @completeImport="completeAction( 'import')"/>
            <ImportInvestments ref="invRef" v-else-if="selectedRef === 'investments'" @completeTransfer="completeAction( 'import')"/>
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