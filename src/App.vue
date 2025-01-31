<script setup lang="ts">
import Sidebar from "./Sidebar.vue";

const authenticated = true;
const initialized = true;
const loggedOut = false;
</script>

<template>
  <div class="app">
    <Sidebar />
    <router-view/>
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
