<template>
  <div class="home-root">
    <div class="home-bg">
      <div class="glow glow-1"></div>
      <div class="glow glow-2"></div>
    </div>
    <div class="home-center">
      <div class="brand">
        <n-icon :size="44" color="#6366f1">
          <svg viewBox="0 0 48 48">
            <circle cx="24" cy="24" r="22" stroke="currentColor" stroke-width="1.5" fill="none" opacity="0.25"/>
            <path d="M18 16v16l14-8z" fill="currentColor"/>
          </svg>
        </n-icon>
        <div style="margin-top:10px;">
          <span class="logo-text">Cee</span>
        </div>
        <n-text depth="3" style="font-size:13px;letter-spacing:3px;margin-top:3px;display:block;">一起看 · 同步播放</n-text>
      </div>

      <n-card :bordered="false" class="enter-card">
        <n-space vertical :size="18">
          <n-button type="primary" size="large" block :loading="creating" @click="createRoom">
            创建房间
          </n-button>

          <div class="or-line">
            <span>加入已有房间</span>
          </div>

          <n-input
            v-model:value="roomCode"
            placeholder="输入房间码"
            size="large"
            :clearable="true"
            @keyup.enter="joinRoom"
          />

          <n-button
            size="large"
            block
            quaternary
            :disabled="!roomCode.trim()"
            @click="joinRoom"
          >
            加入房间
          </n-button>
        </n-space>
      </n-card>

      <n-text depth="3" style="font-size:11px;letter-spacing:1px;margin-top:28px;opacity:0.4;">轻量级多人同步播放工具</n-text>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { normalizeRoomCode } from '../utils'

const router = useRouter()
const message = useMessage()
const roomCode = ref('')
const creating = ref(false)

async function createRoom() {
  creating.value = true
  try {
    const resp = await fetch('/api/rooms', { method: 'POST' })
    if (!resp.ok) { message.error('创建房间失败'); return }
    const data = await resp.json()
    router.push(`/r/${data.room_id}`)
  } catch { message.error('网络错误') }
  finally { creating.value = false }
}

function joinRoom() {
  const code = normalizeRoomCode(roomCode.value)
  if (!code) { message.warning('请输入有效的房间码'); return }
  router.push(`/r/${code}`)
}
</script>

<style scoped>
.home-root {
  min-height: 100vh; display: flex; align-items: center; justify-content: center;
  position: relative; overflow: hidden;
}
.home-bg { position: fixed; inset: 0; z-index: 0; pointer-events: none; }
.glow {
  position: absolute; border-radius: 50%;
  filter: blur(100px); opacity: 0.08;
}
.glow-1 { width: 500px; height: 500px; background: #6366f1; top: -250px; right: -100px; }
.glow-2 { width: 400px; height: 400px; background: #818cf8; bottom: -200px; left: -100px; }
.home-center {
  position: relative; z-index: 1;
  display: flex; flex-direction: column; align-items: center;
}
.brand { text-align: center; margin-bottom: 36px; }
.logo-text {
  font-size: 38px; font-weight: 700;
  background: linear-gradient(135deg, #6366f1 0%, #a5b4fc 100%);
  -webkit-background-clip: text; -webkit-text-fill-color: transparent;
  letter-spacing: 1px;
}
.enter-card {
  width: 380px; max-width: 90vw;
  background: rgba(20, 20, 31, 0.75) !important;
  backdrop-filter: blur(24px) saturate(1.4);
  padding: 8px 0;
}
.or-line {
  display: flex; align-items: center; gap: 16px;
  color: rgba(255,255,255,0.15); font-size: 12px;
}
.or-line::before, .or-line::after {
  content: ''; flex: 1; height: 1px;
  background: rgba(255,255,255,0.06);
}
</style>
