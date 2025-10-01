<script setup lang="ts">
import { computed } from "vue";
import type { MonthlyCashFlowResponse } from "../../../models/chart_models";

const props = defineProps<{
    data: MonthlyCashFlowResponse
}>();

const labels = computed(() =>
    props.data.series.map(m => `M${m.month}`)
);

const inflows = computed(() =>
    props.data.series.map(m => Number(m.inflows))
);

const outflows = computed(() =>
    props.data.series.map(m => Number(m.outflows))
);

const chartData = computed(() => ({
    labels: labels.value,
    datasets: [
        {
            label: "Inflows",
            data: inflows.value,
            borderColor: "green",
            backgroundColor: "rgba(0, 128, 0, 0.2)",
            tension: 0.3
        },
        {
            label: "Outflows",
            data: outflows.value,
            borderColor: "red",
            backgroundColor: "rgba(255, 0, 0, 0.2)",
            tension: 0.3
        }
    ]
}));

const chartOptions = {
    responsive: true,
    plugins: {
        legend: {
            position: "top"
        },
        title: {
            display: true,
            text: "Monthly Inflows vs Outflows"
        }
    }
};
</script>

<template>
    <Chart type="line" :data="chartData" :options="chartOptions" style="width: 100%; height: 400px;" />
</template>