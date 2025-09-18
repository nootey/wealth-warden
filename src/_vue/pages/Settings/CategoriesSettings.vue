<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import {useTransactionStore} from "../../../services/stores/transaction_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {computed, onMounted, ref} from "vue";
import type {Category} from "../../../models/transaction_models.ts";
import CategoriesDisplay from "../../components/data/CategoriesDisplay.vue";
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import CategoryForm from "../../components/forms/CategoryForm.vue";
import {useConfirm} from "primevue/useconfirm";
import {usePermissions} from "../../../utils/use_permissions.ts";

const transactionStore = useTransactionStore();
const toastStore = useToastStore();
const sharedStore = useSharedStore();
const { hasPermission } = usePermissions();

onMounted(async () => {
    await transactionStore.getCategories();
});

const confirm = useConfirm();

const catRef = ref<InstanceType<typeof CategoriesDisplay> | null>(null);
const createModal = ref(false);

const categories = computed<Category[]>(() => transactionStore.categories);

const includeDeleted = ref(false);

async function getCategories() {
    await transactionStore.getCategories(includeDeleted.value);
}

async function handleEmit(type: string, data?: any) {
    switch (type) {
        case 'completeOperation': {
            createModal.value = false;
            await getCategories();
            break;
        }
        case 'openCreate': {

            if(!hasPermission("manage_data")) {
                toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
                return;
            }

            createModal.value = true;
            break;
        }
        case 'deleteCategory': {
            await deleteConfirmation(data.id, data.name, data.deleted);
            break;
        }
        default: {
            break;
        }
    }
}

async function deleteConfirmation(id: number, name: string, deleted: Date | null) {
    confirm.require({
        header: 'Confirm operation',
        message: `You are about to ${!deleted ? 'archive' : 'delete'} category: "${name}". ${!deleted ? '' : 'This action is irreversible!'}`,
        rejectProps: { label: 'Cancel' },
        acceptProps: { label: 'Continue', severity: 'danger' },
        accept: () => deleteRecord(id),
    });
}

async function deleteRecord(id: number) {

    if(!hasPermission("manage_data")) {
        toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
        return;
    }

    try {
        let response = await sharedStore.deleteRecord(
            "transactions/categories",
            id,
        );
        toastStore.successResponseToast(response);
        await getCategories();

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
                <div class="flex flex-row justify-content-between align-items-center gap-3">
                    <div class="w-full flex flex-column gap-2">
                        <h3>Categories</h3>
                        <h5 style="color: var(--text-secondary)">View and manage transaction categories.</h5>
                    </div>

                    <div class="flex align-items-center gap-2" style="margin-left: auto;">
                        <span class="text-sm">Archived?</span>
                        <ToggleSwitch style="transform: scale(0.75)" v-model="includeDeleted"
                            @update:model-value="getCategories()"/>
                    </div>
                    <Button class="main-button w-4" label="New category" icon="pi pi-plus" @click="handleEmit('openCreate')"/>
                </div>


                <div v-if="categories" class="w-full flex flex-column gap-2 w-full">
                    <CategoriesDisplay
                            ref="catRef"
                            :categories="categories"
                            @completeOperation="handleEmit('completeOperation')"
                            @deleteCategory="(id, name, deleted_at) => handleEmit('deleteCategory', {id: id, name: name, deleted: deleted_at})">

                    </CategoriesDisplay>
                </div>
            </div>
        </SettingsSkeleton>
    </div>
</template>

<style scoped>

</style>