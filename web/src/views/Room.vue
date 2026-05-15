<template>
  <div class="room-layout">
    <n-modal v-model:show="showNicknameModal" :mask-closable="false" preset="dialog" title="加入房间" class="nickname-modal">
      <template #default>
        <div style="padding: 8px 0;">
          <n-input v-model:value="nicknameInput" placeholder="输入你的昵称" size="large" @keyup.enter="confirmNickname" />
        </div>
      </template>
      <template #action>
        <n-button type="primary" @click="confirmNickname" style="width: 100%;">加入</n-button>
      </template>
    </n-modal>

    <!-- Top Bar -->
    <header class="topbar">
      <div class="topbar-left">
        <span class="topbar-brand">Cee</span>
        <span class="topbar-sep">/</span>
        <span class="topbar-room">{{ roomStore.roomId }}</span>
        <span class="topbar-dot" :class="{ connected: roomStore.connected }"></span>
      </div>
      <div class="topbar-right">
        <n-button text style="color: rgba(255,255,255,0.4); font-size: 12px;" @click="forceDesktop" v-if="isRemoteMode">
          完整视图
        </n-button>
        <n-button text style="color: rgba(255,255,255,0.4); font-size: 12px;" @click="forceMobile" v-if="!isRemoteMode && isMobile">
          遥控器
        </n-button>
      </div>
    </header>

    <div class="room-body">
      <!-- Remote Mode -->
      <template v-if="isRemoteMode">
        <div class="remote-wrapper">
          <RemoteControl
            :media-title="roomStore.media?.title || ''"
            :is-playing="roomStore.player.playing"
            :current-time="expectedPosition"
            :duration="roomDuration"
            :progress-percent="progressPercent"
            :messages="roomStore.chatHistory"
            :members="roomStore.members"
            :self-id="roomStore.selfId"
            @toggle-play="togglePlay"
            @skip="handleSkip"
            @seek="handleSeek"
            @send-chat="handleChatSend"
          />
        </div>
      </template>

      <!-- Full Mode -->
      <template v-else>
        <div class="main-area">
          <div class="player-section">
            <PlayerArea
              ref="playerAreaRef"
              :media="roomStore.media"
              :player="roomStore.player"
              :sync-diff="expectedPosition"
              :clock-offset="clockStore.offset.value"
              :server-now="clockStore.serverNow"
              @buffering="handleBuffering"
              @heartbeat="handleHeartbeat"
              @seek-debounce="handleSeekDebounce"
            />

            <!-- Progress bar -->
            <div class="progress-bar-wrapper" v-if="roomStore.media">
              <div class="progress-bar" ref="progressBarRef" @click="handleProgressClick">
                <div class="progress-fill" :style="{ width: progressPercent + '%' }"></div>
                <div class="progress-thumb" :style="{ left: progressPercent + '%' }"></div>
              </div>
              <div class="progress-time">
                <span class="time-display">{{ formatTime(expectedPosition) }}</span>
                <span class="time-display dim">{{ formatTime(roomDuration) }}</span>
              </div>
            </div>

            <!-- Controls -->
            <div class="player-controls" v-if="roomStore.media">
              <button class="ctrl-btn" @click="handleSkip(-10)" title="后退 10s">
                <svg viewBox="0 0 24 24" fill="currentColor" width="20" height="20"><path d="M12 5V1L7 6l5 5V7c3.31 0 6 2.69 6 6s-2.69 6-6 6-6-2.69-6-6H4c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8z"/></svg>
              </button>
              <button class="ctrl-btn ctrl-play" @click="togglePlay">
                <svg v-if="roomStore.player.playing" viewBox="0 0 24 24" fill="currentColor" width="28" height="28"><path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/></svg>
                <svg v-else viewBox="0 0 24 24" fill="currentColor" width="28" height="28"><path d="M8 5v14l11-7z"/></svg>
              </button>
              <button class="ctrl-btn" @click="handleSkip(10)" title="前进 10s">
                <svg viewBox="0 0 24 24" fill="currentColor" width="20" height="20"><path d="M4 13c0 4.42 3.58 8 8 8s8-3.58 8-8h-2c0 3.31-2.69 6-6 6s-6-2.69-6-6 2.69-6 6-6v4l5-5-5-5v4c-4.42 0-8 3.58-8 8z"/></svg>
              </button>
            </div>

            <!-- Media Input -->
            <MediaInput
              :send-ws="send"
              :connected="roomStore.connected"
              @submit="handleMediaSubmit"
              @upload-start="handleUploadStart"
              style="margin-top: 8px;"
            />
          </div>

          <div class="sidebar-section">
            <!-- Members -->
            <div class="sidebar-card">
              <div class="sidebar-card-header">
                成员 · {{ roomStore.members.length }}
              </div>
              <div class="member-list">
                <div v-for="member in roomStore.members" :key="member.id" class="member-item">
                  <div class="member-avatar">{{ member.nickname.charAt(0).toUpperCase() }}</div>
                  <span class="member-name">{{ member.nickname }}</span>
                  <span v-if="member.id === roomStore.selfId" class="member-self">我</span>
                </div>
                <div v-if="roomStore.members.length === 0" class="member-empty">暂无成员</div>
              </div>
            </div>

            <!-- Chat -->
            <ChatPanel
              :messages="roomStore.chatHistory"
              :connected="roomStore.connected"
              :self-id="roomStore.selfId"
              @send="handleChatSend"
            />
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useRoomStore } from '../stores/room'
import { useWebSocket } from '../composables/useWebSocket'
import { useClockSync } from '../composables/useClockSync'
import PlayerArea from '../components/PlayerArea.vue'
import MediaInput from '../components/MediaInput.vue'
import ChatPanel from '../components/ChatPanel.vue'
import RemoteControl from '../components/RemoteControl.vue'
import { isMobileDevice, getMobileOverride, setMobileOverride } from '../utils/mobile'
import type { Message, RoomStatePayload, MemberInfo, ChatMessage, PlayerUpdatePayload, MediaUpdatePayload } from '../types/message'

