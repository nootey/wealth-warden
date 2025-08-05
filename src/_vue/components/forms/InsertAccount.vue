<script setup lang="ts">

import ValidationError from "../../components/validation/ValidationError.vue";
import {numeric, required, minValue, maxValue} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useAccountStore} from "../../../services/stores/account_store.ts";
import {computed, ref, watch} from "vue";

const shared_store = useSharedStore();
const account_store = useAccountStore();
const toast_store = useToastStore();

const newRecord = ref(initData(false));
const selectedClassification = ref("Asset");

const accountTypes = computed(() => account_store.accountTypes);

const uniqueAccountTypes = computed(() => {
  const filtered = accountTypes.value.filter(account =>
      account.classification.toLowerCase() === selectedClassification.value.toLowerCase()
  );

  const types = filtered.map(account => account.type);
  return [...new Set(types)];
});
const selectedType = computed(() => newRecord.value.account_type.type);
const selectedSubtype = ref(null);

const uniqueAccountSubtypes = computed(() => {
  if (!selectedType.value) return [];

  const filtered = accountTypes.value.filter(account =>
      account.type === selectedType.value &&
      account.classification.toLowerCase() === selectedClassification.value.toLowerCase()
  );

  const subtypes = filtered.map(account => account.subtype);
  return [...new Set(subtypes)];
});

const filteredAccountTypes = ref([]);
const filteredSubtypeOptions = ref([]);

watch(selectedSubtype, (newVal) => {
  newRecord.value.account_type.subtype = newVal ?? '';
});

watch(selectedClassification, () => {
  newRecord.value.account_type.type = "";
  selectedSubtype.value = null;
  newRecord.value.account_type.subtype = "";
});

const searchAccountType = (event: any) => {
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

const searchSubtype = (event: any) => {
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
      classification: {
        required,
        $autoDirty: true
      },
    },
    balance: {
      required,
      numeric,
      minValue: minValue(0),
      maxValue: maxValue(1000000000),
      $autoDirty: true
    }
  },
};

const v$ = useVuelidate(rules, { newRecord });

function initData(isReoccurring: boolean = false): Record<string, any> {

  return {
    name: "",
    account_type: {
      type: "",
      subtype: "",
      classification: "",
    },
    balance: null,
  };
}

async function isRecordValid() {
  const isValid = await v$.value.newRecord.$validate();
  if (!isValid) return false;
  return true;
}

async function createNewRecord() {

  if (await isRecordValid()) return;

  newRecord.value.subtype = selectedSubtype.value ?? "";
  const currentAccType = accountTypes.value.find(
      acc => acc.type === selectedType.value && acc.subtype === selectedSubtype.value
  );
  newRecord.value.classification = currentAccType.classification;
  newRecord.value.account_type_id = currentAccType.id;

  try {
    let response = await shared_store.createRecord(
      "accounts",
      {
        account_type_id: newRecord.value.account_type_id,
        name: newRecord.value.name,
        type: newRecord.value.account_type.type,
        subtype: newRecord.value.subtype,
        classification: newRecord.value.classification,
        balance: newRecord.value.balance,
    });

    newRecord.value = initData(false);
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


    <div class="flex flex-roww-full">
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
      <InputNumber size="small" v-model="newRecord.balance" mode="currency" currency="EUR" locale="de-DE" placeholder="0,00"></InputNumber>
    </div>

    <div class="flex flex-row w-full">
      <div class="flex flex-column gap-1 w-full">
        <ValidationError :isRequired="true" :message="v$.newRecord.account_type.type.$errors[0]?.$message">
          <label>Type</label>
        </ValidationError>
        <AutoComplete size="small" v-model="newRecord.account_type.type" :suggestions="filteredAccountTypes"
                      @complete="searchAccountType" placeholder="Select type" dropdown></AutoComplete>
      </div>
    </div>

    <div class="flex flex-row gap-2 w-full">
      <div class="flex flex-column gap-1 w-full">
        <ValidationError :isRequired="true" :message="v$.newRecord.account_type.subtype.$errors[0]?.$message">
          <label>Subtype</label>
        </ValidationError>
        <AutoComplete
            size="small" v-model="selectedSubtype" :suggestions="filteredSubtypeOptions"
            @complete="searchSubtype" :disabled="!selectedType" placeholder="Select subtype" dropdown />
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