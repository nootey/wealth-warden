<script setup lang="ts">
import { ref, watch} from "vue";
import { Chart as ChartJS, ArcElement, Tooltip, Legend, PieController } from "chart.js";
import { PieChart } from "vue-chart-3";
import type { ChartData } from "chart.js";

// Register necessary Chart.js components
ChartJS.register(PieController, ArcElement, Tooltip, Legend);

const props = defineProps<{
  values: number[];
  labels: string[];
}>();

const chartData = ref<ChartData<"pie">>({
  labels: props.labels,
  datasets: [
    {
      data: props.values,
      backgroundColor: ["#36A2EB", "#FF6384", "#FFCE56", "#4BC0C0", "#9966FF", "#FF9F40"]
    }
  ]
});

const generateColors = (count: number) => {
  const colors = [
    "#36A2EB", "#FF6384", "#FFCE56", "#4BC0C0", "#9966FF", "#FF9F40"
  ];

  // If there are more categories than predefined colors, generate random colors
  while (colors.length < count) {
    colors.push(`#${Math.floor(Math.random() * 16777215).toString(16)}`);
  }

  return colors.slice(0, count);
};

watch(
    [() => props.values, () => props.labels],
    ([newValues, newLabels]) => {
      chartData.value = {
        labels: newLabels,
        datasets: [
          {
            data: newValues,
            backgroundColor: generateColors(newLabels.length)
          }
        ]
      };
    },
    { immediate: true }
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

