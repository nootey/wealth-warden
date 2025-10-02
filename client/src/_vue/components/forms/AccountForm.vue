<script setup lang="ts">

import ValidationError from "../../components/validation/ValidationError.vue";
import {required} from "@vuelidate/validators";
import {decimalValid, decimalMin, decimalMax} from "../../../validators/currency.ts";
import useVuelidate from "@vuelidate/core";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useAccountStore} from "../../../services/stores/account_store.ts";
import {computed, nextTick, onMounted, ref, watch} from "vue";
import vueHelper from "../../../utils/vue_helper.ts"
import type {Account, AccountType} from "../../../models/account_models.ts"
import currencyHelper from "../../../utils/currency_helper.ts";
import {useConfirm} from "primevue/useconfirm";
import toastHelper from "../../../utils/toast_helper.ts";
import Decimal from "decimal.js";
import ShowLoading from "../base/ShowLoading.vue";

const props = defineProps<{
    mode?: "create" | "update";
    recordId?: number | null;
}>();

const emit = defineEmits<{
    (event: 'completeOperation'): void;
}>();

const sharedStore = useSharedStore();
const accountStore = useAccountStore();
const toastStore = useToastStore();

onMounted(async () => {
    if (props.mode === "update" && props.recordId) {
        await loadRecord(props.recordId);
    }
});

const readOnly = ref(false);

const confirm = useConfirm();
const initializing = ref(false);

const record = ref<Account>(initData());
const balanceFieldRef = computed({
    get: () => {
        if (props.mode === "create") {
            return record.value.balance.start_balance;
        }
        return record.value.balance.end_balance;
    },
    set: (val) => {
        if (props.mode === "create") {
            record.value.balance.start_balance = val;
        } else {
            record.value.balance.end_balance = val;
        }
    },
});

const balanceNumber = currencyHelper.useMoneyField(balanceFieldRef, 2).number;
const balanceAdjusted = ref(false);

const selectedClassification = ref<"Asset" | "Liability">("Asset");
const selectedType = ref<string>("");
const selectedSubtype = ref<string>("");

const accountTypes = computed<AccountType[]>(() => accountStore.accountTypes);

// Unique type options for chosen classification
const typeOptions = computed<string[]>(() => {
  const filtered = accountTypes.value.filter(
      a => a.classification.toLowerCase() === selectedClassification.value.toLowerCase()
  );
  return [...new Set(filtered.map(a => a.type))];
});

// Unique subtype options for chosen type + classification
const subtypeOptions = computed<string[]>(() => {
  if (!selectedType.value) return [];
  const filtered = accountTypes.value.filter(
      a =>
          a.classification.toLowerCase() === selectedClassification.value.toLowerCase() &&
          a.type === selectedType.value
  );
  return [...new Set(filtered.map(a => a.sub_type))];
});

const filteredAccountTypes = ref<string[]>([]);
const filteredSubtypeOptions = ref<string[]>([]);

const searchAccountType = (event: { query: string }) => {
  const q = event.query.trim().toLowerCase();
  const all = typeOptions.value;
  filteredAccountTypes.value = !q ? [...all] : all.filter(t => t.toLowerCase().startsWith(q));
};

const searchSubtype = (event: { query: string }) => {
  const q = event.query.trim().toLowerCase();
  const all = subtypeOptions.value;
  filteredSubtypeOptions.value = !q ? [...all] : all.filter(s => s.toLowerCase().startsWith(q));
};

const rules = {
    record: {
        name: { required, $autoDirty: true },
        account_type: {
            type: { required, $autoDirty: true },
            sub_type: { required, $autoDirty: true },
        },
        balance: {
            start_balance: props.mode === "create" ? {
                required,
                decimalValid,
                decimalMin: decimalMin(0),
                decimalMax: decimalMax(1_000_000_000),
                $autoDirty: true,
            } : {},
            end_balance: props.mode === "update" ? {
                required,
                decimalValid,
                decimalMin: decimalMin(0),
                decimalMax: decimalMax(1_000_000_000),
                $autoDirty: true,
            } : {},
        },
    },
};

const v$ = useVuelidate(rules, { record });

