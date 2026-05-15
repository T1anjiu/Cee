<template>
  <div class="remote-card">
    <div class="remote-header">
      <div class="remote-title">{{ mediaTitle || '暂无媒体' }}</div>
    </div>

    <!-- Progress -->
    <div class="remote-progress">
      <div class="rp-bar" @click="handleProgressClick">
        <div class="rp-fill" :style="{ width: safePercent + '%' }"></div>
        <div class="rp-thumb" :style="{ left: safePercent + '%' }"></div>
      </div>
      <div class="rp-time">
        <span>{{ formatTime(currentTime) }}</span>
        <span>{{ formatTime(duration) }}</span>
      </div>
    </div>

    <!-- Controls -->
    <div class="remote-controls">
      <button class="rc-btn" @click="skipBack">
        <svg viewBox="0 0 24 24" fill="currentColor" width="22" height="22"><path d="M12 5V1L7 6l5 5V7c3.31 0 6 2.69 6 6s-2.69 6-6 6-6-2.69-6-6H4c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8z"/></svg>
      </button>
      <button class="rc-btn rc-play" @click="$emit('toggle-play')">
        <svg v-if="isPlaying" viewBox="0 0 24 24" fill="currentColor" width="32" height="32"><path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/></svg>
        <svg v-else viewBox="0 0 24 24" fill="currentColor" width="32" height="32"><path d="M8 5v14l11-7z"/></svg>
      </button>
      <button class="rc-btn" @click="skipForward">
        <svg viewBox="0 0 24 24" fill="currentColor" width="22" height="22"><path d="M4 13c0 4.42 3.58 8 8 8s8-3.58 8-8h-2c0 3.31-2.69 6-6 6s-6-2.69-6-6 2.69-6 6-6v4l5-5-5-5v4c-4.42 0-8 3.58-8 8z"/></svg>
      </button>
    </div>

    <!-- Members -->
    <div class="remote-section">
      <div class="remote-section-title">成员</div>
      <div class="rm-members">
        <div v-for="m in members" :key="m.id" class="rm-member">
          <div class="rm-avatar">{{ m.nickname.charAt(0).toUpperCase() }}</div>
          <span class="rm-name">{{ m.nickname }}{{ m.id === selfId ? ' (我)' : '' }}</span>
        </div>
        <div v-if="members.length === 0" class="rm-empty">暂无成员</div>
      </div>
    </div>

    <!-- Chat -->
    <div class="remote-section">
      <div class="remote-section-title">聊天</div>
      <div class="rm-chat" ref="chatRef">
        <div v-for="(msg, i) in messages" :key="i" class="rm-chat-msg">
          <span class="rm-chat-nick">{{ msg.nickname }}</span>
          <span class="rm-chat-text">{{ msg.text }}</span>
        </div>
        <div v-if="messages.length === 0" class="rm-empty">暂无消息</div>
      </div>
      <div class="rm-chat-input">
        <input v-model="chatInput" placeholder="消息" @keyup.enter="sendChat" />
        <button @click="sendChat" :disabled="!chatInput.trim()">
          <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16"><path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/></svg>
        </button>
      </div>
    </div>
  </div>
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

const safePercent = computed(() => {
  const dur = props.duration
  if (!dur || dur <= 0) return 0
  return Math.min(100, Math.max(0, (props.currentTime / dur) * 100))
})

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
  background: rgba(26, 26, 46, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 16px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  backdrop-filter: blur(20px);
}

.remote-header {
  text-align: center;
}

.remote-title {
  font-size: 16px;
  font-weight: 600;
  color: #e0e0f0;
}

/* Progress */
.remote-progress {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.rp-bar {
  position: relative;
  height: 8px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 4px;
  cursor: pointer;
}

.rp-fill {
  height: 100%;
  background: linear-gradient(90deg, #6c5ce7, #a29bfe);
  border-radius: 4px;
  transition: width 0.1s linear;
}

.rp-thumb {
  position: absolute;
  top: 50%;
  width: 16px;
  height: 16px;
  background: #fff;
  border-radius: 50%;
  transform: translate(-50%, -50%);
  box-shadow: 0 2px 6px rgba(0,0,0,0.3);
}

.rp-time {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.3);
  font-variant-numeric: tabular-nums;
}

/* Controls */
.remote-controls {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 32px;
}

.rc-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  border: none;
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.7);
  cursor: pointer;
  transition: all 0.2s;
}

.rc-btn:hover {
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
}

.rc-play {
  width: 64px;
  height: 64px;
  background: linear-gradient(135deg, #6c5ce7, #7d6ef0);
  color: #fff;
  box-shadow: 0 4px 20px rgba(108, 92, 231, 0.3);
}

.rc-play:hover {
  transform: scale(1.05);
  box-shadow: 0 6px 25px rgba(108, 92, 231, 0.4);
}

/* Sections */
.remote-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.remote-section-title {
  font-size: 12px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.3);
  letter-spacing: 0.5px;
}

.rm-members {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.rm-member {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px 4px 4px;
  background: rgba(255, 255, 255, 0.04);
  border-radius: 20px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.7);
}

.rm-avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: linear-gradient(135deg, #6c5ce7, #a29bfe);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 700;
  color: #fff;
}

.rm-name {
  white-space: nowrap;
}

.rm-chat {
  max-height: 120px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.rm-chat-msg {
  font-size: 13px;
  line-height: 1.5;
}

.rm-chat-nick {
  font-weight: 600;
  color: #a29bfe;
  margin-right: 4px;
}

.rm-chat-text {
  color: rgba(255, 255, 255, 0.75);
}

.rm-empty {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.15);
  text-align: center;
  padding: 8px 0;
}

.rm-chat-input {
  display: flex;
  gap: 6px;
}

.rm-chat-input input {
  flex: 1;
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(22, 22, 42, 0.8);
  color: #e0e0f0;
  font-size: 13px;
  outline: none;
}

.rm-chat-input input:focus {
  border-color: #6c5ce7;
}

.rm-chat-input button {
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
}

.rm-chat-input button:disabled {
  opacity: 0.3;
}
</style>
