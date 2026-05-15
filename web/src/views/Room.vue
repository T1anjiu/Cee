<template>
  <n-modal v-model:show="showNicknameModal" :mask-closable="false" preset="dialog" title="加入房间" positive-text="加入" @positive-click="confirmNickname">
    <n-input v-model:value="nicknameInput" placeholder="输入你的昵称" size="large" @keyup.enter="confirmNickname" />
  </n-modal>

  <n-layout style="height:100vh;background:#0c0c14;">
    <!-- Top Bar -->
    <n-layout-header style="display:flex;align-items:center;justify-content:space-between;padding:0 20px;height:46px;background:rgba(12,12,20,0.85);backdrop-filter:blur(12px);z-index:100;border-bottom:1px solid rgba(255,255,255,0.04);">
      <n-space align="center" :size="8">
        <span style="font-weight:700;font-size:15px;background:linear-gradient(135deg,#6366f1,#a5b4fc);-webkit-background-clip:text;-webkit-text-fill-color:transparent;letter-spacing:0.5px;">Cee</span>
        <span style="color:rgba(255,255,255,0.12);font-weight:200;font-size:14px;">/</span>
        <span style="font-weight:500;font-size:14px;letter-spacing:2px;color:#e8e8f0;">{{ roomStore.roomId }}</span>
        <div :style="{ width:6, height:6, borderRadius:3, background: roomStore.connected ? '#34d399' : '#f87171', transition:'background 0.3s' }"></div>
      </n-space>
      <n-space :size="6">
        <n-button text size="tiny" depth="3" @click="forceDesktop" v-if="isRemoteMode">完整视图</n-button>
        <n-button text size="tiny" depth="3" @click="forceMobile" v-if="!isRemoteMode && isMobile">遥控模式</n-button>
      </n-space>
    </n-layout-header>

    <!-- Remote Mode -->
    <template v-if="isRemoteMode">
      <n-layout-content style="padding:24px;max-width:420px;margin:0 auto;">
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
      </n-layout-content>
    </template>

    <!-- Full Mode -->
    <template v-else>
      <n-layout-content style="padding:12px;" :native-scrollbar="false">
        <div class="room-grid">
          <!-- Left: Player Column -->
          <div class="player-col">
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

            <!-- Controls -->
            <div v-if="roomStore.media" class="controls">
              <div class="pr-bar" @click="handleProgressClick">
                <div class="pr-track">
                  <div class="pr-fill" :style="{ width: progressPercent + '%' }"></div>
                  <div class="pr-thumb" :style="{ left: progressPercent + '%' }"></div>
                </div>
              </div>
              <div class="pr-time">
                <span>{{ formatTime(expectedPosition) }}</span>
                <span style="opacity:0.3;">{{ formatTime(roomDuration) }}</span>
              </div>
              <div class="pr-ctrls">
                <n-button circle quaternary size="small" @click="handleSkip(-10)">
                  <template #icon><n-icon :size="16"><svg viewBox="0 0 24 24"><path d="M12 5V1L7 6l5 5V7c3.31 0 6 2.69 6 6s-2.69 6-6 6-6-2.69-6-6H4c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8z"/></svg></n-icon></template>
                </n-button>
                <n-button :type="roomStore.player.playing ? 'warning' : 'primary'" circle style="width:44px;height:44px;" @click="togglePlay">
                  <template #icon><n-icon :size="22"><svg v-if="roomStore.player.playing" viewBox="0 0 24 24"><path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/></svg><svg v-else viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg></n-icon></template>
                </n-button>
                <n-button circle quaternary size="small" @click="handleSkip(10)">
                  <template #icon><n-icon :size="16"><svg viewBox="0 0 24 24"><path d="M4 13c0 4.42 3.58 8 8 8s8-3.58 8-8h-2c0 3.31-2.69 6-6 6s-6-2.69-6-6 2.69-6 6-6v4l5-5-5-5v4c-4.42 0-8 3.58-8 8z"/></svg></n-icon></template>
                </n-button>
              </div>
            </div>

            <MediaInput :send-ws="send" :connected="roomStore.connected" @submit="handleMediaSubmit" @upload-start="handleUploadStart" />
          </div>

          <!-- Right: Sidebar -->
          <div class="sidebar-col">
            <div class="sd-card">
              <div class="sd-title">成员 · {{ roomStore.members.length }}</div>
              <div class="sd-members">
                <div v-for="member in roomStore.members" :key="member.id" class="sd-member">
                  <div class="sd-avatar">{{ member.nickname.charAt(0) }}</div>
                  <span class="sd-name">{{ member.nickname }}</span>
                  <span v-if="member.id === roomStore.selfId" class="sd-badge">我</span>
                </div>
                <n-empty v-if="roomStore.members.length === 0" description="暂无成员" size="small" style="padding:12px 0;" />
              </div>
            </div>

            <!-- Chat -->
            <ChatPanel :messages="roomStore.chatHistory" :connected="roomStore.connected" :self-id="roomStore.selfId" @send="handleChatSend" />
          </div>
        </div>
      </n-layout-content>
    </template>
  </n-layout>
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
const playerAreaRef = ref<any>(null)
const showNicknameModal = ref(false)
const nicknameInput = ref('')

