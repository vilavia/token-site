import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiClient } from '@/api/client'

export interface ChatConversation {
  id: number
  user_id: number
  title: string
  model: string
  created_at: string
  updated_at: string
}

export interface ChatMessageRecord {
  id: number
  conversation_id: number
  role: 'user' | 'assistant' | 'system'
  content: string
  created_at: string
}

export const useChatStore = defineStore('chat', () => {
  const conversations = ref<ChatConversation[]>([])
  const activeId = ref<number | null>(null)
  const messages = ref<ChatMessageRecord[]>([])
  const loading = ref(false)

  async function loadConversations() {
    try {
      const res = await apiClient.get<ChatConversation[]>('/conversations')
      conversations.value = res.data || []
      // If active conversation was deleted, reset
      if (activeId.value && !conversations.value.find(c => c.id === activeId.value)) {
        activeId.value = null
        messages.value = []
      }
    } catch { /* ignore */ }
  }

  async function createConversation(model: string, title?: string): Promise<ChatConversation | null> {
    try {
      const res = await apiClient.post<ChatConversation>('/conversations', {
        model,
        title: title || 'New Chat'
      })
      const conv = res.data
      conversations.value.unshift(conv)
      activeId.value = conv.id
      messages.value = []
      return conv
    } catch { return null }
  }

  async function switchTo(convId: number) {
    activeId.value = convId
    await loadMessages(convId)
  }

  async function loadMessages(convId: number) {
    loading.value = true
    try {
      const res = await apiClient.get<ChatMessageRecord[]>(`/conversations/${convId}/messages`)
      messages.value = res.data || []
    } catch {
      messages.value = []
    } finally {
      loading.value = false
    }
  }

  async function addMessage(convId: number, role: string, content: string): Promise<ChatMessageRecord | null> {
    try {
      const res = await apiClient.post<ChatMessageRecord>(`/conversations/${convId}/messages`, { role, content })
      const msg = res.data
      if (activeId.value === convId) {
        messages.value.push(msg)
      }
      return msg
    } catch { return null }
  }

  // Append to the last message's content locally (for streaming)
  function appendToLastMessage(content: string) {
    if (messages.value.length > 0) {
      const last = messages.value[messages.value.length - 1]
      last.content += content
    }
  }

  // Add empty assistant message locally for streaming
  function addLocalAssistant() {
    messages.value.push({
      id: 0,
      conversation_id: activeId.value || 0,
      role: 'assistant',
      content: '',
      created_at: new Date().toISOString()
    })
  }

  // Remove last message if it's empty (on error)
  function removeLastIfEmpty() {
    if (messages.value.length > 0 && messages.value[messages.value.length - 1].content === '') {
      messages.value.pop()
    }
  }

  // Save the streamed assistant message to backend
  async function saveLastAssistantMessage(convId: number) {
    if (messages.value.length === 0) return
    const last = messages.value[messages.value.length - 1]
    if (last.role !== 'assistant' || !last.content) return
    try {
      await apiClient.post(`/conversations/${convId}/messages`, {
        role: 'assistant',
        content: last.content
      })
    } catch { /* ignore */ }
  }

  async function deleteConversation(convId: number) {
    try {
      await apiClient.delete(`/conversations/${convId}`)
      conversations.value = conversations.value.filter(c => c.id !== convId)
      if (activeId.value === convId) {
        activeId.value = conversations.value[0]?.id || null
        if (activeId.value) {
          await loadMessages(activeId.value)
        } else {
          messages.value = []
        }
      }
    } catch { /* ignore */ }
  }

  async function updateModel(convId: number, model: string) {
    const conv = conversations.value.find(c => c.id === convId)
    if (conv) conv.model = model
    try {
      await apiClient.put(`/conversations/${convId}`, { model, title: conv?.title || '' })
    } catch { /* ignore */ }
  }

  function getActive(): ChatConversation | null {
    if (!activeId.value) return null
    return conversations.value.find(c => c.id === activeId.value) || null
  }

  return {
    conversations,
    activeId,
    messages,
    loading,
    loadConversations,
    createConversation,
    switchTo,
    loadMessages,
    addMessage,
    addLocalAssistant,
    appendToLastMessage,
    removeLastIfEmpty,
    saveLastAssistantMessage,
    deleteConversation,
    updateModel,
    getActive
  }
})