// Format selected types
const formattedTypeModel = computed({
  get: () => vueHelper.formatString(selectedType.value),
  set: (val: string) => {
    selectedType.value = val;
  },
});

const formattedSubtypeModel = computed({
  get: () => vueHelper.formatString(selectedSubtype.value ?? ""),
  set: (val: string) => {
    selectedSubtype.value = val;
  },
});

// Keep classification in the account_type, reset selections
watch(selectedClassification, (cls) => {
    if (initializing.value) {
        // keep classification in the model without resetting type/subtype
        record.value.account_type.classification = cls;
        return;
    }
    selectedType.value = "";
    selectedSubtype.value = "";
    record.value.account_type = {
    id: null,
    name: "",
    type: "",
    sub_type: "",
    classification: cls,
    };
});

// Watch type changes
watch(
    [selectedType, selectedSubtype, selectedClassification],
    ([typeVal, subVal, clsVal], [oldType, oldSub, oldCls]) => {
        if (initializing.value) return;
        if (typeVal === oldType && subVal === oldSub && clsVal === oldCls) return;

        // keep current selections on the model
        record.value.account_type.type = typeVal || "";
        record.value.account_type.sub_type = subVal || "";
        record.value.account_type.classification = clsVal;

        // if subtype is no longer valid for the chosen type, clear it
        const stillValid =
            !!subVal && subtypeOptions.value.includes(subVal);
        if (!stillValid) {
            selectedSubtype.value = "";
            record.value.account_type.sub_type = "";
        }

        // resolve the exact AccountType
        const match = accountTypes.value.find(
            a =>
                a.classification.toLowerCase() === clsVal.toLowerCase() &&
                a.type === (typeVal || "") &&
                a.sub_type === (stillValid ? subVal : "")
        );

        record.value.account_type = match
            ? { ...match }
            : {
                id: null,
                name: "",
                type: typeVal || "",
                sub_type: stillValid ? subVal : "",
                classification: clsVal,
            };
    }
);

function initData(): Account {

  return {
    id: null,
    name: "",
    is_active: true,
    closed_at: null,
    account_type: {
      id: null,
      name: "",
      type: "",
      sub_type: "",
      classification: "",
    },
    balance: {
      id: null,
      start_balance: null,
      end_balance: null,
      as_of: null,
    },
  };
}

async function loadRecord(id: number) {
    try {
        initializing.value = true;
        const data = await sharedStore.getRecordByID(accountStore.apiPrefix, id);

        readOnly.value = !!data?.deleted_at || !data.is_active

        record.value = {
            ...initData(),
            ...data,
        };

        selectedClassification.value = vueHelper.capitalize(
            data.account_type.classification
        ) as "Asset" | "Liability";

        selectedType.value = data.account_type.type;
        selectedSubtype.value = data.account_type.sub_type;

        // keep form value non-negative for liabilities
        if (props.mode === "update" && selectedClassification.value === "Liability") {
            const b = record.value.balance.end_balance;
            if (b !== null) {
                record.value.balance.end_balance = new Decimal(b).abs().toString(); // NEW
            }
        }

        await nextTick();

    } catch (err) {
        toastStore.errorResponseToast(err);
    } finally {
        initializing.value = false;
    }
}

async function isRecordValid() {
  const isValid = await v$.value.record.$validate();
  if (!isValid) return false;
  return true;
}

async function confirmManage() {
    if(props.mode === "update" && balanceAdjusted.value) {
        confirm.require({
            header: 'Confirm balance adjustment',
            message: 'You have made a manual balance adjustment. Do you want to continue?',
            rejectProps: { label: 'Cancel' },
            acceptProps: { label: 'Confirm' },
            accept: () => manageRecord(),
        });
    } else {
        await manageRecord()
    }
}