const isMobile = isMobileDevice()
const remoteOverride = ref(!getMobileOverride())
const isRemoteMode = computed(() => !remoteOverride.value && !!(roomStore.media))

function forceMobile() { remoteOverride.value = true; setMobileOverride(true) }
function forceDesktop() { remoteOverride.value = false; setMobileOverride(false) }

let nickname = ''
let pingTimer: ReturnType<typeof setInterval> | null = null

const expectedPosition = computed(() => {
  const p = roomStore.player
  if (!p) return -1
  return p.playing ? p.position + (clockStore.serverNow() - p.updated_at) / 1000 : p.position
})
const roomDuration = computed(() => 0)
const progressPercent = computed(() => {
  const d = roomDuration.value; return (!d || d <= 0) ? 0 : Math.min(100, Math.max(0, (expectedPosition.value / d) * 100))
})

watch(connected, (val) => {
  if (!val) return
  send({ type: 'join', payload: { nickname, member_token: sessionStorage.getItem('member_token') || undefined } })
  clockStore.sendPing(send)
})

onMounted(() => {
  roomStore.roomId = route.params.roomId as string
  nickname = sessionStorage.getItem('nickname') || ''
  nickname ? connectAndJoin() : (showNicknameModal.value = true)
})

function confirmNickname() {
  nickname = nicknameInput.value || '匿名用户'
  sessionStorage.setItem('nickname', nickname)
  showNicknameModal.value = false; connectAndJoin()
}

function connectAndJoin() {
  const wsP = location.protocol === 'https:' ? 'wss' : 'ws'
  connect(`${wsP}://${location.host}/ws/${roomStore.roomId}`)

  const handlers: Array<[string, (m: Message) => void]> = []
  const reg = (t: string, fn: (m: Message) => void) => { on(t, fn); handlers.push([t, fn]) }

  reg('room_state', (m) => { const p = m.payload as unknown as RoomStatePayload; roomStore.setRoomState(p); roomStore.connected = true })
  reg('member_join', (m) => { const p = m.payload as unknown as MemberInfo; roomStore.addMember(p); message.success(`${p.nickname} 加入了房间`) })
  reg('member_resume', (m) => roomStore.addMember(m.payload as unknown as MemberInfo))
  reg('member_leave', (m) => roomStore.removeMember((m.payload as unknown as { id: string }).id))
  reg('chat', (m) => roomStore.addChatMessage(m.payload as unknown as ChatMessage))
  reg('player_update', (m) => { const p = m.payload as unknown as PlayerUpdatePayload; roomStore.player = { playing: p.playing, position: p.position, updated_at: p.updated_at, reason: p.reason } })
  reg('media_update', (m) => { const p = m.payload as unknown as MediaUpdatePayload; roomStore.media = { kind: p.kind, source_url: p.source_url, media_type: p.media_type, title: p.title, status: p.status, uploader_id: p.uploader_id } })
  reg('pong', (m) => clockStore.handlePong(m))
  reg('error', (m) => message.error(`错误: ${(m.payload as unknown as { message: string }).message}`))

  let cleaned = false
  onUnmounted(() => { if (!cleaned) { cleaned = true; handlers.forEach(([t, h]) => off(t, h)) } })

  pingTimer = setInterval(() => { if (clockStore.sampleCount < clockStore.targetSamples) clockStore.sendPing(send) }, 200)
  setTimeout(() => { clearInterval(pingTimer!); pingTimer = setInterval(() => clockStore.sendPing(send), 300000) }, 2000)
}

