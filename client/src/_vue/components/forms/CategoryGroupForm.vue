<script setup lang="ts">
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {computed, nextTick, onMounted, ref, watch} from "vue";
import type {Category, CategoryGroup} from "../../../models/transaction_models.ts";
import {required} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import ValidationError from "../validation/ValidationError.vue";
import ShowLoading from "../base/ShowLoading.vue";
import {usePermissions} from "../../../utils/use_permissions.ts";

const props = defineProps<{
    mode?: "create" | "update";
    recordId?: number | null;
    categories: Category[];
}>();

const emit = defineEmits<{
    (event: 'completeOperation'): void;
}>();

const apiPrefix = "transactions/categories/groups"

const sharedStore = useSharedStore();
const toastStore = useToastStore();

const { hasPermission } = usePermissions();

onMounted(async () => {
    if (props.mode === "update" && props.recordId) {
        await loadRecord(props.recordId);
    }
});

const loading = ref(false);

const parentCategories = computed(() => {
    return props.categories.filter(c =>
        c.display_name === "Expense" || c.display_name === "Income"
    )
});

const selectedParentCategory = computed<Category | null>(() => {
    const classification = record.value.classification || "income";
    return parentCategories.value.find(cat => cat.name === classification.toLowerCase()) || null;
});

const availableCategories = computed<Category[]>(() => {
    return props.categories.filter(
        (category) => category.parent_id === selectedParentCategory.value?.id
    );
});

const selectedCategories = ref<Category[]>([]);
const record = ref<CategoryGroup>(initData());

const classifications = ref<string[]>(['income', 'expense']);
const filteredClassifications = ref<string[]>([]);

const rules = {
    record: {
        name: { required, $autoDirty: true },
        classification: { required, $autoDirty: true },
        description: { $autoDirty: true },
    },
};

const v$ = useVuelidate(rules, { record });

watch(() => record.value.classification, () => {
    if (!loading.value) {
        selectedCategories.value = [];
    }
});

function initData(): CategoryGroup {

    return {
        name: "",
        classification: "income",
        description: null,
    };
}

async function loadRecord(id: number) {
    try {
        loading.value = true;
        const data = await sharedStore.getRecordByID(apiPrefix, id, { deleted: true});

        record.value = {
            ...initData(),
            ...data,
        };

        if (data.categories && Array.isArray(data.categories)) {
            selectedCategories.value = data.categories.map((cat: any) =>
                props.categories.find(c => c.id === cat.id)
            ).filter(Boolean) as Category[];
        }

        await nextTick();
        loading.value = false;

    } catch (err) {
        toastStore.errorResponseToast(err);
    }
}

async function isRecordValid() {
    const isValid = await v$.value.record.$validate();
    if (!isValid) return false;
    return true;
}

async function manageRecord() {

    if(!hasPermission("manage_data")) {
        toastStore.createInfoToast("Access denied", "You don't have permission to perform this action.");
        return;
    }

    if (!await isRecordValid()) return;

    const recordData: any = {
        name: record.value.name,
        classification: record.value.classification,
        description: record.value.description,
        selected_categories: selectedCategories.value.map(cat => cat.id)
    }

    try {

        let response = null;

        switch (props.mode) {
            case "create":
                response = await sharedStore.createRecord(
                    apiPrefix,
                    recordData
                );
                break;
            case "update":
                response = await sharedStore.updateRecord(
                    apiPrefix,
                    record.value.id!,
                    recordData
                );
                break;
            default:
                emit("completeOperation")
                break;
        }

        v$.value.record.$reset();
        toastStore.successResponseToast(response);
        emit("completeOperation")

    } catch (error) {
        toastStore.errorResponseToast(error);
    }
}

const searchClassifications = (event: { query: string }) => {
    const q = event.query.trim().toLowerCase();
    const all = classifications.value;
    filteredClassifications.value = !q ? [...all] : all.filter(t => t.toLowerCase().startsWith(q));
};

</script>

<template>

    <div v-if="!loading" class="flex flex-column gap-3 p-1">
        <div class="flex flex-column gap-3 p-1">
            <div class="flex flex-row w-full">
                <div class="flex flex-column w-full gap-1">
                    <ValidationError :isRequired="true" :message="v$.record.name.$errors[0]?.$message">
                        <label>Name</label>
                    </ValidationError>
                    <InputText size="small" v-model="record.name"></InputText>
                </div>
            </div>

            <div class="flex flex-row w-full">
                <div class="flex flex-column w-full gap-1">
                    <ValidationError :isRequired="false" :message="v$.record.description.$errors[0]?.$message">
                        <label>Description</label>
                    </ValidationError>
                    <InputText size="small" v-model="record.description"></InputText>
                </div>
            </div>

            <div class="flex flex-row w-full">
                <div class="flex flex-column gap-1 w-full">
                    <ValidationError :isRequired="true" :message="v$.record.classification.$errors[0]?.$message">
                        <label>Classification</label>
                    </ValidationError>
                    <AutoComplete size="small" v-model="record.classification"
                                  :suggestions="filteredClassifications" @complete="searchClassifications"
                                  placeholder="Select classification" dropdown>
                    </AutoComplete>
                </div>
            </div>

            <div class="flex flex-row w-full">
                <div class="flex flex-column gap-1 w-full">
                    <label>Selected categories</label>
                    <MultiSelect size="small" v-model="selectedCategories"
                                 placeholder="Select categories"
                                 :options="availableCategories" optionLabel="display_name"
                    >
                    </MultiSelect>
                </div>
            </div>

        </div>

        <div class="flex flex-row gap-2 w-full">
            <div class="flex flex-column w-full">
                <Button class="main-button" :label="(mode == 'create' ? 'Add' : 'Update') +  ' group'"
                        @click="manageRecord" style="height: 42px;" />
            </div>
        </div>

    </div>
    <ShowLoading v-else :numFields="4" />

</template>

<style scoped>

</style>