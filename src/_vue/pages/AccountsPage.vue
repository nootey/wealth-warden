<script setup lang="ts">
import AccountsPanel from '../features/AccountsPanel.vue';
</script>

<template>
    <AccountsPanel
            :advanced="false"
            :showHeader="true"
            :allowCreate="true"
            :allowEdit="true"
    />
</template>
