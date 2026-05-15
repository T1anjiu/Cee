<template>
  <n-card title="切换媒体" :bordered="false">
    <n-tabs type="line" v-model:value="activeTab">
      <n-tab-pane name="url" tab="直链">
        <n-space vertical :size="12">
          <n-input
            v-model:value="urlInput"
            placeholder="https://example.com/video.mp4"
            @update:value="onUrlChange"
            :disabled="!props.connected"
          />
          <n-alert v-if="showM3u8Warning" type="warning" :bordered="false">
            m3u8 仅在源站允许跨域时可用，公网 m3u8 大概率会失败，建议改为上传本地文件
          </n-alert>
          <n-button type="primary" @click="submitUrl" :loading="submitting" :disabled="!props.connected" block>
            提交
          </n-button>
          <n-alert v-if="!props.connected" type="info" :bordered="false">
            等待连接服务器后可切换媒体
          </n-alert>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="upload" tab="上传">
        <n-space vertical :size="12">
          <n-alert v-if="!props.connected" type="info" :bordered="false">
            等待连接服务器后可上传文件
          </n-alert>
          <n-upload
            ref="uploadRef"
            :default-upload="false"
            :multiple="false"
            accept=".mp4,.webm"
            :disabled="!props.connected"
            @change="handleFileSelect"
          >
            <n-button :disabled="!props.connected">选择文件 (.mp4 / .webm)</n-button>
          </n-upload>

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

          <n-alert v-if="uploadError" type="error" :bordered="false">
            {{ uploadError }}
          </n-alert>

          <n-alert v-if="uploadSuccess" type="success" :bordered="false">
            {{ uploadSuccess }}
          </n-alert>
        </n-space>
      </n-tab-pane>
    </n-tabs>
  </n-card>
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

const showM3u8Warning = computed(() => {
  return urlInput.value.toLowerCase().endsWith('.m3u8') ||
         urlInput.value.toLowerCase().includes('.m3u8?')
})

function onUrlChange() {}

async function submitUrl() {
  if (!urlInput.value.trim()) {
    message.warning('请输入 URL')
    return
  }
  const url = urlInput.value.trim()
  if (!url.startsWith('http://') && !url.startsWith('https://')) {
    message.error('URL 必须以 http:// 或 https:// 开头')
    return
  }

  submitting.value = true
  try {
    const title = url.split('/').pop()?.split('?')[0] || 'Unknown'
    emit('submit', 'url', url, title)
  } finally {
    submitting.value = false
  }
}

async function handleFileSelect(data: { file: { file: File } }) {
  if (!props.connected) {
    uploadError.value = '等待服务器连接完成'
    return
  }

  const file = data.file.file
  if (!file) return

  selectedFile.value = file
  uploadError.value = ''
  uploadSuccess.value = ''

  const memberToken = roomStore.selfToken
  if (!memberToken) {
    uploadError.value = '等待加入房间完成，请稍后再试'
    return
  }

  try {
    const suggestedChunkSize = 8 * 1024 * 1024
    const createResp = await fetch(`/api/rooms/${roomStore.roomId}/uploads`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Member-Token': memberToken,
      },
      body: JSON.stringify({
        filename: file.name,
        size: file.size,
        mime: file.type || 'video/mp4',
        chunk_size: suggestedChunkSize,
      }),
    })

    if (!createResp.ok) {
      const err = await createResp.json()
      throw new Error(err.error || 'create upload failed')
    }

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

    // 立即通知房间开始上传，让其他成员看到 uploading 状态
    emit('upload-start', upload_id)

    const uploadId = await uploader.uploadWithTask(
      roomStore.roomId,
      file,
      memberToken,
      upload_id,
      chunk_size,
      total_chunks,
      (bytesUploaded, bytesTotal) => {
        props.sendWs({
          type: 'upload_progress',
          payload: {
            upload_id: uploader.task.uploadId,
            bytes_uploaded: bytesUploaded,
            bytes_total: bytesTotal,
          },
        })
      },
    )

    if (uploadId) {
      uploadSuccess.value = `"${file.name}" 上传完成`
    } else {
      uploadError.value = uploader.task.error || '上传失败'
    }
  } catch (e: any) {
    uploadError.value = e.message || '上传失败'
    uploader.task.active = false
    uploader.uploading.value = false
  }
}
</script>