const route = useRoute()
const message = useMessage()
const roomStore = useRoomStore()
const { connect, send, on, off, connected } = useWebSocket()
const clockStore = useClockSync()
const playerAreaRef = ref<InstanceType<typeof PlayerArea> | null>(null)
const showNicknameModal = ref(false)
const nicknameInput = ref('')
const progressBarRef = ref<HTMLElement | null>(null)

const isMobile = isMobileDevice()
const remoteOverride = ref(!getMobileOverride())

const isRemoteMode = computed(() => !remoteOverride.value && !!(roomStore.media))

function forceMobile() {
  remoteOverride.value = true
  setMobileOverride(true)
}

function forceDesktop() {
  remoteOverride.value = false
  setMobileOverride(false)
}

let nickname = ''
let pingTimer: ReturnType<typeof setInterval> | null = null

const expectedPosition = computed(() => {
  const p = roomStore.player
  if (!p) return -1
  if (p.playing) {
    return p.position + (clockStore.serverNow() - p.updated_at) / 1000
  }
  return p.position
})

const roomDuration = computed(() => {
  return 0
})

const progressPercent = computed(() => {
  const dur = roomDuration.value
  if (!dur || dur <= 0) return 0
  const ep = Math.max(0, expectedPosition.value)
  return Math.min(100, (ep / dur) * 100)
})

watch(connected, (val) => {
  if (val) {
    send({
      type: 'join',
      payload: {
        nickname,
        member_token: sessionStorage.getItem('member_token') || undefined,
      },
    })
    clockStore.sendPing(send)
  }
})

onMounted(() => {
  const roomId = route.params.roomId as string
  roomStore.roomId = roomId

  nickname = sessionStorage.getItem('nickname') || ''
  if (!nickname) {
    showNicknameModal.value = true
  } else {
    connectAndJoin()
  }
})

function confirmNickname() {
  nickname = nicknameInput.value || '匿名用户'
  sessionStorage.setItem('nickname', nickname)
  showNicknameModal.value = false
  connectAndJoin()
}

