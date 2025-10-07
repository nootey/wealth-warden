<template>
    <Drawer v-model:visible="open" header="Monthly stats" position="right" style="width: 400px;">
        <template #container="{ closeCallback }">
            <div class="flex flex-column h-full w-full p-2">
                <div class="flex flex-row justify-content-between p-2">
                    <h3>Monthly stats</h3>
                    <i class="pi pi-times hover-icon" @click="closeCallback" />
                </div>
            </div>
        </template>
    </Drawer>
</template>

<script setup lang="ts">
import { ref, defineExpose } from 'vue';

const open = ref(false);

const toggle = () => (open.value = !open.value);

defineExpose({ open, toggle });
</script>

<style scoped lang="scss">
@media (max-width: 1400px) {

}
</style>