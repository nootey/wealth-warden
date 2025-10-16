<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import Select from 'primevue/select'
import type {Account} from "../../../models/account_models.ts";

const props = defineProps<{
    importedCategories: string[]
    investmentAccounts: Account[]
    modelValue?: Record<string, number | null>
}>()

const emit = defineEmits<{
    (e: 'update:modelValue', value: Record<string, number | null>): void
    (e: 'save', value: Record<string, number | null>): void
}>()

const mapping = ref<Record<string, number | null>>({})

const tableData = computed(() =>
    props.importedCategories.map(name => ({ name }))
)

const prefill = () => {
    const next: Record<string, number | null> = {}
    for (const name of props.importedCategories) {
        next[name] = props.modelValue?.[name] ?? null
    }
    mapping.value = next
    emit('update:modelValue', mapping.value)
}

watch(() => props.importedCategories, prefill, { immediate: true, deep: true })

function onSelect(imported: string, val: number | null) {
    mapping.value[imported] = val
    emit('update:modelValue', mapping.value)
}

function clearAll() {
    const cleared: Record<string, number | null> = {}
    for (const name of props.importedCategories) cleared[name] = null
    mapping.value = cleared
    emit('update:modelValue', mapping.value)
}

function save() {
    emit('save', mapping.value)
}
</script>

<template>
    <div class="flex flex-column gap-3">

        <div class="flex align-items-center gap-2">
            <span style="color: var(--text-secondary)">
            Map each investment category to one of your investment accounts.
          </span>
            <div class="ml-auto flex gap-2">
                <Button class="outline-button" @click="clearAll" label="Clear all" />
                <Button class="main-button" icon="pi pi-save" label="Save" @click="save" />
            </div>
        </div>

        <div class="surface-card border-round p-2">
            <DataTable :value="tableData" dataKey="name" class="w-full" :rows="10"
                       paginator :rowsPerPageOptions="[10,25,50]" responsiveLayout="scroll">

                <Column header="Imported">
                    <template #body="{ data }">
                        <div class="flex align-items-center gap-2">{{ data?.name }}</div>
                    </template>
                </Column>

                <Column header="Investment Account">
                    <template #body="{ data }">
                        <Select class="w-full"
                                :modelValue="mapping[data.name] ?? null"
                                @update:modelValue="val => onSelect(data.name, val)"
                                :options="investmentAccounts"
                                optionLabel="name"
                                optionValue="id"
                                showClear
                                filter
                                placeholder="Select account">
                            <template #value="slotProps">
                                <span v-if="slotProps.value">
                  {{ investmentAccounts.find(a => a.id === slotProps.value)?.name || 'Select account' }}
                </span>
                                <span v-else class="text-color-secondary">Select account</span>
                            </template>
                        </Select>
                    </template>
                </Column>

            </DataTable>
        </div>
    </div>
</template>