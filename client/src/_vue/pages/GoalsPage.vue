<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { useSavingsStore } from "../../services/stores/savings_store.ts";
import { useAccountStore } from "../../services/stores/account_store.ts";
import { useSettingsStore } from "../../services/stores/settings_store.ts";
import { usePermissions } from "../../utils/use_permissions.ts";
import { useConfirm } from "primevue/useconfirm";
import SavingGoalForm from "../components/forms/SavingGoalForm.vue";
import ShowLoading from "../components/base/ShowLoading.vue";
import vueHelper from "../../utils/vue_helper.ts";
import currencyHelper from "../../utils/currency_helper.ts";
import dateHelper from "../../utils/date_helper.ts";
import { required } from "@vuelidate/validators";
import { decimalMin, decimalValid } from "../../validators/currency.ts";
import useVuelidate from "@vuelidate/core";
import ValidationError from "../components/validation/ValidationError.vue";
import type {
  SavingGoalWithProgress,
  SavingContribution,
  SavingContributionReq,
} from "../../models/savings_models.ts";
import Decimal from "decimal.js";

const toastStore = useToastStore();
const savingsStore = useSavingsStore();
const accountStore = useAccountStore();
const settingsStore = useSettingsStore();
const confirm = useConfirm();
const { hasPermission } = usePermissions();

const loading = ref(false);
const goals = ref<SavingGoalWithProgress[]>([]);

const createModal = ref(false);
const updateModal = ref(false);
const selectedGoalID = ref<number | null>(null);

const contribModal = ref(false);
const selectedGoal = ref<SavingGoalWithProgress | null>(null);
const contributions = ref<SavingContribution[]>([]);
const contribLoading = ref(false);

const addingContrib = ref(false);
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

onMounted(async () => {
  await loadGoals();
  await accountStore.getAllAccounts(false, true);
});

async function loadGoals() {
  loading.value = true;
  try {
    goals.value = await savingsStore.fetchGoals();
  } catch (err) {
    toastStore.errorResponseToast(err);
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }
  createModal.value = true;
}

function openUpdate(goal: SavingGoalWithProgress) {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }
  selectedGoalID.value = goal.id!;
  updateModal.value = true;
}

async function openContributions(goal: SavingGoalWithProgress) {
  selectedGoal.value = goal;
  contribModal.value = true;
  contribLoading.value = true;
  try {
    contributions.value = await savingsStore.fetchContributions(goal.id!);
  } catch (err) {
    toastStore.errorResponseToast(err);
  } finally {
    contribLoading.value = false;
  }
}

async function handleGoalCreated() {
  createModal.value = false;
  await loadGoals();
}

async function handleGoalUpdated() {
  updateModal.value = false;
  await loadGoals();
}

async function handleGoalDeleted() {
  updateModal.value = false;
  contribModal.value = false;
  await loadGoals();
}

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
    const res = await savingsStore.insertContribution(
      selectedGoal.value!.id!,
      req,
    );
    toastStore.successResponseToast(res);
    contributions.value = await savingsStore.fetchContributions(
      selectedGoal.value!.id!,
    );
    await loadGoals();
    selectedGoal.value =
      goals.value.find((g) => g.id === selectedGoal.value!.id) ??
      selectedGoal.value;
    contribForm.value = { amount: null, month: new Date(), note: "" };
    cv$.value.$reset();
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
          selectedGoal.value!.id!,
          contrib.id!,
        );
        contributions.value = await savingsStore.fetchContributions(
          selectedGoal.value!.id!,
        );
        await loadGoals();
        selectedGoal.value =
          goals.value.find((g) => g.id === selectedGoal.value!.id) ??
          selectedGoal.value;
        toastStore.successResponseToast(res);
      } catch (err) {
        toastStore.errorResponseToast(err);
      }
    },
  });
}

function progressPercent(goal: SavingGoalWithProgress): number {
  const p = Number(goal.progress_percent);
  return isNaN(p) ? 0 : Math.min(p, 100);
}

function trackStatusLabel(status: string): string {
  const map: Record<string, string> = {
    on_track: "On track",
    early: "Ahead",
    late: "Behind",
    completed: "Completed",
    no_target: "No target",
  };
  return map[status] ?? status;
}

function trackStatusSeverity(status: string): string {
  const map: Record<string, string> = {
    on_track: "success",
    early: "info",
    late: "warn",
    completed: "success",
    no_target: "secondary",
  };
  return map[status] ?? "secondary";
}

