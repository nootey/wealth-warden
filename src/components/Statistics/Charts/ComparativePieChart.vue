<script setup lang="ts">
import {onMounted, ref, watch} from "vue";
import { Chart as ChartJS, ArcElement, Tooltip, Legend, PieController } from "chart.js";
import { PieChart } from "vue-chart-3";
import type { ChartData } from "chart.js";

// Register necessary Chart.js components
ChartJS.register(PieController, ArcElement, Tooltip, Legend);

const props = defineProps<{
  firstValue: number;
  firstLabel: string;
  secondValue: number;
  secondLabel: string;
}>();


// Initialize the chart with both values set to 0
const chartData = ref<ChartData<"pie">>({
  labels: [props.firstLabel, props.secondLabel],
  datasets: [
    {
      data: [0, 0],
      backgroundColor: ["#36A2EB", "#FF6384"]
    }
  ]
});

// Watch for prop changes and update chart data
watch(
    () => [props.firstValue, props.secondValue, props.firstLabel, props.secondLabel],
    ([newFirstValue, newSecondValue, newFirstLabel, newSecondLabel]) => {
      chartData.value = {
        labels: [newFirstLabel, newSecondLabel],
        datasets: [
          {
            data: [newFirstValue, newSecondValue],
            backgroundColor: ["#36A2EB", "#FF6384"]
          }
        ]
      };
    },
    { immediate: true } // Run the watcher immediately to set the initial values
);

const chartOptions = ref({
  responsive: true,
  animation: {
    animateScale: true,
    animateRotate: true,
    duration: 1500,
    easing: "easeOutCubic",
  },
  plugins: {
    legend: {
      position: "top",
    },
  },
});

</script>

<template>
  <div>
    <PieChart :chart-data="chartData" :chart-options="chartOptions" />
  </div>
</template>

