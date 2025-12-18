<script setup lang="ts">
import {onMounted, ref} from "vue";
import type {Account} from "../../models/account_models.ts";
import {useAccountStore} from "../../services/stores/account_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import ShowLoading from "../components/base/ShowLoading.vue";
import {usePermissions} from "../../utils/use_permissions.ts";
import DefaultAccountForm from "../components/forms/DefaultAccountForm.vue";
import vueHelper from "../../utils/vue_helper.ts";
import {colorForAccountType} from "../../style/theme/accountColors.ts";

const accStore = useAccountStore();
const toastStore = useToastStore();
const { hasPermission } = usePermissions();
const logoColor = (type?: string) => colorForAccountType(type);

const loading = ref(true);
const accounts = ref<Account[]>([]);
const createModal = ref(false);

onMounted(async () => {
    await getData();
});

async function getData() {
    loading.value = true;

    try {
        accounts.value = await accStore.getAllDefaultAccounts();
    } catch (error) {
        toastStore.errorResponseToast(error);
    } finally {
        loading.value = false;
    }
}

async function unsetDefault(id: number) {
    loading.value = true;

    try {
        const res = await accStore.unsetDefaultAccount(id);
        toastStore.successResponseToast(res);
        await getData();
    } catch (error) {
        toastStore.errorResponseToast(error);
    } finally {
        loading.value = false;
    }
}

function manipulateDialog(modal: string, value: any) {
    switch (modal) {
        case 'insertDefault': {
            if(accounts.value.length < 1){
                toastStore.createInfoToast("Access denied", "No accounts exist for this account.");
                return;
            }
            if(!hasPermission("manage_data")) {
                toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
                return;
            }
            createModal.value = value;
            break;
        }
        default: {
            break;
        }
    }
}

async function handleEmit(type: string) {
    switch (type) {
        case 'completeCatOperation': {
            createModal.value = false;
            await getData();
            break;
        }
        default: {
            break;
        }
    }
}

</script>

<template>

    <Dialog class="rounded-dialog" v-model:visible="createModal"
            :breakpoints="{ '501px': '90vw' }" :modal="true" :style="{ width: '500px' }" header="Create default">
        <DefaultAccountForm @completeOperation="handleEmit('completeCatOperation')"/>
    </Dialog>

    <div class="w-full flex flex-column gap-2">
        <div class="flex flex-row justify-content-between align-items-center gap-3">
            <div class="w-full flex flex-column gap-2">
                <h3>Default accounts</h3>
                <h5 style="color: var(--text-secondary)">Define default accounts for each account. This might help optimize some flows.</h5>
            </div>
            <Button class="main-button w-4" :disabled="accounts.length < 1"
                    @click="manipulateDialog('insertDefault', true)">
                <div class="flex flex-row gap-1 align-items-center">
                    <i class="pi pi-plus"></i>
                    <span class="mobile-hide"> New default </span>
                </div>
            </Button>
        </div>
    </div>

    <div class="flex-1 w-full border-round-md overflow-y-auto"
         :style="{ maxWidth: '1000px' }">

        <template v-if="loading">
            <ShowLoading :numFields="5" />
        </template>

        <div v-else-if="accounts.length === 0" class="flex flex-row p-2 w-full justify-content-center">
            <div class="flex flex-column gap-2 justify-content-center align-items-center">
                <i style="color: var(--text-secondary)" class="pi pi-eye-slash text-4xl"></i>
                <span>No defaults set</span>
            </div>
        </div>

        <div v-else class="w-full p-3 mb-2 border-round-md"
             style="background: var(--background-primary)">

            <div v-for="(account, i) in accounts" :key="account.id ?? i"
                 class="account-row flex align-items-center justify-content-between p-2 border-round-md mt-1">

                <div class="flex align-items-center">
                    <div class="flex align-items-center justify-content-center font-bold"
                         :style="{
                                    width: '32px',
                                    height: '32px',
                                    border: '1px solid',
                                    borderColor: logoColor(account.account_type?.type).border,
                                    borderRadius: '50%',
                                    background: logoColor(account.account_type.type).bg,
                                    color: logoColor(account.account_type.type).fg,
                                }">
                        {{ account.name.charAt(0).toUpperCase() }}
                    </div>

                    <div class="ml-2">
                        <div class="font-bold">
                            {{ account.name }}
                        </div>

                        <div class="text-sm" style="color: var(--text-secondary)">
                            {{ vueHelper.formatString(account.account_type?.sub_type) }}
                        </div>
                    </div>

                </div>

                <div class="flex align-items-center gap-2">
                    <i class="pi pi-trash hover-icon" style="color: var(--p-red-300)" @click="unsetDefault(account?.id!)" />
                </div>
            </div>
        </div>

    </div>

</template>

<style scoped>

</style>