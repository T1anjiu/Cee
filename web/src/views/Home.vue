<template>
  <div class="home-container">
    <div class="home-bg">
      <div class="bg-orb orb-1"></div>
      <div class="bg-orb orb-2"></div>
      <div class="bg-orb orb-3"></div>
    </div>
    <div class="home-content">
      <div class="logo-area">
        <div class="logo-icon">
          <svg viewBox="0 0 48 48" fill="none" width="56" height="56">
            <circle cx="24" cy="24" r="22" stroke="#6c5ce7" stroke-width="2.5" fill="rgba(108,92,231,0.1)"/>
            <path d="M18 16v16l14-8z" fill="#6c5ce7"/>
          </svg>
        </div>
        <h1 class="logo-title">Cee</h1>
        <p class="logo-subtitle">一起看 · 同步播放</p>
      </div>

      <div class="home-card">
        <div class="card-inner">
          <button class="btn btn-primary btn-block" @click="createRoom" :disabled="creating">
            <svg viewBox="0 0 24 24" fill="currentColor" width="20" height="20" style="margin-right: 8px;">
              <path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
            </svg>
            {{ creating ? '创建中...' : '创建房间' }}
          </button>

          <div class="divider">
            <span>或加入已有房间</span>
          </div>

          <div class="input-group">
            <input
              v-model="roomCode"
              placeholder="输入房间码"
              class="input"
              @keyup.enter="joinRoom"
            />
            <button class="btn btn-secondary btn-block" @click="joinRoom" :disabled="!roomCode.trim()">
              加入房间
            </button>
          </div>
        </div>
      </div>

      <div class="footer-info">
        <span>轻量级多人同步播放工具</span>
      </div>
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
    if (!resp.ok) {
      message.error('创建房间失败')
      return
    }
    const data = await resp.json()
    router.push(`/r/${data.room_id}`)
  } catch {
    message.error('网络错误')
  } finally {
    creating.value = false
  }
}

function joinRoom() {
  const code = normalizeRoomCode(roomCode.value)
  if (!code) {
    message.warning('请输入有效的房间码')
    return
  }
  router.push(`/r/${code}`)
}
</script>

<style scoped>
.home-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.home-bg {
  position: fixed;
  inset: 0;
  z-index: 0;
}

.bg-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.15;
}

.orb-1 {
  width: 600px;
  height: 600px;
  background: #6c5ce7;
  top: -200px;
  right: -200px;
}

.orb-2 {
  width: 500px;
  height: 500px;
  background: #a29bfe;
  bottom: -150px;
  left: -150px;
}

.orb-3 {
  width: 300px;
  height: 300px;
  background: #fd79a8;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
}

.home-content {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 32px;
  padding: 24px;
  width: 100%;
  max-width: 420px;
}

.logo-area {
  text-align: center;
}

.logo-icon {
  margin-bottom: 16px;
  animation: float 3s ease-in-out infinite;
}

@keyframes float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-6px); }
}

.logo-title {
  font-size: 36px;
  font-weight: 700;
  background: linear-gradient(135deg, #6c5ce7, #a29bfe);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  letter-spacing: 2px;
}

.logo-subtitle {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.4);
  margin-top: 6px;
  letter-spacing: 4px;
}

.home-card {
  width: 100%;
  background: rgba(26, 26, 46, 0.8);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 16px;
  padding: 32px;
  box-shadow: 0 8px 40px rgba(0, 0, 0, 0.3);
}

.card-inner {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 12px 24px;
  border-radius: 10px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
  outline: none;
  letter-spacing: 0.5px;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-block {
  width: 100%;
}

.btn-primary {
  background: linear-gradient(135deg, #6c5ce7, #7d6ef0);
  color: #fff;
  box-shadow: 0 4px 15px rgba(108, 92, 231, 0.3);
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 6px 20px rgba(108, 92, 231, 0.4);
}

.btn-primary:active:not(:disabled) {
  transform: translateY(0);
}

.btn-secondary {
  background: rgba(255, 255, 255, 0.06);
  color: #e0e0f0;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.btn-secondary:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.1);
  border-color: rgba(255, 255, 255, 0.15);
}

.divider {
  display: flex;
  align-items: center;
  gap: 16px;
  color: rgba(255, 255, 255, 0.25);
  font-size: 13px;
}

.divider::before,
.divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: rgba(255, 255, 255, 0.08);
}

.input-group {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.input {
  width: 100%;
  padding: 12px 16px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(22, 22, 42, 0.8);
  color: #e0e0f0;
  font-size: 15px;
  outline: none;
  transition: border-color 0.2s;
}

.input:focus {
  border-color: #6c5ce7;
}

.input::placeholder {
  color: rgba(255, 255, 255, 0.3);
}

.footer-info {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.2);
  letter-spacing: 1px;
}
</style>
