<script setup lang="ts">
import {defineProps, onMounted, ref} from "vue";
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

const chartDataReady = ref(false);

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

onMounted(() => {
  // Simulate async data load
  setTimeout(() => {
    chartData.value = {
      labels: [props.firstLabel, props.secondLabel],
      datasets: [
        {
          data: [props.firstValue, props.secondValue],
          backgroundColor: ["#36A2EB", "#FF6384"]
        }
      ]
    };
    chartDataReady.value = true;
  }, 250);

});


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

