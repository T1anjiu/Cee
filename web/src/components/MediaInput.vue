<template>
  <div class="media-card">
    <div class="media-tabs">
      <button class="media-tab" :class="{ active: activeTab === 'url' }" @click="activeTab = 'url'">
        <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16"><path d="M3.9 12c0-1.71 1.39-3.1 3.1-3.1h4V7H7c-2.76 0-5 2.24-5 5s2.24 5 5 5h4v-1.9H7c-1.71 0-3.1-1.39-3.1-3.1zM8 13h8v-2H8v2zm9-6h-4v1.9h4c1.71 0 3.1 1.39 3.1 3.1s-1.39 3.1-3.1 3.1h-4V17h4c2.76 0 5-2.24 5-5s-2.24-5-5-5z"/></svg>
        直链
      </button>
      <button class="media-tab" :class="{ active: activeTab === 'upload' }" @click="activeTab = 'upload'">
        <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16"><path d="M9 16h6v-6h4l-7-7-7 7h4zm-4 2h14v2H5z"/></svg>
        上传
      </button>
    </div>

    <!-- URL Tab -->
    <div v-if="activeTab === 'url'" class="tab-content">
      <input
        v-model="urlInput"
        placeholder="https://example.com/video.mp4"
        class="media-input"
        :disabled="!props.connected"
        @keyup.enter="submitUrl"
      />
      <div v-if="showM3u8Warning" class="m3u8-warning">
        <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/></svg>
        m3u8 仅在源站允许跨域时可用，公网大概率会失败，建议上传本地文件
      </div>
      <button class="action-btn" @click="submitUrl" :disabled="!props.connected || submitting">
        <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16"><path d="M9 16h6v-6h4l-7-7-7 7h4zm-4 2h14v2H5z"/></svg>
        提交
      </button>
      <div v-if="!props.connected" class="info-msg">等待连接服务器…</div>
    </div>

    <!-- Upload Tab -->
    <div v-if="activeTab === 'upload'" class="tab-content">
      <div class="upload-area" @click="triggerFileInput">
        <input type="file" ref="fileInputRef" accept=".mp4,.webm" style="display:none" :disabled="!props.connected" @change="onFileChange" />
        <svg viewBox="0 0 24 24" fill="currentColor" width="28" height="28"><path d="M9 16h6v-6h4l-7-7-7 7h4zm-4 2h14v2H5z"/></svg>
        <span>{{ selectedFile ? selectedFile.name : '选择 .mp4 / .webm 文件' }}</span>
      </div>

      <UploadProgress
        :show="!!selectedFile"
        :filename="selectedFile?.name || ''"
        :percentage="uploader.progress.value"
        :uploaded="uploader.task.bytesUploaded"
        :total="uploader.task.size"
        :active="uploader.uploading.value"
        :error="uploader.task.error"
        @cancel="uploader.cancel()"
      />

      <div v-if="uploadError" class="msg msg-error">{{ uploadError }}</div>
      <div v-if="uploadSuccess" class="msg msg-success">{{ uploadSuccess }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useMessage } from 'naive-ui'
import { useRoomStore } from '../stores/room'
import { useUploader } from '../composables/useUploader'
import UploadProgress from './UploadProgress.vue'

const emit = defineEmits<{
  (e: 'submit', kind: 'url', sourceUrl: string, title: string): void
  (e: 'upload-start', uploadId: string): void
}>()

const props = defineProps<{
  sendWs: (msg: any) => void
  connected: boolean
}>()

const message = useMessage()
const roomStore = useRoomStore()
const uploader = useUploader()

const activeTab = ref('url')
const urlInput = ref('')
const submitting = ref(false)
const selectedFile = ref<File | null>(null)
const uploadError = ref('')
const uploadSuccess = ref('')
const fileInputRef = ref<HTMLInputElement | null>(null)

const showM3u8Warning = computed(() =>
  urlInput.value.toLowerCase().endsWith('.m3u8') || urlInput.value.toLowerCase().includes('.m3u8?')
)

function triggerFileInput() {
  fileInputRef.value?.click()
}

function onFileChange(e: Event) {
  const target = e.target as HTMLInputElement
  if (target.files && target.files[0]) {
    selectedFile.value = target.files[0]
    handleFileSelect()
  }
}

async function submitUrl() {
  if (!urlInput.value.trim()) { message.warning('请输入 URL'); return }
  const url = urlInput.value.trim()
  if (!url.startsWith('http://') && !url.startsWith('https://')) { message.error('URL 必须以 http:// 或 https:// 开头'); return }
  submitting.value = true
  try {
    const title = url.split('/').pop()?.split('?')[0] || 'Unknown'
    emit('submit', 'url', url, title)
  } finally { submitting.value = false }
}

