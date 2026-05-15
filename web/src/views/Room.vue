<template>
  <n-config-provider>
    <n-message-provider>
      <n-modal v-model:show="showNicknameModal" :mask-closable="false" preset="dialog" title="输入昵称">
        <template #default>
          <n-input v-model:value="nicknameInput" placeholder="你的昵称" @keyup.enter="confirmNickname" />
        </template>
        <template #action>
          <n-button type="primary" @click="confirmNickname">加入</n-button>
        </template>
      </n-modal>

      <div class="room-container">
        <n-space vertical :size="16" style="width: 100%; max-width: 900px;">
          <n-card title-placement="left">
            <template #header>
              <n-space>
                <span>房间: {{ roomStore.roomId }}</span>
                <n-tag :type="roomStore.connected ? 'success' : 'error'" size="small">
                  {{ roomStore.connected ? '已连接' : '未连接' }}
                </n-tag>
                <n-tag v-if="clockStore.offset.value !== 0" size="small" type="info">
                  时钟偏差: {{ clockStore.offset.value.toFixed(0) }}ms
                </n-tag>
              </n-space>
            </template>

            <n-space vertical :size="16">
              <template v-if="isRemoteMode">
                <RemoteControl
                  :media-title="roomStore.media?.title || ''"
                  :is-playing="roomStore.player.playing"
                  :current-time="expectedPosition"
                  :duration="0"
                  :progress-percent="0"
                  :messages="roomStore.chatHistory"
                  :members="roomStore.members"
                  :self-id="roomStore.selfId"
                  @toggle-play="togglePlay"
                  @skip="handleSkip"
                  @send-chat="handleChatSend"
                />
                <n-button size="tiny" @click="forceDesktop">切换到完整视图</n-button>
              </template>

              <template v-else>
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

                <MediaInput
                  :send-ws="send"
                  :connected="roomStore.connected"
                  @submit="handleMediaSubmit"
                  @upload-start="handleUploadStart"
                />

                <n-space vertical :size="8">
                  <n-button-group style="width: 100%;">
                    <n-button :type="roomStore.player.playing ? 'warning' : 'primary'" @click="togglePlay" style="flex: 1;">
                      {{ roomStore.player.playing ? '暂停' : '播放' }}
                    </n-button>
                    <n-button @click="handleSkip(-10)" style="flex: 1;">-10s</n-button>
                    <n-button @click="handleSkip(10)" style="flex: 1;">+10s</n-button>
                  </n-button-group>
                </n-space>

                <n-divider title-placement="left">成员 ({{ roomStore.members.length }})</n-divider>

                <n-space wrap>
                  <n-tag v-for="member in roomStore.members" :key="member.id" type="info">
                    {{ member.nickname }}{{ member.id === roomStore.selfId ? ' (我)' : '' }}
                  </n-tag>
                </n-space>

                <n-empty v-if="roomStore.members.length === 0" description="暂无成员" />

                <ChatPanel
                  :messages="roomStore.chatHistory"
                  :connected="roomStore.connected"
                  :self-id="roomStore.selfId"
                  @send="handleChatSend"
                />

                <n-button v-if="isMobile" size="tiny" @click="forceMobile">切换到遥控器模式</n-button>
              </template>
            </n-space>
          </n-card>
        </n-space>
      </div>
    </n-message-provider>
  </n-config-provider>
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

const isMobile = isMobileDevice()
const remoteOverride = ref(!getMobileOverride())

const isRemoteMode = computed(() => {
  return remoteOverride.value && !!(roomStore.media)
})

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

// Send join when WebSocket connects, and on reconnect
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

  register('upload_cancel', (msg: Message) => {})

  register('pong', (msg: Message) => {
    clockStore.handlePong(msg)
  })

  register('error', (msg: Message) => {
    const payload = msg.payload as unknown as { code: string; message: string }
    message.error(`错误: ${payload.message}`)
  })

  // Clean up handlers on component unmount to prevent leaks across reconnects
  const origOnUnmounted = onUnmounted
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
  send({
    type: 'change_media',
    payload: { kind, source_url: sourceUrl, title },
  })
}

function handleUploadStart(uploadId: string) {
  send({
    type: 'change_media',
    payload: { kind: 'upload', upload_id: uploadId },
  })
}

function handleChatSend(text: string) {
  send({
    type: 'chat',
    payload: { text },
  })
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

function handleSeekDebounce(position: number) {
  send({ type: 'seek', payload: { position } })
}

function handleBuffering(buffering: boolean) {
  send({ type: 'buffering', payload: { buffering } })
}

function handleHeartbeat(position: number, playing: boolean, duration?: number) {
  send({
    type: 'heartbeat',
    payload: { position, playing, duration },
  })
}
</script>

<style scoped>
.room-container {
  display: flex;
  justify-content: center;
  padding: 24px;
  background: #f5f5f5;
  min-height: 100vh;
}
</style>
