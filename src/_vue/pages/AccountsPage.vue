<script setup lang="ts">

import InsertAccount from "../components/forms/InsertAccount.vue";
import {onMounted, ref} from "vue";
import {useAccountStore} from "../../services/stores/account_store.ts";

const account_store = useAccountStore();

const addAccountModal = ref(false);

onMounted(async () => {
  await account_store.getAccountTypes();
})

function manipulateDialog(modal: string, value: boolean) {
  switch (modal) {
    case 'add-account': {
      addAccountModal.value = value;
      break;
    }
    default: {
      break;
    }
  }
}

async function handleEmit(emitType: any) {
  switch (emitType) {
    case 'insert-account': {
      console.log("hello there")
      break;
    }
    default: {
      break;
    }
  }
}


</script>

<template>

  <main>

    <Dialog v-model:visible="addAccountModal" :breakpoints="{'801px': '90vw'}"
            :modal="true" :style="{width: '500px'}" header="Add account">
      <InsertAccount entity="account" @insertAccount="handleEmit('insert-account')"></InsertAccount>
    </Dialog>

    <div class="flex flex-column w-100 gap-3 justify-content-center align-items-center">

      <div class="main-item flex flex-row justify-content-between">
        <div style="font-weight: bold;">Accounts</div>
        <Button class="main-button" label="New Account" icon="pi pi-plus" @click="manipulateDialog('add-account', true)"></Button>
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