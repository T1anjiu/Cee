<template>
  <div class="chat-card">
    <div class="chat-header">
      <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14"><path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2z"/></svg>
      <span>聊天</span>
    </div>

    <div class="chat-messages" ref="messagesRef">
      <div v-if="messages.length === 0" class="chat-empty">暂无消息</div>
      <div v-for="(msg, i) in messages" :key="i" class="chat-msg-row" :class="{ self: msg.sender_id === selfId }">
        <div class="msg-bubble">
          <div class="msg-header">
            <span class="msg-nick">{{ msg.nickname }}</span>
            <span class="msg-time">{{ formatTime(msg.ts) }}</span>
          </div>
          <div class="msg-text">
            <template v-for="(token, j) in tokenizeChatMessage(msg.text)" :key="j">
              <span v-if="token.type === 'text'">{{ token.value }}</span>
              <a v-else :href="token.value" target="_blank" rel="noopener noreferrer" class="chat-link">{{ token.value }}</a>
            </template>
          </div>
        </div>
      </div>
    </div>

    <div class="chat-input-area">
      <input
        v-model="input"
        placeholder="发送消息…"
        class="chat-input"
        :disabled="!connected"
        @keyup.enter="sendMessage"
      />
      <button class="chat-send-btn" :disabled="!connected || !input.trim()" @click="sendMessage">
        <svg viewBox="0 0 24 24" fill="currentColor" width="18" height="18"><path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/></svg>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { tokenizeChatMessage } from '../utils/chat'
import type { ChatMessage } from '../types/message'

const props = defineProps<{
  messages: ChatMessage[]
  connected: boolean
  selfId: string
}>()

const emit = defineEmits<{ (e: 'send', text: string): void }>()

const message = useMessage()
const input = ref('')
const messagesRef = ref<HTMLElement | null>(null)
let lastSendTime = 0

function sendMessage() {
  const text = input.value.trim()
  if (!text) return
  if (text.length > 500) { message.warning('消息不能超过 500 字符'); return }
  const now = Date.now()
  if (now - lastSendTime < 333) { message.warning('发送太快，请稍候'); return }
  lastSendTime = now
  emit('send', text)
  input.value = ''
}

function formatTime(ts: number): string {
  const d = new Date(ts)
  return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
}

watch(() => props.messages.length, async () => {
  await nextTick()
  if (messagesRef.value) messagesRef.value.scrollTop = messagesRef.value.scrollHeight
})
</script>

<style scoped>
.chat-card {
  display: flex;
  flex-direction: column;
  background: rgba(26, 26, 46, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.04);
  border-radius: 12px;
  overflow: hidden;
  height: 380px;
}

.chat-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 16px;
  font-size: 13px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.5);
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  letter-spacing: 0.5px;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.chat-empty {
  text-align: center;
  color: rgba(255, 255, 255, 0.15);
  font-size: 13px;
  padding: 32px 0;
}

.chat-msg-row {
  display: flex;
}

.chat-msg-row.self {
  justify-content: flex-end;
}

.msg-bubble {
  max-width: 85%;
  background: rgba(255, 255, 255, 0.04);
  border-radius: 10px;
  padding: 8px 12px;
  border: 1px solid rgba(255, 255, 255, 0.04);
}

.self .msg-bubble {
  background: rgba(108, 92, 231, 0.15);
  border-color: rgba(108, 92, 231, 0.2);
}

.msg-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 3px;
}

.msg-nick {
  font-size: 12px;
  font-weight: 600;
  color: #a29bfe;
}

.self .msg-nick {
  color: #6c5ce7;
}

.msg-time {
  font-size: 10px;
  color: rgba(255, 255, 255, 0.2);
}

.msg-text {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.85);
  line-height: 1.5;
  word-break: break-all;
}

.chat-link {
  color: #6c5ce7;
  text-decoration: underline;
  font-size: 12px;
}

.chat-input-area {
  display: flex;
  gap: 8px;
  padding: 10px 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.04);
  background: rgba(0, 0, 0, 0.15);
}

.chat-input {
  flex: 1;
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(22, 22, 42, 0.8);
  color: #e0e0f0;
  font-size: 13px;
  outline: none;
  transition: border-color 0.2s;
}

.chat-input:focus {
  border-color: #6c5ce7;
}

.chat-input::placeholder {
  color: rgba(255, 255, 255, 0.25);
}

.chat-input:disabled {
  opacity: 0.4;
}

.chat-send-btn {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  border: none;
  background: linear-gradient(135deg, #6c5ce7, #7d6ef0);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: opacity 0.2s;
  flex-shrink: 0;
}

.chat-send-btn:hover:not(:disabled) {
  opacity: 0.85;
}

.chat-send-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}
</style>
