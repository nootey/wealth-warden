<template>
    <div class="mobile-hide flex justify-center" style="width: 70px; position: sticky; top: 0;">
        <Drawer v-model:visible="visible" header="Drawer" position="right">
            <p>Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.</p>
        </Drawer>
        <Button icon="pi pi-arrow-right" @click="visible = true" />
    </div>
</template>

<script setup>
import { ref } from "vue";

const visible = ref(false);
</script>

<style scoped>

</style>