function accountName(accountID: number): string {
  return accountStore.accounts.find((a) => a.id === accountID)?.name ?? "—";
}
</script>

<template>
  <ConfirmDialog />

  <Dialog
    v-model:visible="createModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '460px' }"
    header="New goal"
    position="right"
  >
    <SavingGoalForm mode="create" @complete-operation="handleGoalCreated" />
  </Dialog>

  <!-- Update goal dialog -->
  <Dialog
    v-model:visible="updateModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '460px' }"
    header="Edit goal"
    position="right"
  >
    <SavingGoalForm
      mode="update"
      :record-id="selectedGoalID"
      @complete-operation="handleGoalUpdated"
      @complete-delete="handleGoalDeleted"
    />
  </Dialog>

  <!-- Contributions dialog -->
  <Dialog
    v-model:visible="contribModal"
    class="rounded-dialog"
    :breakpoints="{ '851px': '90vw' }"
    :modal="true"
    :style="{ width: '600px' }"
    position="top"
  >
    <template #header>
      <div class="flex flex-column gap-1">
        <div class="font-bold">{{ selectedGoal?.name }}</div>
        <div class="text-sm" style="color: var(--text-secondary)">
          {{ accountName(selectedGoal?.account_id!) }}
        </div>
      </div>
    </template>

    <div class="flex flex-column gap-4">
      <!-- Progress summary -->
      <div
        class="flex flex-column gap-2 p-3 border-round-xl bordered"
        style="background: var(--background-primary)"
      >
        <div class="flex flex-row justify-content-between align-items-center">
          <div class="text-sm" style="color: var(--text-secondary)">
            Progress
          </div>
          <Tag
            :value="trackStatusLabel(selectedGoal?.track_status ?? '')"
            :severity="
              trackStatusSeverity(selectedGoal?.track_status ?? '') as any
            "
          />
        </div>
        <ProgressBar
          :value="progressPercent(selectedGoal!)"
          style="height: 8px"
        />
        <div class="flex flex-row justify-content-between">
          <div class="text-sm">
            <span class="font-bold">{{
              vueHelper.displayAsCurrency(selectedGoal?.current_amount ?? null)
            }}</span>
            <span style="color: var(--text-secondary)"> saved</span>
          </div>
          <div class="text-sm" style="color: var(--text-secondary)">
            {{
              vueHelper.displayAsCurrency(selectedGoal?.target_amount ?? null)
            }}
            target
          </div>
        </div>
        <div
          v-if="selectedGoal?.monthly_needed"
          class="text-sm"
          style="color: var(--text-secondary)"
        >
          {{ vueHelper.displayAsCurrency(selectedGoal.monthly_needed) }}/mo
          needed
          <span v-if="selectedGoal.months_remaining">
            &middot; {{ selectedGoal.months_remaining }} months left</span
          >
        </div>
      </div>

      <!-- Add contribution -->
      <div v-if="hasPermission('manage_data')" class="flex flex-column gap-2">
        <div class="font-medium text-sm">Add contribution</div>
        <div class="flex flex-row gap-2 align-items-start">
          <div class="flex flex-column gap-1 flex-1">
            <InputNumber
              v-model="contribAmountNumber"
              mode="currency"
              :currency="settingsStore.defaultCurrency"
              :locale="
                vueHelper.getCurrencyLocale(settingsStore.defaultCurrency)
              "
              :placeholder="vueHelper.displayAsCurrency(0) ?? '0.00'"
              class="w-full"
              :class="{ 'p-invalid': cv$.amount.$error }"
            />
            <ValidationError :state="cv$.amount" />
          </div>
          <div class="flex flex-column gap-1">
            <DatePicker
              v-model="contribForm.month"
              view="month"
              date-format="mm/yy"
              placeholder="Month"
              :class="{ 'p-invalid': cv$.month.$error }"
            />
            <ValidationError :state="cv$.month" />
          </div>
          <Button
            icon="pi pi-plus"
            class="main-button"
            :loading="addingContrib"
            @click="addContribution"
          />
        </div>
        <InputText
          v-model="contribForm.note"
          placeholder="Note (optional)"
          class="w-full"
        />
      </div>

      <!-- Contributions list -->
      <div class="flex flex-column gap-1">
        <div class="font-medium text-sm">History</div>

        <ShowLoading v-if="contribLoading" :num-fields="3" />

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

        <div
          v-for="contrib in contributions"
          v-else
          :key="contrib.id"
          class="flex flex-row align-items-center justify-content-between p-2 border-round-xl bordered"
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
              style="color: var(--text-secondary)"
              @click="confirmDeleteContrib(contrib)"
            />
          </div>
        </div>
      </div>
    </div>
  </Dialog>

  <main class="flex flex-column w-full align-items-center">
    <div
      id="mobile-container"
      class="flex flex-column justify-content-center w-full gap-3 border-round-xl"
    >
      <!-- Header row -->
      <div
        class="w-full flex flex-row justify-content-between p-1 gap-2 align-items-center"
      >
        <div class="w-full flex flex-column gap-2">
          <div class="flex flex-row gap-2 align-items-center w-full">
            <div class="font-bold">Goals</div>
          </div>
          <div>Scope your savings goals and plan ahead.</div>
        </div>
        <Button class="main-button" @click="openCreate">
          <div class="flex flex-row gap-1 align-items-center">
            <i class="pi pi-plus" />
            <span> New </span>
            <span class="mobile-hide"> Goal </span>
          </div>
        </Button>
      </div>

      <!-- Goals panel -->
      <Panel :collapsed="false" header="Savings goals">
        <ShowLoading v-if="loading" :num-fields="4" />

        <div
          v-else-if="goals.length === 0"
          class="flex flex-row p-4 w-full justify-content-center"
        >
          <div
            class="flex flex-column gap-2 justify-content-center align-items-center"
          >
            <i
              style="color: var(--text-secondary)"
              class="pi pi-flag text-4xl"
            />
            <span>No goals yet - create one to get started</span>
          </div>
        </div>

        <div v-else class="flex flex-column gap-2">
          <div
            v-for="goal in goals"
            :key="goal.id"
            class="flex flex-column gap-2 p-3 border-round-xl bordered"
            style="background: var(--background-primary); cursor: pointer"
            @click="openContributions(goal)"
          >
            <!-- Top row: name + badges + edit -->
            <div
              class="flex flex-row align-items-center justify-content-between gap-2"
            >
              <div
                class="flex flex-row align-items-center gap-2 flex-1 min-w-0"
              >
                <div
                  class="font-bold"
                  style="
                    white-space: nowrap;
                    overflow: hidden;
                    text-overflow: ellipsis;
                  "
                >
                  {{ goal.name }}
                </div>
                <div
                  class="text-sm"
                  style="color: var(--text-secondary); white-space: nowrap"
                >
                  {{ accountName(goal.account_id) }}
                </div>
              </div>
              <div class="flex flex-row align-items-center gap-2" @click.stop>
                <Tag
                  :value="trackStatusLabel(goal.track_status)"
                  :severity="trackStatusSeverity(goal.track_status) as any"
                />
                <i
                  v-if="hasPermission('manage_data')"
                  class="pi pi-pencil hover-icon text-sm"
                  style="color: var(--text-secondary)"
                  @click="openUpdate(goal)"
                />
              </div>
            </div>

            <!-- Progress bar -->
            <ProgressBar :value="progressPercent(goal)" style="height: 6px" />

            <!-- Bottom row: amounts + target date -->
            <div
              class="flex flex-row justify-content-between align-items-center"
            >
              <div class="text-sm">
                <span class="font-bold">{{
                  vueHelper.displayAsCurrency(goal.current_amount)
                }}</span>
                <span style="color: var(--text-secondary)">
                  / {{ vueHelper.displayAsCurrency(goal.target_amount) }}</span
                >
              </div>
              <div class="text-sm" style="color: var(--text-secondary)">
                <span v-if="goal.target_date">
                  {{
                    dateHelper.formatDate(goal.target_date, false, "MMM YYYY")
                  }}
                </span>
                <span v-else>No deadline</span>
              </div>
            </div>

            <!-- Monthly needed (if behind or on track with target) -->
            <div
              v-if="goal.monthly_needed && goal.track_status !== 'completed'"
              class="text-sm"
              style="color: var(--text-secondary)"
            >
              {{ vueHelper.displayAsCurrency(goal.monthly_needed) }}/mo
              <span v-if="goal.months_remaining"
                >&middot; {{ goal.months_remaining }} months left</span
              >
            </div>
          </div>
        </div>
      </Panel>
    </div>
  </main>
</template>

<style scoped lang="scss">
@media (max-width: 768px) {
  #mobile-container {
    padding: 0.5rem;
  }
}
</style>
