<script setup lang="ts">
import AccountsPanel from '../features/AccountsPanel.vue';
import AccountForm from "../components/forms/AccountForm.vue";
import {ref} from "vue";
import {useRouter} from "vue-router";

const createModal = ref(false);
const router = useRouter();

const accountsPanelRef = ref<InstanceType<typeof AccountsPanel> | null>(null);

function openCreate() {
    createModal.value = true;
}

async function handleCreate() {
    createModal.value = false;
    await accountsPanelRef.value?.refresh?.();
}

</script>

<template>

    <Dialog class="rounded-dialog" v-model:visible="createModal" :breakpoints="{ '801px': '90vw' }"
            :modal="true" :style="{ width: '500px' }" header="Create account">
        <AccountForm mode="create" @completeOperation="handleCreate" />
    </Dialog>

    <main class="flex flex-column w-full p-2 align-items-center gap-2">

        <div class="flex flex-row justify-content-between align-items-center p-3 w-full border-round-md bordered gap-2"
             style="max-width: 1000px">
            <div class="font-bold">Accounts</div>
            <i class="pi pi-map hover-icon mr-auto text-sm" @click="router.push('settings/accounts')" v-tooltip="'Go to accounts settings.'"></i>
            <Button class="main-button" label="New Account" icon="pi pi-plus" @click="openCreate"/>
        </div>

        <AccountsPanel ref="accountsPanelRef"
                       :advanced="false"
                       :allowEdit="true"
                       :maxHeight="80"
        />

    </main>


</template>

<style scoped>
.bordered {
    border: 1px solid var(--border-color);
    background: var(--background-secondary);
}
.hover {
    font-weight: bold;
}
.hover:hover {
    cursor: pointer;
    text-decoration: underline;
}
</style>
