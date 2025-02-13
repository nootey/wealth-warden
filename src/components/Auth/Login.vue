<script setup lang="ts">
import {ref} from "vue";
import {required, email } from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {useRouter} from "vue-router";
import InputValidation from "../Validation/InputValidation.vue";
import ValidationError from "../Validation/ValidationError.vue";
import vueHelper from "../../utils/vueHelper.ts"
import {useAuthStore} from "../../services/stores/authStore.ts";

const authStore = useAuthStore();

const router = useRouter();
const form = ref({
  email: "",
  password: "",
  rememberMe: false,
});

const rules = {
  form: {
    email : {
      required,
      $autoDirty: true,
    },
    password : {
      required,
      $autoDirty: true,
    },
  }
}

const v$ = useVuelidate(rules, {form})

async function login() {
  v$.value.$touch();
  if (v$.value.$error) return;

  try {
    await authStore.login(form.value);
    if (authStore.authenticated){
      await router.push({name: "Dashboard"})
    }
  } catch (error) {
    console.error('Error during login:', error);
  }

}

function register() {
  router.push({name: "register"});
}

function forgotPassword() {
  router.push({name: "forgot.password"});
}

</script>

<template>
  <div class="auth-background">
    <Transition appear name="auth-animation">
      <div class="auth-card gap-2">
        <div class="flex flex-column gap-2 justify-content-center">

<!--          <div class="flex flex-row p-2 justify-content-center">-->
<!--            <img src="../../../assets/images/ng-logo.png" width="200"/>-->
<!--          </div>-->

          <div class="flex flex-row w-100 gap-3 justify-content-center">
            <div class="flex flex-column w-100 gap-1">
              <ValidationError :message="v$.form.email.$errors[0]?.$message.replace('Value', 'Email').replace('The value', 'Email')">
                <label>{{ "Email or username" }}</label>
              </ValidationError>
              <InputValidation :validationObject="v$.form.email">
                <InputText v-model="form.email" class="auth-input-field" style="border-radius: 15px;"
                           type="email"
                           :class="vueHelper.getValidationClass(v$.form.email, 'shake-form')"
                />
              </InputValidation>
            </div>
          </div>

          <div class="flex flex-row w-100 gap-3 justify-content-center">
            <div class="flex flex-column w-100 gap-1">
              <ValidationError :message="v$.form.password.$errors[0]?.$message.replace('Value', 'Password').replace('The value', 'Password')">
                <label>{{ "Password" }}</label>
              </ValidationError>
              <InputValidation :validationObject="v$.form.password">
                <InputText v-model="form.password" class="auth-input-field" type="password" style="border-radius: 15px;"
                           :class="vueHelper.getValidationClass(v$.form.password, 'shake-form')" @keydown.enter="login"/>
              </InputValidation>
            </div>
          </div>

          <div class="flex flex-row w-100 gap-3 justify-content-center">
            <div class="flex flex-column w-100 gap-2">
              <div class="flex flex-row align-items-center gap-1">
                <Checkbox id="binary" v-model="form.rememberMe" style="transform: scale(0.7)"
                          :binary="true"
                          :falseValue="false"
                          :trueValue="true"/>
                <label>{{ "Remember me" }}</label>
              </div>
            </div>
          </div>

          <div class="flex flex-row w-100 gap-3 justify-content-center mt-2">
            <div class="flex flex-column w-50 gap-1">
              <Button :label='"Login"' @click="login" class="auth-button" />
            </div>
          </div>

          <div class="link flex flex-row w-100 justify-content-center mt-2">
            <div class="flex flex-column justify-content-center align-items-center">
              <span class="auth-link-text" @click="register">{{"Create new account" }}</span>
              <span class="auth-link-text" @click="forgotPassword">{{"Forgot your password?" }}</span>
            </div>
          </div>

        </div>
      </div>
    </Transition>

    <Transition appear name="auth-animation">
      <div class="flex flex-row p-5 justify-content-center mb-2" style="position: absolute; bottom: 0; left: 0; width: 100%">
        <div class="flex flex-column justify-content-center gap-2">
          <span style="color: white; margin: 0 auto;"> POWERED BY</span>
<!--          <img src="../../../assets/images/nglogo_small.png" width="150" />-->
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>

</style>