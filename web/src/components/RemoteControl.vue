<template>
  <n-card :bordered="false" size="small" class="remote-card">
    <n-space vertical :size="12">
      <div class="remote-media-info">
        <n-text v-if="mediaTitle" strong>{{ mediaTitle }}</n-text>
        <n-text v-else depth="3">暂无媒体</n-text>
      </div>

      <n-progress
        :percentage="progressPercent"
        :height="6"
        :rail-color="'#e0e0e0'"
        style="cursor: pointer;"
        @click="handleProgressClick"
      />

      <n-space justify="space-between" style="font-size: 12px;">
        <n-text depth="3">{{ formatTime(currentTime) }}</n-text>
        <n-text depth="3">{{ formatTime(duration) }}</n-text>
      </n-space>

      <n-space justify="center" :size="24">
        <n-button circle @click="skipBack">
          <template #icon><n-icon><svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 5V1L7 6l5 5V7c3.31 0 6 2.69 6 6s-2.69 6-6 6-6-2.69-6-6H4c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8z"/></svg></n-icon></template>
        </n-button>
        <n-button circle type="primary" :size="'large'" @click="togglePlay">
          <template #icon>
            <n-icon :size="28">
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path v-if="isPlaying" d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/>
                <path v-else d="M8 5v14l11-7z"/>
              </svg>
            </n-icon>
          </template>
        </n-button>
        <n-button circle @click="skipForward">
          <template #icon><n-icon><svg viewBox="0 0 24 24" fill="currentColor"><path d="M4 13c0 4.42 3.58 8 8 8s8-3.58 8-8h-2c0 3.31-2.69 6-6 6s-6-2.69-6-6 2.69-6 6-6v4l5-5-5-5v4c-4.42 0-8 3.58-8 8z"/></svg></n-icon></template>
        </n-button>
      </n-space>

      <n-divider style="margin: 4px 0;" />

      <div class="chat-section">
        <div class="chat-messages" ref="chatRef">
          <div v-for="(msg, i) in messages" :key="i" class="chat-msg">
            <n-text strong depth="primary" style="font-size: 12px;">{{ msg.nickname }}</n-text>
            <n-text style="font-size: 12px; margin-left: 4px;">{{ msg.text }}</n-text>
          </div>
        </div>
        <n-space :size="4">
          <n-input
            v-model:value="chatInput"
            placeholder="消息"
            size="small"
            :maxlength="500"
            @keyup.enter="sendChat"
          />
          <n-button size="small" type="primary" @click="sendChat" :disabled="!chatInput.trim()">
            发送
          </n-button>
        </n-space>
      </div>

      <n-space justify="space-between">
        <n-tag v-for="m in members" :key="m.id" size="small" type="info">
          {{ m.nickname }}{{ m.id === selfId ? ' (我)' : '' }}
        </n-tag>
      </n-space>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { MemberInfo, ChatMessage } from '../types/message'

const props = defineProps<{
  mediaTitle: string
  isPlaying: boolean
  currentTime: number
  duration: number
  progressPercent: number
  messages: ChatMessage[]
  members: MemberInfo[]
  selfId: string
}>()

const emit = defineEmits<{
  (e: 'toggle-play'): void
  (e: 'skip', seconds: number): void
  (e: 'seek', position: number): void
  (e: 'send-chat', text: string): void
}>()

const chatInput = ref('')
const chatRef = ref<HTMLElement | null>(null)

function togglePlay() { emit('toggle-play') }
function skipBack() { emit('skip', -15) }
function skipForward() { emit('skip', 15) }

function handleProgressClick(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  const rect = el.getBoundingClientRect()
  const pct = (e.clientX - rect.left) / rect.width
  emit('seek', pct * props.duration)
}

function sendChat() {
  const text = chatInput.value.trim()
  if (!text) return
  emit('send-chat', text)
  chatInput.value = ''
}

function formatTime(seconds: number): string {
  if (!seconds || !isFinite(seconds)) return '0:00'
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${s.toString().padStart(2, '0')}`
}
</script>

<style scoped>
.remote-card {
  max-width: 400px;
  margin: 0 auto;
}

.remote-media-info {
  text-align: center;
}

.chat-section {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.chat-messages {
  max-height: 120px;
  overflow-y: auto;
}

.chat-msg {
  padding: 2px 0;
}
</style>
