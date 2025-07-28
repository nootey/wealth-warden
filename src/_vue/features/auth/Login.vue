<script setup lang="ts">
import {ref} from "vue";
import {required, email } from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {useRouter} from "vue-router";
import ValidationError from "../../components/validation/ValidationError.vue";
import {useAuthStore} from "../../../services/stores/auth_store.ts";
import AuthSkeleton from "../../components/forms/AuthSkeleton.vue";
import {useToastStore} from "../../../services/stores/toast_store.ts";

const authStore = useAuthStore();
const toastStore = useToastStore()

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

async function login() {
  v$.value.$touch();
  if (v$.value.$error) return;

  try {
    await authStore.login(form.value);
    if (authStore.authenticated){
      await router.push({name: "Dashboard"})
    }
  } catch (error) {
    toastStore.errorResponseToast(error)
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
    <div class="login-container">
      <div class="login-header">
        <h2 class="login-title">Welcome back</h2>
        <p class="login-subtitle">Sign in to your account to continue</p>
      </div>

      <form @submit.prevent="login" class="login-form">
        <div class="form-group">
          <div class="flex flex-row">
            <label for="email" class="form-label">Username</label>
          <ValidationError 
              :isRequired="true" 
              :message="v$.form.email.$errors[0]?.$message"
              class="error-message"
            />
          </div>
          <div class="input-wrapper">
            <InputText 
              id="email"
              v-model="form.email" 
              class="form-input" 
              type="email" 
              placeholder="Enter your email or username"
              :class="{ 'input-error': !!v$.form.email.$errors[0]?.$message }"
              :invalid="!!v$.form.email.$errors[0]?.$message"
            />
          </div>
        </div>

        <div class="form-group">
          <div class="flex flex-row">
            <label for="password" class="form-label">Password</label>
            <ValidationError 
              :isRequired="true" 
              :message="v$.form.password.$errors[0]?.$message"
              class="error-message"
            />
          </div>
          <div class="input-wrapper">
            <InputText 
              id="password"
              v-model="form.password" 
              class="form-input" 
              type="password" 
              placeholder="Enter your password"
              :class="{ 'input-error': !!v$.form.password.$errors[0]?.$message }"
              :invalid="!!v$.form.password.$errors[0]?.$message"
              @keydown.enter="login"
            />
          </div>
        </div>

        <div class="form-options" style="margin-top: -0.75rem;">
          <div class="remember-me">
            <Checkbox 
              id="rememberMe" 
              v-model="form.rememberMe" 
              :binary="true"
              class="remember-checkbox"
            />
            <label for="rememberMe" class="remember-label">Remember me</label>
          </div>
          <button type="button" class="forgot-link" @click="forgotPassword">
            Forgot password?
          </button>
        </div>

        <Button 
          type="submit"
          :label="'Sign in'" 
          class="login-button"
        />
      </form>

      <div class="login-footer">
        <span class="footer-text">Don't have an account?</span>
        <button type="button" class="register-link" @click="register">
          Create account
        </button>
      </div>
    </div>
  </AuthSkeleton>
</template>

<style scoped>
.login-container {
  width: 100%;
  max-width: 400px;
  margin: 0 auto;
}

.login-header {
  text-align: center;
  margin-bottom: 2rem;
}

.login-title {
  font-size: 1.875rem;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 0.5rem;
  letter-spacing: -0.025em;
}

.login-subtitle {
  color: var(--text-secondary);
  font-size: 1rem;
  line-height: 1.5;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.form-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: 0.025em;
}

.input-wrapper {
  position: relative;
}

.form-input {
  width: 100%;
  padding: 0.75rem 1rem;
  border: 2px solid var(--border-color);
  border-radius: 12px;
  background: var(--background-primary);
  color: var(--text-primary);
  font-size: 1rem;
  transition: all 0.2s ease;
  outline: none;
}

.form-input:focus {
  border-color: var(--accent-primary);
  box-shadow: 0 0 0 3px rgba(var(--accent-primary-rgb, 99, 102, 241), 0.1);
}

.form-input::placeholder {
  color: var(--text-secondary);
  opacity: 0.7;
}

.input-error {
  border-color: #ef4444;
}

.input-error:focus {
  border-color: #ef4444;
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.1);
}

.error-message {
  margin-top: 0.25rem;
  font-size: 0.875rem;
  color: #ef4444;
}

.form-options {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 0.5rem;
}

.remember-me {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.remember-checkbox {
  transform: scale(0.9);
}

.remember-label {
  font-size: 0.875rem;
  color: var(--text-secondary);
  cursor: pointer;
}

.forgot-link {
  background: none;
  border: none;
  color: var(--accent-primary);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  text-decoration: none;
  transition: color 0.2s ease;
}

.forgot-link:hover {
  color: var(--accent-secondary);
  text-decoration: underline;
}

.login-button {
  width: 100%;
  padding: 0.875rem 1.5rem;
  background: linear-gradient(135deg, var(--accent-primary) 0%, var(--accent-secondary) 100%);
  border: none;
  border-radius: 12px;
  color: white;
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  margin-top: 1rem;
}

.login-button:hover {
  transform: translateY(-1px);
  box-shadow: 0 8px 25px rgba(var(--accent-primary-rgb, 99, 102, 241), 0.3);
}

.login-button:active {
  transform: translateY(0);
}

.login-footer {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 0.5rem;
  margin-top: 2rem;
  padding-top: 1.5rem;
  border-top: 1px solid var(--border-color);
}

.footer-text {
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.register-link {
  background: none;
  border: none;
  color: var(--accent-primary);
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  text-decoration: none;
  transition: color 0.2s ease;
}

.register-link:hover {
  color: var(--accent-secondary);
  text-decoration: underline;
}

/* Responsive adjustments */
@media (max-width: 480px) {
  .login-container {
    padding: 0 1rem;
  }
  
  .login-title {
    font-size: 1.5rem;
  }
  
  .form-options {
    flex-direction: column;
    align-items: flex-start;
    gap: 1rem;
  }
}
</style>