async function manageRecord() {

    if (readOnly.value) {
        toastStore.infoResponseToast(toastHelper.formatInfoToast("Not allowed", "This record is read only!"))
        return;
    }

  if (!await isRecordValid()) return;

  const at =
      accountTypes.value.find(
          a =>
              a.classification.toLowerCase() === selectedClassification.value.toLowerCase() &&
              a.type === selectedType.value &&
              a.sub_type === selectedSubtype.value
      ) || null;

  if (!at) {
    toastStore.errorResponseToast("Account type not found!");
    return;
  }

    let balanceToSend =
        props.mode === "create"
            ? record.value.balance.start_balance
            : record.value.balance.end_balance

    const recordData: any = {
        account_type_id: at.id,
        name: record.value.name,
        type: at.type,
        sub_type: at.sub_type,
        classification: at.classification,
    }

    if (props.mode === "create") {
        recordData.balance = balanceToSend
    } else if (props.mode === "update" && balanceAdjusted.value) {
        // only send on update if the user actually edited it
        recordData.balance = balanceToSend
    }
  
  try {

    let response = null;

    switch (props.mode) {
      case "create":
          response = await sharedStore.createRecord(
              accountStore.apiPrefix,
              recordData
          );
          break;
      case "update":
          response = await sharedStore.updateRecord(
              accountStore.apiPrefix,
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

</script>

<template>

    <div v-if="!initializing" class="flex flex-column gap-3 p-1">

        <div v-if="!readOnly" class="flex flex-row w-full justify-content-center">
          <div class="flex flex-column w-50">
            <SelectButton
                style="font-size: 0.875rem;" size="small"
                v-model="selectedClassification" :options="['Asset', 'Liability']" :allowEmpty="false" />
          </div>
        </div>
        <div v-else>
           <h5 style="color: var(--text-secondary)">Read-only mode.</h5>
        </div>


        <div class="flex flex-row w-full">
          <div class="flex flex-column w-full">
            <ValidationError :isRequired="true" :message="v$.record.name.$errors[0]?.$message">
              <label>Name</label>
            </ValidationError>
            <InputText :readonly="readOnly" :disabled="readOnly" size="small" v-model="record.name"></InputText>
          </div>
        </div>

        <div class="flex flex-column gap-1">
          <ValidationError :isRequired="true" :message="v$.record.balance.$errors[0]?.$message">
            <label>Current balance</label>
          </ValidationError>
          <InputNumber :readonly="readOnly" :disabled="readOnly" size="small" v-model="balanceNumber"
                       mode="currency" currency="EUR" locale="de-DE" :min="0"
                       placeholder="0,00 €" :minFractionDigits="2" :maxFractionDigits="2"
                       @update:model-value="balanceAdjusted = true">
          </InputNumber>
        </div>

        <div class="flex flex-row w-full">
          <div class="flex flex-column gap-1 w-full">
            <ValidationError :isRequired="true" :message="v$.record.account_type.type.$errors[0]?.$message">
              <label>Type</label>
            </ValidationError>
            <AutoComplete :readonly="readOnly" :disabled="readOnly" size="small" v-model="formattedTypeModel" :suggestions="filteredAccountTypes"
                          @complete="searchAccountType" placeholder="Select type" dropdown>
              <template #option="slotProps">
                <div class="flex items-center">
                  {{ vueHelper.formatString(slotProps.option)}}
                </div>
              </template>
            </AutoComplete>
          </div>
        </div>

        <div class="flex flex-row gap-2 w-full">
          <div class="flex flex-column gap-1 w-full">
            <ValidationError :isRequired="true" :message="v$.record.account_type.sub_type.$errors[0]?.$message">
              <label>Subtype</label>
            </ValidationError>
            <AutoComplete :readonly="readOnly" :disabled="!selectedType || readOnly "
                size="small" v-model="formattedSubtypeModel" :suggestions="filteredSubtypeOptions"
                @complete="searchSubtype" placeholder="Select subtype" dropdown>
              <template #option="slotProps">
                <div class="flex items-center">
                  {{ vueHelper.formatString(slotProps.option)}}
                </div>
              </template>
            </AutoComplete>
          </div>
        </div>

        <div class="flex flex-row gap-2 w-full">
          <div class="flex flex-column w-full">
            <Button v-if="!readOnly" class="main-button" :label="(mode == 'create' ? 'Add' : 'Update') +  ' account'" @click="confirmManage" style="height: 42px;" />
          </div>
        </div>

    </div>
    <ShowLoading v-else :numFields="6" />


</template>

<style scoped>

</style>