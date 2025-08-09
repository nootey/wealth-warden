<script setup lang="ts">

import ValidationError from "../../components/validation/ValidationError.vue";
import {numeric, required, minValue, maxValue} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useAccountStore} from "../../../services/stores/account_store.ts";
import {computed, ref, watch} from "vue";
import vueHelper from "../../../utils/vueHelper.ts"
import type {Account, AccountType} from "../../../models/account_models.ts"

const shared_store = useSharedStore();
const account_store = useAccountStore();
const toast_store = useToastStore();

const newRecord = ref<Account>(initData());

const selectedClassification = ref<"Asset" | "Liability">("Asset");
const selectedType = ref<string>("");
const selectedSubtype = ref<string>("");

const accountTypes = computed<AccountType[]>(() => account_store.accountTypes);

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
  return [...new Set(filtered.map(a => a.subtype))];
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

const emit = defineEmits<{
  (event: 'addAccount'): void;
}>();

const rules = {
  newRecord: {
    name: {
      required,
      $autoDirty: true
    },
    account_type: {
      type: {
        required,
        $autoDirty: true
      },
      subtype: {
        required,
        $autoDirty: true
      },
    },
    balance: {
      start_balance: {
        required,
        numeric,
        minValue: minValue(0),
        maxValue: maxValue(1000000000),
        $autoDirty: true
      },
    },
  },
};

const v$ = useVuelidate(rules, { newRecord });

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
  selectedType.value = "";
  selectedSubtype.value = "";
  newRecord.value.account_type = {
    id: null,
    name: "",
    type: "",
    subtype: "",
    classification: cls,
  };
});

// Watch type changes
watch(selectedType, (val) => {
  newRecord.value.account_type.type = val || "";
  newRecord.value.account_type.subtype = "";

  selectedSubtype.value = "";

  const firstMatch = accountTypes.value.find(
      a =>
          a.classification.toLowerCase() === selectedClassification.value.toLowerCase() &&
          a.type === val
  );

  if (firstMatch) {
    newRecord.value.account_type = { ...firstMatch };
  } else {
    newRecord.value.account_type = {
      id: null,
      name: "",
      type: val || "",
      subtype: "",
      classification: selectedClassification.value,
    };
  }
});

function initData(): Account {

  return {
    id: null,
    name: "",
    account_type: {
      id: null,
      name: "",
      type: "",
      subtype: "",
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

async function isRecordValid() {
  const isValid = await v$.value.newRecord.$validate();
  if (!isValid) return false;
  return true;
}

async function createNewRecord() {

  if (!await isRecordValid()) return;

  const at =
      accountTypes.value.find(
          a =>
              a.classification.toLowerCase() === selectedClassification.value.toLowerCase() &&
              a.type === selectedType.value &&
              a.subtype === selectedSubtype.value
      ) || null;

  if (!at) {
    toast_store.errorResponseToast("Account type not found!");
    return;
  }
  
  try {
    let response = await shared_store.createRecord(
      "accounts",
        {
          account_type_id: at.id,
          name: newRecord.value.name,
          type: at.type,
          subtype: at.subtype,
          classification: at.classification,
          balance: newRecord.value.balance.start_balance,
        }
        );

    newRecord.value = initData();
    v$.value.newRecord.$reset();

    toast_store.successResponseToast(response);

    emit("addAccount")

  } catch (error) {
    toast_store.errorResponseToast(error);
  }
}

</script>

<template>
  <div class="flex flex-column gap-3 p-1">

    <div class="flex flex-row w-full justify-content-center">
      <div class="flex flex-column w-50">
        <SelectButton
            style="font-size: 0.875rem;" size="small"
            v-model="selectedClassification" :options="['Asset', 'Liability']" :allowEmpty="false" />
      </div>
    </div>


    <div class="flex flex-row w-full">
      <div class="flex flex-column w-full">
        <ValidationError :isRequired="true" :message="v$.newRecord.name.$errors[0]?.$message">
          <label>Name</label>
        </ValidationError>
        <InputText size="small" v-model="newRecord.name"></InputText>
      </div>
    </div>

    <div class="flex flex-column gap-1">
      <ValidationError :isRequired="true" :message="v$.newRecord.balance.$errors[0]?.$message">
        <label>Current balance</label>
      </ValidationError>
      <InputNumber size="small" v-model="newRecord.balance.start_balance" mode="currency" currency="EUR" locale="de-DE" placeholder="0,00 €"></InputNumber>
    </div>

    <div class="flex flex-row w-full">
      <div class="flex flex-column gap-1 w-full">
        <ValidationError :isRequired="true" :message="v$.newRecord.account_type.type.$errors[0]?.$message">
          <label>Type</label>
        </ValidationError>
        <AutoComplete size="small" v-model="formattedTypeModel" :suggestions="filteredAccountTypes"
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
        <ValidationError :isRequired="true" :message="v$.newRecord.account_type.subtype.$errors[0]?.$message">
          <label>Subtype</label>
        </ValidationError>
        <AutoComplete
            size="small" v-model="formattedSubtypeModel" :suggestions="filteredSubtypeOptions"
            @complete="searchSubtype" :disabled="!selectedType" placeholder="Select subtype" dropdown>
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
        <Button class="main-button" label="Create account" @click="createNewRecord" style="height: 42px;" />
      </div>
    </div>

  </div>
</template>

<style scoped>

</style>