<template>
  <n-config-provider>
    <n-message-provider>
      <div class="home-container">
        <n-card title="Cee - 一起看" class="home-card">
          <n-space vertical :size="24">
            <n-button type="primary" size="large" block @click="createRoom" :loading="creating">
              创建房间
            </n-button>

            <n-divider>或加入已有房间</n-divider>

            <n-input
              v-model:value="roomCode"
              placeholder="输入房间码"
              size="large"
              @keyup.enter="joinRoom"
            />

            <n-button type="info" size="large" block @click="joinRoom" :disabled="!roomCode.trim()">
              加入房间
            </n-button>
          </n-space>
        </n-card>
      </div>
    </n-message-provider>
  </n-config-provider>
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
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: #f5f5f5;
}

.home-card {
  width: 400px;
  max-width: 90vw;
}
</style>