onUnmounted(() => { if (pingTimer) clearInterval(pingTimer) })

const handleMediaSubmit = (kind: 'url', sourceUrl: string, title: string) => send({ type: 'change_media', payload: { kind, source_url: sourceUrl, title } })
const handleUploadStart = (uploadId: string) => send({ type: 'change_media', payload: { kind: 'upload', upload_id: uploadId } })
const handleChatSend = (text: string) => send({ type: 'chat', payload: { text } })
const togglePlay = () => { const p = roomStore.player; send({ type: p.playing ? 'pause' : 'play', payload: { position: expectedPosition.value } }) }
const handleSkip = (s: number) => send({ type: 'seek', payload: { position: Math.max(0, expectedPosition.value + s) } })
const handleSeek = (pos: number) => send({ type: 'seek', payload: { position: pos } })
const handleProgressClick = (e: MouseEvent) => { const el = e.currentTarget as HTMLElement; const r = el.getBoundingClientRect(); handleSeek(((e.clientX - r.left) / r.width) * roomDuration.value) }
const handleSeekDebounce = (pos: number) => send({ type: 'seek', payload: { position: pos } })
const handleBuffering = (b: boolean) => send({ type: 'buffering', payload: { buffering: b } })
const handleHeartbeat = (pos: number, playing: boolean, duration?: number) => send({ type: 'heartbeat', payload: { position: pos, playing, duration } })
const formatTime = (s: number) => (!s || !isFinite(s)) ? '0:00' : `${Math.floor(s / 60)}:${Math.floor(s % 60).toString().padStart(2, '0')}`
</script>

<style scoped>
.room-grid {
  display: flex; gap: 16px; align-items: flex-start;
}
.player-col { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 8px; }
.sidebar-col { width: 280px; flex-shrink: 0; display: flex; flex-direction: column; gap: 8px; }

/* Controls */
.controls { padding: 12px 14px; background: rgba(20,20,31,0.5); border-radius: 8px; }
.pr-bar { cursor: pointer; padding: 5px 0; }
.pr-track { position: relative; height: 4px; background: rgba(255,255,255,0.06); border-radius: 2px; transition: height 0.12s; }
.pr-bar:hover .pr-track { height: 6px; }
.pr-fill { height: 100%; background: linear-gradient(90deg, #6366f1, #a5b4fc); border-radius: 2px; transition: width 0.1s linear; }
.pr-thumb {
  position: absolute; top: 50%; width: 11px; height: 11px; background: #e8e8f0; border-radius: 50%;
  transform: translate(-50%, -50%); opacity: 0; transition: opacity 0.12s;
  box-shadow: 0 0 0 2px rgba(99,102,241,0.15);
}
.pr-bar:hover .pr-thumb { opacity: 1; }
.pr-time { display: flex; justify-content: space-between; margin-top: 3px; font-size: 11px; color: rgba(255,255,255,0.2); font-variant-numeric: tabular-nums; }
.pr-ctrls { display: flex; align-items: center; justify-content: center; gap: 18px; margin-top: 6px; }

/* Sidebar cards */
.sd-card { background: rgba(20,20,31,0.5); border-radius: 8px; padding: 14px; }
.sd-title { font-size: 10px; font-weight: 600; color: rgba(255,255,255,0.2); letter-spacing: 0.5px; text-transform: uppercase; margin-bottom: 10px; }
.sd-members { display: flex; flex-direction: column; gap: 4px; }
.sd-member { display: flex; align-items: center; gap: 8px; padding: 4px 6px; border-radius: 6px; transition: background 0.12s; }
.sd-member:hover { background: rgba(255,255,255,0.02); }
.sd-avatar { width: 26px; height: 26px; border-radius: 50%; background: linear-gradient(135deg, #6366f1, #a5b4fc); display: flex; align-items: center; justify-content: center; font-size: 10px; font-weight: 700; color: #fff; flex-shrink: 0; }
.sd-name { flex: 1; font-size: 13px; color: #e8e8f0; }
.sd-badge { font-size: 10px; color: rgba(255,255,255,0.15); background: rgba(255,255,255,0.03); padding: 1px 5px; border-radius: 3px; }

@media (max-width: 860px) { .room-grid { flex-direction: column; align-items: stretch; } .sidebar-col { width: auto; } }
</style>
