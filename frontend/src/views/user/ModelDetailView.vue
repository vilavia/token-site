<template>
  <AppLayout>
    <!-- Loading state -->
    <div v-if="loading" class="flex items-center justify-center py-20">
      <svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
    </div>

    <div v-else-if="model" class="mx-auto max-w-3xl space-y-6">
      <!-- Back button -->
      <div>
        <button
          type="button"
          @click="router.push('/models')"
          class="inline-flex items-center gap-1.5 text-sm text-gray-500 transition-colors hover:text-gray-800 dark:text-dark-400 dark:hover:text-dark-200"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18" />
          </svg>
          {{ t('models.backToModels') }}
        </button>
      </div>

      <!-- Model header -->
      <div class="card p-6">
        <div class="flex items-start justify-between gap-4">
          <div class="space-y-2">
            <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ model.display_name }}</h1>
            <span
              class="inline-flex items-center rounded-full px-3 py-1 text-sm font-medium"
              :class="providerBadgeClass"
            >
              {{ model.provider }}
            </span>
          </div>
        </div>

        <!-- Description -->
        <p v-if="model.description" class="mt-4 text-sm text-gray-600 dark:text-dark-300 leading-relaxed">
          {{ model.description }}
        </p>

        <!-- Action buttons -->
        <div class="mt-6 flex flex-wrap gap-3">
          <button
            v-if="chatEnabled"
            type="button"
            @click="router.push(`/chat?model=${encodeURIComponent(model.id)}`)"
            class="btn btn-primary"
          >
            <svg class="mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z" />
            </svg>
            {{ t('models.tryInChat') }}
          </button>
          <button
            type="button"
            @click="showApiKeyDialog = true"
            class="btn btn-secondary"
          >
            <svg class="mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
            </svg>
            {{ t('models.getApiKey') }}
          </button>
        </div>
      </div>

      <!-- Specifications -->
      <div class="card p-6">
        <h2 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">
          {{ t('models.specifications') }}
        </h2>
        <dl class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div class="space-y-1">
            <dt class="text-xs font-medium text-gray-500 dark:text-dark-400 uppercase tracking-wide">
              {{ t('models.contextWindow') }}
            </dt>
            <dd class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ contextWindowLabel }}
            </dd>
          </div>
          <div class="space-y-1">
            <dt class="text-xs font-medium text-gray-500 dark:text-dark-400 uppercase tracking-wide">
              {{ t('models.maxOutput') }}
            </dt>
            <dd class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ maxOutputLabel }}
            </dd>
          </div>
          <div class="space-y-1">
            <dt class="text-xs font-medium text-gray-500 dark:text-dark-400 uppercase tracking-wide">
              {{ t('models.inputFormats') }}
            </dt>
            <dd class="flex flex-wrap gap-1.5 pt-0.5">
              <span
                v-for="fmt in model.input_formats"
                :key="fmt"
                class="inline-flex items-center rounded-md bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-dark-300"
              >{{ fmt }}</span>
            </dd>
          </div>
          <div class="space-y-1">
            <dt class="text-xs font-medium text-gray-500 dark:text-dark-400 uppercase tracking-wide">
              {{ t('models.outputFormats') }}
            </dt>
            <dd class="flex flex-wrap gap-1.5 pt-0.5">
              <span
                v-for="fmt in model.output_formats"
                :key="fmt"
                class="inline-flex items-center rounded-md bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-dark-300"
              >{{ fmt }}</span>
            </dd>
          </div>
        </dl>
      </div>

      <!-- Pricing -->
      <div class="card p-6">
        <h2 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">
          {{ t('models.pricing') }}
        </h2>
        <dl class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800">
            <dt class="text-xs font-medium text-gray-500 dark:text-dark-400 uppercase tracking-wide">
              {{ t('models.inputPrice') }}
            </dt>
            <dd class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">
              ${{ model.input_price_per_mtok.toFixed(2) }}
            </dd>
            <dd class="text-xs text-gray-500 dark:text-dark-400">{{ t('models.perMTokens') }}</dd>
            <dd v-if="model.official_input_price_per_mtok > 0 && model.official_input_price_per_mtok !== model.input_price_per_mtok" class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              {{ t('models.officialPrice') }}: ${{ model.official_input_price_per_mtok.toFixed(2) }}
            </dd>
          </div>
          <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800">
            <dt class="text-xs font-medium text-gray-500 dark:text-dark-400 uppercase tracking-wide">
              {{ t('models.outputPrice') }}
            </dt>
            <dd class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">
              ${{ model.output_price_per_mtok.toFixed(2) }}
            </dd>
            <dd class="text-xs text-gray-500 dark:text-dark-400">{{ t('models.perMTokens') }}</dd>
            <dd v-if="model.official_output_price_per_mtok > 0 && model.official_output_price_per_mtok !== model.output_price_per_mtok" class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              {{ t('models.officialPrice') }}: ${{ model.official_output_price_per_mtok.toFixed(2) }}
            </dd>
          </div>
        </dl>
      </div>
    </div>

    <!-- Not found state -->
    <div v-else class="flex flex-col items-center justify-center py-20">
      <p class="text-gray-500 dark:text-dark-400">Model not found.</p>
      <button type="button" @click="router.push('/models')" class="btn btn-primary mt-4">
        {{ t('models.backToModels') }}
      </button>
    </div>

    <!-- API Key Dialog -->
    <ApiKeyDialog
      v-if="model"
      :show="showApiKeyDialog"
      :model-id="model.id"
      :group-id="null"
      @close="showApiKeyDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { modelsAPI, type ModelInfo } from '@/api/models'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import ApiKeyDialog from '@/components/models/ApiKeyDialog.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const model = ref<ModelInfo | null>(null)
const loading = ref(false)
const showApiKeyDialog = ref(false)
const chatEnabled = computed(() => appStore.cachedPublicSettings?.chat_enabled !== false)

const providerBadgeClass = computed(() => {
  if (!model.value) return ''
  const p = model.value.provider.toLowerCase()
  if (p.includes('anthropic')) {
    return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
  }
  if (p.includes('openai') || p.includes('open ai')) {
    return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
  }
  if (p.includes('google')) {
    return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
  }
  return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
})

const contextWindowLabel = computed(() => {
  if (!model.value) return ''
  const ctx = model.value.context_window
  if (ctx >= 1000) {
    return `${Math.round(ctx / 1000)}K ${t('models.tokens')}`
  }
  return `${ctx} ${t('models.tokens')}`
})

const maxOutputLabel = computed(() => {
  if (!model.value) return ''
  const max = model.value.max_output_tokens
  if (max >= 1000) {
    return `${Math.round(max / 1000)}K ${t('models.tokens')}`
  }
  return `${max} ${t('models.tokens')}`
})

async function fetchModel() {
  const id = route.params.id as string
  if (!id) return

  loading.value = true
  try {
    const res = await modelsAPI.getById(id)
    model.value = res.data
  } catch (error) {
    console.error('Failed to fetch model:', error)
    model.value = null
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchModel()
})
</script>
