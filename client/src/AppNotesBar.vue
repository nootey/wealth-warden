<template>
  <Drawer
    id="drawer"
    v-model:visible="open"
    header="Notes"
    position="right"
    style="width: 100%; max-width: 468px; overflow-y: auto"
  >
    <template #container="{ closeCallback }">
      <div class="flex flex-column w-full p-3 gap-3">
        <div
          class="flex flex-row justify-content-between align-items-center p-2"
        >
          <h3>Notes</h3>
          <i class="pi pi-times hover-icon" @click="closeCallback" />
        </div>

        <div class="flex flex-row align-items-center gap-2">
          <Textarea
            v-model="newNoteContent"
            placeholder="Add a new note ..."
            rows="1"
            class="w-full border-round-xl"
            :style="{
              borderColor: 'var(--border-color)',
              resize: 'none',
            }"
            @keydown.enter.exact.prevent="createNote"
          />
        </div>

        <SimplePaginator
          v-if="paginator.total > paginator.rowsPerPage"
          :current-page="page"
          :total-records="paginator.total"
          :rows-per-page="paginator.rowsPerPage!"
          @page-change="loadNotes"
        />

        <div
          v-for="note in notes"
          :key="note.id"
          class="p-3 border-round-xl"
          :style="{
            backgroundColor: note.resolved_at
              ? 'var(--background-secondary)'
              : 'var(--background-primary)',
            border: '1px solid var(--border-color)',
          }"
        >
          <div class="flex flex-column gap-2">
            <Textarea
              v-model="note.content"
              :disabled="!!note.resolved_at"
              rows="2"
              class="w-full border-round-xl"
              :style="{
                borderColor: 'var(--border-color)',
                backgroundColor: note.resolved_at
                  ? 'transparent'
                  : 'var(--surface-ground)',
                cursor: note.resolved_at ? 'default' : 'text',
                resize: 'none',
              }"
              @focus="storeOriginalContent(note)"
              @blur="updateNoteIfChanged(note)"
            />

            <div
              class="flex flex-row gap-2 text-xs justify-content-between"
              style="color: var(--text-secondary)"
            >
              <div class="flex flex-row align-items-center gap-1">
                <span
                  >Created: {{ dateHelper.formatDate(note.created_at) }}</span
                >
                <i
                  v-if="note.updated_at && note.updated_at !== note.created_at"
                  v-tooltip="
                    'Updated: ' + dateHelper.formatDate(note.updated_at)
                  "
                  class="pi pi-info-circle ml-1"
                  style="font-size: 0.7rem"
                ></i>
              </div>
              <span>
                {{
                  note.resolved_at
                    ? "Resolved: " + dateHelper.formatDate(note.resolved_at)
                    : "Resolve: "
                }}
                <i
                  v-if="!note.resolved_at"
                  class="pi pi-check-square ml-2 text-sm"
                  style="cursor: pointer"
                  @click="toggleResolve(note.id!)"
                ></i>
                <template v-else>
                  <i
                    class="pi pi-replay ml-2 text-sm"
                    style="cursor: pointer"
                    @click="toggleResolve(note.id!)"
                  ></i>
                  <i
                    class="pi pi-trash ml-2 text-sm"
                    style="color: var(--p-red-300); cursor: pointer"
                    @click="deleteNote(note.id!)"
                  ></i>
                </template>
              </span>
            </div>
          </div>
        </div>

        <div
          v-if="notes.length === 0"
          class="text-center p-4"
          style="color: var(--text-secondary)"
        >
          No notes yet
        </div>
      </div>
    </template>
  </Drawer>
</template>

<script setup lang="ts">
import { ref, defineExpose } from "vue";
import { useNotesStore } from "./services/stores/notes_store.ts";
import { useSharedStore } from "./services/stores/shared_store.ts";
import type { Note } from "./models/notes_models.ts";
import dateHelper from "./utils/date_helper.ts";
import { useToastStore } from "./services/stores/toast_store.ts";
import SimplePaginator from "./_vue/components/base/SimplePaginator.vue";
import type { PaginatorState } from "./models/shared_models.ts";

const notesStore = useNotesStore();
const sharedStore = useSharedStore();
const toastStore = useToastStore();

const open = ref(false);
const notes = ref<Note[]>([]);
const newNoteContent = ref("");

const rows = ref([5]);
const default_rows = ref(rows.value[0]);
const paginator = ref<PaginatorState>({
  total: 0,
  from: 0,
  to: 0,
  rowsPerPage: default_rows.value!,
});
const page = ref(1);
const originalContent = ref<string>("");

const toggle = async () => {
  open.value = !open.value;
  if (open.value) {
    await loadNotes();
  }
};

const loadNotes = async (page_num = 1) => {
  try {
    const response = await sharedStore.getRecordsPaginated(
      notesStore.apiPrefix,
      { rowsPerPage: paginator.value.rowsPerPage },
      page_num,
    );

    notes.value = response.data || [];
    paginator.value.total = response.total_records;
    paginator.value.to = response.to;
    paginator.value.from = response.from;
    page.value = page_num;
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
};

const createNote = async () => {
  if (!newNoteContent.value.trim()) return;

  try {
    await sharedStore.createRecord(notesStore.apiPrefix, {
      content: newNoteContent.value.trim(),
    });
    newNoteContent.value = "";
    await loadNotes();
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
};

const storeOriginalContent = (note: Note) => {
  originalContent.value = note.content;
};

const updateNoteIfChanged = async (note: Note) => {
  if (!note.content.trim() || note.content === originalContent.value) return;

  try {
    await sharedStore.updateRecord(notesStore.apiPrefix, note.id!, {
      content: note.content.trim(),
    });
    await loadNotes(page.value);
    toastStore.successResponseToast({
      title: "Success",
      message: "Note updated successfully.",
    });
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
};

const toggleResolve = async (id: number) => {
  try {
    await notesStore.toggleResolveState(id);
    await loadNotes(page.value);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
};

const deleteNote = async (id: number) => {
  try {
    await sharedStore.deleteRecord("notes", id);
    await loadNotes(page.value);
  } catch (error) {
    toastStore.errorResponseToast(error);
  }
};

defineExpose({ open, toggle });
</script>

<style scoped lang="scss">
@media (max-width: 768px) {
  #drawer {
    max-width: 100% !important;
  }
}
</style>
