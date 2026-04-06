<script setup lang="ts">
import ValidationError from "../validation/ValidationError.vue";
import ShowLoading from "../base/ShowLoading.vue";
import vueHelper from "../../../utils/vue_helper.ts";
import savingsHelper from "../../../utils/savings_helper.ts";
import { useSettingsStore } from "../../../services/stores/settings_store.ts";
import { useConfirm } from "primevue/useconfirm";
import { usePermissions } from "../../../utils/use_permissions.ts";
import type {
  SavingContribution,
  SavingContributionReq,
  SavingGoalWithProgress,
} from "../../../models/savings_models.ts";
import { useSavingsStore } from "../../../services/stores/savings_store.ts";
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import Decimal from "decimal.js";
import dateHelper from "../../../utils/date_helper.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { computed, onMounted, ref } from "vue";
import currencyHelper from "../../../utils/currency_helper.ts";
import { required } from "@vuelidate/validators";
import { decimalMin, decimalValid } from "../../../validators/currency.ts";
import useVuelidate from "@vuelidate/core";
import filterHelper from "../../../utils/filter_helper.ts";
import type { PaginatorState } from "../../../models/shared_models.ts";
import CustomPaginator from "../base/CustomPaginator.vue";

const props = defineProps<{
  goal: SavingGoalWithProgress;
}>();

const emit = defineEmits<{
  refresh: [];
}>();

const settingsStore = useSettingsStore();
const savingsStore = useSavingsStore();
const sharedStore = useSharedStore();
const toastStore = useToastStore();
const confirm = useConfirm();
const { hasPermission } = usePermissions();

const loading = ref(false);
const contributions = ref<SavingContribution[]>([]);
const addingContrib = ref(false);

const rows = [5, 10, 25];
const paginator = ref<PaginatorState>({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: rows[0]!,
});
const page = ref(1);
const sort = ref(filterHelper.initSort("month"));

const apiPrefix = computed(() => `savings/${props.goal.id}/contributions`);
const params = computed(() => ({
  rowsPerPage: paginator.value.rowsPerPage,
  sort: sort.value,
}));

onMounted(async () => {
  await getData();
});

async function getData(new_page: number | null = null) {
  loading.value = true;
  if (new_page) page.value = new_page;
  try {
    const res = await sharedStore.getRecordsPaginated(
      apiPrefix.value,
      params.value,
      page.value,
    );
    contributions.value = res.data;
    paginator.value.total = res.total_records;
    paginator.value.from = res.from;
    paginator.value.to = res.to;
  } catch (err) {
    toastStore.errorResponseToast(err);
  } finally {
    loading.value = false;
  }
}

async function onPage(event: any) {
  paginator.value.rowsPerPage = event.rows;
  await getData(event.page + 1);
}

const contribForm = ref({
  amount: null as string | null,
  month: new Date() as Date | null,
  note: "" as string,
});

const contribAmountRef = computed({
  get: () => contribForm.value.amount,
  set: (v) => (contribForm.value.amount = v),
});
const { number: contribAmountNumber } = currencyHelper.useMoneyField(
  contribAmountRef,
  2,
);

const contribRules = computed(() => ({
  amount: { required, decimalValid, decimalMin: decimalMin("0.01") },
  month: { required },
}));
const cv$ = useVuelidate(contribRules, contribForm);

async function addContribution() {
  const valid = await cv$.value.$validate();
  if (!valid) return;

  addingContrib.value = true;
  try {
    const req: SavingContributionReq = {
      amount: new Decimal(contribForm.value.amount!).toFixed(2),
      month: dateHelper.formatDate(contribForm.value.month!),
      note: contribForm.value.note || null,
    };
    const res = await savingsStore.insertContribution(props.goal.id!, req);
    toastStore.successResponseToast(res);
    emit("refresh");
    contribForm.value = { amount: null, month: new Date(), note: "" };
    cv$.value.$reset();
    await getData();
  } catch (err) {
    toastStore.errorResponseToast(err);
  } finally {
    addingContrib.value = false;
  }
}

function confirmDeleteContrib(contrib: SavingContribution) {
  confirm.require({
    message: "Remove this contribution?",
    header: "Confirm",
    icon: "pi pi-exclamation-triangle",
    rejectProps: { label: "Cancel", severity: "secondary", outlined: true },
    acceptProps: { label: "Remove", severity: "danger" },
    accept: async () => {
      try {
        const res = await savingsStore.deleteContribution(
          props.goal.id!,
          contrib.id!,
        );
        toastStore.successResponseToast(res);
        emit("refresh");
        await getData();
      } catch (err) {
        toastStore.errorResponseToast(err);
      }
    },
  });
}
</script>

