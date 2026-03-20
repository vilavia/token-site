<template>
  <AppLayout>
    <div class="chat-container">
      <!-- Top bar -->
      <div class="chat-topbar">
        <div class="chat-topbar-left">
          <button class="chat-sidebar-toggle" @click="showSidebar = !showSidebar">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" />
            </svg>
          </button>
          <ModelSelector
            v-model="currentModel"
            :models="models"
            :placeholder="t('chat.selectModel')"
            @update:modelValue="onModelChange"
          />
        </div>
        <div class="chat-topbar-right">
          <span class="chat-balance">${{ balance }}</span>
          <button class="chat-new-btn" @click="newChat">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
            <span class="chat-new-btn-text">{{ t('chat.newChat') }}</span>
          </button>
        </div>
      </div>

      <div class="chat-body">
        <!-- Conversation sidebar -->
        <aside v-if="showSidebar" class="chat-sidebar">
          <div class="chat-sidebar-list">
            <div
              v-for="conv in chatStore.conversations"
              :key="conv.id"
              class="chat-sidebar-item"
              :class="{ active: conv.id === chatStore.activeId }"
              @click="switchConversation(conv.id)"
            >
              <div class="chat-sidebar-item-info">
                <span class="chat-sidebar-item-title">{{ conv.title }}</span>
                <span class="chat-sidebar-item-model">{{ conv.model || 'No model' }}</span>
              </div>
              <button class="chat-sidebar-item-delete" @click.stop="deleteConversation(conv.id)">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div v-if="chatStore.conversations.length === 0" class="chat-sidebar-empty">
              {{ t('chat.noHistory') }}
            </div>
          </div>
        </aside>

        <!-- Messages area -->
        <div class="chat-main">
          <div ref="messagesContainer" class="chat-messages">
            <div v-if="chatStore.loading" class="chat-loading">
              <svg class="animate-spin" width="24" height="24" viewBox="0 0 24 24" fill="none">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
              </svg>
            </div>
            <div v-else-if="chatStore.messages.length === 0 && !isStreaming" class="chat-empty">
              <svg class="chat-empty-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z" />
              </svg>
              <p class="chat-empty-text">{{ t('chat.startPrompt') }}</p>
            </div>
            <template v-else>
              <ChatMessage v-for="(msg, idx) in chatStore.messages" :key="idx" :message="msg" />
            </template>

            <div v-if="isStreaming && !lastAssistantContent" class="chat-typing">
              <div class="chat-typing-bubble">
                <span class="chat-typing-dot"></span>
                <span class="chat-typing-dot"></span>
                <span class="chat-typing-dot"></span>
              </div>
            </div>
          </div>

          <ChatInput
            :disabled="isStreaming || !currentModel"
            :placeholder="t('chat.placeholder')"
            :send-label="t('chat.send')"
            @send="handleSend"
          />
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useChatStore } from '@/stores/chat'
import { getModels, streamChat } from '@/api/chat'
import AppLayout from '@/components/layout/AppLayout.vue'
import ChatMessage from '@/components/chat/ChatMessage.vue'
import ChatInput from '@/components/chat/ChatInput.vue'
import ModelSelector from '@/components/chat/ModelSelector.vue'

const { t } = useI18n()
const route = useRoute()
const authStore = useAuthStore()
const appStore = useAppStore()
const chatStore = useChatStore()

const models = ref<string[]>([])
const isStreaming = ref(false)
const messagesContainer = ref<HTMLElement | null>(null)
const showSidebar = ref(true)
let abortController: AbortController | null = null

const balance = computed(() => authStore.user?.balance?.toFixed(2) || '0.00')

const currentModel = computed({
  get: () => chatStore.getActive()?.model || '',
  set: (val: string) => {
    const active = chatStore.getActive()
    if (active) chatStore.updateModel(active.id, val)
  }
})

const lastAssistantContent = computed(() => {
  const msgs = chatStore.messages
  if (msgs.length === 0) return ''
  const last = msgs[msgs.length - 1]
  return last.role === 'assistant' ? last.content : ''
})

function scrollToBottom() {
  nextTick(() => {
    const el = messagesContainer.value
    if (el) el.scrollTop = el.scrollHeight
  })
}

watch(() => chatStore.messages.length, () => scrollToBottom())

