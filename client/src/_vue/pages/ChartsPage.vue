<script setup lang="ts">
import SlotSkeleton from "../components/layout/SlotSkeleton.vue";
import {onMounted} from "vue";
import MonthlyCashFlowWidget from "../features/MonthlyCashFlowWidget.vue";
import MonthlyCategoryBreakdownWidget from "../features/MonthlyCategoryBreakdownWidget.vue";
import {useTransactionStore} from "../../services/stores/transaction_store.ts";

const transactionStore = useTransactionStore();

onMounted(async () => {
    await transactionStore.getCategories();
});

</script>

<template>
    <main class="flex flex-column w-full p-2 align-items-center" style="height: 100%;">

        <div class="flex flex-column justify-content-center p-3 w-full gap-2 border-round-md"
             style="max-width: 1000px;">

            <SlotSkeleton bg="transparent">
                <div class="w-full flex flex-row justify-content-between p-1 gap-2">
                    <div class="w-full flex flex-column gap-2">
                        <h3 style="font-weight: bold;"> Chart view </h3>
                        <div> A better look at your cash-flow. </div>
                    </div>
                </div>
            </SlotSkeleton>

            <div class="w-full flex flex-row justify-content-between p-1">
                <h4>Monthly cash-flow breakdown </h4>
            </div>

            <SlotSkeleton bg="secondary">
                <MonthlyCashFlowWidget />
            </SlotSkeleton>

            <div class="w-full flex flex-row justify-content-between p-1">
                <h4>Monthly category display</h4>
            </div>
            <SlotSkeleton bg="secondary">
                <MonthlyCategoryBreakdownWidget />
            </SlotSkeleton>
        </div>



    </main>
</template>

<style scoped>

</style>