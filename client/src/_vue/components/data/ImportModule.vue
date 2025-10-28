<script setup lang="ts">
import {ref} from "vue";
import ImportCash from "../../features/ImportCash.vue";

const emit = defineEmits<{
    (e: 'refreshData', value: string): void;
}>();

const selectedRef = ref("");

async function completeAction(val: string) {
    emit("refreshData", val);
    selectedRef.value = "";
}

</script>

<template>
    <div style="min-height: 350px;">
        <div class="flex flex-row gap-2 p-2 mb-2 align-items-center cursor-pointer font-bold hoverable"
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
            <div v-else-if="selectedRef === 'accounts'">Account imports not currently supported</div>
            <div v-else-if="selectedRef === 'categories'">Category imports not currently supported</div>
            <ImportCash v-else-if="selectedRef === 'transactions'" @completeImport="completeAction( 'import')"/>
            <div v-else-if="selectedRef === 'investments'">Investment transfers not currently supported</div>
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