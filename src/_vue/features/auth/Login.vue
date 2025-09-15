<script setup lang="ts">
import {ref} from "vue";
import {required, email } from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {useRoute, useRouter} from "vue-router";
import ValidationError from "../../components/validation/ValidationError.vue";
import {useAuthStore} from "../../../services/stores/auth_store.ts";
import AuthSkeleton from "../../components/layout/AuthSkeleton.vue";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import type {AuthForm} from "../../../models/auth_models.ts";

const authStore = useAuthStore();
const toastStore = useToastStore()

const router = useRouter();
const route = useRoute();

const loading = ref<boolean>(false);

const form = ref<AuthForm>({
  email: "",
  password: "",
  remember_me: false,
});

const rules = {
  form: {
    email : {
      required,
      email,
      $autoDirty: true,
    },
    password : {
      required,
      $autoDirty: true,
    },
  }
}

const v$ = useVuelidate(rules, {form})

function resolveRedirect(): string {
    const q = route.query.redirect as string | string[] | undefined;
    const redirect = Array.isArray(q) ? q[0] : q;

    if (typeof redirect !== "string") return "/";

    // Disallow absolute URLs or protocol-relative
    if (/^https?:\/\//i.test(redirect) || redirect.startsWith("//")) return "/";

    // Allow only root-relative paths
    if (!redirect.startsWith("/")) return "/";

    // Avoid looping back to login
    if (redirect === "/login") return "/";

    return redirect;
}

async function login() {
  v$.value.$touch();
  if (v$.value.$error) return;

    loading.value = true;
    try {
        await authStore.login(form.value);

        if (authStore.authenticated){
            const target = resolveRedirect();
            await router.replace(target);
        }

    } catch (error) {
        toastStore.errorResponseToast(error)
    } finally {
        loading.value = false;
    }
}

function signUp() {
  router.push({name: "sign.up"});
}

function forgotPassword() {
  router.push({name: "forgot.password"});
}

</script>

<template>
    <AuthSkeleton>
        <div class="w-full mx-auto px-3 sm:px-0" style="max-width: 400px;">

            <div id="hideOnMobile" class="text-center mb-4">
                <h2 class="m-0 text-2xl sm:text-3xl font-bold"
                    style="color: var(--text-primary); letter-spacing: -0.025em;">
                    Welcome back
                </h2>
                <p class="mt-2 text-base line-height-3" style="color: var(--text-secondary);">
                    Sign in to your account to continue
                </p>
            </div>

            <div class="flex flex-column gap-3">

                <div class="flex flex-row w-full">
                    <div class="flex flex-column gap-1 w-full">
                        <ValidationError :isRequired="true" :message="v$.form.email.$errors[0]?.$message">
                            <label>Email</label>
                        </ValidationError>
                        <InputText id="email" v-model="form.email" type="email"
                                   :placeholder="'Email'"
                                   class="w-full border-round-xl"/>
                    </div>
                </div>

                <div class="flex flex-row w-full">
                    <div class="flex flex-column gap-1 w-full">
                        <ValidationError :isRequired="true" :message="v$.form.password.$errors[0]?.$message">
                            <label>Password</label>
                        </ValidationError>
                        <InputText id="password" v-model="form.password" type="password"
                                   :placeholder="'Password'"
                                   class="w-full border-round-xl"
                                   @keydown.enter="login"/>
                    </div>
                </div>

                <div class="flex flex-row w-full justify-content-between">
                    <div class="flex flex-row align-items-center gap-2">
                        <Checkbox inputId="rememberMe" v-model="form.remember_me" :binary="true" class="scale-90" />
                        <label for="rememberMe" class="text-sm cursor-pointer" style="color: var(--text-secondary);">
                            Remember me
                        </label>
                    </div>

                    <span class="text-sm hover-icon hover-dim"
                          @click="forgotPassword">
                        Forgot password?</span>
                </div>

                <Button :label="loading ? 'Signing in...' : 'Sign in'"
                        :icon="loading ? 'pi pi-spin pi-spinner mr-2' : ''" class="w-full auth-accent-button"
                        :disabled="loading || v$.$error" @click="login"/>

            </div>

            <div class="flex align-items-center justify-content-center gap-2 mt-4 pt-3"
                 style="border-top: 1px solid var(--border-color);">
                <span class="text-sm" style="color: var(--text-secondary);">
                  Don't have an account?
                </span>
                <span class="text-sm hover-icon hover-dim"
                      @click="signUp">
                        Create account</span>
            </div>
        </div>
    </AuthSkeleton>
</template>

<style scoped>

@media (max-width: 768px) {
    #hideOnMobile {
        display: none;
    }
}

.hover-dim {
    color: var(--accent-primary);
}
.hover-dim:hover {
    color: var(--accent-secondary);
}
</style>