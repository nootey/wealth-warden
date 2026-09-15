<script setup lang="ts">
import { ref, computed, watch } from "vue";
import type { SelectChangeEvent } from "primevue/select";
import type { Category } from "../../../models/transaction_models.ts";
import Select from "primevue/select";
import { useRouter } from "vue-router";

const props = defineProps<{
  importedCategories: string[];
  appCategories: Category[];
  modelValue?: Record<string, number | null>;
}>();

const emit = defineEmits<{
  (e: "update:modelValue", value: Record<string, number | null>): void;
  (e: "save", value: Record<string, number | null>): void;
}>();

const router = useRouter();

const tableData = computed(() =>
  props.importedCategories.map((name) => ({ name })),
);

const normalize = (s: string) =>
  s.toLowerCase().replace(/[_-]/g, " ").replace(/\s+/g, " ").trim();

const defaultCategory = computed(() => {
  return (
    props.appCategories.find((c) => normalize(c.name) === "(uncategorized)") ||
    props.appCategories.find((c) => c.is_default) ||
    null
  );
});

type OptionItem = { label: string; value: number; meta: Category };
type OptionGroup = { label: string; items: OptionItem[] };

const groupedOptions = computed<OptionGroup[]>(() => {
  const groups = new Map<string, OptionItem[]>();
  for (const c of props.appCategories) {
    const g = groups.get(c.classification) ?? [];
    g.push({ label: c.display_name || c.name, value: c.id!, meta: c });
    groups.set(c.classification, g);
  }
  for (const [k, items] of groups.entries()) {
    items.sort((a, b) => a.label.localeCompare(b.label));
    groups.set(k, items);
  }
  const order = ["income", "expense", "investments", "savings"];
  const rest = [...groups.keys()].filter((k) => !order.includes(k)).sort();
  const labels = [...order, ...rest].filter((k) => groups.has(k));
  return labels.map((k) => ({
    label: k.charAt(0).toUpperCase() + k.slice(1),
    items: groups.get(k)!,
  }));
});

const byNormalizedName = computed<Map<string, Category>>(() => {
  const m = new Map<string, Category>();
  for (const c of props.appCategories) {
    m.set(normalize(c.name), c);
    m.set(normalize(c.display_name || c.name), c);
  }
  return m;
});

const mapping = ref<Record<string, number | null>>({});

const prefill = () => {
  const next: Record<string, number | null> = {};
  for (const raw of props.importedCategories) {
    const key = raw;
    const n = normalize(raw);
    let picked: Category | undefined = byNormalizedName.value.get(n);
    for (const c of props.appCategories) {
      if (picked) break;
      if (normalize(c.name).includes(n)) picked = c;
      if (!picked && n.includes(normalize(c.name))) picked = c;
    }

    // A prefilled default is not a manual choice; leave it unset so rules can run on the server.
    if (picked && picked.id !== defaultCategory.value?.id) {
      next[key] = picked.id ?? null;
    } else {
      next[key] = null;
    }
  }
  mapping.value = next;
  emit("update:modelValue", mapping.value);
  emit("save", mapping.value);
};

watch(() => [props.importedCategories, props.appCategories], prefill, {
  immediate: true,
  deep: true,
});

function onSelect(imported: string, e: SelectChangeEvent) {
  mapping.value[imported] = e.value ?? null;
  emit("update:modelValue", mapping.value);
  emit("save", mapping.value);
}

function mapAllToDefault() {
  const id = defaultCategory.value?.id ?? null;
  const next: Record<string, number | null> = {};
  for (const raw of props.importedCategories) next[raw] = id;
  mapping.value = next;
  emit("update:modelValue", mapping.value);
}

function clearAll() {
  const next: Record<string, number | null> = {};
  for (const raw of props.importedCategories) next[raw] = null;
  mapping.value = next;
  emit("update:modelValue", mapping.value);
}
</script>

<template>
  <div class="flex flex-col gap-1 w-full">
    <div class="flex flex-col items-center w-full">
      <div class="flex flex-col items-center text-center">
        <span style="color: var(--text-secondary)">
          These are the distinct categories. Map them to existing ones. A mapped
          category always wins.
        </span>
        <span class="text-xs" style="color: var(--text-secondary)">
          <i class="pi pi-info-circle text-xs" />
          Rows left on Auto get their category from your active
          <span
            class="hover-icon font-bold"
            @click="router.push({ name: 'settings.rules' })"
            >rules</span
          >. If no rule matches, they stay uncategorized.
        </span>
      </div>
      <div class="flex flex-row gap-4">
        <Button
          size="small"
          class="delete-button"
          label="Clear"
          @click="clearAll"
        />
        <Button
          size="small"
          class="outline-button"
          label="Defaults"
          @click="mapAllToDefault"
        />
      </div>
    </div>

    <DataTable
      :value="tableData"
      data-key="name"
      class="w-full"
      :rows="10"
      paginator
      :rows-per-page-options="[10, 25, 50]"
      responsive-layout="scroll"
    >
      <Column header="Imported">
        <template #body="{ data }">
          <div class="flex items-center gap-2">
            {{ data.name }}
          </div>
        </template>
      </Column>

      <Column header="Mapping">
        <template #body="{ data }">
          <Select
            class="w-full"
            size="small"
            :model-value="mapping[data.name] ?? null"
            :options="groupedOptions"
            option-group-label="label"
            option-group-children="items"
            option-label="label"
            option-value="value"
            show-clear
            filter
            placeholder="Select category"
            @update:model-value="
              (val) => onSelect(data.name, { value: val } as any)
            "
          >
            <template #value="slotProps">
              <span v-if="slotProps.value">
                {{
                  appCategories.find((c) => c.id === slotProps.value)
                    ?.display_name ??
                  appCategories.find((c) => c.id === slotProps.value)?.name ??
                  "Select category"
                }}
              </span>
              <span v-else class="text-muted-color">Auto (rules)</span>
            </template>

            <template #option="opt">
              <div class="flex justify-between w-full">
                <span>{{ opt.option.label }}</span>
                <small class="text-muted-color">
                  {{ opt.option.meta.classification }}
                </small>
              </div>
            </template>
          </Select>
        </template>
      </Column>
    </DataTable>
  </div>
</template>
