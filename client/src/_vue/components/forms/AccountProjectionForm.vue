<script setup lang="ts">
import type {Account} from "../../../models/account_models.ts";
import ShowLoading from "../base/ShowLoading.vue";
import SlotSkeleton from "../layout/SlotSkeleton.vue";
import ValidationError from "../validation/ValidationError.vue";
import {computed, onMounted, ref, watch} from "vue";
import {required} from "@vuelidate/validators";
import {decimalMax, decimalMin, decimalValid} from "../../../validators/currency.ts";
import useVuelidate from "@vuelidate/core";
import {useTransactionStore} from "../../../services/stores/transaction_store.ts";
import type {CategoryOrGroup} from "../../../models/transaction_models.ts";
import {useStatisticsStore} from "../../../services/stores/statistics_store.ts";
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import Decimal from "decimal.js";
import currencyHelper from "../../../utils/currency_helper.ts";
import {useAccountStore} from "../../../services/stores/account_store.ts";
import {useToastStore} from "../../../services/stores/toast_store.ts";

const props = defineProps<{
    accID: number;
}>();

const transactionStore = useTransactionStore();
const statStore = useStatisticsStore();
const sharedStore = useSharedStore();
// const accountStore = useAccountStore();
const toastStore = useToastStore();

const categories = ref<CategoryOrGroup[]>([]);
const account = ref<Account|null>(null);

const categoryAverage = ref<number>(0);
const loadingAverage = ref<boolean>(false);

onMounted(async () => {
    categories.value = await transactionStore.getCategoriesWithGroups();
    account.value = await sharedStore.getRecordByID("accounts", props.accID, );
})

const record = ref({
    balance_projection: '',
    expected_balance: '',
    percentage_value: 0,
    multiplier_category_id: null as number|null,
    multiplier_value: 1
});

const expectedBalanceFieldRef = computed({
    get: () => record.value.expected_balance,
    set: (val) => {
        record.value.expected_balance = val;
    },
});

const expectedBalanceNumber = currencyHelper.useMoneyField(expectedBalanceFieldRef, 2).number;

const projectionOptions = [
    { label: 'Fixed', value: 'fixed' },
    { label: 'Multiplier', value: 'multiplier' },
    { label: 'Percentage', value: 'percentage' }
];

const categoryOptions = computed(() => {
    if (!categories.value || !Array.isArray(categories.value)) {
        return [];
    }

    return categories.value.map(cat => ({
        label: cat.name,
        value: cat.id
    }));
});

watch(() => record.value.multiplier_category_id, async (newCategoryId) => {
    if (newCategoryId && record.value.balance_projection === 'multiplier') {
        loadingAverage.value = true;
        categoryAverage.value = 0;

        try {
            const selectedCategory = categories.value.find(cat => cat.id === newCategoryId);
            const isGroup = selectedCategory?.is_group || false;
            const avg = await statStore.getCategoryAverage(newCategoryId, props.accID, isGroup);
            categoryAverage.value = Math.abs(avg);
        } catch (error) {
            categoryAverage.value = 0;
        } finally {
            loadingAverage.value = false;
        }
    } else {
        categoryAverage.value = 0;
    }
});

const currentBalanceNumber = computed(() => {
    if (!account.value?.balance?.end_balance) return 0;
    try {
        return new Decimal(account.value.balance.end_balance).toNumber();
    } catch {
        return 0;
    }
});

const expectedBalance = computed(() => {
    if (record.value.balance_projection === 'fixed') {
        if (!record.value.expected_balance) return 0;
        try {
            const balance = new Decimal(record.value.expected_balance);
            return balance.toNumber();
        } catch {
            return 0;
        }
    }

    if (record.value.balance_projection === 'percentage') {
        if (!account.value?.balance?.end_balance || record.value.percentage_value === 0) {
            return currentBalanceNumber.value; // Return current balance instead of 0
        }
        try {
            const currentBalance = new Decimal(account.value.balance.end_balance);
            const percentage = new Decimal(record.value.percentage_value || 0);
            const percentageIncrease = currentBalance.mul(percentage).div(100);
            const result = currentBalance.plus(percentageIncrease);
            return result.toNumber();
        } catch (error) {
            return currentBalanceNumber.value;
        }
    }

    if (record.value.balance_projection === 'multiplier') {
        if (loadingAverage.value || !record.value.multiplier_category_id || !record.value.multiplier_value) {
            return 0;
        }
        return categoryAverage.value * record.value.multiplier_value;
    }

    return 0;
});

const rules = computed(() => ({
    record: {
        balance_projection: { required, $autoDirty: true },
        expected_balance: {
            required,
            decimalValid,
            decimalMin: decimalMin(0),
            decimalMax: decimalMax(1_000_000_000),
            $autoDirty: true,
        },
        percentage_value: {
            decimalMin: decimalMin(0),
            decimalMax: decimalMax(100),
            $autoDirty: true,
        },
        multiplier_category_id: {
            required,
            $autoDirty: true,
        },
        multiplier_value: {
            required,
            minValue: 1,
            maxValue: 10,
            $autoDirty: true,
        }
    },
}));

const v$ = useVuelidate(rules, { record });

