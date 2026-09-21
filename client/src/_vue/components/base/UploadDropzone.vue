<script setup lang="ts">
import { ref } from "vue";

withDefaults(
  defineProps<{
    accept: string;
    files: File[];
    hint?: string;
    info?: string;
    multiple?: boolean;
    maxFileSize?: number;
    disabled?: boolean;
    status?: { label: string; severity: string };
  }>(),
  {
    hint: undefined,
    info: undefined,
    multiple: false,
    maxFileSize: 10485760,
    disabled: false,
    status: () => ({ label: "Pending", severity: "warn" }),
  },
);

const emit = defineEmits<{
  (e: "select", value: { files: File[] }): void;
  (e: "remove", value: { files: File[] }): void;
  (e: "clear"): void;
}>();

const uploadRef = ref<{ clear?: () => void } | null>(null);

function clear() {
  try {
    uploadRef.value?.clear?.();
  } catch {
    /* no-op */
  }
}

defineExpose({ clear });
</script>

<template>
  <div class="flex flex-col w-full">
    <FileUpload
      ref="uploadRef"
      :accept="accept"
      :max-file-size="maxFileSize"
      :multiple="multiple"
      custom-upload
      :show-upload-button="false"
      :show-cancel-button="false"
      @select="emit('select', $event)"
      @remove="emit('remove', $event)"
      @clear="emit('clear')"
    >
      <template #header="{ chooseCallback }">
        <div class="flex flex-col gap-3 w-full">
          <div
            v-if="info"
            class="flex flex-row w-full gap-2 items-center p-3 rounded-lg text-xs"
            style="
              background: var(--background-secondary);
              border: 1px solid var(--border-color);
              color: var(--text-secondary);
            "
          >
            <i class="pi pi-info-circle" style="flex-shrink: 0" />
            <span>{{ info }}</span>
          </div>

          <div class="flex flex-col items-center gap-2 py-4">
            <i
              class="pi pi-cloud-upload text-3xl"
              style="color: var(--text-secondary)"
            />
            <span class="text-sm" style="color: var(--text-secondary)"
              >Drag files here, or</span
            >
            <Button
              class="outline-button"
              :disabled="disabled"
              label="Upload"
              @click="chooseCallback()"
            />
            <span
              v-if="hint"
              class="text-xs"
              style="color: var(--text-secondary)"
              >{{ hint }}</span
            >
          </div>
        </div>
      </template>

      <template #content="{ removeFileCallback }">
        <div v-if="files.length > 0" class="flex flex-col gap-2 w-full">
          <h5>Pending</h5>
          <div class="flex flex-col gap-2 w-full">
            <div
              v-for="(file, index) in files"
              :key="file.name + file.type + file.size"
              class="flex flex-row gap-2 items-center justify-between p-2 rounded-lg"
              style="border: 1px solid var(--border-color)"
            >
              <div class="flex flex-row gap-2 items-center min-w-0">
                <i
                  class="pi pi-file"
                  style="color: var(--text-secondary); flex-shrink: 0"
                />
                <span
                  class="font-semibold text-ellipsis whitespace-nowrap overflow-hidden"
                  >{{ file.name }}</span
                >
              </div>
              <div
                class="flex flex-row gap-2 items-center"
                style="flex-shrink: 0"
              >
                <Badge :value="status.label" :severity="status.severity" />
                <i
                  class="pi pi-times hover-icon"
                  style="color: var(--p-red-300)"
                  @click="removeFileCallback(index)"
                />
              </div>
            </div>
          </div>
        </div>
      </template>
    </FileUpload>
  </div>
</template>
