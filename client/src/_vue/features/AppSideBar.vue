<template>
  <Drawer
    id="drawer"
    v-model:visible="open"
    header="Monthly stats"
    position="right"
    style="width: 100%; max-width: 468px"
  >
    <template #container="{ closeCallback }">
      <div class="flex flex-col w-full p-2">
        <div class="flex flex-row justify-between p-2">
          <h3>Daily stats</h3>
          <i class="pi pi-times hover-icon" @click="closeCallback" />
        </div>

        <div class="flex flex-col w-full ml-2">
          <DailyStats />
        </div>
      </div>

      <div class="flex flex-col w-full ml-2 gap-1">
        <div class="flex flex-row justify-between ml-2">
          <h3>Monthly stats</h3>
        </div>

        <div class="flex flex-col w-full p-2">
          <MonthlyStats />
        </div>
      </div>
    </template>
  </Drawer>
</template>

<script setup lang="ts">
import { ref } from "vue";
import MonthlyStats from "../components/data/MonthlyStats.vue";
import DailyStats from "../components/data/DailyStats.vue";

const open = ref(false);

const toggle = () => (open.value = !open.value);

defineExpose({ open, toggle });
</script>

<style scoped lang="scss">
@media (max-width: 768px) {
  #drawer {
    max-width: 100% !important;
  }
}
</style>
