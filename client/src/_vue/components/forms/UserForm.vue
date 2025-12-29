<script setup lang="ts">
import { useSharedStore } from "../../../services/stores/shared_store.ts";
import { useToastStore } from "../../../services/stores/toast_store.ts";
import { nextTick, onMounted, ref } from "vue";
import type { Role, User } from "../../../models/user_models.ts";
import { email, required, requiredIf } from "@vuelidate/validators";
import useVuelidate from "@vuelidate/core";
import { useUserStore } from "../../../services/stores/user_store.ts";
import ShowLoading from "../base/ShowLoading.vue";
import ValidationError from "../validation/ValidationError.vue";
import { usePermissions } from "../../../utils/use_permissions.ts";
import { useConfirm } from "primevue/useconfirm";
import AuditTrail from "../base/AuditTrail.vue";

const props = defineProps<{
  mode?: "create" | "update";
  recordId?: number | null;
  roles: Role[];
}>();
const emit = defineEmits<{
  (event: "completeOperation"): void;
  (event: "completeUserDelete"): void;
}>();

const sharedStore = useSharedStore();
const userStore = useUserStore();
const toastStore = useToastStore();

const { hasPermission } = usePermissions();
const confirm = useConfirm();

onMounted(async () => {
  if (props.mode === "update" && props.recordId) {
    await loadRecord(props.recordId);
  }
});

const readOnly = ref(false);
const loading = ref(false);

const record = ref<User>(initData());
const filteredRoles = ref<Role[]>([]);

const rules = {
  record: {
    role: {
      name: {
        required,
        $autoDirty: true,
      },
    },
    display_name: {
      required: requiredIf(() => props.mode === "update"),
      $autoDirty: true,
    },
    email: {
      required,
      email,
      $autoDirty: true,
    },
  },
};

const v$ = useVuelidate(rules, { record });

function initData(): User {
  return {
    role: {
      name: "",
      is_default: false,
    },
    display_name: "",
    email: "",
    deleted_at: null,
  };
}

const searchRole = (event: { query: string }) => {
  setTimeout(() => {
    if (!event.query.trim().length) {
      filteredRoles.value = [...props.roles];
    } else {
      filteredRoles.value = props.roles.filter((record) => {
        return record.name.toLowerCase().startsWith(event.query.toLowerCase());
      });
    }
  }, 250);
};

async function loadRecord(id: number) {
  try {
    loading.value = true;
    const data = await sharedStore.getRecordByID("users", id);
    record.value = {
      ...initData(),
      ...data,
    };

    await nextTick();
    loading.value = false;
  } catch (err) {
    toastStore.errorResponseToast(err);
  }
}

async function isRecordValid() {
  const isValid = await v$.value.record.$validate();
  if (!isValid) return false;
  return true;
}

async function manageRecord() {
  if (!hasPermission("manage_users")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }

  if (readOnly.value) {
    toastStore.createInfoToast("Not allowed", "This record is read only!");
    return;
  }

  if (!(await isRecordValid())) return;

  loading.value = true;

  let recordData = {
    email: record.value.email,
    role_id: record.value?.role?.id,
    ...(props.mode === "update" && { display_name: record.value.display_name }),
  };

  try {
    let response = null;

    switch (props.mode) {
      case "create":
        response = await sharedStore.createRecord(
          userStore.apiPrefix + "/invitations",
          recordData,
        );
        break;
      case "update":
        response = await sharedStore.updateRecord(
          userStore.apiPrefix,
          record.value.id!,
          recordData,
        );
        break;
      default:
        loading.value = false;
        emit("completeOperation");
        break;
    }

    loading.value = false;
    v$.value.record.$reset();
    toastStore.successResponseToast(response);
    emit("completeOperation");
  } catch (error) {
    toastStore.errorResponseToast(error);
  } finally {
    loading.value = false;
  }
}

async function deleteConfirmation(id: number) {
  confirm.require({
    header: "Delete record?",
    message: `This will delete record: "${id}".`,
    rejectProps: { label: "Cancel" },
    acceptProps: { label: "Delete", severity: "danger" },
    accept: () => deleteRecord(id),
  });
}

async function deleteRecord(id: number) {
  if (!hasPermission("delete_users")) {
    toastStore.createInfoToast(
      "Access denied",
      "You don't have permission to perform this action.",
    );
    return;
  }

  try {
    let response = await sharedStore.deleteRecord("users", id);
    toastStore.successResponseToast(response);
    emit("completeUserDelete");
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
}
</script>

<template>
  <div v-if="!loading" class="flex flex-column gap-3 p-1">
    <div v-if="readOnly">
      <h5 style="color: var(--text-secondary)">Read-only mode.</h5>
    </div>

    <div class="flex flex-column gap-3">
      <div v-if="mode === 'update'" class="flex flex-row w-full">
        <div class="flex flex-column gap-1 w-full">
          <ValidationError
            :is-required="true"
            :message="v$.record.display_name.$errors[0]?.$message"
          >
            <label>Display name</label>
          </ValidationError>
          <InputText
            v-model="record.display_name"
            :disabled="readOnly"
            size="small"
            placeholder="Change display name"
          />
        </div>
      </div>

      <div class="flex flex-row w-full">
        <div class="flex flex-column gap-1 w-full">
          <ValidationError
            :is-required="true"
            :message="v$.record.email.$errors[0]?.$message"
          >
            <label>Email</label>
          </ValidationError>
          <InputText
            v-model="record.email"
            :readonly="readOnly || mode == 'update'"
            :disabled="readOnly"
            size="small"
            placeholder="Input email"
          />
        </div>
      </div>

      <div class="flex flex-row w-full">
        <div class="flex flex-column gap-1 w-full">
          <ValidationError
            :is-required="true"
            :message="v$.record.role.name.$errors[0]?.$message"
          >
            <label>Role</label>
          </ValidationError>
          <AutoComplete
            v-model="record.role"
            :readonly="readOnly"
            :disabled="readOnly"
            size="small"
            :suggestions="filteredRoles"
            option-label="name"
            force-selection
            placeholder="Select role"
            dropdown
            @complete="searchRole"
          />
        </div>
      </div>

      <div v-if="mode == 'update'" class="flex flex-row gap-2 w-full">
        <AuditTrail
          :record-id="props.recordId!"
          :events="['create', 'update', 'delete', 'restore']"
          category="user"
        />
      </div>

      <div class="flex flex-row gap-2 w-full">
        <div class="flex flex-column w-full gap-2">
          <Button
            v-if="!readOnly"
            class="main-button"
            :label="(mode == 'create' ? 'Invite' : 'Update') + ' user'"
            style="height: 42px"
            @click="manageRecord"
          />
          <Button
            v-if="!readOnly && mode == 'update'"
            label="Delete user"
            class="delete-button"
            style="height: 42px"
            @click="deleteConfirmation(record.id!)"
          />
        </div>
      </div>
    </div>
  </div>
  <ShowLoading v-else :num-fields="7" />
</template>

<style scoped></style>
