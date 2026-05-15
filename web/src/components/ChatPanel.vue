<template>
  <n-card title="聊天" :bordered="false" size="small" class="chat-card">
    <div class="chat-messages" ref="messagesRef">
      <n-empty v-if="messages.length === 0" description="暂无消息" :size="'small'" />

      <div v-for="(msg, i) in messages" :key="i" class="chat-message">
        <n-space align="baseline" :size="4">
          <n-text strong depth="primary" style="font-size: 13px;">{{ msg.nickname }}</n-text>
          <n-text depth="3" style="font-size: 11px;">
            {{ formatTime(msg.ts) }}
          </n-text>
        </n-space>
        <div class="chat-text">
          <template v-for="(token, j) in tokenizeChatMessage(msg.text)" :key="j">
            <span v-if="token.type === 'text'">{{ token.value }}</span>
            <a
              v-else
              :href="token.value"
              target="_blank"
              rel="noopener noreferrer"
              class="chat-link"
            >{{ token.value }}</a>
          </template>
        </div>
      </div>
    </div>

    <n-space :size="8" style="margin-top: 8px;">
      <n-input
        v-model:value="input"
        placeholder="发送消息"
        :maxlength="500"
        @keyup.enter="sendMessage"
        :disabled="!connected"
        size="small"
      />
      <n-button
        size="small"
        type="primary"
        @click="sendMessage"
        :disabled="!connected || !input.trim()"
      >
        发送
      </n-button>
    </n-space>
  </n-card>
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

const emit = defineEmits<{
  (e: 'send', text: string): void
}>()

const message = useMessage()
const input = ref('')
const messagesRef = ref<HTMLElement | null>(null)
let lastSendTime = 0

function sendMessage() {
  const text = input.value.trim()
  if (!text) return
  if (text.length > 500) {
    message.warning('消息不能超过 500 字符')
    return
  }

  // Rate limiting: 3 messages per second
  const now = Date.now()
  if (now - lastSendTime < 333) {
    message.warning('发送太快，请稍候')
    return
  }
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
  if (messagesRef.value) {
    messagesRef.value.scrollTop = messagesRef.value.scrollHeight
  }
})
</script>

<style scoped>
.chat-card {
  height: 350px;
  display: flex;
  flex-direction: column;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

.chat-message {
  margin-bottom: 8px;
  padding: 4px 8px;
  border-radius: 6px;
}

.chat-message:hover {
  background: #f5f5f5;
}

.chat-text {
  font-size: 13px;
  line-height: 1.5;
  word-break: break-all;
}

.chat-link {
  color: #2080f0;
  text-decoration: underline;
  font-size: 12px;
}

.chat-link:hover {
  color: #4098fc;
}
</style>
