<script setup lang="ts">
import {ref} from "vue";
import {required, email, minLength, helpers} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {useRouter} from "vue-router";
import ValidationError from "../../components/validation/ValidationError.vue";
import AuthSkeleton from "../../components/layout/AuthSkeleton.vue";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import type {AuthForm} from "../../../models/auth_models.ts";
import {useAuthStore} from "../../../services/stores/auth_store.ts";

const authStore = useAuthStore();
const toastStore = useToastStore()

const router = useRouter();

const loading = ref(false);

const form = ref<AuthForm>({
    display_name: '',
    email: '',
    password: '',
    password_confirmation: '',
})

const noSpaces = helpers.withMessage(
    'Password cannot contain spaces',
    (value: string) => !/\s/.test(value ?? '')
)

const hasNumber = helpers.withMessage(
    'Password must contain at least one number',
    helpers.regex(/\d/)
)

const hasUppercase = helpers.withMessage(
    'Password must contain at least one uppercase letter',
    helpers.regex(/[A-Z]/)
)

const hasSpecialChar = helpers.withMessage(
    'Password must contain at least one special character',
    helpers.regex(/[!@#$%^&*(),.?":{}|<>]/)
)

const rules = {
    form: {
        display_name : {
            required,
            $autoDirty: true,
        },
        email : {
            required,
            email,
            $autoDirty: true,
        },
        password : {
            required,
            $autoDirty: true,
            minLength: minLength(6),
            noSpaces,
            hasNumber,
            hasUppercase,
            hasSpecialChar,
        },
        password_confirmation : {
            required,
            $autoDirty: true,
            repeatPassword: helpers.withMessage(': must match password', value => value === form.value.password),
        },
    }
}

const v$ = useVuelidate(rules, {form})

async function createInvitation() {
    v$.value.$touch();
    if (v$.value.$error) return;
    loading.value = true;

    try {
        await authStore.signUp(form.value);
        await router.push({name: "Login"})
    } catch (error) {
        toastStore.errorResponseToast(error)
    } finally
    {
        loading.value = false;
    }
}

function login() {
    router.push({name: "Login"});
}


</script>

<template>
    <AuthSkeleton>
        <div class="w-full mx-auto px-3 sm:px-0" style="max-width: 400px;">

            <div id="hideOnMobile" class="text-center mb-4">
                <h2 class="m-0 text-2xl sm:text-3xl font-bold"
                    style="color: var(--text-primary); letter-spacing: -0.025em;">
                    Create an account
                </h2>
            </div>

            <div class="flex flex-column gap-3">

                <div class="flex flex-row w-full">
                    <div class="flex flex-column gap-1 w-full">
                        <ValidationError :isRequired="true" :message="v$.form.display_name.$errors[0]?.$message">
                            <label>Display name</label>
                        </ValidationError>
                        <InputText id="display_name" v-model="form.display_name" type="text"
                                   :placeholder="'Display name'" :disabled="loading" :readonly="loading"
                                   class="w-full border-round-xl"/>
                    </div>
                </div>

                <div class="flex flex-row w-full">
                    <div class="flex flex-column gap-1 w-full">
                        <ValidationError :isRequired="true" :message="v$.form.email.$errors[0]?.$message">
                            <label>Email</label>
                        </ValidationError>
                        <InputText id="email" v-model="form.email" type="email"
                                   :placeholder="'Email'" :disabled="loading" :readonly="loading"
                                   class="w-full border-round-xl"/>
                    </div>
                </div>

                <div class="flex flex-row w-full">
                    <div class="flex flex-column gap-1 w-full">
                        <ValidationError :isRequired="true" :message="v$.form.password.$errors[0]?.$message">
                            <label>Password</label>
                        </ValidationError>
                        <InputText id="password" v-model="form.password" type="password"
                                   :placeholder="'Password'" :disabled="loading" :readonly="loading"
                                   class="w-full border-round-xl"/>
                    </div>
                </div>

                <div class="flex flex-row w-full">
                    <div class="flex flex-column gap-1 w-full">
                        <ValidationError :isRequired="true" :message="v$.form.password_confirmation.$errors[0]?.$message">
                            <label>Confirm password</label>
                        </ValidationError>
                        <InputText id="password_confirmation" v-model="form.password_confirmation" type="password"
                                   :placeholder="'Confirm password'"
                                   class="w-full border-round-xl" :disabled="loading" :readonly="loading"
                                   @keydown.enter="createInvitation"/>
                    </div>
                </div>

                <Button label="Sign up" class="w-full auth-accent-button"
                        :disabled="loading"  @click="createInvitation"></Button>

            </div>

            <div class="flex align-items-center justify-content-center gap-2 mt-4 pt-3"
                 style="border-top: 1px solid var(--border-color);">
                <span class="text-sm" style="color: var(--text-secondary);">
                  Already have an account?
                </span>
                <span class="text-sm hover-icon hover-dim"
                      @click="login">
                        Login</span>
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