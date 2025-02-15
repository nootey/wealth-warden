<script setup lang="ts">
import {ref} from "vue";
import {required, email } from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {useRouter} from "vue-router";
import ValidationError from "../Validation/ValidationError.vue";
import {useAuthStore} from "../../services/stores/authStore.ts";
import AuthSkeleton from "./AuthSkeleton.vue";

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
    <AuthSkeleton>
      <div class="flex flex-column gap-2 w-full">

        <div class="flex flex-row w-full gap-3 justify-content-center">
          <div class="flex flex-column w-full gap-1">
            <ValidationError :isRequired="true" :message="v$.form.email.$errors[0]?.$message">
              <label>{{ "Email" }}</label>
            </ValidationError>
            <InputText v-model="form.email" class="auth-input-field" style="border-radius: 15px;"
                       type="email" :invalid="!!v$.form.email.$errors[0]?.$message"/>
          </div>
        </div>

        <div class="flex flex-row w-full gap-3 justify-content-center">
          <div class="flex flex-column w-full gap-1">
            <ValidationError :isRequired="true" :message="v$.form.password.$errors[0]?.$message">
              <label>{{ "Password" }}</label>
            </ValidationError>
            <InputText v-model="form.password" class="auth-input-field" type="password" style="border-radius: 15px;"
                       :invalid="!!v$.form.password.$errors[0]?.$message" @keydown.enter="login"/>
          </div>
        </div>

        <div class="flex flex-row w-full gap-3 justify-content-center">
          <div class="flex flex-column w-full gap-2">
            <div class="flex flex-row align-items-center gap-1">
              <span class="fine-text link-text" @click="forgotPassword">{{"Forgot your password?" }}</span>
<!--              <Checkbox id="binary" v-model="form.rememberMe" style="transform: scale(0.7)"-->
<!--                        :binary="true"-->
<!--                        :falseValue="false"-->
<!--                        :trueValue="true"/>-->
<!--              <label>{{ "Remember me" }}</label>-->
            </div>
          </div>
        </div>

        <div class="flex flex-row w-full gap-3 justify-content-center mt-2">
          <div class="flex flex-column w-full gap-1">
            <Button :label='"Login"' @click="login" class="auth-button" />
          </div>
        </div>

        <div class="link flex flex-row w-100 align-items-center gap-2 fine-text">
          <span>{{"Need an account?" }}</span>
          <span class="link-text" @click="register"> {{ "Register" }} </span>
        </div>

      </div>
    </AuthSkeleton>
</template>

<style scoped>
  .fine-text {
    font-size: 0.85rem;
  }
  .link-text {
    color: var(--accent-primary);
    &:hover {
      text-decoration: underline;
      cursor: pointer;
    }
  }
</style>