async function loadModels() {
  try {
    const res = await getModels()
    models.value = res.models || []
  } catch (err) {
    console.error('Failed to load models:', err)
  }
}

function onModelChange(model: string) {
  const active = chatStore.getActive()
  if (active) chatStore.updateModel(active.id, model)
}

async function newChat() {
  if (abortController) { abortController.abort(); abortController = null }
  isStreaming.value = false
  const model = currentModel.value || models.value[0] || ''
  await chatStore.createConversation(model)
}

async function switchConversation(id: number) {
  if (isStreaming.value && abortController) {
    abortController.abort(); abortController = null; isStreaming.value = false
  }
  await chatStore.switchTo(id)
  scrollToBottom()
}

async function deleteConversation(id: number) {
  await chatStore.deleteConversation(id)
}

async function handleSend(text: string) {
  if (!currentModel.value || isStreaming.value) return

  let conv = chatStore.getActive()
  if (!conv) {
    conv = await chatStore.createConversation(currentModel.value)
    if (!conv) return
  }

  // Save user message to backend
  await chatStore.addMessage(conv.id, 'user', text)

  // Add empty local assistant message for streaming
  chatStore.addLocalAssistant()
  isStreaming.value = true
  scrollToBottom()

  const convId = conv.id

  abortController = streamChat(
    {
      model: currentModel.value,
      messages: chatStore.messages.slice(0, -1).map(m => ({ role: m.role, content: m.content }))
    },
    (chunk: string) => {
      chatStore.appendToLastMessage(chunk)
      scrollToBottom()
    },
    async () => {
      isStreaming.value = false
      abortController = null
      // Save streamed assistant response to backend
      await chatStore.saveLastAssistantMessage(convId)
      // Refresh conversation list to update title
      await chatStore.loadConversations()
      authStore.refreshUser().catch(() => {})
    },
    (err: Error) => {
      isStreaming.value = false
      abortController = null
      chatStore.removeLastIfEmpty()
      appStore.showError(err.message || t('chat.errorStream'))
    }
  )
}

onMounted(async () => {
  await loadModels()
  await chatStore.loadConversations()

  const qModel = route.query.model as string | undefined

  if (chatStore.conversations.length === 0) {
    await chatStore.createConversation(qModel || models.value[0] || '')
  } else {
    // Resume last conversation
    if (!chatStore.activeId && chatStore.conversations.length > 0) {
      await chatStore.switchTo(chatStore.conversations[0].id)
    } else if (chatStore.activeId) {
      await chatStore.loadMessages(chatStore.activeId)
    }
    if (qModel && chatStore.getActive()) {
      chatStore.updateModel(chatStore.getActive()!.id, qModel)
    }
  }
  scrollToBottom()
})
</script>

<style scoped>
.chat-container {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 64px);
  max-height: calc(100vh - 64px);
  margin: -24px;
}

