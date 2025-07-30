<script setup lang="ts">

import ValidationError from "../../components/validation/ValidationError.vue";
import {required} from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import {useToastStore} from "../../../services/stores/toast_store.ts";
import {useSharedStore} from "../../../services/stores/shared_store.ts";
import {useAccountStore} from "../../../services/stores/account_store.ts";
import {computed, ref, watch} from "vue";

const shared_store = useSharedStore();
const account_store = useAccountStore();
const toast_store = useToastStore();

const NewRecord = ref(initData(false));
const selectedClassification = ref("Asset");

const accountTypes = computed(() => account_store.accountTypes);

const uniqueAccountTypes = computed(() => {
  const filtered = accountTypes.value.filter(account =>
      account.classification.toLowerCase() === selectedClassification.value.toLowerCase()
  );

  const types = filtered.map(account => account.type);
  return [...new Set(types)];
});
const selectedType = computed(() => NewRecord.value.account_type.type);
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
  NewRecord.value.account_type.subtype = newVal?.name ?? '';
});

watch(selectedClassification, () => {
  NewRecord.value.account_type.type = "";
  selectedSubtype.value = null;
  NewRecord.value.account_type.subtype = "";
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
      filteredSubtypeOptions.value = [...uniqueAccountSubtypes.value.map(s => ({ name: s }))];
    } else {
      filteredSubtypeOptions.value = uniqueAccountSubtypes.value
          .filter(subtype =>
              subtype.toLowerCase().startsWith(event.query.toLowerCase())
          )
          .map(s => ({ name: s }));
    }
  }, 250);
};

const emit = defineEmits<{
  (event: 'insertAccount'): void;
}>();

const rules = {
  NewRecord: {
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
  },
};

const v$ = useVuelidate(rules, { NewRecord });

function initData(isReoccurring: boolean = false): Record<string, any> {

  return {
    name: "",
    account_type: {
      type: "",
      subtype: "",
      classification: "",
    },
  };
}

async function validateRecord(reoccurring = false) {
  const isValidInflow = await v$.value.NewRecord.$validate();
  let isValidReoccurring = true;

  if (reoccurring) {
    isValidReoccurring = await v$.value?.newReoccurringRecord.$validate();
  }

  if (!isValidReoccurring) return true;
  if (!isValidInflow) return true;

  return false;
}

async function createNewRecord() {

  if (await validateRecord(false)) return;

  NewRecord.value.subtype = selectedSubtype.value?.name ?? "";
  NewRecord.value.classification = accountTypes.value.find(
      acc => acc.type === selectedType.value?.name && acc.subtype === selectedSubtype.value?.name
  )?.classification ?? "";

  try {
    let response = await shared_store.createRecord(
      "accounts",
      {
        id: null,
        name: NewRecord.value.name,
        subtype: NewRecord.value.subtype,
        classification: NewRecord.value.classification,
    });

    NewRecord.value = initData(false);
    v$.value.NewRecord.$reset();

    toast_store.successResponseToast(response);

  } catch (error) {
    toast_store.errorResponseToast(error);
  }
}


</script>

<template>
  <div class="flex flex-column gap-4 p-1">

    <div class="flex flex-row gap-2 w-full justify-content-center">
      <div class="flex flex-column w-50">
        <SelectButton style="font-size: 0.875rem;" size="small" v-model="selectedClassification" :options="['Asset', 'Liability']" />
      </div>
    </div>


    <div class="flex flex-row gap-2 w-full">
      <div class="flex flex-column w-full">
        <ValidationError :isRequired="true" :message="v$.NewRecord.name.$errors[0]?.$message">
          <label>Name</label>
        </ValidationError>
        <InputText size="small" v-model="NewRecord.name"></InputText>
      </div>
    </div>

    <div class="flex flex-row gap-2 w-full">
      <div class="flex flex-column w-full">
        <ValidationError :isRequired="true" :message="v$.NewRecord.account_type.type.$errors[0]?.$message">
          <label>Type</label>
        </ValidationError>
        <AutoComplete size="small" v-model="NewRecord.account_type.type" :suggestions="filteredAccountTypes"
                      @complete="searchAccountType" placeholder="Select type" dropdown></AutoComplete>
      </div>
    </div>

    <div class="flex flex-row gap-2 w-full">
      <div class="flex flex-column w-full">
        <ValidationError :isRequired="true" :message="v$.NewRecord.account_type.subtype.$errors[0]?.$message">
          <label>Subtype</label>
        </ValidationError>
        <AutoComplete
            size="small" v-model="selectedSubtype" :suggestions="filteredSubtypeOptions"
            @complete="searchSubtype" :disabled="!selectedType" option-label="name" placeholder="Select subtype" dropdown />
      </div>
    </div>

    <div class="flex flex-row gap-2 w-full">
      <div class="flex flex-column w-full">
        <Button class="main-button" icon="pi pi-cart-plus" label="Create account" @click="createNewRecord" style="height: 42px;" />
      </div>
    </div>

  </div>
</template>

<style scoped>

</style>