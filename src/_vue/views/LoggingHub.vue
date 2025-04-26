<script setup lang="ts">
import {computed, ref} from "vue";
import LogsActivity from "../features/logging/LogsActivity.vue";
import LogsAccess from "../features/logging/LogsAccess.vue";

const logIndex = ref("activity");

function setIndex(index: string): void {
  logIndex.value = index;
}

const currentComponent = computed(() => {
  switch (logIndex.value) {
    case 'activity':
      return LogsActivity;
    case 'access':
      return LogsAccess;
    default:
      return LogsActivity;
  }
});
</script>

<template>
  <div style="width: 95%; margin: 0 auto; padding: 5px;" class="flex flex-column gap-3">

    <div class="flex flex-row justify-content-start gap-3 p-2" style="border-bottom: 2px solid var(--border-color)">
      <div class="flex flex-column header" :class="{'active' : logIndex === 'activity'}" @click="setIndex('activity')">
        {{ "Activity" }}
      </div>
      <div class="flex flex-column header" :class="{'active' : logIndex === 'access'}"  @click="setIndex('access')">
        {{ "Access" }}
      </div>
    </div>

    <Transition name="fade" mode="out-in">
      <component :is="currentComponent" key="logComponent"></component>
    </Transition>
  </div>
</template>

<style scoped>
.header{
  font-size: 1.1rem;
  transition: 0.3s ease-in-out;
}
.header:hover{
  cursor: pointer;
}
.active{
  color: var(--accent-primary);
  border-bottom: 1px solid var(--accent-primary);
  transition: 0.3s ease-in-out;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>