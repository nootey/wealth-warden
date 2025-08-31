<script setup lang="ts">
import {computed, ref, watch} from "vue";
import type {Account} from "../../../models/account_models.ts";
import type {Transfer} from "../../../models/transaction_models.ts";
import useVuelidate from "@vuelidate/core";
import {required} from "@vuelidate/validators";
import ValidationError from "../validation/ValidationError.vue";
import {decimalMax, decimalMin, decimalValid} from "../../../validators/currency.ts";
import currencyHelper from "../../../utils/currency_helper.ts";

const props = defineProps<{
    accounts: Account[];
    transfer: Transfer;
}>();

const emit = defineEmits<{
    (event: "update:transfer", value: Transfer): void;
}>();

const localTransfer = ref<{
    source: Account | null;
    destination: Account | null;
    amount: string | null;
    notes: string | null;
}>({
    source: null,
    destination: null,
    amount: null,
    notes: props.transfer.notes ?? null
});

const rules = {
    localTransfer: {
        source: { required, $autoDirty: true },
        destination: { required, $autoDirty: true },
        amount: {
            required,
            decimalValid,
            decimalMin: decimalMin(0),
            decimalMax: decimalMax(1_000_000_000),
            $autoDirty: true
        },
        notes: { $autoDirty: true },
    }
};

const v$ = useVuelidate(rules, { localTransfer });
const amountRef = computed({
    get: () => localTransfer.value.amount,
    set: v => localTransfer.value.amount = v
});
const { number: amountNumber } = currencyHelper.useMoneyField(amountRef, 2);

watch(
    localTransfer,
    (val) => {
        emit("update:transfer", {
            ...props.transfer,
            transaction_outflow_id: val.source?.id ?? null,
            transaction_inflow_id: val.destination?.id ?? null,
            notes: val.notes ?? null
        });
    },
    { deep: true }
);

const filteredSourceAccounts = ref<Account[]>([]);
const filteredDestinationAccounts = ref<Account[]>([]);

function searchAccount(type: "source" | "destination", event: { query: string }) {
    setTimeout(() => {
        let results = props.accounts;

        if (event.query?.trim().length) {
            results = results.filter((a) =>
                a.name.toLowerCase().startsWith(event.query.toLowerCase())
            );
        }

        if (type === "source") {
            // exclude currently selected destination
            if (localTransfer.value.destination) {
                results = results.filter((a) => a.id !== localTransfer.value?.destination?.id);
            }
            filteredSourceAccounts.value = results;
        } else {
            // exclude currently selected source
            if (localTransfer.value.source) {
                results = results.filter((a) => a.id !== localTransfer.value?.source?.id);
            }
            filteredDestinationAccounts.value = results;
        }
    }, 200);
}

// Expose validator so parent can call it
defineExpose({ v$, localTransfer });

</script>

<template>
    <div class="flex flex-column gap-3">

        <div class="flex flex-row w-full">
            <div class="flex flex-column gap-1 w-full">
                <ValidationError :isRequired="true" :message="v$.localTransfer.source.$errors[0]?.$message">
                    <label>Source</label>
                </ValidationError>
                <AutoComplete
                        v-model="localTransfer.source"
                        :suggestions="filteredSourceAccounts"
                        optionLabel="name" @complete="(e) => searchAccount('source', e)"
                        placeholder="Select source account" dropdown/>
            </div>
        </div>

        <div class="flex flex-row w-full">
            <div class="flex flex-column gap-1 w-full">
                <ValidationError :isRequired="true" :message="v$.localTransfer.destination.$errors[0]?.$message">
                    <label>Destination</label>
                </ValidationError>
            <AutoComplete
                    v-model="localTransfer.destination"
                    :suggestions="filteredDestinationAccounts"
                    optionLabel="name" @complete="(e) => searchAccount('destination', e)"
                    placeholder="Select destination account" dropdown />
            </div>
        </div>

        <div class="flex flex-row w-full">
            <div class="flex flex-column gap-1 w-full">
                <ValidationError :isRequired="true" :message="v$.localTransfer.amount.$errors[0]?.$message">
                    <label>Amount</label>
                </ValidationError>
                <InputNumber size="small" v-model="amountNumber" mode="currency" currency="EUR" locale="de-DE" placeholder="0,00 €"></InputNumber>
            </div>
        </div>

        <div class="flex flex-row w-full">
            <div class="flex flex-column gap-1 w-full">
                <ValidationError :isRequired="false" :message="v$.localTransfer.notes.$errors[0]?.$message">
                    <label>Notes</label>
                </ValidationError>
            <InputText
                    v-model="localTransfer.notes" placeholder="Describe transfer"/>
        </div>
    </div>

    </div>
</template>