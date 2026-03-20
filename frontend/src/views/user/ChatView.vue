<template>
  <div class="chat-root">
    <div class="chat-page">
      <!-- Header -->
      <header class="chat-header">
        <div class="chat-header-left">
          <router-link :to="dashboardPath" class="chat-back-btn" :title="t('common.back')">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M15 19l-7-7 7-7" />
            </svg>
          </router-link>
          <ModelSelector
            v-model="selectedModel"
            :models="models"
            :placeholder="t('chat.selectModel')"
          />
        </div>
        <div class="chat-header-right">
          <span class="chat-balance">${{ balance }}</span>
          <button class="chat-new-btn" @click="newChat">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
            <span class="chat-new-btn-text">{{ t('chat.newChat') }}</span>
          </button>
        </div>
      </header>

      <!-- Messages area -->
      <main ref="messagesContainer" class="chat-messages">
        <!-- Empty state -->
        <div v-if="messages.length === 0" class="chat-empty">
          <svg class="chat-empty-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z" />
          </svg>
          <p class="chat-empty-text">{{ t('chat.startPrompt') }}</p>
        </div>

        <!-- Message list -->
        <ChatMessage
          v-for="(msg, idx) in messages"
          :key="idx"
          :message="msg"
        />

        <!-- Typing indicator -->
        <div v-if="isStreaming && !lastAssistantContent" class="chat-typing">
          <div class="chat-typing-bubble">
            <span class="chat-typing-dot"></span>
            <span class="chat-typing-dot"></span>
            <span class="chat-typing-dot"></span>
          </div>
        </div>
      </main>

      <!-- Input area -->
      <ChatInput
        :disabled="isStreaming || !selectedModel"
        :placeholder="t('chat.placeholder')"
        :send-label="t('chat.send')"
        @send="handleSend"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { getModels, streamChat, type ChatMessage as ChatMsg } from '@/api/chat'
import ChatMessage from '@/components/chat/ChatMessage.vue'
import ChatInput from '@/components/chat/ChatInput.vue'
import ModelSelector from '@/components/chat/ModelSelector.vue'

const { t } = useI18n()
const route = useRoute()
const authStore = useAuthStore()
const appStore = useAppStore()

const models = ref<string[]>([])
const selectedModel = ref('')
const messages = ref<ChatMsg[]>([])
const isStreaming = ref(false)
const messagesContainer = ref<HTMLElement | null>(null)
let abortController: AbortController | null = null

const dashboardPath = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
const balance = computed(() => authStore.user?.balance?.toFixed(2) || '0.00')

const lastAssistantContent = computed(() => {
  if (messages.value.length === 0) return ''
  const last = messages.value[messages.value.length - 1]
  return last.role === 'assistant' ? last.content : ''
})

function scrollToBottom() {
  nextTick(() => {
    const el = messagesContainer.value
    if (el) {
      el.scrollTop = el.scrollHeight
    }
  })
}

// Watch messages for auto-scroll
watch(
  () => messages.value.length,
  () => scrollToBottom()
)

async function loadModels() {
  try {
    const res = await getModels()
    models.value = res.models || []
  } catch (err) {
    console.error('Failed to load models:', err)
    appStore.showError(t('chat.errorStream'))
  }
}

function newChat() {
  if (abortController) {
    abortController.abort()
    abortController = null
  }
  isStreaming.value = false
  messages.value = []
}

function handleSend(text: string) {
  if (!selectedModel.value || isStreaming.value) return

  // Add user message
  messages.value.push({ role: 'user', content: text })
  scrollToBottom()

  // Add empty assistant message to fill in
  messages.value.push({ role: 'assistant', content: '' })
  isStreaming.value = true

  const msgIndex = messages.value.length - 1

  abortController = streamChat(
    {
      model: selectedModel.value,
      messages: messages.value.slice(0, -1) // send all except the empty assistant message
    },
    (chunk: string) => {
      // Accumulate content into the assistant message
      messages.value[msgIndex] = {
        ...messages.value[msgIndex],
        content: messages.value[msgIndex].content + chunk
      }
      scrollToBottom()
    },
    () => {
      isStreaming.value = false
      abortController = null
      // Refresh balance after chat
      authStore.refreshUser().catch(() => {})
    },
    (err: Error) => {
      isStreaming.value = false
      abortController = null
      // Remove empty assistant message on error
      if (messages.value[msgIndex]?.content === '') {
        messages.value.splice(msgIndex, 1)
      }
      appStore.showError(err.message || t('chat.errorStream'))
    }
  )
}

onMounted(() => {
  loadModels()
  // Pre-select model from query param
  const qModel = route.query.model as string | undefined
  if (qModel) {
    selectedModel.value = qModel
  }
})
</script>

