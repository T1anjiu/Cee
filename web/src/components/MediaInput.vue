<template>
  <n-card :bordered="false" size="small" style="background:rgba(20,20,31,0.6);border:1px solid rgba(255,255,255,0.03);">
    <template #header>
      <n-tabs v-model:value="activeTab" size="small" :tabs-padding="0" style="margin:-2px 0;">
        <n-tab-pane name="url" tab="直链">
          <n-space vertical :size="10" style="padding-top:6px;">
            <n-input v-model:value="urlInput" placeholder="https://example.com/video.mp4" size="small" :disabled="!props.connected" @keyup.enter="submitUrl" />
            <n-alert v-if="showM3u8Warning" type="warning" :bordered="false" closable style="font-size:11px;padding:8px 10px;">
              m3u8 仅在源站允许跨域时可用，公网大概率会失败
            </n-alert>
            <n-button type="primary" block size="small" :loading="submitting" :disabled="!props.connected" @click="submitUrl">
              提交直链
            </n-button>
          </n-space>
        </n-tab-pane>
        <n-tab-pane name="upload" tab="上传">
          <n-space vertical :size="10" style="padding-top:6px;">
            <n-upload ref="uploadRef" :default-upload="false" :multiple="false" accept=".mp4,.webm" :disabled="!props.connected" @change="handleChange" list-type="text" :show-file-list="false">
              <n-button secondary block size="small" :disabled="!props.connected">
                <template #icon><n-icon><svg viewBox="0 0 24 24"><path d="M9 16h6v-6h4l-7-7-7 7h4zm-4 2h14v2H5z"/></svg></n-icon></template>
                选择文件
              </n-button>
            </n-upload>

            <UploadProgress :show="!!selectedFile" :filename="selectedFile?.name || ''" :percentage="uploader.progress.value" :uploaded="uploader.task.bytesUploaded" :total="uploader.task.size" :active="uploader.uploading.value" :error="uploader.task.error" @cancel="uploader.cancel()" />

            <n-alert v-if="uploadError" type="error" :bordered="false" closable @close="uploadError=''" style="font-size:12px;">{{ uploadError }}</n-alert>
            <n-alert v-if="uploadSuccess" type="success" :bordered="false" closable @close="uploadSuccess=''" style="font-size:12px;">{{ uploadSuccess }}</n-alert>
          </n-space>
        </n-tab-pane>
      </n-tabs>
    </template>
  </n-card>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useMessage } from 'naive-ui'
import { useRoomStore } from '../stores/room'
import { useUploader } from '../composables/useUploader'
import UploadProgress from './UploadProgress.vue'

const emit = defineEmits<{ (e: 'submit', kind: 'url', sourceUrl: string, title: string): void; (e: 'upload-start', uploadId: string): void }>()
const props = defineProps<{ sendWs: (msg: any) => void; connected: boolean }>()
const msg = useMessage(); const roomStore = useRoomStore(); const uploader = useUploader()
const activeTab = ref('url'); const urlInput = ref(''); const submitting = ref(false); const selectedFile = ref<File | null>(null); const uploadError = ref(''); const uploadSuccess = ref('')
const showM3u8Warning = computed(() => urlInput.value.toLowerCase().endsWith('.m3u8') || urlInput.value.toLowerCase().includes('.m3u8?'))

function handleChange(data: { file: { file?: File } }) {
  if (!data.file.file) return; selectedFile.value = data.file.file; doUpload()
}
async function submitUrl() {
  if (!urlInput.value.trim()) { msg.warning('请输入 URL'); return }
  const url = urlInput.value.trim()
  if (!url.startsWith('http://') && !url.startsWith('https://')) { msg.error('URL 必须合法'); return }
  submitting.value = true; try { emit('submit', 'url', url, url.split('/').pop()?.split('?')[0] || 'Unknown') } finally { submitting.value = false }
}
async function doUpload() {
  if (!props.connected) { uploadError.value = '等待连接'; return }; const file = selectedFile.value; if (!file) return
  uploadError.value = ''; uploadSuccess.value = ''; const token = roomStore.selfToken
  if (!token) { uploadError.value = '等待加入房间…'; return }
  try {
    const r = await fetch(`/api/rooms/${roomStore.roomId}/uploads`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-Member-Token': token }, body: JSON.stringify({ filename: file.name, size: file.size, mime: file.type || 'video/mp4', chunk_size: 8 * 1024 * 1024 }) })
    if (!r.ok) { const e = await r.json(); throw new Error(e.error || '失败') }
    const { upload_id, chunk_size, total_chunks } = await r.json()
    Object.assign(uploader.task, { uploadId: upload_id, chunkSize: chunk_size, totalChunks: total_chunks, completedChunks: 0, bytesUploaded: 0, filename: file.name, size: file.size, active: true }); uploader.uploading.value = true
    emit('upload-start', upload_id)
    const uid = await uploader.uploadWithTask(roomStore.roomId, file, token, upload_id, chunk_size, total_chunks, (bu: number, bt: number) => props.sendWs({ type: 'upload_progress', payload: { upload_id: uploader.task.uploadId, bytes_uploaded: bu, bytes_total: bt } }))
    uid ? uploadSuccess.value = `"${file.name}" 完成` : uploadError.value = uploader.task.error || '上传失败'
  } catch (e: any) { uploadError.value = e.message || '上传失败'; uploader.task.active = false; uploader.uploading.value = false }
}
</script>
