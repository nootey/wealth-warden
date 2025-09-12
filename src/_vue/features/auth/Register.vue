<script setup lang="ts">
import {ref} from "vue";
import {required, email, minLength, helpers} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {useRouter} from "vue-router";
import ValidationError from "../../components/validation/ValidationError.vue";
import {useAuthStore} from "../../../services/stores/auth_store.ts";
import AuthSkeleton from "../../components/layout/AuthSkeleton.vue";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import type {AuthForm} from "../../../models/auth_models.ts";

const authStore = useAuthStore();
const toastStore = useToastStore()

const router = useRouter();

const form = ref<AuthForm>({
    email: '',
    password: '',
    passwordConfirmation: '',
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
        passwordConfirmation : {
            required,
            $autoDirty: true,
            repeatPassword: helpers.withMessage('Password confirmation must match password', value => value === form.value.password),
        },
    }
}

const v$ = useVuelidate(rules, {form})

async function register() {
    v$.value.$touch();
    if (v$.value.$error) return;

    try {
        await authStore.register(form.value);
        await router.push({name: "Login"})
    } catch (error) {
        toastStore.errorResponseToast(error)
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
                                   class="w-full border-round-xl"/>
                    </div>
                </div>

                <div class="flex flex-row w-full">
                    <div class="flex flex-column gap-1 w-full">
                        <ValidationError :isRequired="true" :message="v$.form.passwordConfirmation.$errors[0]?.$message">
                            <label>Confirm password</label>
                        </ValidationError>
                        <InputText id="password_confirmation" v-model="form.passwordConfirmation" type="password"
                                   :placeholder="'Confirm password'"
                                   class="w-full border-round-xl"
                                   @keydown.enter="register"/>
                    </div>
                </div>

                <Button label="Sign up" class="w-full auth-accent-button" @click="register"></Button>

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