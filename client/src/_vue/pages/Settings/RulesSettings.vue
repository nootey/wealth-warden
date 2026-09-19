<script setup lang="ts">
import SettingsSkeleton from "../../components/layout/SettingsSkeleton.vue";
import { useRulesStore } from "../../../services/stores/rules_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { computed, onMounted, ref } from "vue";
import type { Rule } from "../../../models/rules_models.ts";
import RulesDisplay from "../../components/data/RulesDisplay.vue";
import RuleForm from "../../components/forms/RuleForm.vue";
import { usePermissions } from "../../../utils/use_permissions.ts";

const rulesStore = useRulesStore();
const toastStore = useToastStore();
const { hasPermission } = usePermissions();

onMounted(async () => {
  await getRules();
});

const createModal = ref(false);
const applying = ref(false);

const rules = computed<Rule[]>(() => rulesStore.rules);

async function getRules() {
  await rulesStore.getRules();
}

async function applyRules() {
  if (!hasPermission("manage_data")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }

  applying.value = true;
  try {
    const res = await rulesStore.applyRules();
    toastStore.successResponseToast(res);
  } catch (e) {
    toastStore.errorResponseToast(e);
  } finally {
    applying.value = false;
  }
}

async function handleEmit(type: string) {
  switch (type) {
    case "completeOperation": {
      createModal.value = false;
      await getRules();
      break;
    }
    case "completeDelete": {
      await getRules();
      break;
    }
    case "openRuleCreate": {
      if (!hasPermission("manage_data")) {
        toastStore.createInfoToast(
          "Access denied",
          "You don't have permission to perform this action.",
        );
        return;
      }

      createModal.value = true;
      break;
    }
    default: {
      break;
    }
  }
}
</script>

<template>
  <Dialog
    v-model:visible="createModal"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="Create rule"
  >
    <RuleForm
      mode="create"
      @complete-operation="handleEmit('completeOperation')"
    />
  </Dialog>

  <div class="flex flex-col w-full gap-4">
    <SettingsSkeleton class="w-full">
      <div class="w-full flex flex-col gap-4 p-2">
        <div class="flex flex-row justify-between items-center gap-4">
          <div class="w-full flex flex-col gap-2">
            <h3>Rules</h3>
            <h5 class="mobile-hide" style="color: var(--text-secondary)">
              View and manage rules for imported transactions.
            </h5>
          </div>

          <div class="flex flex-row gap-2 shrink-0">
            <Button
              class="outline-button"
              :disabled="applying"
              @click="applyRules"
            >
              <div class="flex flex-row gap-1 items-center">
                <i :class="applying ? 'pi pi-spinner pi-spin' : 'pi pi-bolt'" />
                <span class="mobile-hide"> Apply rules </span>
              </div>
            </Button>

            <Button class="main-button" @click="handleEmit('openRuleCreate')">
              <div class="flex flex-row gap-1 items-center">
                <i class="pi pi-plus" />
                <span class="mobile-hide"> New rule </span>
              </div>
            </Button>
          </div>
        </div>

        <div v-if="rules" class="w-full flex flex-col gap-2">
          <RulesDisplay
            :rules="rules"
            @complete-operation="handleEmit('completeOperation')"
            @complete-delete="handleEmit('completeDelete')"
          />
        </div>
      </div>
    </SettingsSkeleton>
  </div>
</template>

<style scoped></style>
