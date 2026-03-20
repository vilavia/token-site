<template>
  <div class="chat-input-wrap">
    <textarea
      ref="textareaRef"
      v-model="text"
      :placeholder="placeholder"
      :disabled="disabled"
      class="chat-input-textarea"
      rows="1"
      @input="autoResize"
      @keydown="onKeydown"
    ></textarea>
    <button
      class="chat-input-send"
      :disabled="disabled || !text.trim()"
      :title="sendLabel"
      @click="handleSend"
    >
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M6 12L3.269 3.126A59.768 59.768 0 0121.485 12 59.77 59.77 0 013.27 20.876L5.999 12zm0 0h7.5" />
      </svg>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'

defineProps<{
  disabled?: boolean
  placeholder?: string
  sendLabel?: string
}>()

const emit = defineEmits<{
  (e: 'send', text: string): void
}>()

const text = ref('')
const textareaRef = ref<HTMLTextAreaElement | null>(null)

function autoResize() {
  const el = textareaRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 200) + 'px'
}

function handleSend() {
  const trimmed = text.value.trim()
  if (!trimmed) return
  emit('send', trimmed)
  text.value = ''
  nextTick(() => {
    if (textareaRef.value) {
      textareaRef.value.style.height = 'auto'
    }
  })
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}
</script>

<style scoped>
.chat-input-wrap {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  padding: 12px 16px;
  background: var(--chat-bg-secondary, #ffffff);
  border-top: 1px solid var(--chat-border-subtle, #f3f4f6);
}

.chat-input-textarea {
  flex: 1;
  resize: none;
  border: 1px solid var(--chat-border-color, #e5e7eb);
  border-radius: 12px;
  padding: 10px 14px;
  font-size: 14px;
  line-height: 1.5;
  background: var(--chat-bg-input, #ffffff);
  color: var(--chat-text-primary, #111827);
  outline: none;
  transition: border-color 150ms ease;
  max-height: 200px;
  overflow-y: auto;
  font-family: inherit;
}

.chat-input-textarea::placeholder {
  color: var(--chat-text-tertiary, #9ca3af);
}

.chat-input-textarea:focus {
  border-color: var(--chat-accent-primary, #14b8a6);
}

.chat-input-textarea:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.chat-input-send {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  border: none;
  background: var(--chat-accent-primary, #14b8a6);
  color: #fff;
  cursor: pointer;
  flex-shrink: 0;
  transition: opacity 150ms ease, background 150ms ease;
}

.chat-input-send:hover:not(:disabled) {
  background: var(--chat-accent-secondary, #0d9488);
}

.chat-input-send:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
</style>
