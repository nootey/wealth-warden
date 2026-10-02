<template>
  <Drawer
    id="drawer"
    v-model:visible="open"
    header="Notes"
    position="left"
    style="width: 100%; max-width: 468px"
  >
    <template #container="{ closeCallback }">
      <div class="flex flex-col h-full overflow-y-auto bg-canvas">
        <div
          class="sticky top-0 z-10 flex flex-row items-center justify-between gap-2 px-5 py-4 border-b border-line bg-canvas"
        >
          <div class="flex flex-col gap-0.5">
            <span class="label">Workspace</span>
            <div class="flex items-baseline gap-2">
              <span class="text-lg font-medium tracking-tight text-ink">
                Notes
              </span>
              <span v-if="paginator.total" class="text-xs text-faint">
                {{ paginator.total }}
              </span>
            </div>
          </div>
          <button
            type="button"
            class="flex items-center justify-center w-8 h-8 rounded-lg text-muted hover:text-ink hover:bg-sunken transition-colors cursor-pointer"
            aria-label="Close"
            @click="closeCallback"
          >
            <i class="pi pi-times text-sm" />
          </button>
        </div>

        <div class="flex flex-col gap-4 p-4">
          <section
            class="flex flex-col gap-2 rounded-2xl border border-line bg-card p-4 shadow-[var(--shadow-card)]"
          >
            <Textarea
              v-model="newNoteContent"
              placeholder="Add a new note ..."
              rows="2"
              class="w-full rounded-xl"
              :style="{
                borderColor: 'var(--border-color)',
                backgroundColor: 'var(--background-alt)',
                resize: 'none',
              }"
              @keydown.enter.exact.prevent="createNote"
            />
            <span class="text-xs text-faint">
              Press Enter to save, Shift + Enter for a new line.
            </span>
          </section>

          <div
            v-for="note in notes"
            :key="note.id"
            class="flex flex-col gap-3 rounded-2xl border p-4"
            :class="
              note.resolved_at
                ? 'border-transparent bg-sunken'
                : 'border-line bg-card shadow-[var(--shadow-card)]'
            "
          >
            <Textarea
              v-model="note.content"
              :disabled="!!note.resolved_at"
              rows="2"
              auto-resize
              class="w-full rounded-xl"
              :style="{
                borderColor: note.resolved_at
                  ? 'transparent'
                  : 'var(--border-color)',
                backgroundColor: 'transparent',
                color: note.resolved_at
                  ? 'var(--text-secondary)'
                  : 'var(--text-primary)',
                textDecoration: note.resolved_at ? 'line-through' : 'none',
                cursor: note.resolved_at ? 'default' : 'text',
                resize: 'none',
              }"
              @focus="storeOriginalContent(note)"
              @blur="updateNoteIfChanged(note)"
            />

            <div class="flex flex-row items-center justify-between gap-2">
              <div class="flex flex-row items-center gap-2 text-xs text-muted">
                <span
                  v-if="note.resolved_at"
                  class="rounded-full bg-card px-2 py-0.5 font-medium text-gain"
                >
                  Resolved
                </span>
                <span>
                  {{
                    dateHelper.formatDate(note.resolved_at ?? note.created_at)
                  }}
                </span>
                <i
                  v-if="note.updated_at && note.updated_at !== note.created_at"
                  v-tooltip="
                    'Updated: ' + dateHelper.formatDate(note.updated_at)
                  "
                  class="pi pi-pencil text-[0.65rem] text-faint"
                />
              </div>

              <div class="flex flex-row items-center gap-1">
                <button
                  v-if="!note.resolved_at"
                  v-tooltip.top="'Resolve'"
                  type="button"
                  class="flex items-center justify-center w-7 h-7 rounded-lg text-muted hover:text-gain hover:bg-sunken transition-colors cursor-pointer"
                  aria-label="Resolve"
                  @click="toggleResolve(note.id!)"
                >
                  <i class="pi pi-check text-xs" />
                </button>
                <template v-else>
                  <button
                    v-tooltip.top="'Reopen'"
                    type="button"
                    class="flex items-center justify-center w-7 h-7 rounded-lg text-muted hover:text-ink hover:bg-card transition-colors cursor-pointer"
                    aria-label="Reopen"
                    @click="toggleResolve(note.id!)"
                  >
                    <i class="pi pi-replay text-xs" />
                  </button>
                  <button
                    v-tooltip.top="'Delete'"
                    type="button"
                    class="flex items-center justify-center w-7 h-7 rounded-lg text-muted hover:text-loss hover:bg-card transition-colors cursor-pointer"
                    aria-label="Delete"
                    @click="deleteNote(note.id!)"
                  >
                    <i class="pi pi-trash text-xs" />
                  </button>
                </template>
              </div>
            </div>
          </div>

          <EmptyState
            v-if="notes.length === 0"
            icon="pi pi-file-edit"
            title="No notes yet."
          />

          <SimplePaginator
            v-if="paginator.total > paginator.rowsPerPage"
            :current-page="page"
            :total-records="paginator.total"
            :rows-per-page="paginator.rowsPerPage!"
            @page-change="loadNotes"
          />
        </div>
      </div>
    </template>
  </Drawer>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useNotesStore } from "../../services/stores/notes_store.ts";
import { useSharedStore } from "../../services/stores/shared_store.ts";
import type { Note } from "../../models/notes_models.ts";
import dateHelper from "../../utils/date_helper.ts";
import { useToastStore } from "../../services/stores/toast_store.ts";
import SimplePaginator from "../components/base/SimplePaginator.vue";
import EmptyState from "../components/base/EmptyState.vue";
import type { PaginatorState } from "../../models/shared_models.ts";

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
