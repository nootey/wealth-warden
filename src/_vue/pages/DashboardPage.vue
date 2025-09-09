<script setup lang="ts">
import {useAuthStore} from "../../services/stores/auth_store.ts";
import {useAccountStore} from "../../services/stores/account_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import {onMounted, ref} from "vue";

const authStore = useAuthStore();
const accountStore = useAccountStore();
const toastStore = useToastStore();

const points = ref([]);

onMounted(async () => {
    points.value = await accountStore.getNetWorth();
    console.log(points.value);
})

async function backfillBalances(){
    try {
        const response = await accountStore.backfillBalances();
        toastStore.successResponseToast(response.data);
    } catch (err) {
        toastStore.errorResponseToast(err)
    }
}

</script>

<template>
  <main>
    <div class="flex flex-column w-100 gap-3 justify-content-center align-items-center">

      <div class="main-item flex flex-column justify-content-center gap-1">
        <div style="font-weight: bold;">WealthWarden </div>
        <br>
        <div> Welcome back {{ authStore?.user?.display_name }} </div>
        <div>{{ "Here's what's happening with your finances." }} </div>
      </div>

        <Button label="magic" @click="backfillBalances"></Button>

        <div v-if="points">
            {{ points }}
        </div>

    </div>
  </main>
</template>

<style scoped>
.main-item {
  width: 100%;
  max-width: 1000px;
  align-items: center;
  padding: 1rem;
  border-radius: 8px; border: 1px solid var(--border-color); background-color: var(--background-secondary)
}
</style>