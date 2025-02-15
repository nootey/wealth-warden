<script setup lang="ts">


</script>

<template>

  <div class="auth-background">
    <Transition appear name="auth-animation">
      <div class="auth-card gap-2">
        <div class="flex flex-column w-7">
          <slot></slot>
        </div>
        <div class="flex flex-column w-5 p-2 justify-content-center text-center">
          <img src="../../assets/images/logo.png" style="width: 100%;"/>
          <span class="flex flex-column "> {{ "WealthWarden" }}</span>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
  .auth-background{
    background: var(--p-indigo-950);
    height: 100vh;
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
  }

  .auth-card{
    display: flex;
    flex-direction: row;
    color: black;
    justify-content: center;
    align-items: center;
    background: var(--p-slate-100);
    width: 600px;
    padding: 30px;
    height: min-content;
    margin: 0 auto;
    border-radius: 20px;
    z-index: 2;
  }

  @media(max-width: 700px) {
    .auth-card {
      width: 85%;
    }
  }

  .auth-button{
    background: #C3181D;
    border-color: #D65B5F;
    color: white;
    border-radius: 15px;
    line-height: 1;
  }

  .auth-button:hover{
    background: #D65B5F;
    cursor: pointer;
  }

  .auth-link-text{
    color: #707070;
    text-decoration: underline;
  }

  .auth-link-text:hover{
    color: #C3181D;
    cursor: pointer;
  }

  .auth-animation-enter-active,
  .auth-animation-leave-active {
    transition: opacity 1s ease;
  }

  .auth-animation-enter-from,
  .auth-animation-leave-to {
    opacity: 0;
  }
</style>