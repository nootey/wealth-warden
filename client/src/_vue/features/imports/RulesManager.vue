<script setup lang="ts">
import { computed, ref } from "vue";
import { useRulesStore } from "../../../services/stores/rules_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { usePermissions } from "../../../utils/use_permissions.ts";
import RulesDisplay from "../../components/data/RulesDisplay.vue";
import RuleForm from "../../components/forms/RuleForm.vue";
import type { Rule } from "../../../models/rules_models.ts";

const emit = defineEmits<{
  (e: "changed"): void;
}>();

const rulesStore = useRulesStore();
const toastStore = useToastStore();
const { hasPermission } = usePermissions();

const manageModal = ref(false);
const createModal = ref(false);

const rules = computed<Rule[]>(() => rulesStore.rules);

async function getRules() {
  try {
    await rulesStore.getRules();
  } catch (e) {
    toastStore.errorResponseToast(e);
  }
}

async function openManage() {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }
  manageModal.value = true;
  await getRules();
}

async function onCreated() {
  createModal.value = false;
  await getRules();
  emit("changed");
}

async function onUpdated() {
  await getRules();
  emit("changed");
}
</script>

<template>
  <Button class="outline-button" @click="openManage">
    <div class="flex flex-row gap-1 items-center">
      <i class="pi pi-sliders-h" />
      <span> Manage rules </span>
    </div>
  </Button>

  <Dialog
    v-model:visible="manageModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '700px' }"
    header="Manage rules"
  >
    <div class="flex flex-col w-full gap-4">
      <div class="flex flex-row items-center justify-between gap-2">
        <span class="text-sm" style="color: var(--text-secondary)">
          Add or edit rules, then close to refresh the guessed categories.
        </span>
        <Button class="main-button shrink-0" @click="createModal = true">
          <div class="flex flex-row gap-1 items-center">
            <i class="pi pi-plus" />
            <span> New rule </span>
          </div>
        </Button>
      </div>

      <RulesDisplay
        :rules="rules"
        @complete-operation="onUpdated"
        @complete-delete="onUpdated"
      />
    </div>
  </Dialog>

  <Dialog
    v-model:visible="createModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Create rule"
  >
    <RuleForm mode="create" @complete-operation="onCreated" />
  </Dialog>
</template>

<style scoped></style>
