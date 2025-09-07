<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import {useTransactionStore} from "../../../services/stores/transaction_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {computed, onMounted, ref} from "vue";
import type {Category} from "../../../models/transaction_models.ts";
import CategoriesDisplay from "../../components/data/CategoriesDisplay.vue";
import {useSharedStore} from "../../../services/stores/shared_store.ts";

const transactionStore = useTransactionStore();
const toastStore = useToastStore();
const sharedStore = useSharedStore();

onMounted(async () => {
    await transactionStore.getCategories();
});

const catRef = ref<InstanceType<typeof CategoriesDisplay> | null>(null);

const categories = computed<Category[]>(() => transactionStore.categories);

async function handleEmit(type: string, data: any) {
    switch (type) {
        case 'deleteCategory': {
            await deleteCategory(data);
            break;
        }
        default: {
            break;
        }
    }
}

async function deleteCategory(id: number) {
    try {
        let response = await sharedStore.deleteRecord(
            "transactions/categories",
            id,
        );
        toastStore.successResponseToast(response);
        await transactionStore.getCategories();
    } catch (err) {
        toastStore.errorResponseToast(err)
    }
}

</script>

<template>
    <div class="flex flex-column w-full gap-3">
        <SettingsSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-2">
                <div class="w-full flex flex-column gap-2">
                    <h3>Categories</h3>
                    <h5 style="color: var(--text-secondary)">View and manage transaction categories.</h5>
                </div>

                <div v-if="categories" class="w-full flex flex-column gap-2 w-full">
                    <CategoriesDisplay
                            ref="catRef"
                            :categories="categories"
                            @deleteCategory="(id) => handleEmit('deleteCategory', id)"></CategoriesDisplay>
                </div>
            </div>
        </SettingsSkeleton>
    </div>
</template>

<style scoped>

</style>