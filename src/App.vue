<script setup lang="ts">
import Sidebar from "./Sidebar.vue";

const authenticated = false;
const initialized = true;
const loggedOut = true;
</script>

<template>
  <div class="app">

    <router-view v-if="!authenticated && loggedOut"/>
    <div v-else-if="initialized">
      <Sidebar />
      <router-view/>
    </div>

  </div>
</template>

<style scoped lang="scss">

.app {
  display: flex;
  main {
    flex: 1 1 0;
    padding: 2rem;

    @media (max-width: 768px) {
      padding-left: 6rem;
    }
  }
}
</style>