<style scoped>
/* ============================================================
   Chat Theme CSS Variables - Light Mode
   ============================================================ */
.chat-root {
  --chat-bg-primary: #f9fafb;
  --chat-bg-secondary: #ffffff;
  --chat-bg-tertiary: #f3f4f6;
  --chat-bg-input: #ffffff;
  --chat-bg-code: rgba(0, 0, 0, 0.06);
  --chat-text-primary: #111827;
  --chat-text-secondary: #6b7280;
  --chat-text-tertiary: #9ca3af;
  --chat-accent-primary: #14b8a6;
  --chat-accent-secondary: #0d9488;
  --chat-border-color: #e5e7eb;
  --chat-border-subtle: #f3f4f6;
  --chat-radius-md: 12px;
  --chat-shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.05);
  --chat-header-height: 56px;
  --chat-header-bg: rgba(249, 250, 251, 0.85);

  min-height: 100vh;
  background: var(--chat-bg-primary);
  color: var(--chat-text-primary);
  font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', 'Segoe UI', 'PingFang SC', 'Noto Sans SC', sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

.chat-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  width: 100%;
}

/* ============================================================
   Header
   ============================================================ */
.chat-header {
  position: sticky;
  top: 0;
  z-index: 30;
  height: var(--chat-header-height);
  background: var(--chat-header-bg);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border-bottom: 1px solid var(--chat-border-subtle);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  flex-shrink: 0;
}

.chat-header-left,
.chat-header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.chat-back-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  color: var(--chat-text-secondary);
  text-decoration: none;
  transition: all 150ms ease;
}

.chat-back-btn:hover {
  background: var(--chat-bg-tertiary);
  color: var(--chat-text-primary);
}

.chat-balance {
  font-size: 13px;
  font-weight: 600;
  color: var(--chat-text-secondary);
  padding: 4px 10px;
  background: var(--chat-bg-tertiary);
  border-radius: 999px;
}

.chat-new-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border-radius: 8px;
  border: 1px solid var(--chat-border-color);
  background: var(--chat-bg-secondary);
  color: var(--chat-text-primary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 150ms ease;
}

.chat-new-btn:hover {
  border-color: var(--chat-accent-primary);
  color: var(--chat-accent-primary);
}

/* ============================================================
   Messages Area
   ============================================================ */
.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 24px 0;
}

.chat-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  text-align: center;
  padding: 40px;
}

.chat-empty-icon {
  width: 56px;
  height: 56px;
  color: var(--chat-text-tertiary);
  margin-bottom: 12px;
}

.chat-empty-text {
  font-size: 15px;
  color: var(--chat-text-secondary);
}

/* ============================================================
   Typing Indicator
   ============================================================ */
.chat-typing {
  display: flex;
  padding: 0 16px;
  margin-bottom: 16px;
}

.chat-typing-bubble {
  display: flex;
  gap: 4px;
  align-items: center;
  padding: 12px 16px;
  background: var(--chat-bg-secondary);
  border-radius: var(--chat-radius-md);
  border-bottom-left-radius: 4px;
}

.chat-typing-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--chat-text-tertiary);
  animation: chat-dot-bounce 1.4s ease-in-out infinite;
}

.chat-typing-dot:nth-child(2) {
  animation-delay: 0.2s;
}

.chat-typing-dot:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes chat-dot-bounce {
  0%, 60%, 100% { transform: translateY(0); opacity: 0.4; }
  30% { transform: translateY(-4px); opacity: 1; }
}

/* ============================================================
   Responsive
   ============================================================ */
@media (max-width: 640px) {
  .chat-header {
    padding: 0 10px;
  }

  .chat-new-btn-text {
    display: none;
  }
}

/* ============================================================
   Scrollbar
   ============================================================ */
.chat-root ::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

.chat-root ::-webkit-scrollbar-track {
  background: transparent;
}

.chat-root ::-webkit-scrollbar-thumb {
  background: var(--chat-bg-tertiary);
  border-radius: 3px;
}

.chat-root ::-webkit-scrollbar-thumb:hover {
  background: var(--chat-text-tertiary);
}
</style>

<style>
/* Dark Mode */
html.dark .chat-root {
  --chat-bg-primary: #020617;
  --chat-bg-secondary: #0f172a;
  --chat-bg-tertiary: #1e293b;
  --chat-bg-input: #0f172a;
  --chat-bg-code: rgba(255, 255, 255, 0.08);
  --chat-text-primary: #f1f5f9;
  --chat-text-secondary: #94a3b8;
  --chat-text-tertiary: #64748b;
  --chat-border-color: #334155;
  --chat-border-subtle: #1e293b;
  --chat-shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.3);
  --chat-header-bg: rgba(2, 6, 23, 0.85);
}
</style>
