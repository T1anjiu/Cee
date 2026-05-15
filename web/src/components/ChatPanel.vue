<template>
  <div class="chat-panel">
    <div class="chat-head">
      <n-icon :size="12"><svg viewBox="0 0 24 24"><path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2z"/></svg></n-icon>
      <span>聊天</span>
    </div>
    <div ref="messagesRef" class="chat-body-scroll">
      <n-empty v-if="messages.length === 0" description="暂无消息" size="small" style="padding:32px 0;" />
      <div v-for="(msg, i) in messages" :key="i" class="msg-row" :class="{ mine: msg.sender_id === selfId }">
        <div class="msg-bubble">
          <div class="msg-meta">
            <span class="msg-author">{{ msg.nickname }}</span>
            <span class="msg-ts">{{ formatTime(msg.ts) }}</span>
          </div>
          <div class="msg-body">
            <template v-for="(token, j) in tokenizeChatMessage(msg.text)" :key="j">
              <span v-if="token.type === 'text'">{{ token.value }}</span>
              <a v-else :href="token.value" target="_blank" rel="noopener noreferrer" class="msg-link">{{ token.value }}</a>
            </template>
          </div>
        </div>
      </div>
    </div>
    <div class="chat-foot">
      <n-input v-model:value="input" placeholder="消息" size="small" :disabled="!connected" :maxlength="500" @keyup.enter="sendMessage" />
      <n-button size="small" type="primary" circle :disabled="!connected || !input.trim()" @click="sendMessage">
        <template #icon><n-icon :size="12"><svg viewBox="0 0 24 24"><path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/></svg></n-icon></template>
      </n-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { tokenizeChatMessage } from '../utils/chat'
import type { ChatMessage } from '../types/message'

const props = defineProps<{ messages: ChatMessage[]; connected: boolean; selfId: string }>()
const emit = defineEmits<{ (e: 'send', text: string): void }>()
const msg = useMessage(); const input = ref(''); const messagesRef = ref<HTMLElement | null>(null); let lastSendTime = 0

function sendMessage() {
  const t = input.value.trim(); if (!t) return
  if (t.length > 500) { msg.warning('消息不能超过 500 字符'); return }
  const n = Date.now(); if (n - lastSendTime < 333) { msg.warning('发送太快'); return }
  lastSendTime = n; emit('send', t); input.value = ''
}
function formatTime(ts: number) { const d = new Date(ts); return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}` }
watch(() => props.messages.length, async () => { await nextTick(); if (messagesRef.value) messagesRef.value.scrollTop = messagesRef.value.scrollHeight })
</script>

<style scoped>
.chat-panel {
  display: flex; flex-direction: column; height: 300px;
  background: rgba(20,20,31,0.6); border-radius: 10px;
  border: 1px solid rgba(255,255,255,0.03); overflow: hidden;
}
.chat-head {
  display: flex; align-items: center; gap: 6px;
  padding: 12px 14px; font-size: 11px; font-weight: 600;
  color: rgba(255,255,255,0.25);
  border-bottom: 1px solid rgba(255,255,255,0.04);
  letter-spacing: 0.5px; text-transform: uppercase;
}
.chat-body-scroll { flex: 1; overflow-y: auto; padding: 10px 12px; }
.msg-row { display: flex; margin-bottom: 8px; }
.msg-row.mine { justify-content: flex-end; }
.msg-bubble {
  max-width: 88%; background: rgba(255,255,255,0.03);
  border-radius: 8px; padding: 6px 10px;
}
.mine .msg-bubble { background: rgba(99,102,241,0.1); }
.msg-meta { display: flex; align-items: center; gap: 6px; margin-bottom: 2px; }
.msg-author { font-size: 11px; font-weight: 600; color: #818cf8; }
.msg-ts { font-size: 10px; color: rgba(255,255,255,0.15); }
.msg-body { font-size: 13px; color: rgba(255,255,255,0.8); line-height: 1.5; word-break: break-all; }
.msg-link { color: #6366f1; text-decoration: underline; font-size: 12px; }
.chat-foot {
  display: flex; gap: 6px; padding: 8px 10px;
  border-top: 1px solid rgba(255,255,255,0.04);
  background: rgba(0,0,0,0.1);
}
</style>
