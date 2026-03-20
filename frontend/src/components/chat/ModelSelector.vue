<template>
  <select
    class="model-selector"
    :value="modelValue"
    @change="onChange"
  >
    <option value="" disabled>{{ placeholder }}</option>
    <template v-for="group in groupedModels" :key="group.provider">
      <optgroup :label="group.provider">
        <option v-for="model in group.models" :key="model" :value="model">
          {{ model }}
        </option>
      </optgroup>
    </template>
  </select>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  modelValue: string
  models: string[]
  placeholder?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

interface ModelGroup {
  provider: string
  models: string[]
}

function inferProvider(name: string): string {
  const lower = name.toLowerCase()
  if (lower.startsWith('claude') || lower.includes('claude')) return 'Anthropic'
  if (lower.startsWith('gpt') || lower.startsWith('o1') || lower.startsWith('o3') || lower.startsWith('o4')) return 'OpenAI'
  if (lower.startsWith('gemini')) return 'Google'
  if (lower.startsWith('deepseek')) return 'DeepSeek'
  if (lower.startsWith('llama') || lower.startsWith('meta')) return 'Meta'
  if (lower.startsWith('mistral') || lower.startsWith('mixtral')) return 'Mistral'
  return 'Other'
}

const groupedModels = computed((): ModelGroup[] => {
  const map = new Map<string, string[]>()
  for (const model of props.models) {
    const provider = inferProvider(model)
    if (!map.has(provider)) map.set(provider, [])
    map.get(provider)!.push(model)
  }
  // Sort: known providers first, then "Other"
  const order = ['OpenAI', 'Anthropic', 'Google', 'DeepSeek', 'Meta', 'Mistral', 'Other']
  return Array.from(map.entries())
    .sort((a, b) => {
      const ai = order.indexOf(a[0])
      const bi = order.indexOf(b[0])
      return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
    })
    .map(([provider, models]) => ({ provider, models }))
})

function onChange(e: Event) {
  const value = (e.target as HTMLSelectElement).value
  emit('update:modelValue', value)
}
</script>

<style scoped>
.model-selector {
  appearance: none;
  -webkit-appearance: none;
  background: var(--chat-bg-input, #ffffff);
  border: 1px solid var(--chat-border-color, #e5e7eb);
  border-radius: 8px;
  padding: 6px 28px 6px 10px;
  font-size: 13px;
  color: var(--chat-text-primary, #111827);
  outline: none;
  cursor: pointer;
  max-width: 220px;
  text-overflow: ellipsis;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 20 20'%3E%3Cpath stroke='%236b7280' stroke-linecap='round' stroke-linejoin='round' stroke-width='1.5' d='M6 8l4 4 4-4'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 6px center;
  background-size: 16px;
  transition: border-color 150ms ease;
}

.model-selector:focus {
  border-color: var(--chat-accent-primary, #14b8a6);
}
</style>
