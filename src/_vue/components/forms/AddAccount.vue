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
const selectedClassification = ref("Asset");

const accountTypes = computed<AccountType[]>(() => account_store.accountTypes);

const uniqueAccountTypes = computed(() => {
  const filtered = accountTypes.value.filter(account =>
      account.classification.toLowerCase() === selectedClassification.value.toLowerCase()
  );

  const types = filtered.map(account => account.type);
  return [...new Set(types)];
});
const selectedType = computed(() => newRecord.value.account_type.type);
const selectedSubtype = ref<string | null>(null);

const uniqueAccountSubtypes = computed(() => {
  if (!selectedType.value) return [];

  const filtered = accountTypes.value.filter(account =>
      account.type === selectedType.value &&
      account.classification.toLowerCase() === selectedClassification.value.toLowerCase()
  );

  const subtypes = filtered.map(account => account.subtype);
  return [...new Set(subtypes)];
});

const filteredAccountTypes = ref<string[]>([]);
const filteredSubtypeOptions = ref<string[]>([]);

watch(selectedSubtype, (newVal) => {
  newRecord.value.account_type.subtype = newVal ?? '';
});

watch(selectedClassification, () => {
  newRecord.value.account_type.type = "";
  selectedSubtype.value = null;
  newRecord.value.account_type.subtype = "";
});

const formattedTypeModel = computed({
  get: () => vueHelper.formatString(newRecord.value.account_type.type),
  set: (val: string) => {
    newRecord.value.account_type.type = val;
  },
});

const formattedSubtypeModel = computed({
  get: () => vueHelper.formatString(selectedSubtype.value ?? ""),
  set: (val: string) => {
    selectedSubtype.value = val;
  },
});

const searchAccountType = (event: { query: string }) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredAccountTypes.value = [...uniqueAccountTypes.value];
    } else {
      filteredAccountTypes.value = uniqueAccountTypes.value.filter((record) => {
        return record.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
}

const searchSubtype = (event: { query: string }) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredSubtypeOptions.value = [...uniqueAccountSubtypes.value];
    } else {
      filteredSubtypeOptions.value = uniqueAccountSubtypes.value
          .filter(subtype => subtype.toLowerCase().startsWith(event.query.toLowerCase()));
    }
  }, 250);
};

const emit = defineEmits<{
  (event: 'insertAccount'): void;
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

  const currentAccType = accountTypes.value.find(
      acc => acc.type === selectedType.value && acc.subtype === selectedSubtype.value
  );

  if (!currentAccType) {
    toast_store.errorResponseToast("Account type not found!");
    return;
  }

  newRecord.value.account_type.subtype = selectedSubtype.value ?? "";
  newRecord.value.account_type.classification = currentAccType?.classification;

  try {
    let response = await shared_store.createRecord(
      "accounts",
        {
          account_type_id: currentAccType.id,
          name: newRecord.value.name,
          type: newRecord.value.account_type.type,
          subtype: newRecord.value.account_type.subtype,
          classification: newRecord.value.account_type.classification,
          balance: newRecord.value.balance.start_balance,
        }
        );

    newRecord.value = initData();
    v$.value.newRecord.$reset();

    toast_store.successResponseToast(response);

    emit("insertAccount")

  } catch (error) {
    toast_store.errorResponseToast(error);
  }
}

</script>

<template>
  <div class="flex flex-column gap-3 p-1">

    <div class="flex flex-row w-full justify-content-center">
      <div class="flex flex-column w-50">
        <SelectButton style="font-size: 0.875rem;" size="small" v-model="selectedClassification" :options="['Asset', 'Liability']" />
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
      <InputNumber size="small" v-model="newRecord.balance.start_balance" mode="currency" currency="EUR" locale="de-DE" placeholder="0,00"></InputNumber>
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