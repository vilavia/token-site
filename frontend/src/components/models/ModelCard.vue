<template>
  <router-link
    :to="`/models/${encodeURIComponent(model.id)}`"
    class="card block cursor-pointer transition-shadow hover:shadow-md"
  >
    <div class="p-5 space-y-4">
      <!-- Header: name + provider badge -->
      <div class="flex items-start justify-between gap-3">
        <h3 class="text-base font-semibold text-gray-900 dark:text-white leading-snug">
          {{ model.display_name }}
        </h3>
        <span
          class="inline-flex shrink-0 items-center rounded-full px-2.5 py-0.5 text-xs font-medium"
          :class="providerBadgeClass"
        >
          {{ model.provider }}
        </span>
      </div>

      <!-- Pricing -->
      <div class="space-y-1">
        <div class="flex items-center justify-between text-sm">
          <span class="text-gray-500 dark:text-dark-400">{{ t('models.inputPrice') }}</span>
          <span class="font-medium text-gray-800 dark:text-dark-200">
            ${{ model.input_price_mtok.toFixed(2) }} {{ t('models.perMTokens') }}
          </span>
        </div>
        <div class="flex items-center justify-between text-sm">
          <span class="text-gray-500 dark:text-dark-400">{{ t('models.outputPrice') }}</span>
          <span class="font-medium text-gray-800 dark:text-dark-200">
            ${{ model.output_price_mtok.toFixed(2) }} {{ t('models.perMTokens') }}
          </span>
        </div>
      </div>

      <!-- Context window -->
      <div class="flex items-center gap-1.5 text-sm text-gray-500 dark:text-dark-400">
        <svg class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" />
        </svg>
        <span>{{ contextWindowLabel }}</span>
      </div>

      <!-- Input format tags -->
      <div v-if="model.input_formats && model.input_formats.length > 0" class="flex flex-wrap gap-1.5">
        <span
          v-for="fmt in model.input_formats"
          :key="fmt"
          class="inline-flex items-center rounded-md bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-dark-300"
        >
          {{ fmt }}
        </span>
      </div>
    </div>
  </router-link>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelInfo } from '@/api/models'

const props = defineProps<{ model: ModelInfo }>()

const { t } = useI18n()

const providerBadgeClass = computed(() => {
  const p = props.model.provider.toLowerCase()
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
  const ctx = props.model.context_window
  if (ctx >= 1000) {
    return `${Math.round(ctx / 1000)}K ${t('models.contextWindow')}`
  }
  return `${ctx} ${t('models.contextWindow')}`
})
</script>
