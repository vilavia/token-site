<template>
  <BaseDialog :show="show" :title="t('models.yourApiKey')" width="wide" @close="emit('close')">
    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-10">
      <svg class="h-6 w-6 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
    </div>

    <div v-else class="space-y-6">
      <!-- Existing key or newly generated key -->
      <div v-if="currentKey">
        <label class="input-label">{{ t('models.yourApiKey') }}</label>
        <div class="mt-1 flex items-center gap-2">
          <code class="flex-1 rounded-xl border border-gray-200 bg-gray-50 px-4 py-2.5 font-mono text-sm text-gray-800 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200 overflow-x-auto">
            {{ isNewKey ? currentKey.key : maskKey(currentKey.key) }}
          </code>
          <button
            type="button"
            @click="copyKey"
            class="shrink-0 rounded-xl border border-gray-200 p-2.5 transition-colors hover:bg-gray-100 dark:border-dark-600 dark:hover:bg-dark-700"
            :class="copied ? 'text-emerald-500' : 'text-gray-500 dark:text-dark-400'"
            :title="copied ? t('keys.copied') : t('keys.copyToClipboard')"
          >
            <svg v-if="copied" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
            </svg>
            <svg v-else class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M8.25 7.5V6.108c0-1.135.845-2.098 1.976-2.192.373-.03.748-.057 1.123-.08M15.75 18H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08M15.75 18.75v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5A3.375 3.375 0 006.375 7.5H5.25m11.9-3.664A2.251 2.251 0 0015 4.5h-1.5a2.251 2.251 0 00-2.15 1.586m5.8 0c.065.21.1.433.1.664v.75h-6V6.75c0-.231.035-.454.1-.664M6.75 7.5H4.875c-.621 0-1.125.504-1.125 1.125v12c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V16.5a9 9 0 00-9-9z" />
            </svg>
          </button>
        </div>
        <p v-if="isNewKey" class="mt-2 text-xs text-amber-600 dark:text-amber-400">
          Make sure to copy your API key now. You won't be able to see the full key again.
        </p>
      </div>

      <!-- No key yet: generate button -->
      <div v-else class="flex flex-col items-center gap-4 py-4">
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('models.noKeyYet') }}</p>
        <button
          type="button"
          :disabled="generating"
          @click="generateKey"
          class="btn btn-primary"
        >
          <svg v-if="generating" class="-ml-1 mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          {{ t('models.generateKey') }}
        </button>
      </div>

      <!-- Usage Guide -->
      <div v-if="currentKey" class="space-y-3">
        <h4 class="font-medium text-gray-900 dark:text-white">{{ t('models.usageGuide') }}</h4>

        <!-- Base URL info -->
        <div>
          <label class="input-label text-xs">Base URL</label>
          <code class="mt-1 block rounded-xl border border-gray-200 bg-gray-50 px-4 py-2 font-mono text-xs text-gray-700 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-300">
            {{ baseUrl }}
          </code>
        </div>

        <!-- cURL -->
        <div>
          <label class="input-label text-xs">cURL</label>
          <div class="relative mt-1">
            <pre class="overflow-x-auto rounded-xl border border-gray-200 bg-gray-50 p-4 font-mono text-xs text-gray-700 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-300"><code>{{ curlExample }}</code></pre>
            <button
              type="button"
              @click="copyCurl"
              class="absolute right-2 top-2 rounded-lg p-1.5 transition-colors hover:bg-gray-200 dark:hover:bg-dark-600"
              :class="copiedCurl ? 'text-emerald-500' : 'text-gray-400 dark:text-dark-500'"
            >
              <svg v-if="copiedCurl" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
              </svg>
              <svg v-else class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M8.25 7.5V6.108c0-1.135.845-2.098 1.976-2.192.373-.03.748-.057 1.123-.08M15.75 18H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08M15.75 18.75v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5A3.375 3.375 0 006.375 7.5H5.25m11.9-3.664A2.251 2.251 0 0015 4.5h-1.5a2.251 2.251 0 00-2.15 1.586m5.8 0c.065.21.1.433.1.664v.75h-6V6.75c0-.231.035-.454.1-.664M6.75 7.5H4.875c-.621 0-1.125.504-1.125 1.125v12c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V16.5a9 9 0 00-9-9z" />
              </svg>
            </button>
          </div>
        </div>

        <!-- Python openai SDK -->
        <div>
          <label class="input-label text-xs">Python (openai SDK)</label>
          <div class="relative mt-1">
            <pre class="overflow-x-auto rounded-xl border border-gray-200 bg-gray-50 p-4 font-mono text-xs text-gray-700 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-300"><code>{{ pythonExample }}</code></pre>
            <button
              type="button"
              @click="copyPython"
              class="absolute right-2 top-2 rounded-lg p-1.5 transition-colors hover:bg-gray-200 dark:hover:bg-dark-600"
              :class="copiedPython ? 'text-emerald-500' : 'text-gray-400 dark:text-dark-500'"
            >
              <svg v-if="copiedPython" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
              </svg>
              <svg v-else class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M8.25 7.5V6.108c0-1.135.845-2.098 1.976-2.192.373-.03.748-.057 1.123-.08M15.75 18H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08M15.75 18.75v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5A3.375 3.375 0 006.375 7.5H5.25m11.9-3.664A2.251 2.251 0 0015 4.5h-1.5a2.251 2.251 0 00-2.15 1.586m5.8 0c.065.21.1.433.1.664v.75h-6V6.75c0-.231.035-.454.1-.664M6.75 7.5H4.875c-.621 0-1.125.504-1.125 1.125v12c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V16.5a9 9 0 00-9-9z" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { keysAPI } from '@/api/keys'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { ApiKey } from '@/types'