.chat-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  border-bottom: 1px solid var(--color-border, #e5e7eb);
  background: var(--color-bg-primary, #fff);
  flex-shrink: 0;
}
.chat-topbar-left, .chat-topbar-right { display: flex; align-items: center; gap: 10px; }

.chat-sidebar-toggle {
  display: flex; align-items: center; justify-content: center;
  width: 32px; height: 32px; border-radius: 6px; border: none;
  background: none; color: #6b7280; cursor: pointer;
}
.chat-sidebar-toggle:hover { background: #f3f4f6; color: #111827; }

.chat-balance {
  font-size: 13px; font-weight: 600; color: #6b7280;
  padding: 4px 10px; background: #f3f4f6; border-radius: 999px;
}
.chat-new-btn {
  display: flex; align-items: center; gap: 6px; padding: 6px 12px;
  border-radius: 8px; border: 1px solid #e5e7eb; background: #fff;
  color: #111827; font-size: 13px; font-weight: 500; cursor: pointer;
}
.chat-new-btn:hover { border-color: #14b8a6; color: #14b8a6; }

.chat-body { display: flex; flex: 1; overflow: hidden; }

/* Sidebar */
.chat-sidebar {
  width: 240px; flex-shrink: 0;
  border-right: 1px solid #e5e7eb; background: #f9fafb; overflow-y: auto;
}
.chat-sidebar-list { padding: 8px; }
.chat-sidebar-item {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 10px; border-radius: 8px; cursor: pointer; margin-bottom: 2px;
}
.chat-sidebar-item:hover { background: #e5e7eb; }
.chat-sidebar-item.active { background: #dbeafe; }

.chat-sidebar-item-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.chat-sidebar-item-title {
  font-size: 13px; color: #374151;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.chat-sidebar-item.active .chat-sidebar-item-title { color: #1d4ed8; font-weight: 500; }

.chat-sidebar-item-model {
  font-size: 11px; color: #9ca3af;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.chat-sidebar-item.active .chat-sidebar-item-model { color: #60a5fa; }

.chat-sidebar-item-delete {
  display: none; align-items: center; justify-content: center;
  width: 20px; height: 20px; border: none; background: none;
  color: #9ca3af; cursor: pointer; border-radius: 4px; flex-shrink: 0;
}
.chat-sidebar-item:hover .chat-sidebar-item-delete { display: flex; }
.chat-sidebar-item-delete:hover { color: #ef4444; background: #fee2e2; }

.chat-sidebar-empty { padding: 20px 10px; text-align: center; font-size: 13px; color: #9ca3af; }

/* Main */
.chat-main { flex: 1; display: flex; flex-direction: column; overflow: hidden; min-width: 0; }
.chat-messages { flex: 1; overflow-y: auto; padding: 24px 0; }
.chat-loading { display: flex; justify-content: center; padding: 40px; color: #9ca3af; }
.chat-empty { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100%; padding: 40px; }
.chat-empty-icon { width: 56px; height: 56px; color: #9ca3af; margin-bottom: 12px; }
.chat-empty-text { font-size: 15px; color: #6b7280; }

.chat-typing { display: flex; padding: 0 16px; margin-bottom: 16px; }
.chat-typing-bubble { display: flex; gap: 4px; align-items: center; padding: 12px 16px; background: #f9fafb; border-radius: 12px; border-bottom-left-radius: 4px; }
.chat-typing-dot { width: 6px; height: 6px; border-radius: 50%; background: #9ca3af; animation: dot-bounce 1.4s ease-in-out infinite; }
.chat-typing-dot:nth-child(2) { animation-delay: 0.2s; }
.chat-typing-dot:nth-child(3) { animation-delay: 0.4s; }
@keyframes dot-bounce {
  0%, 60%, 100% { transform: translateY(0); opacity: 0.4; }
  30% { transform: translateY(-4px); opacity: 1; }
}

@media (max-width: 768px) { .chat-sidebar { width: 200px; } .chat-new-btn-text { display: none; } }
@media (max-width: 640px) {
  .chat-sidebar { position: absolute; z-index: 20; left: 0; top: 0; bottom: 0; width: 260px; box-shadow: 2px 0 8px rgba(0,0,0,0.1); }
}
</style>

<style>
html.dark .chat-topbar { background: #0f172a; border-color: #1e293b; }
html.dark .chat-sidebar-toggle:hover { background: #1e293b; color: #f1f5f9; }
html.dark .chat-balance { color: #94a3b8; background: #1e293b; }
html.dark .chat-new-btn { background: #0f172a; border-color: #334155; color: #f1f5f9; }
html.dark .chat-sidebar { background: #020617; border-color: #1e293b; }
html.dark .chat-sidebar-item:hover { background: #1e293b; }
html.dark .chat-sidebar-item.active { background: #1e3a5f; }
html.dark .chat-sidebar-item-title { color: #cbd5e1; }
html.dark .chat-sidebar-item.active .chat-sidebar-item-title { color: #93c5fd; }
html.dark .chat-sidebar-item-model { color: #64748b; }
html.dark .chat-sidebar-item.active .chat-sidebar-item-model { color: #60a5fa; }
html.dark .chat-sidebar-item-delete:hover { color: #f87171; background: #450a0a; }
html.dark .chat-sidebar-empty { color: #64748b; }
html.dark .chat-empty-icon { color: #64748b; }
html.dark .chat-empty-text { color: #94a3b8; }
html.dark .chat-typing-bubble { background: #1e293b; }
html.dark .chat-typing-dot { background: #64748b; }
html.dark .chat-loading { color: #64748b; }
</style>