<template>
  <div class="flex flex-column gap-3 w-full">
    <div
      class="flex flex-column gap-2 p-3 border-round-xl w-full"
      style="background: var(--background-primary)"
    >
      <div class="flex flex-row justify-content-between align-items-center">
        <div class="text-sm" style="color: var(--text-secondary)">Progress</div>
        <Tag
          :value="savingsHelper.trackStatusLabel(goal?.track_status ?? '')"
          :severity="
            savingsHelper.trackStatusSeverity(goal?.track_status ?? '') as any
          "
        />
      </div>
      <ProgressBar
        :value="savingsHelper.progressPercent(goal)"
        style="height: 8px"
      />
      <div class="flex flex-row justify-content-between">
        <div class="text-sm">
          <span class="font-bold">{{
            vueHelper.displayAsCurrency(goal?.current_amount ?? null)
          }}</span>
          <span style="color: var(--text-secondary)"> saved</span>
        </div>
        <div class="text-sm" style="color: var(--text-secondary)">
          {{ vueHelper.displayAsCurrency(goal?.target_amount ?? null) }} target
        </div>
      </div>
      <div
        v-if="goal?.monthly_needed"
        class="text-sm"
        style="color: var(--text-secondary)"
      >
        {{ vueHelper.displayAsCurrency(goal.monthly_needed) }}/mo needed
        <span v-if="goal.months_remaining"
          >&middot; {{ goal.months_remaining }} months left</span
        >
      </div>
    </div>

    <div class="flex flex-column gap-2 w-full">
      <div class="flex flex-row w-full text-center align-items-center">
        <div class="font-medium text-sm">Add contribution</div>
        <Button
          size="small"
          icon="pi pi-plus"
          class="main-button ml-auto"
          :loading="addingContrib"
          @click="addContribution"
        />
      </div>
      <div class="flex flex-row gap-2 w-full">
        <div class="flex flex-column w-6">
          <InputNumber
            v-model="contribAmountNumber"
            fluid
            mode="currency"
            size="small"
            :currency="settingsStore.defaultCurrency"
            :locale="vueHelper.getCurrencyLocale(settingsStore.defaultCurrency)"
            :placeholder="vueHelper.displayAsCurrency(0) ?? '0.00'"
            :class="{ 'p-invalid': cv$.amount.$error }"
          />
          <ValidationError :state="cv$.amount" />
        </div>
        <div class="flex flex-column w-6">
          <DatePicker
            v-model="contribForm.month"
            size="small"
            fluid
            view="month"
            date-format="mm/yy"
            placeholder="Month"
            :class="{ 'p-invalid': cv$.month.$error }"
          />
          <ValidationError :state="cv$.month" />
        </div>
      </div>
      <div class="flex flex-row w-full">
        <InputText
          v-model="contribForm.note"
          placeholder="Note"
          class="w-full"
          size="small"
        />
      </div>
    </div>

    <div class="flex flex-column gap-2">
      <div class="font-medium text-sm">History</div>

      <ShowLoading v-if="loading" :num-fields="3" />

      <div
        v-else-if="contributions.length === 0"
        class="flex flex-row justify-content-center p-3"
        style="color: var(--text-secondary)"
      >
        <div class="flex flex-column align-items-center gap-2">
          <i class="pi pi-inbox text-3xl" />
          <span class="text-sm">No contributions yet</span>
        </div>
      </div>

      <template v-else>
        <div
          v-for="contrib in contributions"
          :key="contrib.id"
          class="flex flex-row align-items-center justify-content-between p-3 border-round-xl bordered"
          style="background: var(--background-primary)"
        >
          <div class="flex flex-column gap-1">
            <div class="font-medium text-sm">
              {{ vueHelper.displayAsCurrency(contrib.amount) }}
            </div>
            <div class="text-sm" style="color: var(--text-secondary)">
              {{ dateHelper.formatDate(contrib.month, false, "MMM YYYY") }}
              <span v-if="contrib.note"> &middot; {{ contrib.note }}</span>
            </div>
          </div>
          <div class="flex flex-row align-items-center gap-2">
            <Tag :value="contrib.source" severity="secondary" />
            <i
              v-if="hasPermission('manage_data')"
              class="pi pi-trash hover-icon text-sm"
              style="color: var(--p-red-300)"
              @click="confirmDeleteContrib(contrib)"
            />
          </div>
        </div>

        <CustomPaginator
          :paginator="paginator"
          :rows="rows"
          @on-page="onPage"
        />
      </template>
    </div>
  </div>
</template>

<style scoped></style>
