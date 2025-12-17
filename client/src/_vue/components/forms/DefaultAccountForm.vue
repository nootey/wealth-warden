<script setup lang="ts">
import {useAccountStore} from "../../../services/stores/account_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {onMounted, ref} from "vue";
import type {Account, AccountType} from "../../../models/account_models.ts";

const emit = defineEmits<{
    (event: 'completeOperation'): void;
}>();

const accStore = useAccountStore();
const toastStore = useToastStore();

const accTypes = ref<AccountType[]>([]);
const selectedType = ref<AccountType | null>(null);
const selectedAccount = ref<Account | null>(null);
const accounts = ref<Account[]>([]);
const loading = ref(false);

onMounted(async () => {
    await getAccTypes();
});

async function getAccTypes() {
    try {
        accTypes.value = await accStore.getAccountTypesWithoutDefaults();
    } catch (e) {
        toastStore.errorResponseToast(e)
    }
}

async function onTypeChange() {
    if (!selectedType.value) {
        accounts.value = [];
        return;
    }

    loading.value = true;
    try {
        accounts.value = await accStore.getAccountsByType(selectedType.value.type);
    } catch (e) {
        toastStore.errorResponseToast(e);
        accounts.value = [];
    } finally {
        loading.value = false;
    }
}

async function setAsDefault() {
    try {
        const res = await accStore.setDefaultAccount(selectedAccount.value?.id!);
        toastStore.successResponseToast(res);
        selectedType.value = null;
        selectedAccount.value = null;
        emit("completeOperation")
    } catch (e) {
        toastStore.errorResponseToast(e);
    }
}

</script>

<template>
    <div class="flex flex-column gap-3 p-1">

        <span class="text-sm" style="color: var(--text-secondary)">Select an account type, and define which account should be the default for it.</span>
        <span class="text-sm" style="color: var(--text-secondary)">Only types which do not already have a default account assigned, are shown.</span>

        <div class="flex flex-row w-full">
            <div class="flex flex-column gap-1 w-full">
                <label>Account Type</label>
                <Select v-model="selectedType" :options="accTypes" filter
                        optionLabel="sub_type" placeholder="Select account type"
                        @change="onTypeChange" class="w-full" size="small"/>
            </div>
        </div>

        <div v-if="selectedType" class="flex flex-row w-full">
            <div class="flex flex-column gap-1 w-full">
                <label>Account</label>
                <Select v-model="selectedAccount" filter
                        :options="accounts" :loading="loading"
                        optionLabel="name" size="small"
                        placeholder="Select account"
                        class="w-full"/>
            </div>
        </div>
        <div class="flex flex-row w-full">
            <div class="flex flex-column gap-1 w-full">
                <Button v-if="selectedAccount"
                        label="Set as Default"
                        @click="setAsDefault"
                        class="main-button"
                />
            </div>
        </div>
    </div>
</template>

<style scoped>

</style>