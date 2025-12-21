<script setup lang="ts">
import { useSharedStore } from "../../services/stores/shared_store.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { usePermissions } from "../../utils/use_permissions.ts";

const sharedStore = useSharedStore();
const toastStore = useToastStore();
const { hasPermission } = usePermissions();


const router = useRouter();

const createModal = ref(false);
const updateModal = ref(false);
const updateRecordID = ref(null);

function manipulateDialog(modal: string, value: any) {
  switch (modal) {
    default: {
      break;
    }
  }
}

async function handleEmit(emitType: any) {
  switch (emitType) {
    default: {
      break;
    }
  }
}

</script>

<template>
  <main class="flex flex-column w-full p-2 align-items-center">
    <div
      id="mobile-container"
      class="flex flex-column justify-content-center p-3 w-full gap-3 border-round-md"
      style="
        border: 1px solid var(--border-color);
        background: var(--background-secondary);
      "
    >
      <div
        class="flex flex-row justify-content-between align-items-center text-center gap-2 w-full"
      >
        <div style="font-weight: bold">Investments</div>
        <Button
          class="main-button"
          @click="manipulateDialog('inviteUser', true)"
        >
          <div class="flex flex-row gap-1 align-items-center">
            <i class="pi pi-plus" />
            <span> New </span>
            <span class="mobile-hide"> Holding </span>
          </div>
        </Button>
      </div>
    </div>
  </main>
</template>

<style scoped></style>
