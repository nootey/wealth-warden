<script setup lang="ts">
import AccountsPanel from '../features/AccountsPanel.vue';
import AccountForm from "../components/forms/AccountForm.vue";
import {ref} from "vue";
import {useRouter} from "vue-router";
import {usePermissions} from "../../utils/use_permissions.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";

const createModal = ref(false);
const router = useRouter();
const { hasPermission } = usePermissions();
const toastStore = useToastStore();

const accountsPanelRef = ref<InstanceType<typeof AccountsPanel> | null>(null);

function openCreate() {

    if(!hasPermission("manage_data")) {
        toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
        return;
    }

    createModal.value = true;
}

async function handleCreate() {
    createModal.value = false;
    await accountsPanelRef.value?.refresh?.();
}

</script>

<template>

    <Dialog class="rounded-dialog" v-model:visible="createModal" :breakpoints="{'501px':'90vw'}"
            :modal="true" :style="{ width: '500px' }" header="Create account">
        <AccountForm mode="create" @completeOperation="handleCreate" />
    </Dialog>

    <main id="main-row" class="flex flex-column w-full p-2 align-items-center">
        <div id="inner-row" class="flex flex-column justify-content-center p-3 w-full gap-3 border-round-md"
             style="border: 1px solid var(--border-color); background: var(--background-secondary); max-width: 1000px;">
            
            <div class="flex flex-row justify-content-between align-items-center text-center gap-2 w-full">
                <div class="font-bold">Accounts</div>
                <i v-if="hasPermission('manage_data')" class="pi pi-external-link hover-icon mr-auto text-sm" @click="router.push('settings/accounts')" v-tooltip="'Go to accounts settings.'"></i>
                <Button class="main-button" @click="openCreate">
                    <div class="flex flex-row gap-1 align-items-center">
                        <i class="pi pi-plus"></i>
                        <span> New </span>
                        <span class="mobile-hide"> Account </span>
                    </div>
                </Button>
            </div>

            <AccountsPanel ref="accountsPanelRef"
                           :advanced="false"
                           :allowEdit="true"
                           :maxHeight="80"
            />
        </div>
    </main>

</template>

<style scoped>
@media (max-width: 768px) {
    #main-row {
        padding: 0 !important;
    }
    #inner-row {
        padding: 0.75rem !important;
        margin-bottom: -7px !important;
    }
}
</style>
