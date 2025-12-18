<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import Select from 'primevue/select'
import type {Account} from "../../../models/account_models.ts";

const props = defineProps<{
    importedCategories: string[]
    accounts: Account[]
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
    emit('save', mapping.value)
}

watch(() => props.importedCategories, prefill, { immediate: true, deep: true })

function onSelect(imported: string, val: number | null) {
    mapping.value[imported] = val
    emit('update:modelValue', mapping.value)
    emit('save', mapping.value)
}

function clearAll() {
    const cleared: Record<string, number | null> = {}
    for (const name of props.importedCategories) cleared[name] = null
    mapping.value = cleared
    emit('update:modelValue', mapping.value)
}
</script>

<template>
  <div class="flex flex-column w-full">
    <div class="flex flex-column align-items-center w-full gap-3">
      <span style="color: var(--text-secondary)">
        Map each category to one of your accounts.
      </span>
      <div class="flex flex-row gap-3">
        <Button
          size="small"
          class="delete-button"
          label="Clear"
          @click="clearAll"
        />
      </div>
    </div>

    <DataTable
      :value="tableData"
      data-key="name"
      class="w-full"
      :rows="10"
      paginator
      :rows-per-page-options="[10,25,50]"
      responsive-layout="scroll"
    >
      <Column header="Imported">
        <template #body="{ data }">
          <div class="flex align-items-center gap-2">
            {{ data?.name }}
          </div>
        </template>
      </Column>

      <Column header="Account">
        <template #body="{ data }">
          <Select
            class="w-full"
            :model-value="mapping[data.name] ?? null"
            :options="accounts"
            option-label="name"
            option-value="id"
            show-clear
            filter
            placeholder="Select account"
            @update:model-value="val => onSelect(data.name, val)"
          >
            <template #value="slotProps">
              <span v-if="slotProps.value">
                {{ accounts.find(a => a.id === slotProps.value)?.name || 'Select account' }}
              </span>
              <span
                v-else
                class="text-color-secondary"
              >Select account</span>
            </template>
          </Select>
        </template>
      </Column>
    </DataTable>
  </div>
</template>