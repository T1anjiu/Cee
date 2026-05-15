<template>
  <div class="remote-wrap">
    <!-- Title -->
    <div class="remote-title">{{ mediaTitle || '暂无媒体' }}</div>

    <!-- Progress -->
    <div class="rp-row" @click="handleProgressClick">
      <div class="rp-track">
        <div class="rp-fill" :style="{ width: safePercent + '%' }"></div>
        <div class="rp-thumb" :style="{ left: safePercent + '%' }"></div>
      </div>
    </div>
    <n-space justify="space-between">
      <n-text depth="3" style="font-size:11px;font-variant-numeric:tabular-nums;">{{ formatTime(currentTime) }}</n-text>
      <n-text depth="3" style="font-size:11px;font-variant-numeric:tabular-nums;">{{ formatTime(duration) }}</n-text>
    </n-space>

    <!-- Controls -->
    <n-space justify="center" align="center" :size="24">
      <n-button circle quaternary size="small" @click="skipBack">
        <template #icon><n-icon :size="18"><svg viewBox="0 0 24 24"><path d="M12 5V1L7 6l5 5V7c3.31 0 6 2.69 6 6s-2.69 6-6 6-6-2.69-6-6H4c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8z"/></svg></n-icon></template>
      </n-button>
      <n-button :type="isPlaying ? 'warning' : 'primary'" circle size="large" style="width:56px;height:56px;" @click="$emit('toggle-play')">
        <template #icon>
          <n-icon :size="28">
            <svg v-if="isPlaying" viewBox="0 0 24 24"><path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/></svg>
            <svg v-else viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg>
          </n-icon>
        </template>
      </n-button>
      <n-button circle quaternary size="small" @click="skipForward">
        <template #icon><n-icon :size="18"><svg viewBox="0 0 24 24"><path d="M4 13c0 4.42 3.58 8 8 8s8-3.58 8-8h-2c0 3.31-2.69 6-6 6s-6-2.69-6-6 2.69-6 6-6v4l5-5-5-5v4c-4.42 0-8 3.58-8 8z"/></svg></n-icon></template>
      </n-button>
    </n-space>

    <n-divider style="margin:4px 0;" />

    <!-- Members -->
    <n-text depth="3" style="font-size:10px;font-weight:600;letter-spacing:0.5px;text-transform:uppercase;display:block;margin-bottom:8px;">成员</n-text>
    <n-space :size="6" wrap style="margin-bottom:12px;">
      <div v-for="m in members" :key="m.id" class="rm-tag">
        <span class="rm-tag-avatar">{{ m.nickname.charAt(0) }}</span>
        <span class="rm-tag-name">{{ m.nickname }}{{ m.id === selfId ? ' (我)' : '' }}</span>
      </div>
    </n-space>

    <!-- Chat -->
    <n-text depth="3" style="font-size:10px;font-weight:600;letter-spacing:0.5px;text-transform:uppercase;display:block;margin-bottom:8px;">聊天</n-text>
    <div ref="chatRef" class="rm-chat-box">
      <div v-for="(msg, i) in messages" :key="i" class="rm-msg">
        <span class="rm-msg-author">{{ msg.nickname }}</span>
        <span class="rm-msg-text">{{ msg.text }}</span>
      </div>
      <n-empty v-if="messages.length === 0" description="暂无消息" size="small" style="padding:12px 0;" />
    </div>
    <n-space :size="6" style="margin-top:8px;">
      <n-input v-model:value="chatInput" placeholder="消息" size="small" @keyup.enter="sendChat" />
      <n-button size="small" type="primary" circle :disabled="!chatInput.trim()" @click="sendChat">
        <template #icon><n-icon :size="12"><svg viewBox="0 0 24 24"><path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/></svg></n-icon></template>
      </n-button>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { MemberInfo, ChatMessage } from '../types/message'

const props = defineProps<{ mediaTitle: string; isPlaying: boolean; currentTime: number; duration: number; progressPercent: number; messages: ChatMessage[]; members: MemberInfo[]; selfId: string }>()
const emit = defineEmits<{ (e: 'toggle-play'): void; (e: 'skip', seconds: number): void; (e: 'seek', position: number): void; (e: 'send-chat', text: string): void }>()
const chatInput = ref(''); const chatRef = ref<HTMLElement | null>(null)
const safePercent = computed(() => { const d = props.duration; return (!d || d <= 0) ? 0 : Math.min(100, Math.max(0, (props.currentTime / d) * 100)) })
function skipBack() { emit('skip', -15) }; function skipForward() { emit('skip', 15) }
function handleProgressClick(e: MouseEvent) { const el = e.currentTarget as HTMLElement; const r = el.getBoundingClientRect(); emit('seek', ((e.clientX - r.left) / r.width) * props.duration) }
function sendChat() { const t = chatInput.value.trim(); if (t) { emit('send-chat', t); chatInput.value = '' } }
function formatTime(s: number) { return (!s || !isFinite(s)) ? '0:00' : `${Math.floor(s / 60)}:${Math.floor(s % 60).toString().padStart(2, '0')}` }
</script>

<style scoped>
.remote-wrap {
  background: rgba(20,20,31,0.8); border: 1px solid rgba(255,255,255,0.04);
  border-radius: 12px; padding: 20px;
  display: flex; flex-direction: column; gap: 14px;
  backdrop-filter: blur(24px) saturate(1.4);
}
.remote-title { font-size: 15px; font-weight: 600; color: #e8e8f0; text-align: center; }
.rp-row { cursor: pointer; padding: 4px 0; }
.rp-track { position: relative; height: 4px; background: rgba(255,255,255,0.06); border-radius: 2px; }
.rp-fill { height: 100%; background: linear-gradient(90deg, #6366f1, #a5b4fc); border-radius: 2px; transition: width 0.1s linear; }
.rp-thumb { position: absolute; top: 50%; width: 12px; height: 12px; background: #e8e8f0; border-radius: 50%; transform: translate(-50%, -50%); box-shadow: 0 0 0 3px rgba(99,102,241,0.15); }
.rm-tag { display: inline-flex; align-items: center; gap: 4px; padding: 2px 8px 2px 2px; background: rgba(255,255,255,0.04); border-radius: 14px; font-size: 11px; }
.rm-tag-avatar { width: 20px; height: 20px; border-radius: 50%; background: linear-gradient(135deg, #6366f1, #a5b4fc); display: flex; align-items: center; justify-content: center; font-size: 9px; font-weight: 700; color: #fff; }
.rm-tag-name { color: rgba(255,255,255,0.6); }
.rm-chat-box { max-height: 100px; overflow-y: auto; display: flex; flex-direction: column; gap: 4px; }
.rm-msg { font-size: 12px; line-height: 1.5; }
.rm-msg-author { font-weight: 600; color: #818cf8; margin-right: 4px; }
.rm-msg-text { color: rgba(255,255,255,0.7); }
</style>
