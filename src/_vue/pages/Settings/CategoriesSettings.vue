<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import {useTransactionStore} from "../../../services/stores/transaction_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {computed, onMounted, ref} from "vue";
import type {Category} from "../../../models/transaction_models.ts";
import CategoriesDisplay from "../../components/data/CategoriesDisplay.vue";
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import CategoryForm from "../../components/forms/CategoryForm.vue";

const transactionStore = useTransactionStore();
const toastStore = useToastStore();
const sharedStore = useSharedStore();

onMounted(async () => {
    await transactionStore.getCategories();
});

const catRef = ref<InstanceType<typeof CategoriesDisplay> | null>(null);
const createModal = ref(false);

const categories = computed<Category[]>(() => transactionStore.categories);

async function getCategories() {
    await transactionStore.getCategories();
}

async function handleEmit(type: string, data?: any) {
    switch (type) {
        case 'completeOperation': {
            createModal.value = false;
            await getCategories();
            break;
        }
        case 'openCreate': {
            createModal.value = true;
            break;
        }
        case 'deleteCategory': {
            await deleteCategory(data);
            await getCategories();
            break;
        }
        default: {
            break;
        }
    }
}

async function deleteCategory(id: number) {
    console.log(id);
    try {
        let response = await sharedStore.deleteRecord(
            "transactions/categories",
            id,
        );
        toastStore.successResponseToast(response);

    } catch (err) {
        toastStore.errorResponseToast(err)
    }
}

</script>

<template>

    <Dialog class="rounded-dialog" v-model:visible="createModal"
            :breakpoints="{ '501px': '90vw' }" :modal="true" :style="{ width: '500px' }" header="Create category">
        <CategoryForm mode="create"
                      @completeOperation="handleEmit('completeOperation')"/>
    </Dialog>

    <div class="flex flex-column w-full gap-3">
        <SettingsSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-2">
                <div class="flex flex-row justify-content-between align-items-center">
                    <div class="w-full flex flex-column gap-2">
                        <h3>Categories</h3>
                        <h5 style="color: var(--text-secondary)">View and manage transaction categories.</h5>
                    </div>
                    <Button class="main-button w-3" label="New category" icon="pi pi-plus" @click="handleEmit('openCreate')"/>
                </div>


                <div v-if="categories" class="w-full flex flex-column gap-2 w-full">
                    <CategoriesDisplay
                            ref="catRef"
                            :categories="categories"
                            @completeOperation="handleEmit('completeOperation')"
                            @deleteCategory="(id) => handleEmit('deleteCategory', id)">

                    </CategoriesDisplay>
                </div>
            </div>
        </SettingsSkeleton>
    </div>
</template>

<style scoped>

</style>