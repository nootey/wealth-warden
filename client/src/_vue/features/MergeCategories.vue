<script setup lang="ts">
import { computed, ref } from "vue";
import { useConfirm } from "primevue/useconfirm";
import ValidationError from "../components/validation/ValidationError.vue";
import { useTransactionStore } from "../../services/stores/transaction_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import type { Category } from "../../models/transaction_models.ts";
import UserJobsRunner from "./UserJobsRunner.vue";

const props = defineProps<{
  categories: Category[];
  kind: string;
}>();

const emit = defineEmits<{
  (e: "completeOperation"): void;
}>();

const transactionStore = useTransactionStore();
const toastStore = useToastStore();
const confirm = useConfirm();

const merging = ref(false);
const runner = ref<InstanceType<typeof UserJobsRunner> | null>(null);
const sourceCategory = ref<Category | null>(null);
const destinationCategory = ref<Category | null>(null);

const available = computed<Category[]>(() =>
  props.categories.filter((c) => !c.deleted_at),
);

const sourceOptions = computed<Category[]>(() =>
  available.value.filter(
    (c) =>
      c.id !== destinationCategory.value?.id &&
      (!destinationCategory.value ||
        c.classification === destinationCategory.value.classification),
  ),
);

const destinationOptions = computed<Category[]>(() =>
  available.value.filter(
    (c) =>
      c.id !== sourceCategory.value?.id &&
      (!sourceCategory.value ||
        c.classification === sourceCategory.value.classification),
  ),
);

function confirmMerge() {
  confirm.require({
    header: "Confirm category merge",
    message: `You are about to merge "${sourceCategory.value?.display_name}" into "${destinationCategory.value?.display_name}". All transactions will be moved to the destination category. This action is irreversible. Are you sure?`,
    rejectProps: { label: "Cancel" },
    acceptProps: { label: "Merge", severity: "danger" },
    accept: () => doMerge(),
  });
}

async function doMerge() {
  merging.value = true;
  try {
    const res = await transactionStore.mergeCategories(
      sourceCategory.value!.id!,
      destinationCategory.value!.id!,
    );
    toastStore.successResponseToast(res);
    sourceCategory.value = null;
    destinationCategory.value = null;
    await runner.value?.refresh();
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    merging.value = false;
  }
}
</script>

<template>
  <div class="w-full flex flex-col gap-2">
    <div class="flex flex-row justify-between items-center gap-4">
      <div class="w-full flex flex-col gap-2">
        <h3>Merge categories</h3>
        <span class="text-sm" style="color: var(--text-secondary)">
          Merge one category into another. All transactions will be reassigned
          to the destination category. Both categories must be of the same type.
        </span>
      </div>
    </div>

    <div class="flex flex-row gap-4 w-full items-center">
      <div class="flex flex-col gap-2 w-full">
        <ValidationError :is-required="true">
          <label>Source category</label>
        </ValidationError>
        <Select
          v-model="sourceCategory"
          :options="sourceOptions"
          filter
          option-label="display_name"
          placeholder="Select source"
          class="w-full"
          size="small"
        >
          <template #option="{ option }">
            <div class="flex justify-between w-full">
              <span>{{ option.display_name }}</span>
              <small class="text-muted-color">
                {{ option.classification }}
              </small>
            </div>
          </template>
        </Select>
      </div>
      <div class="flex flex-col gap-1 w-full">
        <ValidationError :is-required="true">
          <label>Destination category</label>
        </ValidationError>
        <Select
          v-model="destinationCategory"
          :options="destinationOptions"
          filter
          option-label="display_name"
          placeholder="Select destination"
          class="w-full"
          size="small"
        >
          <template #option="{ option }">
            <div class="flex justify-between w-full">
              <span>{{ option.display_name }}</span>
              <small class="text-muted-color">
                {{ option.classification }}
              </small>
            </div>
          </template>
        </Select>
      </div>
    </div>

    <div class="flex flex-row gap-2 w-full">
      <div id="expand" class="flex flex-col gap-2 ml-auto">
        <Button
          class="main-button"
          label="Merge"
          :disabled="!sourceCategory || !destinationCategory"
          :loading="merging"
          style="height: 42px"
          @click="confirmMerge"
        />
      </div>
    </div>

    <UserJobsRunner
      ref="runner"
      :kind="kind"
      label="Category merge jobs"
      @job-completed="emit('completeOperation')"
    />
  </div>
</template>

<style scoped>
@media (max-width: 768px) {
  #expand {
    width: 100% !important;
  }
}
</style>