async function saveProjection() {
    // TODO: Implement save
    try {
    } catch (e) {
        toastStore.errorResponseToast(e)
    }
}

async function revertProjection() {
    // TODO: Implement revert
    try {

    } catch (e) {
        toastStore.errorResponseToast(e)
    }
}

</script>

<template>
    <div v-if="account" class="flex flex-column w-full gap-3">
        <SlotSkeleton class="w-full" bg="opt">
            <div class="flex flex-column gap-2 p-3 w-full">

                <div class="flex flex-column gap-1">
                    <label>Preview</label>
                    <span v-if="loadingAverage" class="text-lg font-semibold text-gray-500">
                        Loading...
                    </span>
                    <span v-else style="color: var(--text-secondary)">
                        {{ expectedBalance.toLocaleString('de-DE', {
                        style: 'currency',
                        currency: 'EUR',
                        minimumFractionDigits: 2,
                        maximumFractionDigits: 2
                    }) }}
                    </span>
                </div>

                <div class="flex flex-row w-full">
                    <div class="flex flex-column w-full gap-1">
                        <ValidationError :isRequired="true" :message="v$.record.balance_projection.$errors[0]?.$message">
                            <label>Projection</label>
                        </ValidationError>
                        <Select v-model="record.balance_projection"
                                :options="projectionOptions"
                                optionLabel="label"
                                optionValue="value"
                                placeholder="Select projection type"
                                size="small"
                        />
                    </div>
                </div>

                <!-- Fixed projection -->
                <div v-if="record.balance_projection === 'fixed'" class="flex flex-column gap-1">
                    <ValidationError :isRequired="true" :message="v$.record.expected_balance.$errors[0]?.$message">
                        <label>Expected balance</label>
                    </ValidationError>
                    <InputNumber size="small" v-model="expectedBalanceNumber"
                                 mode="currency" currency="EUR" locale="de-DE" :min="0"
                                 placeholder="0,00 €" :minFractionDigits="2" :maxFractionDigits="2">
                    </InputNumber>
                    <span class="text-sm" style="color: var(--text-secondary)">
                        Input a fixed balance. This value will be used as the expected balance for this account.
                    </span>
                </div>

                <!-- Percentage projection -->
                <div v-if="record.balance_projection === 'percentage'" class="flex flex-column gap-2">
                    <div class="flex flex-column gap-1">
                        <label>Current Balance</label>
                        <InputNumber size="small" :modelValue="parseFloat(account.balance.end_balance!)"
                                     mode="currency" currency="EUR" locale="de-DE"
                                     :minFractionDigits="2" :maxFractionDigits="2"
                                     disabled>
                        </InputNumber>
                    </div>
                    <div class="flex flex-column gap-1">
                        <ValidationError :message="v$.record.percentage_value.$errors[0]?.$message">
                            <label>Percentage</label>
                        </ValidationError>
                        <InputNumber size="small" v-model="record.percentage_value"
                                     suffix="%" :min="0" :max="100"
                                     placeholder="0%" :minFractionDigits="0" :maxFractionDigits="2">
                        </InputNumber>
                    </div>
                    <span class="text-sm" style="color: var(--text-secondary)">
                        Input a percentage rate. This value will be used as the expected growth for this account. The growth period is currently unlimited.
                    </span>
                </div>

                <!-- Multiplier projection -->
                <div v-if="record.balance_projection === 'multiplier'" class="flex flex-column gap-2">
                    <div class="flex flex-column gap-1">
                        <ValidationError :isRequired="true" :message="v$.record.multiplier_category_id.$errors[0]?.$message">
                            <label>Category/Group</label>
                        </ValidationError>
                        <Select v-model="record.multiplier_category_id"
                                :options="categoryOptions"
                                optionLabel="label"
                                optionValue="value"
                                placeholder="Select category or group"
                                size="small"
                                :loading="loadingAverage"
                        />
                    </div>

                    <div class="flex flex-column gap-1">
                        <ValidationError :isRequired="true" :message="v$.record.multiplier_value.$errors[0]?.$message">
                            <label>Multiplier</label>
                        </ValidationError>
                        <InputNumber size="small" v-model="record.multiplier_value"
                                     :min="1" :max="10"
                                     placeholder="1" :maxFractionDigits="0">
                        </InputNumber>
                    </div>

                    <span class="text-sm" style="color: var(--text-secondary)">
                        Select a category or group and input a multiplier.
                        The expected balance will be the monthly average of transactions, linked to this category, constrained to the current year - with the provided multiplier.
                    </span>

                    <span class="text-sm" style="color: var(--text-secondary)">
                        For example, you can define a 6 times multiplier and select your salary as the category, which would set the expected balance as 6 times the average monthly salary.
                    </span>
                </div>

                <div class="flex flex-column w-full">
                    <Button class="main-button" label="Save" @click="saveProjection" style="height: 42px;" />
                </div>

                <div class="flex flex-column w-full">
                    <Button class="delete-button" label="Revert" @click="revertProjection" style="height: 42px;" />
                </div>

            </div>
        </SlotSkeleton>
    </div>
    <ShowLoading v-else :numFields="3" />
</template>

<style scoped>

</style>