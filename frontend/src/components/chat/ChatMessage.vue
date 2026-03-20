<template>
  <div class="chat-msg" :class="[message.role === 'user' ? 'chat-msg--user' : 'chat-msg--assistant']">
    <div class="chat-msg-bubble">
      <!-- User: plain text -->
      <template v-if="message.role === 'user'">
        <span class="chat-msg-text">{{ message.content }}</span>
      </template>
      <!-- Assistant: markdown rendered -->
      <template v-else>
        <div class="chat-msg-markdown" v-html="renderedContent"></div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const props = defineProps<{
  message: { role: string; content: string }
}>()

const renderedContent = computed(() => {
  if (!props.message.content) return ''
  const raw = marked.parse(props.message.content) as string
  return DOMPurify.sanitize(raw)
})
</script>

<style scoped>
.chat-msg {
  display: flex;
  margin-bottom: 16px;
  padding: 0 16px;
}

.chat-msg--user {
  justify-content: flex-end;
}

.chat-msg--assistant {
  justify-content: flex-start;
}

.chat-msg-bubble {
  max-width: 75%;
  padding: 10px 16px;
  border-radius: var(--chat-radius-md, 12px);
  line-height: 1.6;
  font-size: 14px;
  word-break: break-word;
}

.chat-msg--user .chat-msg-bubble {
  background: var(--chat-accent-primary, #14b8a6);
  color: #fff;
  border-bottom-right-radius: 4px;
}

.chat-msg--assistant .chat-msg-bubble {
  background: var(--chat-bg-secondary, #f3f4f6);
  color: var(--chat-text-primary, #111827);
  border-bottom-left-radius: 4px;
}

.chat-msg-text {
  white-space: pre-wrap;
}

/* Markdown typography */
.chat-msg-markdown :deep(p) {
  margin: 0 0 8px;
}

.chat-msg-markdown :deep(p:last-child) {
  margin-bottom: 0;
}

.chat-msg-markdown :deep(pre) {
  background: var(--chat-bg-code, rgba(0, 0, 0, 0.06));
  border-radius: 8px;
  padding: 12px;
  overflow-x: auto;
  margin: 8px 0;
  font-size: 13px;
}

.chat-msg-markdown :deep(code) {
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 13px;
}

.chat-msg-markdown :deep(:not(pre) > code) {
  background: var(--chat-bg-code, rgba(0, 0, 0, 0.06));
  padding: 2px 6px;
  border-radius: 4px;
}

.chat-msg-markdown :deep(ul),
.chat-msg-markdown :deep(ol) {
  padding-left: 20px;
  margin: 8px 0;
}

.chat-msg-markdown :deep(blockquote) {
  border-left: 3px solid var(--chat-border-color, #d1d5db);
  padding-left: 12px;
  margin: 8px 0;
  color: var(--chat-text-secondary, #6b7280);
}

.chat-msg-markdown :deep(table) {
  border-collapse: collapse;
  margin: 8px 0;
  width: 100%;
}

.chat-msg-markdown :deep(th),
.chat-msg-markdown :deep(td) {
  border: 1px solid var(--chat-border-color, #d1d5db);
  padding: 6px 10px;
  text-align: left;
}

.chat-msg-markdown :deep(a) {
  color: var(--chat-accent-primary, #14b8a6);
  text-decoration: underline;
}

@media (max-width: 640px) {
  .chat-msg-bubble {
    max-width: 88%;
  }
}
</style>
