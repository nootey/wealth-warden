<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import UserForm from "../components/forms/UserForm.vue";
import InvitationsPaginated from "../components/data/InvitationsPaginated.vue";
import UsersPaginated from "../components/data/UsersPaginated.vue";
import type { Role } from "../../models/user_models.ts";
import { useUserStore } from "../../services/stores/user_store.ts";
import { usePermissions } from "../../utils/use_permissions.ts";
import { useRouter } from "vue-router";

const userStore = useUserStore();

const { hasPermission } = usePermissions();
const router = useRouter();

const createModal = ref(false);
const updateModal = ref(false);
const updateUserID = ref(null);

const usrRef = ref<InstanceType<typeof UsersPaginated> | null>(null);
const invRef = ref<InstanceType<typeof InvitationsPaginated> | null>(null);

const roles = computed<Role[]>(() => userStore.roles);
const activeTab = ref("users");

onMounted(async () => {
  await userStore.getRoles();
});

function manipulateDialog(modal: string, value: any) {
  switch (modal) {
    case "inviteUser": {
      createModal.value = value;
      break;
    }
    case "updateUser": {
      updateModal.value = true;
      updateUserID.value = value;
      break;
    }
    default: {
      break;
    }
  }
}

async function handleEmit(emitType: any) {
  switch (emitType) {
    case "completeOperation": {
      createModal.value = false;
      updateModal.value = false;
      usrRef.value?.refresh();
      break;
    }
    case "deleteUser": {
      createModal.value = false;
      updateModal.value = false;
      usrRef.value?.refresh();
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
    header="Invite user"
  >
    <UserForm
      mode="create"
      :roles="roles"
      @complete-operation="handleEmit('completeOperation')"
    />
  </Dialog>

  <Dialog
    v-model:visible="updateModal"
    position="right"
    class="rounded-dialog"
    :breakpoints="{ '501px': '90vw' }"
    :modal="true"
    :style="{ width: '500px' }"
    header="User details"
  >
    <UserForm
      mode="update"
      :roles="roles"
      :record-id="updateUserID"
      @complete-operation="handleEmit('completeOperation')"
      @complete-user-delete="handleEmit('deleteUser')"
    />
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
            <div style="font-weight: bold">Management</div>
            <i
              v-if="hasPermission('manage_roles')"
              v-tooltip="'Go to roles settings.'"
              class="pi pi-external-link hover-icon mr-auto text-sm"
              @click="router.push('settings/roles')"
            />
          </div>
          <div>View and manage users and invitations.</div>
        </div>
        <Button
          class="main-button"
          @click="manipulateDialog('inviteUser', true)"
        >
          <div class="flex flex-row gap-1 align-items-center">
            <i class="pi pi-plus" />
            <span> New </span>
            <span class="mobile-hide"> Invitation </span>
          </div>
        </Button>
      </div>

      <div class="flex flex-row gap-3 p-2">
        <div
          class="cursor-pointer pb-1"
          style="color: var(--text-secondary)"
          :style="
            activeTab === 'users'
              ? 'color: var(--text-primary); border-bottom: 2px solid var(--text-primary)'
              : ''
          "
          @click="activeTab = 'users'"
        >
          Users
        </div>
        <div
          class="cursor-pointer pb-1"
          style="color: var(--text-secondary)"
          :style="
            activeTab === 'invitations'
              ? 'color: var(--text-primary); border-bottom: 2px solid var(--text-primary)'
              : ''
          "
          @click="activeTab = 'invitations'"
        >
          Invitations
        </div>
      </div>

      <Transition name="fade" mode="out-in">
        <div
          v-if="activeTab === 'users'"
          key="users"
          class="flex flex-column justify-content-center w-full gap-3"
        >
          <Panel :collapsed="false" header="Users">
            <div id="mobile-row" class="flex flex-row w-full">
              <UsersPaginated
                ref="usrRef"
                :roles="roles"
                @update-user="(id) => manipulateDialog('updateUser', id)"
              />
            </div>
          </Panel>
        </div>
        <div v-else key="invitations" class="w-full">
          <Panel :collapsed="false" header="Invitations">
            <div class="flex flex-row gap-2 w-full">
              <InvitationsPaginated ref="invRef" />
            </div>
          </Panel>
        </div>
      </Transition>
    </div>
  </main>
</template>

<style scoped></style>
