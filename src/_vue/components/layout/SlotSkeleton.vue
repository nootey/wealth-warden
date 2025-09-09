<script setup lang="ts">

withDefaults(defineProps<{ bg?: 'primary' | 'secondary' | 'transparent' }>(), {
    bg: 'secondary'
})

</script>

<template>
    <div class="w-full flex flex-column border-round-lg"
            :class="`panel--${bg}`">
        <Transition appear name="animate">
            <slot />
        </Transition>
    </div>
</template>

<style scoped>
.panel--primary { background-color: var(--background-primary); border: 1px solid var(--background-primary); }
.panel--secondary { background-color: var(--background-secondary); border: 1px solid var(--border-color); }
.panel--transparent { background-color: transparent; }

.animate-enter-active, .animate-leave-active { transition: all .2s ease; }
.animate-enter-from, .animate-leave-to { opacity: 0; transform: translateY(4px); }
</style>
