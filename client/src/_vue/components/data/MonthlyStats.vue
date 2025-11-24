<script setup lang="ts">

import {useStatisticsStore} from "../../../services/stores/statistics_store.ts";
import {onMounted, ref} from "vue";
import type {MonthlyStats} from "../../../models/statistics_models.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import vueHelper from "../../../utils/vue_helper.ts";
import ShowLoading from "../base/ShowLoading.vue";


const statsStore = useStatisticsStore();
const toastStore = useToastStore();

const loading = ref(false);

const monthlyStats = ref<MonthlyStats | null>(null);

onMounted(async () => {
    await loadStats();
})

async function loadStats() {
    try {
        loading.value = true;
        const result = await statsStore.getCurrentMonthsStats(null);

        if (!result) {
            monthlyStats.value = null;
        } else {
            monthlyStats.value = result;
        }

    } catch (e) {
        toastStore.errorResponseToast(e);
    } finally {
        loading.value = false;
    }
}

</script>

<template>
    <div v-if="!loading" class="flex flex-column p-2 gap-2">
        <h4>Info</h4>
        <span style="color: var(--text-secondary)">Monthly stats are computed for all checking accounts, which are treated as main accounts.</span>

        <br>

        <h4>Details</h4>
        <div v-if="monthlyStats" class="flex flex-column">

            <div class="flex flex-column w-full gap-2">
                <div class="flex flex-row gap-2 align-items-center">
                    <span>Inflows:</span>
                    <span><b>{{ vueHelper.displayAsCurrency(monthlyStats?.inflow!) }}</b></span>
                </div>
                <div class="flex flex-row gap-2 align-items-center">
                    <span>Outflows:</span>
                    <span><b>{{ vueHelper.displayAsCurrency(monthlyStats?.outflow!) }}</b></span>
                </div>
                <div class="flex flex-row gap-2 align-items-center">
                    <span>Take home:</span>
                    <span><b>{{ vueHelper.displayAsCurrency(monthlyStats?.take_home!) }}</b></span>
                </div>
                <div class="flex flex-row gap-2 align-items-center">
                    <span>Overflow:</span>
                    <span><b>{{ vueHelper.displayAsCurrency(monthlyStats?.overflow!) }}</b></span>
                </div>
            </div>

            <br>

            <div class="flex flex-column w-full gap-2">
                <h4>Rates</h4>

                <div class="flex flex-row gap-2 align-items-center">
                    <span>Savings:</span>
                    <span><b>{{ vueHelper.displayAsCurrency(monthlyStats.savings) }}</b></span>
                    <span>Rate:</span>
                    <span><b>{{ vueHelper.displayAsPercentage(monthlyStats.savings_rate) }}</b></span>
                </div>

                <div class="flex flex-row gap-2 align-items-center">
                    <span>Investments</span>
                    <span><b>{{ vueHelper.displayAsCurrency(monthlyStats.investments) }}</b></span>
                    <span>Rate</span>
                    <span><b>{{ vueHelper.displayAsPercentage(monthlyStats.investments_rate) }}</b></span>
                </div>

                <div class="flex flex-row gap-2 align-items-center">
                    <span>Debt repayments</span>
                    <span><b>{{ vueHelper.displayAsCurrency(monthlyStats.debt_repayments) }}</b></span>
                    <span>Rate</span>
                    <span><b>{{ vueHelper.displayAsPercentage(monthlyStats.debt_repayment_rate) }}</b></span>
                </div>
            </div>
        </div>
        <div v-else>
            <span>No checking accounts are currently available.</span>
        </div>


    </div>
    <ShowLoading v-else :numFields="7" />
</template>

<style scoped>

</style>