function connectAndJoin() {
  const wsProtocol = location.protocol === 'https:' ? 'wss' : 'ws'
  const wsUrl = `${wsProtocol}://${location.host}/ws/${roomStore.roomId}`
  connect(wsUrl)

  const handlers: Array<[string, (msg: Message) => void]> = []

  function register(type: string, handler: (msg: Message) => void) {
    on(type, handler)
    handlers.push([type, handler])
  }

  register('room_state', (msg: Message) => {
    const payload = msg.payload as unknown as RoomStatePayload
    roomStore.setRoomState(payload)
    roomStore.connected = true
  })

  register('member_join', (msg: Message) => {
    const payload = msg.payload as unknown as MemberInfo
    roomStore.addMember(payload)
    message.success(`${payload.nickname} 加入了房间`)
  })

  register('member_resume', (msg: Message) => {
    const payload = msg.payload as unknown as MemberInfo
    roomStore.addMember(payload)
  })

  register('member_leave', (msg: Message) => {
    const payload = msg.payload as unknown as { id: string }
    roomStore.removeMember(payload.id)
  })

  register('chat', (msg: Message) => {
    const payload = msg.payload as unknown as ChatMessage
    roomStore.addChatMessage(payload)
  })

  register('player_update', (msg: Message) => {
    const payload = msg.payload as unknown as PlayerUpdatePayload
    roomStore.player = {
      playing: payload.playing,
      position: payload.position,
      updated_at: payload.updated_at,
      reason: payload.reason,
    }
  })

  register('media_update', (msg: Message) => {
    const payload = msg.payload as unknown as MediaUpdatePayload
    roomStore.media = {
      kind: payload.kind,
      source_url: payload.source_url,
      media_type: payload.media_type,
      title: payload.title,
      status: payload.status,
      uploader_id: payload.uploader_id,
    }
  })

  register('upload_progress', (msg: Message) => {
    const payload = msg.payload as unknown as { member_id: string; upload_id: string; bytes_uploaded: number; bytes_total: number }
  })

  register('upload_cancel', (_msg: Message) => {})

  register('pong', (msg: Message) => {
    clockStore.handlePong(msg)
  })

  register('error', (msg: Message) => {
    const payload = msg.payload as unknown as { code: string; message: string }
    message.error(`错误: ${payload.message}`)
  })

  let cleaned = false
  onUnmounted(() => {
    if (!cleaned) {
      cleaned = true
      for (const [type, handler] of handlers) {
        off(type, handler)
      }
    }
  })

  pingTimer = setInterval(() => {
    if (clockStore.sampleCount < clockStore.targetSamples) {
      clockStore.sendPing(send)
    }
  }, 200)

  setTimeout(() => {
    clearInterval(pingTimer!)
    pingTimer = null
    pingTimer = setInterval(() => {
      clockStore.sendPing(send)
    }, 300000)
  }, 2000)
}

onUnmounted(() => {
  if (pingTimer) clearInterval(pingTimer)
})

function handleMediaSubmit(kind: 'url', sourceUrl: string, title: string) {
  send({ type: 'change_media', payload: { kind, source_url: sourceUrl, title } })
}

function handleUploadStart(uploadId: string) {
  send({ type: 'change_media', payload: { kind: 'upload', upload_id: uploadId } })
}

function handleChatSend(text: string) {
  send({ type: 'chat', payload: { text } })
}

function togglePlay() {
  const p = roomStore.player
  if (p.playing) {
    send({ type: 'pause', payload: { position: expectedPosition.value } })
  } else {
    send({ type: 'play', payload: { position: expectedPosition.value } })
  }
}

function handleSkip(seconds: number) {
  const pos = Math.max(0, expectedPosition.value + seconds)
  send({ type: 'seek', payload: { position: pos } })
}

function handleSeek(position: number) {
  send({ type: 'seek', payload: { position } })
}

function handleProgressClick(e: MouseEvent) {
  const el = progressBarRef.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  const pct = (e.clientX - rect.left) / rect.width
  const pos = pct * roomDuration.value
  handleSeek(pos)
}

function handleSeekDebounce(position: number) {
  send({ type: 'seek', payload: { position } })
}

function handleBuffering(buffering: boolean) {
  send({ type: 'buffering', payload: { buffering } })
}

function handleHeartbeat(position: number, playing: boolean, duration?: number) {
  send({ type: 'heartbeat', payload: { position, playing, duration } })
}

