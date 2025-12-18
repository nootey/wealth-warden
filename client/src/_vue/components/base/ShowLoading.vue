<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
    numFields: number;
}>();

function randInt(min: number, max: number) {
    return Math.floor(Math.random() * (max - min + 1)) + min;
}

const skeletons = computed(() =>
    Array.from({ length: props.numFields }, (_, i) => ({
        id: i,
        width: `${randInt(20, 100)}%`,
        height: `${randInt(1, 3)}rem`,
    }))
);
</script>

<template>
  <div
    class="flex flex-column justify-content-center mt-4 gap-4"
    style="width: 75%; margin: 0 auto;"
  >
    <Skeleton
      v-for="s in skeletons"
      :key="s.id"
      border-radius="16px"
      :width="s.width"
      :height="s.height"
    />
  </div>
</template>
