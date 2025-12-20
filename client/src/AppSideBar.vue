<template>
  <Drawer
    v-model:visible="open"
    header="Monthly stats"
    position="right"
    style="width: 100%; max-width: 420px"
  >
    <template #container="{ closeCallback }">
      <div class="flex flex-column w-full h-4 p-2">
        <div class="flex flex-row justify-content-between p-2">
          <h3>Monthly stats</h3>
          <i class="pi pi-times hover-icon" @click="closeCallback" />
        </div>

        <div class="flex flex-column w-full p-2">
          <MonthlyStats />
        </div>
      </div>

      <div class="flex flex-column w-full h-4 p-2">
        <div class="flex flex-row justify-content-between p-2">
          <h3>Daily stats</h3>
        </div>

        <div class="flex flex-column w-full p-2">
          <DailyStats />
        </div>
      </div>
    </template>
  </Drawer>
</template>

<script setup lang="ts">
import { ref, defineExpose } from "vue";
import MonthlyStats from "./_vue/components/data/MonthlyStats.vue";
import DailyStats from "./_vue/components/data/DailyStats.vue";

const open = ref(false);

const toggle = () => (open.value = !open.value);

defineExpose({ open, toggle });
</script>

<style scoped lang="scss"></style>
