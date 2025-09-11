<script setup lang="ts">
import {useAuthStore} from "../../services/stores/auth_store.ts";
import {useAccountStore} from "../../services/stores/account_store.ts";
import {useToastStore} from "../../services/stores/toast_store.ts";
import SlotSkeleton from "../components/layout/SlotSkeleton.vue";
import NetworthWidget from "../components/widgets/NetworthWidget.vue";
import {ref} from "vue";

const authStore = useAuthStore();
const accountStore = useAccountStore();
const toastStore = useToastStore();

const nWidgetRef = ref<InstanceType<typeof NetworthWidget> | null>(null);

async function backfillBalances(){
    try {
        const response = await accountStore.backfillBalances();
        toastStore.successResponseToast(response.data);
        nWidgetRef.value?.refresh();
    } catch (err) {
        toastStore.errorResponseToast(err)
    }
}


</script>

<template>

    <main class="flex flex-column w-full p-2 align-items-center" style="height: 100vh;">

        <div class="flex flex-column justify-content-center p-2 w-full gap-3 border-round-md"
             style="max-width: 1000px;">

            <SlotSkeleton bg="transparent">
                <div class="w-full flex flex-row justify-content-between p-2 gap-2">
                    <div class="w-full flex flex-column gap-2">
                        <div style="font-weight: bold;"> Welcome back {{ authStore?.user?.display_name }} </div>
                        <div>{{ "Here's what's happening with your finances." }} </div>
                    </div>
                    <Button label="Refresh" icon="pi pi-refresh" class="main-button" @click="backfillBalances"></Button>
                </div>
            </SlotSkeleton>

            <NetworthWidget ref="nWidgetRef" :chartHeight="400" />

            <SlotSkeleton bg="secondary">
                <div class="w-full flex flex-row justify-content-between p-2 gap-2">
                    Assets - WIP
                </div>
            </SlotSkeleton>

            <SlotSkeleton bg="secondary">
                <div class="w-full flex flex-row justify-content-between p-2 gap-2">
                    Liabilities - WIP
                </div>
            </SlotSkeleton>

    </div>
    </main>

</template>

<style scoped>

</style>