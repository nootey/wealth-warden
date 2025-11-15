<script setup lang="ts">

import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import {useTransactionStore} from "../../../services/stores/transaction_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {computed, onMounted, ref} from "vue";
import type {Category, CategoryGroup} from "../../../models/transaction_models.ts";
import CategoriesDisplay from "../../components/data/CategoriesDisplay.vue";
import CategoryForm from "../../components/forms/CategoryForm.vue";
import {usePermissions} from "../../../utils/use_permissions.ts";
import CategoryGroupForm from "../../components/forms/CategoryGroupForm.vue";
import CategoryGroupsDisplay from "../../components/data/CategoryGroupsDisplay.vue";

const transactionStore = useTransactionStore();
const toastStore = useToastStore();
const { hasPermission } = usePermissions();

onMounted(async () => {
    await getCategories();
    await getCategoryGroups();
});


const catRef = ref<InstanceType<typeof CategoriesDisplay> | null>(null);
const groupRef = ref<InstanceType<typeof CategoryGroupsDisplay> | null>(null);
const createCatModal = ref(false);
const createGroupModal = ref(false);

const categories = computed<Category[]>(() => transactionStore.categories);
const categoryGroups = computed<CategoryGroup[]>(() => transactionStore.category_groups);

const includeDeleted = ref(false);

async function getCategories() {
    await transactionStore.getCategories(includeDeleted.value);
}

async function getCategoryGroups() {
    await transactionStore.getCategoryGroups();
}

async function handleEmit(type: string) {
    switch (type) {
        case 'completeCatOperation': {
            createCatModal.value = false;
            await getCategories();
            break;
        }
        case 'completeGroupOperation': {
            createGroupModal.value = false;
            await getCategoryGroups();
            break;
        }
        case 'openCatCreate': {

            if(!hasPermission("manage_data")) {
                toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
                return;
            }

            createCatModal.value = true;
            break;
        }
        case 'openGroupCreate': {

            if(!hasPermission("manage_data")) {
                toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
                return;
            }

            createGroupModal.value = true;
            break;
        }
        case 'completeCatDelete': {
            await getCategories()
            break;
        }
        case 'completeGroupDelete': {
            await getCategoryGroups()
            break;
        }
        default: {
            break;
        }
    }
}

</script>

<template>

    <Dialog class="rounded-dialog" v-model:visible="createCatModal"
            :breakpoints="{ '501px': '90vw' }" :modal="true" :style="{ width: '500px' }" header="Create category">
        <CategoryForm mode="create"
                      @completeOperation="handleEmit('completeCatOperation')"/>
    </Dialog>

    <Dialog class="rounded-dialog" v-model:visible="createGroupModal"
            :breakpoints="{ '501px': '90vw' }" :modal="true" :style="{ width: '500px' }" header="Create category group">
        <CategoryGroupForm mode="create" :categories="categories"
                      @completeOperation="handleEmit('completeGroupOperation')"/>
    </Dialog>

    <div class="flex flex-column w-full gap-3">
        <SettingsSkeleton class="w-full">
            <div class="w-full flex flex-column gap-3 p-2">
                <div class="flex flex-row justify-content-between align-items-center gap-3">
                    <div class="w-full flex flex-column gap-2">
                        <h3>Categories</h3>
                        <h5 class="mobile-hide" style="color: var(--text-secondary)">View and manage transaction categories.</h5>
                    </div>

                    <div class="flex align-items-center gap-2" style="margin-left: auto;">
                        <span class="text-sm">Archived?</span>
                        <ToggleSwitch style="transform: scale(0.75)" v-model="includeDeleted"
                            @update:model-value="getCategories()"/>
                    </div>
                    <Button class="main-button w-4"
                            @click="handleEmit('openCatCreate')">
                        <div class="flex flex-row gap-1 align-items-center">
                            <i class="pi pi-plus"></i>
                            <span class="mobile-hide"> New category </span>
                        </div>
                    </Button>
                </div>

                <div v-if="categories" class="w-full flex flex-column gap-2 w-full">
                    <CategoriesDisplay
                            ref="catRef"
                            :categories="categories"
                            @completeOperation="handleEmit('completeCatOperation')"
                            @completeDelete="handleEmit('completeCatDelete')">
                    </CategoriesDisplay>
                </div>

                <div class="w-full flex flex-column gap-3 p-2">
                    <div class="flex flex-row justify-content-between align-items-center gap-3">
                        <div class="w-full flex flex-column gap-2">
                            <h3>Category groupings</h3>
                            <h5 class="mobile-hide" style="color: var(--text-secondary)">View and manage groupings of your categories.</h5>
                        </div>

                        <Button class="main-button w-4"
                                @click="handleEmit('openGroupCreate')">
                            <div class="flex flex-row gap-1 align-items-center">
                                <i class="pi pi-plus"></i>
                                <span class="mobile-hide"> New grouping </span>
                            </div>
                        </Button>
                    </div>
                </div>

                <div v-if="categoryGroups" class="w-full flex flex-column gap-2 w-full">
                    <CategoryGroupsDisplay
                            ref="groupRef"
                            :categories="categories"
                            :category_groups="categoryGroups"
                            @completeOperation="handleEmit('completeGroupOperation')"
                            @completeDelete="handleEmit('completeGroupDelete')">
                    </CategoryGroupsDisplay>
                </div>

            </div>
        </SettingsSkeleton>
    </div>
</template>

<style scoped>

</style>