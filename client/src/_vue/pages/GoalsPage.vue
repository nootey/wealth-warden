<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { useSavingsStore } from "../../services/stores/savings_store.ts";
import { useAccountStore } from "../../services/stores/account_store.ts";
import { usePermissions } from "../../utils/use_permissions.ts";
import SavingGoalForm from "../components/forms/SavingGoalForm.vue";
import ShowLoading from "../components/base/ShowLoading.vue";
import vueHelper from "../../utils/vue_helper.ts";
import dateHelper from "../../utils/date_helper.ts";
import type { SavingGoalWithProgress } from "../../models/savings_models.ts";
import SavingGoalDetails from "../components/data/SavingGoalDetails.vue";
import savingsHelper from "../../utils/savings_helper.ts";
import Decimal from "decimal.js";

const toastStore = useToastStore();
const savingsStore = useSavingsStore();
const accountStore = useAccountStore();
const { hasPermission } = usePermissions();

const loading = ref(false);
const goals = ref<SavingGoalWithProgress[]>([]);

const createModal = ref(false);
const updateModal = ref(false);
const selectedGoalID = ref<number | null>(null);

const contribModal = ref(false);
const selectedGoal = ref<SavingGoalWithProgress | null>(null);

onMounted(async () => {
  await loadGoals();
  await accountStore.getAllAccountsWithBalance();
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

function openContributions(goal: SavingGoalWithProgress) {
  selectedGoal.value = goal;
  contribModal.value = true;
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

async function handleContribRefresh() {
  await loadGoals();
  selectedGoal.value =
    goals.value.find((g) => g.id === selectedGoal.value!.id) ??
    selectedGoal.value;
}

function accountName(accountID: number): string {
  return accountStore.accounts.find((a) => a.id === accountID)?.name ?? "—";
}

const groupedGoals = computed(() => {
  const map = new Map<
    number,
    {
      accountID: number;
      accountName: string;
      totalBalance: Decimal;
      allocated: Decimal;
      goals: SavingGoalWithProgress[];
    }
  >();

  for (const goal of goals.value) {
    if (!map.has(goal.account_id)) {
      const account = accountStore.accounts.find(
        (a) => a.id === goal.account_id,
      );
      const totalBalance = new Decimal(account?.balance?.end_balance ?? "0");
      map.set(goal.account_id, {
        accountID: goal.account_id,
        accountName: account?.name ?? "—",
        totalBalance,
        allocated: new Decimal(0),
        goals: [],
      });
    }
    const group = map.get(goal.account_id)!;
    group.allocated = group.allocated.add(
      new Decimal(goal.current_amount ?? "0"),
    );
    group.goals.push(goal);
  }

  return Array.from(map.values()).map((g) => ({
    ...g,
    uncategorized: Decimal.max(g.totalBalance.sub(g.allocated), new Decimal(0)),
    goals: [...g.goals].sort(
      (a, b) =>
        savingsHelper.goalSortOrder(a.status) -
        savingsHelper.goalSortOrder(b.status),
    ),
  }));
});
</script>

<template>
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
    <SavingGoalDetails :goal="selectedGoal!" @refresh="handleContribRefresh" />
  </Dialog>

  <main class="flex flex-column w-full align-items-center">
    <div
      id="mobile-container"
      class="flex flex-column justify-content-center w-full gap-3 border-round-xl"
    >
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

      <div
        class="w-full flex flex-row justify-content-between p-1 align-items-center text-sm"
      >
        <i class="pi pi-info-circle" style="flex-shrink: 0" />
        <div style="color: var(--text-secondary)">
          Goals with a monthly allocation are funded automatically each day.
          Ensure your account has enough uncategorized balance before the
          configured fund day.
        </div>
      </div>

      <div
        class="flex-1 w-full border-round-xl overflow-y-auto"
        :style="{ maxWidth: '1000px' }"
      >
        <template v-if="loading">
          <ShowLoading :num-fields="5" />
        </template>

        <div
          v-else-if="goals.length === 0"
          class="flex flex-row p-2 w-full justify-content-center"
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

        <div v-else class="flex flex-column gap-4">
          <div
            v-for="group in groupedGoals"
            :key="group.accountID"
            class="flex flex-column gap-2"
          >
            <div
              class="flex flex-row align-items-center justify-content-between gap-2 px-1"
            >
              <div class="font-bold">{{ group.accountName }}</div>
              <div
                class="flex flex-row gap-3 text-sm"
                style="color: var(--text-secondary)"
              >
                <span>
                  <span
                    class="font-medium"
                    style="color: var(--text-primary)"
                    >{{
                      vueHelper.displayAsCurrency(group.allocated.toString())
                    }}</span
                  >
                  allocated
                </span>
                <span>
                  <span
                    class="font-medium"
                    style="color: var(--text-primary)"
                    >{{
                      vueHelper.displayAsCurrency(
                        group.uncategorized.toString(),
                      )
                    }}</span
                  >
                  free
                </span>
              </div>
            </div>

            <div
              v-for="goal in group.goals"
              :key="goal.id"
              class="flex flex-column p-3 gap-3 border-round-xl bordered"
              :style="{
                background: 'var(--background-secondary)',
                opacity: savingsHelper.isGoalDimmed(goal.status) ? '0.55' : '1',
              }"
            >
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
                      cursor: pointer;
                    "
                    @click="openContributions(goal)"
                  >
                    {{ goal.name }}
                  </div>
                </div>
                <div class="flex flex-row align-items-center gap-2">
                  <Tag
                    v-if="goal.status !== 'active'"
                    :value="savingsHelper.goalStatusLabel(goal.status)"
                    :severity="
                      savingsHelper.goalStatusSeverity(goal.status) as any
                    "
                  />
                  <Tag
                    v-else
                    :value="savingsHelper.trackStatusLabel(goal.track_status)"
                    :severity="
                      savingsHelper.trackStatusSeverity(
                        goal.track_status,
                      ) as any
                    "
                  />
                  <i
                    v-if="hasPermission('manage_data')"
                    class="pi pi-pencil hover-icon text-sm"
                    style="color: var(--text-secondary)"
                    @click="openUpdate(goal)"
                  />
                </div>
              </div>

              <ProgressBar
                :value="savingsHelper.progressPercent(goal)"
                style="height: 14px"
                :pt="{ label: { style: 'color: white' } }"
              />

              <div
                class="flex flex-row justify-content-between align-items-center"
              >
                <div class="text-sm">
                  <span class="font-bold">{{
                    vueHelper.displayAsCurrency(goal.current_amount)
                  }}</span>
                  <span style="color: var(--text-secondary)">
                    /
                    {{ vueHelper.displayAsCurrency(goal.target_amount) }}</span
                  >
                </div>
                <div class="text-sm" style="color: var(--text-secondary)">
                  <span v-if="goal.target_date">{{
                    dateHelper.formatDate(goal.target_date, false, "MMM YYYY")
                  }}</span>
                  <span v-else>No deadline</span>
                </div>
              </div>

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
        </div>
      </div>
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