function formatTime(seconds: number): string {
  if (!seconds || !isFinite(seconds)) return '0:00'
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${s.toString().padStart(2, '0')}`
}
</script>

<style scoped>
.room-layout {
  min-height: 100vh;
  background: #0f0f1a;
  display: flex;
  flex-direction: column;
}

/* Top Bar */
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 24px;
  background: rgba(26, 26, 46, 0.8);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  position: sticky;
  top: 0;
  z-index: 100;
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.topbar-brand {
  font-weight: 700;
  font-size: 16px;
  background: linear-gradient(135deg, #6c5ce7, #a29bfe);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.topbar-sep {
  color: rgba(255, 255, 255, 0.15);
  font-size: 14px;
}

.topbar-room {
  font-size: 15px;
  font-weight: 600;
  color: #e0e0f0;
  letter-spacing: 2px;
}

.topbar-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #e74c3c;
  transition: background 0.3s;
}

.topbar-dot.connected {
  background: #2ecc71;
  box-shadow: 0 0 8px rgba(46, 204, 113, 0.4);
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Body */
.room-body {
  flex: 1;
  padding: 20px;
}

.main-area {
  display: flex;
  gap: 20px;
  max-width: 1200px;
  margin: 0 auto;
  align-items: flex-start;
}

.player-section {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* Progress Bar */
.progress-bar-wrapper {
  padding: 0 4px;
}

.progress-bar {
  position: relative;
  height: 6px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 3px;
  cursor: pointer;
  transition: height 0.15s;
}

.progress-bar:hover {
  height: 8px;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #6c5ce7, #a29bfe);
  border-radius: 3px;
  transition: width 0.1s linear;
}

.progress-thumb {
  position: absolute;
  top: 50%;
  width: 14px;
  height: 14px;
  background: #fff;
  border-radius: 50%;
  transform: translate(-50%, -50%);
  box-shadow: 0 2px 6px rgba(0,0,0,0.3);
  opacity: 0;
  transition: opacity 0.15s;
}

.progress-bar:hover .progress-thumb {
  opacity: 1;
}

.progress-time {
  display: flex;
  justify-content: space-between;
  margin-top: 4px;
}

.time-display {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.4);
  font-variant-numeric: tabular-nums;
}

.time-display.dim {
  color: rgba(255, 255, 255, 0.2);
}

/* Player Controls */
.player-controls {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 24px;
  padding: 8px 0;
}

.ctrl-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: none;
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.7);
  cursor: pointer;
  transition: all 0.2s;
}

.ctrl-btn:hover {
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
}

.ctrl-play {
  width: 56px;
  height: 56px;
  background: linear-gradient(135deg, #6c5ce7, #7d6ef0);
  color: #fff;
  box-shadow: 0 4px 15px rgba(108, 92, 231, 0.3);
}

.ctrl-play:hover {
  transform: scale(1.05);
  box-shadow: 0 6px 20px rgba(108, 92, 231, 0.4);
}

/* Sidebar */
.sidebar-section {
  width: 340px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.sidebar-card {
  background: rgba(26, 26, 46, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.04);
  border-radius: 12px;
  padding: 16px;
}

.sidebar-card-header {
  font-size: 13px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.5);
  margin-bottom: 12px;
  letter-spacing: 0.5px;
}

.member-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.member-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 8px;
  border-radius: 8px;
  transition: background 0.15s;
}

.member-item:hover {
  background: rgba(255, 255, 255, 0.04);
}

.member-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, #6c5ce7, #a29bfe);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 700;
  color: #fff;
  flex-shrink: 0;
}

.member-name {
  flex: 1;
  font-size: 14px;
  color: #e0e0f0;
}

.member-self {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.3);
  background: rgba(255, 255, 255, 0.06);
  padding: 2px 8px;
  border-radius: 4px;
}

.member-empty {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.2);
  text-align: center;
  padding: 16px;
}

/* Remote */
.remote-wrapper {
  max-width: 420px;
  margin: 0 auto;
  padding-top: 20px;
}

@media (max-width: 768px) {
  .main-area {
    flex-direction: column;
  }
  .sidebar-section {
    width: 100%;
  }
  .topbar {
    padding: 10px 16px;
  }
  .room-body {
    padding: 12px;
  }
}
</style>
