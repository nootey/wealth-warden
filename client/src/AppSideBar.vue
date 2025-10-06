<template>
    <div class="mobile-hide flex flex-column flex-shrink-0"
         style="background-color: var(--background-secondary); transition: width .2s ease; overflow: hidden;"
         :style="{
         width: open ? '350px' : '0px',
         minWidth: open ? '350px' : '0px',
         flexShrink: 0
     }"
    >
        <!-- sidebar content -->
    </div>
</template>

<script setup lang="ts">
import { ref, defineExpose, watch, onMounted } from 'vue';

const STORAGE_KEY = 'sidebar-open';

const open = ref(true);

onMounted(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved !== null) {
        open.value = saved === 'true';
    }
});

watch(open, (val) => {
    localStorage.setItem(STORAGE_KEY, String(val));
});

const toggle = () => (open.value = !open.value);

defineExpose({ open, toggle });
</script>

<style scoped lang="scss">
@media (max-width: 1400px) {

}
</style>