async function handleFileSelect() {
  if (!props.connected) { uploadError.value = '等待服务器连接完成'; return }
  const file = selectedFile.value
  if (!file) return

  uploadError.value = ''
  uploadSuccess.value = ''

  const memberToken = roomStore.selfToken
  if (!memberToken) { uploadError.value = '等待加入房间…'; return }

  try {
    const suggestedChunkSize = 8 * 1024 * 1024
    const createResp = await fetch(`/api/rooms/${roomStore.roomId}/uploads`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Member-Token': memberToken },
      body: JSON.stringify({ filename: file.name, size: file.size, mime: file.type || 'video/mp4', chunk_size: suggestedChunkSize }),
    })
    if (!createResp.ok) { const err = await createResp.json(); throw new Error(err.error || 'create upload failed') }

    const { upload_id, chunk_size, total_chunks } = await createResp.json()
    uploader.task.uploadId = upload_id
    uploader.task.chunkSize = chunk_size
    uploader.task.totalChunks = total_chunks
    uploader.task.completedChunks = 0
    uploader.task.bytesUploaded = 0
    uploader.task.filename = file.name
    uploader.task.size = file.size
    uploader.task.active = true
    uploader.uploading.value = true

    emit('upload-start', upload_id)

    const uploadId = await uploader.uploadWithTask(
      roomStore.roomId, file, memberToken,
      upload_id, chunk_size, total_chunks,
      (bytesUploaded: number, bytesTotal: number) => {
        props.sendWs({ type: 'upload_progress', payload: { upload_id: uploader.task.uploadId, bytes_uploaded: bytesUploaded, bytes_total: bytesTotal } })
      },
    )

    if (uploadId) { uploadSuccess.value = `"${file.name}" 上传完成` }
    else { uploadError.value = uploader.task.error || '上传失败' }
  } catch (e: any) {
    uploadError.value = e.message || '上传失败'
    uploader.task.active = false
    uploader.uploading.value = false
  }
}
</script>

<style scoped>
.media-card {
  background: rgba(26, 26, 46, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.04);
  border-radius: 12px;
  overflow: hidden;
}

.media-tabs {
  display: flex;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}

.media-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 12px;
  border: none;
  background: transparent;
  color: rgba(255, 255, 255, 0.35);
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  transition: all 0.2s;
  position: relative;
}

.media-tab:hover {
  color: rgba(255, 255, 255, 0.6);
  background: rgba(255, 255, 255, 0.02);
}

.media-tab.active {
  color: #6c5ce7;
}

.media-tab.active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 20%;
  right: 20%;
  height: 2px;
  background: #6c5ce7;
  border-radius: 1px;
}

.tab-content {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.media-input {
  width: 100%;
  padding: 10px 14px;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(22, 22, 42, 0.8);
  color: #e0e0f0;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
}

.media-input:focus {
  border-color: #6c5ce7;
}

.media-input:disabled {
  opacity: 0.4;
}

.media-input::placeholder {
  color: rgba(255, 255, 255, 0.2);
}

.m3u8-warning {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 10px 12px;
  background: rgba(253, 121, 168, 0.1);
  border: 1px solid rgba(253, 121, 168, 0.2);
  border-radius: 8px;
  color: #fd79a8;
  font-size: 12px;
  line-height: 1.5;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px;
  border-radius: 8px;
  border: none;
  background: linear-gradient(135deg, #6c5ce7, #7d6ef0);
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.2s;
}

.action-btn:hover:not(:disabled) {
  opacity: 0.85;
}

.action-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.info-msg {
  text-align: center;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.2);
}

.upload-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 24px;
  border: 2px dashed rgba(255, 255, 255, 0.08);
  border-radius: 10px;
  color: rgba(255, 255, 255, 0.3);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.upload-area:hover {
  border-color: rgba(108, 92, 231, 0.3);
  color: rgba(255, 255, 255, 0.5);
  background: rgba(108, 92, 231, 0.04);
}

.msg {
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 13px;
}

.msg-error {
  background: rgba(231, 76, 60, 0.1);
  border: 1px solid rgba(231, 76, 60, 0.2);
  color: #e74c3c;
}

.msg-success {
  background: rgba(46, 204, 113, 0.1);
  border: 1px solid rgba(46, 204, 113, 0.2);
  color: #2ecc71;
}
</style>
