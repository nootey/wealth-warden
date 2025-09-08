<script setup lang="ts">
function getDimension(min: number, max: number){
    let rand = Math.floor(Math.random() * (max - min + 1)) + min;
    return rand.toString()
}
</script>

<template>
    <div class="flex flex-column justify-content-center mt-4 gap-4" style="width: 75%; margin: 0 auto;">
        <Skeleton borderRadius="16px" :width="getDimension(20, 100)+'%'" :height="getDimension(1, 2)+'rem'" />
        <Skeleton borderRadius="16px" :width="getDimension(20, 100)+'%'" :height="getDimension(1, 3)+'rem'" />
        <Skeleton borderRadius="16px" :width="getDimension(20, 100)+'%'" :height="getDimension(1, 1)+'rem'" />
        <Skeleton borderRadius="16px" :width="getDimension(20, 100)+'%'" :height="getDimension(1, 1)+'rem'" />
        <Skeleton borderRadius="16px" :width="getDimension(20, 100)+'%'" :height="getDimension(1, 3)+'rem'" />
    </div>
</template>

<style scoped>

</style>