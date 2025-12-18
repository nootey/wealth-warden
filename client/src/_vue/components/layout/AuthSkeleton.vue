<script setup lang="ts">
// AuthSkeleton component with modern design and dark mode support
</script>

<template>
  <div class="auth-background">
    <Transition
      appear
      name="auth-animation"
    >
      <div class="auth-card">
        <div class="auth-content">
          <slot />
        </div>
        <div class="auth-brand">
          <img
            src="../../../assets/images/logo.png"
            alt="WealthWarden Logo"
            class="auth-logo"
          >
          <h1 class="auth-title">
            WealthWarden
          </h1>
          <p
            id="hideOnMobile"
            class="auth-subtitle"
          >
            Tap. Track. Relax.
          </p>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.auth-background {
  background: linear-gradient(135deg, var(--background-primary) 0%, var(--background-secondary) 100%);
  min-height: 100vh;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  position: relative;
}

.auth-background::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: radial-gradient(circle at 20% 80%, var(--accent-primary) 0%, transparent 50%),
              radial-gradient(circle at 80% 20%, var(--accent-secondary) 0%, transparent 50%);
  opacity: 0.1;
  z-index: 1;
}

.auth-card {
  display: flex;
  flex-direction: row;
  background: var(--background-secondary);
  border: 1px solid var(--border-color);
  border-radius: 24px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.1);
  backdrop-filter: blur(10px);
  width: 900px;
  max-width: 100%;
  min-height: 600px;
  position: relative;
  z-index: 2;
  overflow: hidden;
}

.auth-content {
  flex: 1;
  padding: 3rem;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.auth-brand {
  flex: 1;
  background: linear-gradient(135deg, var(--accent-primary) 0%, var(--accent-secondary) 100%);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem 2rem;
  text-align: center;
  color: white;
  position: relative;
}

.auth-brand::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><defs><pattern id="grain" width="100" height="100" patternUnits="userSpaceOnUse"><circle cx="50" cy="50" r="1" fill="white" opacity="0.1"/></pattern></defs><rect width="100" height="100" fill="url(%23grain)"/></svg>');
  opacity: 0.3;
}

.auth-logo {
  width: 120px;
  height: auto;
  margin-bottom: 1.5rem;
  filter: brightness(0) invert(1);
}

.auth-title {
  font-size: 2.5rem;
  font-weight: 700;
  margin-bottom: 0.5rem;
  letter-spacing: -0.02em;
}

.auth-subtitle {
  font-size: 1.1rem;
  opacity: 0.9;
  font-weight: 400;
}

@media (max-width: 768px) {

   #hideOnMobile {
     display: none;
   }

  .auth-card {
    flex-direction: column;
    width: 90%;
    max-width: 400px;
    min-height: auto;
  }
  
  .auth-content {
    padding: 1rem;
  }
  
  .auth-brand {
    padding: 1.5rem 1rem;
    order: -1;
  }
  
  .auth-logo {
    width: 80px;
    margin-bottom: 1rem;
  }
  
  .auth-title {
    font-size: 1.55rem;
  }
}

@media (max-width: 480px) {
  .auth-background {
    padding: 0.5rem;
  }
  
  .auth-content {
    padding: 1.5rem;
  }
  
  .auth-brand {
    padding: 1.5rem 1rem;
  }
}

.auth-animation-enter-active,
.auth-animation-leave-active {
  transition: all 0.6s cubic-bezier(0.4, 0, 0.2, 1);
}

.auth-animation-enter-from {
  opacity: 0;
  transform: translateY(30px) scale(0.95);
}

.auth-animation-leave-to {
  opacity: 0;
  transform: translateY(-30px) scale(0.95);
}
</style>