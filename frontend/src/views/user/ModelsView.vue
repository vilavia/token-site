<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Page header -->
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('models.title') }}</h1>
      </div>

      <!-- Search + provider filters -->
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div class="relative flex-1 sm:max-w-xs">
          <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
            <svg class="h-4 w-4 text-gray-400 dark:text-dark-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" />
            </svg>
          </div>
          <input
            v-model="searchQuery"
            type="text"
            :placeholder="t('models.search')"
            class="input pl-9"
          />
        </div>

        <!-- Provider filter tabs -->
        <div class="flex gap-1.5">
          <button
            v-for="provider in providers"
            :key="provider"
            type="button"
            @click="selectedProvider = provider"
            :class="[
              'rounded-xl border px-3 py-1.5 text-sm font-medium transition-all',
              selectedProvider === provider
                ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300 dark:border-primary-400'
                : 'border-gray-200 bg-white text-gray-600 hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-300 dark:hover:border-primary-500'
            ]"
          >
            {{ provider === 'All' ? t('models.all') : provider }}
          </button>
        </div>
      </div>

      <!-- Loading state -->
      <div v-if="loading" class="flex items-center justify-center py-16">
        <svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>

      <!-- Models grid -->
      <div v-else-if="filteredModels.length > 0" class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <ModelCard v-for="model in filteredModels" :key="model.id" :model="model" />
      </div>

      <!-- Empty state -->
      <div v-else class="flex flex-col items-center justify-center py-16">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-800">
          <svg class="h-8 w-8 text-gray-400 dark:text-dark-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" />
          </svg>
        </div>
        <p class="text-sm text-gray-500 dark:text-dark-400">No models found</p>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { modelsAPI, type ModelInfo } from '@/api/models'
import AppLayout from '@/components/layout/AppLayout.vue'
import ModelCard from '@/components/models/ModelCard.vue'

const { t } = useI18n()

const models = ref<ModelInfo[]>([])
const loading = ref(false)
const searchQuery = ref('')
const selectedProvider = ref('All')

const providers = computed(() => {
  const providerSet = new Set(models.value.map((m) => m.provider))
  return ['All', ...Array.from(providerSet).sort()]
})

const filteredModels = computed(() => {
  let result = models.value

  if (selectedProvider.value !== 'All') {
    result = result.filter((m) => m.provider === selectedProvider.value)
  }

  if (searchQuery.value.trim()) {
    const q = searchQuery.value.trim().toLowerCase()
    result = result.filter(
      (m) =>
        m.display_name.toLowerCase().includes(q) ||
        m.provider.toLowerCase().includes(q) ||
        m.id.toLowerCase().includes(q)
    )
  }

  return result
})

async function fetchModels() {
  loading.value = true
  try {
    const res = await modelsAPI.list()
    models.value = res.data.models || []
  } catch (error) {
    console.error('Failed to fetch models:', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchModels()
})
</script>
