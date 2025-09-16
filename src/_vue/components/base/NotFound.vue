<script setup lang="ts">

import {useRouter} from "vue-router";
import {useAuthStore} from "../../../services/stores/auth_store.ts";
import {storeToRefs} from "pinia";

const router = useRouter();
const authStore = useAuthStore();

const {isAuthenticated} = storeToRefs(authStore);

function handlePrimary() {
        router.push({name: isAuthenticated.value ? 'dashboard' : 'login'});
}
</script>

<template>
    <div class="min-h-screen w-full flex flex-column align-items-center justify-content-center text-center p-6">
        <div class="flex flex-column gap-3" role="region" aria-label="Page not found">
            <h1 ref="headingRef" tabindex="-1" class="text-3xl font-bold">
                404  Page not found
            </h1>
            <p  style="color: var(--text-secondary)">
                The page you're looking for doesn't exist or was moved.
            </p>
            <div class="flex flex-row gap-3 justify-content-center">
                <Button :label="isAuthenticated ? 'Go Home' : 'Login'" class="main-button w-4"
                        @click="handlePrimary"/>
            </div>
        </div>
    </div>
</template>