const props = defineProps<{
  show: boolean
  modelId: string
  groupId?: number | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()

const loading = ref(false)
const generating = ref(false)
const currentKey = ref<ApiKey | null>(null)
const isNewKey = ref(false)
const copied = ref(false)
const copiedCurl = ref(false)
const copiedPython = ref(false)

const baseUrl = computed(() => window.location.origin + '/v1')

const displayKey = computed(() => {
  if (!currentKey.value) return ''
  return isNewKey.value ? currentKey.value.key : maskKey(currentKey.value.key)
})

const curlExample = computed(() => {
  const key = displayKey.value || 'YOUR_API_KEY'
  return `curl ${baseUrl.value}/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${key}" \\
  -d '{
    "model": "${props.modelId}",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'`
})

const pythonExample = computed(() => {
  const key = displayKey.value || 'YOUR_API_KEY'
  return `from openai import OpenAI

client = OpenAI(
    api_key="${key}",
    base_url="${baseUrl.value}"
)

response = client.chat.completions.create(
    model="${props.modelId}",
    messages=[{"role": "user", "content": "Hello!"}]
)
print(response.choices[0].message.content)`
})

function maskKey(key: string): string {
  if (!key) return ''
  if (key.length <= 12) return key.slice(0, 4) + '...' + key.slice(-4)
  return key.slice(0, 8) + '...' + key.slice(-6)
}

async function loadKey() {
  loading.value = true
  isNewKey.value = false
  currentKey.value = null
  try {
    const data = await keysAPI.list(1, 100, props.groupId ? { group_id: props.groupId } : undefined)
    const keys = data.items || []
    if (keys.length > 0) {
      currentKey.value = keys[0]
    }
  } catch (error) {
    console.error('Failed to load keys:', error)
  } finally {
    loading.value = false
  }
}

async function generateKey() {
  generating.value = true
  try {
    const newKey = await keysAPI.create(
      `${props.modelId} Key`,
      props.groupId ?? undefined
    )
    currentKey.value = newKey
    isNewKey.value = true
  } catch (error) {
    console.error('Failed to generate key:', error)
  } finally {
    generating.value = false
  }
}

async function copyKey() {
  if (!currentKey.value) return
  await navigator.clipboard.writeText(currentKey.value.key)
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}

async function copyCurl() {
  await navigator.clipboard.writeText(curlExample.value)
  copiedCurl.value = true
  setTimeout(() => { copiedCurl.value = false }, 2000)
}

async function copyPython() {
  await navigator.clipboard.writeText(pythonExample.value)
  copiedPython.value = true
  setTimeout(() => { copiedPython.value = false }, 2000)
}

watch(
  () => props.show,
  (v) => {
    if (v) {
      loadKey()
    }
  }
)
</script>
