<script setup lang="ts">
import {useAuthStore} from "../../services/stores/authStore.ts";

const authStore = useAuthStore();
</script>

<template>
  <main>
    <h1>Home</h1>
    <p> This is the home page </p>
    <span> {{ "Has user initialized budget? "}}</span>
    <span> {{ authStore?.user?.secrets?.budget_initialized}}</span>
  </main>
</template>

<style scoped>

</style>