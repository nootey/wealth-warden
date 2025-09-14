<script setup lang="ts">
import {ref} from "vue";
import AuthSkeleton from "../../components/layout/AuthSkeleton.vue";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useAuthStore} from "../../../services/stores/auth_store.ts";

const authStore = useAuthStore();
const toastStore = useToastStore()

const loading = ref(false);

async function resendConfirmationEmail() {
    loading.value = true;
    try {
        const response = await authStore.resendConfirmationEmail(authStore.user?.email)
        toastStore.successResponseToast(response)
    } catch (error) {
        toastStore.errorResponseToast(error);
        loading.value = false;
    } finally {
        loading.value = false;
    }
}

</script>

<template>
    <AuthSkeleton>
        <div class="w-full mx-auto px-3 sm:px-0" style="max-width: 400px;">

            <div class="text-center mb-4">
                <h2 class="m-0 text-2xl sm:text-3xl font-bold"
                    style="color: var(--text-primary); letter-spacing: -0.025em;">
                    {{ "Hey " + (authStore.user?.display_name ?? "user") }}
                </h2>
                <p class="mt-2 line-height-3 text-base" style="color: var(--text-secondary);">
                    You need to confirm your email to continue using the app.
                </p>
            </div>

            <div class="flex flex-column gap-3">

                <div class="flex flex-row w-full">
                    <div class="flex flex-column gap-1 w-full">
                        <label>Email</label>
                        <InputText id="email" :value="authStore.user?.email" type="email"
                                   :disabled="loading" :readonly="true"
                                   class="w-full border-round-xl"/>
                    </div>
                </div>

                <Button label="Resend email" class="w-full auth-accent-button"
                        :disabled="loading"  @click="resendConfirmationEmail"></Button>

            </div>

            <div class="flex align-items-center justify-content-center gap-2 mt-4 pt-3"
                 style="border-top: 1px solid var(--border-color);">
                <span class="text-sm" style="color: var(--text-secondary);">
                  Sign in with a different account?
                </span>
                <span class="text-sm hover-icon hover-dim"
                      @click="authStore.logoutUser()">
                        Log in</span>
            </div>
        </div>
    </AuthSkeleton>
</template>

<style scoped>

.hover-dim {
    color: var(--accent-primary);
}
.hover-dim:hover {
    color: var(--accent-secondary);
}
</style>