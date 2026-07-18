<script setup lang="ts">
import LoadingSpinner from "../base/LoadingSpinner.vue";
import ColumnHeader from "../base/ColumnHeader.vue";
import vueHelper from "../../../utils/vue_helper.ts";
import savingsHelper from "../../../utils/savings_helper.ts";
import { useConfirm } from "primevue/useconfirm";
import { usePermissions } from "../../../utils/use_permissions.ts";
import type {
  SavingContribution,
  SavingGoalWithProgress,
} from "../../../models/savings_models.ts";
import { useSavingsStore } from "../../../services/stores/savings_store.ts";
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import dateHelper from "../../../utils/date_helper.ts";
import dayjs from "dayjs";
import Decimal from "decimal.js";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { computed, onMounted, provide, ref } from "vue";
import filterHelper from "../../../utils/filter_helper.ts";
import type { PaginatorState } from "../../../models/shared_models.ts";
import type { Column } from "../../../services/filter_registry.ts";
import CustomPaginator from "../base/CustomPaginator.vue";

const props = defineProps<{
  goal: SavingGoalWithProgress;
}>();

const emit = defineEmits<{
  refresh: [];
}>();

const savingsStore = useSavingsStore();
const sharedStore = useSharedStore();
const toastStore = useToastStore();
const confirm = useConfirm();
const { hasPermission } = usePermissions();

const loading = ref(false);
const contributions = ref<SavingContribution[]>([]);

const rows = [5, 10, 25];
const paginator = ref<PaginatorState>({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: rows[0]!,
});
const page = ref(1);
const sort = ref(filterHelper.initSort("month"));

const columns: Column[] = [
  { field: "amount", header: "Amount", type: "number" },
  { field: "month", header: "Month", type: "date" },
  { field: "source", header: "Source" },
  { field: "note", header: "Note", hideOnMobile: true },
];

const remaining = computed(() =>
  Decimal.max(
    new Decimal(props.goal.target_amount ?? "0").sub(
      new Decimal(props.goal.current_amount ?? "0"),
    ),
    new Decimal(0),
  ),
);

const monthlyPace = computed(() => {
  const months = Math.max(
    dayjs().diff(dayjs(props.goal.created_at), "month"),
    1,
  );
  return new Decimal(props.goal.current_amount ?? "0").div(months);
});

const projectedFinish = computed(() => {
  if (remaining.value.lte(0) || monthlyPace.value.lte(0)) return null;
  const months = remaining.value.div(monthlyPace.value).ceil().toNumber();
  return dayjs().add(months, "month").format("MMM YYYY");
});

const allocationShortfall = computed(() => {
  if (props.goal.status !== "active" || props.goal.track_status === "completed")
    return false;
  if (!props.goal.monthly_allocation || !props.goal.monthly_needed)
    return false;
  return new Decimal(props.goal.monthly_allocation).lt(
    new Decimal(props.goal.monthly_needed),
  );
});

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

async function switchSort(column: string) {
  if (sort.value.field === column) {
    sort.value.order = filterHelper.toggleSort(sort.value.order);
  } else {
    sort.value.order = 1;
  }
  sort.value.field = column;
  await getData();
}

provide("switchSort", switchSort);

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
        :pt="{ label: { style: 'color: white' } }"
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

    <div
      v-if="remaining.gt(0)"
      class="flex flex-column gap-2 p-3 border-round-xl w-full"
      style="background: var(--background-primary)"
    >
      <div class="text-sm" style="color: var(--text-secondary)">Insights</div>
      <div class="text-sm">
        {{ vueHelper.displayAsCurrency(remaining.toString()) }} to go
        <span v-if="monthlyPace.gt(0)">
          &middot; averaging
          {{ vueHelper.displayAsCurrency(monthlyPace.toFixed(2)) }}/mo since
          {{ dateHelper.formatDate(goal.created_at, false, "MMM YYYY") }}
        </span>
        <span v-if="projectedFinish">
          &middot; at this pace, done around {{ projectedFinish }}
        </span>
      </div>
    </div>

    <div
      v-if="allocationShortfall"
      class="flex flex-column gap-2 p-3 border-round-xl text-sm"
      style="
        background: var(--background-primary);
        border: 1px solid var(--border-color);
        color: var(--text-secondary);
      "
    >
      <div class="flex flex-row gap-2 align-items-center">
        <i class="pi pi-exclamation-triangle" style="flex-shrink: 0" />
        <div class="flex flex-column gap-1 text-xs">
          <span
            >Your {{ vueHelper.displayAsCurrency(goal.monthly_allocation!) }}/mo
            allocation is below the
            {{ vueHelper.displayAsCurrency(goal.monthly_needed!) }}/mo needed to
            reach the target on time.</span
          >
        </div>
      </div>
    </div>

    <div class="flex flex-column gap-2">
      <div class="flex flex-row align-items-center justify-content-between">
        <div class="font-medium text-sm">History</div>
        <div
          v-if="paginator.total > 0"
          class="text-sm"
          style="color: var(--text-secondary)"
        >
          {{ paginator.total }}
          {{ paginator.total === 1 ? "contribution" : "contributions" }}
        </div>
      </div>

      <div
        class="flex flex-column w-full border-round-2xl"
        style="
          padding: 0.25rem 0.25rem 0 0.25rem;
          border: 1px solid var(--border-color);
        "
      >
        <DataTable
          data-key="id"
          class="w-full enhanced-table"
          :loading="loading"
          :value="contributions"
          scrollable
          scroll-height="30vh"
          column-resize-mode="fit"
          scroll-direction="both"
        >
          <template #empty>
            <div style="padding: 10px">No contributions yet.</div>
          </template>
          <template #loading>
            <LoadingSpinner />
          </template>
          <template #footer>
            <CustomPaginator
              :paginator="paginator"
              :rows="rows"
              @on-page="onPage"
            />
          </template>

          <Column
            v-for="col of columns"
            :key="col.field"
            :field="col.field"
            :header-class="col.hideOnMobile ? 'mobile-hide ' : ''"
            :body-class="col.hideOnMobile ? 'mobile-hide ' : ''"
          >
            <template #header>
              <ColumnHeader
                :header="col.header"
                :field="col.field"
                :sort="sort"
                :sortable="true"
              />
            </template>
            <template #body="{ data }">
              <template v-if="col.field === 'amount'">
                {{ vueHelper.displayAsCurrency(data.amount) }}
              </template>
              <template v-else-if="col.field === 'month'">
                {{ dateHelper.formatDate(data.month, false, "MMM YYYY") }}
              </template>
              <template v-else-if="col.field === 'source'">
                <Tag :value="data.source" severity="secondary" />
              </template>
              <template v-else-if="col.field === 'note'">
                <span v-tooltip.top="data.note" class="truncate-text">
                  {{ data.note ?? "" }}
                </span>
              </template>
              <template v-else>
                {{ data[col.field] ?? "" }}
              </template>
            </template>
          </Column>

          <Column v-if="hasPermission('manage_data')" header="">
            <template #body="{ data }">
              <i
                class="pi pi-trash hover-icon text-sm"
                style="color: var(--p-red-300)"
                @click="confirmDeleteContrib(data)"
              />
            </template>
          </Column>
        </DataTable>
      </div>
    </div>
  </div>
</template>

<style scoped></style>
