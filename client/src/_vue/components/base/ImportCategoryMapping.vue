<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { SelectChangeEvent } from 'primevue/select'
import type {Category} from "../../../models/transaction_models.ts";
import Select from 'primevue/select'


const props = defineProps<{
    importedCategories: string[]
    appCategories: Category[]
    modelValue?: Record<string, number | null>
}>()

const emit = defineEmits<{
    (e: 'update:modelValue', value: Record<string, number | null>): void
    (e: 'save', value: Record<string, number | null>): void
}>()

const tableData = computed(() =>
    props.importedCategories.map(name => ({ name }))
)

const normalize = (s: string) =>
    s
        .toLowerCase()
        .replace(/[_-]/g, ' ')
        .replace(/\s+/g, ' ')
        .trim()

const defaultCategory = computed(() => {
    return (
        props.appCategories.find(c => normalize(c.name) === '(uncategorized)') ||
        props.appCategories.find(c => c.is_default) ||
        null
    )
})

type OptionItem = { label: string; value: number; meta: Category }
type OptionGroup = { label: string; items: OptionItem[] }

const groupedOptions = computed<OptionGroup[]>(() => {
    const groups = new Map<string, OptionItem[]>()
    for (const c of props.appCategories) {
        const g = groups.get(c.classification) ?? []
        g.push({ label: c.display_name || c.name, value: c.id!, meta: c })
        groups.set(c.classification, g)
    }
    for (const [k, items] of groups.entries()) {
        items.sort((a, b) => a.label.localeCompare(b.label))
        groups.set(k, items)
    }
    const order = ['income', 'expense', 'investments', 'savings']
    const rest = [...groups.keys()].filter(k => !order.includes(k)).sort()
    const labels = [...order, ...rest].filter(k => groups.has(k))
    return labels.map(k => ({ label: k.charAt(0).toUpperCase() + k.slice(1), items: groups.get(k)! }))
})

const byNormalizedName = computed<Map<string, Category>>(() => {
    const m = new Map<string, Category>()
    for (const c of props.appCategories) {
        m.set(normalize(c.name), c)
        m.set(normalize(c.display_name || c.name), c)
    }
    return m
})

const mapping = ref<Record<string, number | null>>({})

const prefill = () => {
    const next: Record<string, number | null> = {}
    for (const raw of props.importedCategories) {
        const key = raw // keep original for server
        const n = normalize(raw)
        const exact = byNormalizedName.value.get(n)

        if (exact) {
            next[key] = exact.id
            continue
        }

        let picked: Category | undefined
        for (const c of props.appCategories) {
            if (normalize(c.name) === n || normalize(c.display_name || c.name) === n) {
                picked = c
                break
            }
            if (!picked && normalize(c.name).includes(n)) picked = c
            if (!picked && n.includes(normalize(c.name))) picked = c
        }

        if (picked) {
            next[key] = picked.id
        } else {
            next[key] = defaultCategory.value ? defaultCategory.value.id : null
        }
    }
    mapping.value = next
    emit('update:modelValue', mapping.value)
}

watch(
    () => [props.importedCategories, props.appCategories],
    prefill,
    { immediate: true, deep: true }
)

function onSelect(imported: string, e: SelectChangeEvent) {
    mapping.value[imported] = e.value ?? null
    emit('update:modelValue', mapping.value)
}

function mapAllToDefault() {
    const id = defaultCategory.value?.id ?? null
    const next: Record<string, number | null> = {}
    for (const raw of props.importedCategories) next[raw] = id
    mapping.value = next
    emit('update:modelValue', mapping.value)
}

function clearAll() {
    const next: Record<string, number | null> = {}
    for (const raw of props.importedCategories) next[raw] = null
    mapping.value = next
    emit('update:modelValue', mapping.value)
}

function save() {
    emit('save', mapping.value)
}
</script>

<template>
    <div class="flex flex-column gap-1 w-full">

        <div class="flex align-items-center w-full">
            <span style="color: var(--text-secondary)">
                These are the distinct categories. Map them to existing ones. If none selected, default will be used.
            </span>
            <div class="ml-auto flex gap-2">
                <Button size="small" class="outline-button" @click="clearAll" label="Clear" />
                <Button size="small" class="main-button" @click="mapAllToDefault" label="Defaults" />
                <Button size="small" class="main-button" icon="pi pi-save" label="Save" @click="save" />
            </div>
        </div>

        <DataTable :value="tableData" dataKey="name" class="w-full" :rows="10"
                   paginator :rowsPerPageOptions="[10,25,50]" responsiveLayout="scroll">

            <Column header="Imported">
                <template #body="{ data }">
                    <div class="flex align-items-center gap-2">
                        {{ data.name }}
                    </div>
                </template>
            </Column>

            <Column header="Mapping">
                <template #body="{ data }">
                    <Select class="w-full" size="small"
                            :modelValue="mapping[data.name] ?? null"
                            @update:modelValue="val => onSelect(data.name, { value: val } as any)"
                            :options="groupedOptions"
                            optionGroupLabel="label"
                            optionGroupChildren="items"
                            optionLabel="label"
                            optionValue="value"
                            showClear
                            filter
                            placeholder="Select category">

                        <template #value="slotProps">
                                <span v-if="slotProps.value">
                                    {{
                                        appCategories.find(c => c.id === slotProps.value)?.display_name
                                        ?? appCategories.find(c => c.id === slotProps.value)?.name
                                        ?? 'Select category'
                                    }}
                                </span>
                            <span v-else class="text-color-secondary">
                                    {{ defaultCategory ? `Default: ${defaultCategory.display_name || defaultCategory.name}` : 'Select category' }}
                                </span>
                        </template>

                        <template #option="opt">
                            <div class="flex justify-content-between w-full">
                                <span>{{ opt.option.label }}</span>
                                <small class="text-color-secondary">
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
