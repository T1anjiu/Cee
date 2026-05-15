import { ref, reactive } from 'vue'

export interface UploadTask {
  uploadId: string
  filename: string
  size: number
  chunkSize: number
  totalChunks: number
  completedChunks: number
  bytesUploaded: number
  active: boolean
  error: string
}

export function useUploader() {
  const task = reactive<UploadTask>({
    uploadId: '',
    filename: '',
    size: 0,
    chunkSize: 0,
    totalChunks: 0,
    completedChunks: 0,
    bytesUploaded: 0,
    active: false,
    error: '',
  })

  const progress = ref(0)
  const uploading = ref(false)
  let aborted = false

  async function uploadWithTask(
    roomId: string,
    file: File,
    memberToken: string,
    uploadId: string,
    chunkSize: number,
    totalChunks: number,
    onProgress?: (bytesUploaded: number, bytesTotal: number) => void,
  ): Promise<string | null> {
    aborted = false
    task.uploadId = uploadId
    task.chunkSize = chunkSize
    task.totalChunks = totalChunks
    task.completedChunks = 0
    task.bytesUploaded = 0

    try {
      await uploadChunks(file, memberToken, uploadId, chunkSize, totalChunks, onProgress)

      const completeResp = await fetch(`/api/uploads/${uploadId}/complete`, {
        method: 'POST',
        headers: { 'X-Member-Token': memberToken },
      })

      if (!completeResp.ok) {
        const err = await completeResp.json()
        throw new Error(err.error || 'complete upload failed')
      }

      task.active = false
      uploading.value = false
      return uploadId
    } catch (e: any) {
      task.error = e.message || 'upload failed'
      task.active = false
      uploading.value = false
      return null
    }
  }

  async function uploadChunks(
    file: File,
    memberToken: string,
    uploadId: string,
    chunkSize: number,
    totalChunks: number,
    onProgress?: (bytesUploaded: number, bytesTotal: number) => void,
  ) {
    let lastReportedBytes = 0
    let lastReportedTime = Date.now()
    const activePromises = new Set<Promise<void>>()

    for (let i = 0; i < totalChunks; i++) {
      if (aborted) throw new Error('cancelled')

      const p = uploadSingleChunk(i, file, memberToken, uploadId, chunkSize)
      activePromises.add(p)
      p.finally(() => activePromises.delete(p))

      // Throttled progress reporting: ≥ 1 MiB or ≥ 250ms
      const now = Date.now()
      const bytesSinceLastReport = task.bytesUploaded - lastReportedBytes
      if (onProgress && (bytesSinceLastReport >= 1024 * 1024 || now - lastReportedTime >= 250)) {
        onProgress(task.bytesUploaded, file.size)
        lastReportedBytes = task.bytesUploaded
        lastReportedTime = now
      }

      // Wait when concurrency limit reached
      if (activePromises.size >= 2) {
        await Promise.race(activePromises)
      }
    }

    // Wait for remaining
    await Promise.all(activePromises)

    if (onProgress) {
      onProgress(file.size, file.size)
    }
  }

  async function uploadSingleChunk(
    index: number,
    file: File,
    memberToken: string,
    uploadId: string,
    chunkSize: number,
  ): Promise<void> {
    const start = index * chunkSize
    const end = Math.min(start + chunkSize, file.size)
    const blob = file.slice(start, end)

    for (let attempt = 0; attempt < 3; attempt++) {
      try {
        const resp = await fetch(`/api/uploads/${uploadId}/chunks/${index}`, {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/octet-stream',
            'X-Member-Token': memberToken,
          },
          body: blob,
        })

        if (!resp.ok) {
          throw new Error(`chunk ${index} failed`)
        }

        task.completedChunks++
        task.bytesUploaded += blob.size
        progress.value = (task.bytesUploaded / file.size) * 100
        return
      } catch (e) {
        if (attempt === 2) throw e
        await new Promise(r => setTimeout(r, 1000 * (attempt + 1)))
      }
    }
  }

  async function upload(
    roomId: string,
    file: File,
    memberToken: string,
    onProgress?: (bytesUploaded: number, bytesTotal: number) => void,
  ): Promise<string | null> {
    aborted = false
    uploading.value = true
    task.filename = file.name
    task.size = file.size
    task.active = true
    task.error = ''

    try {
      const suggestedChunkSize = 8 * 1024 * 1024
      const createResp = await fetch(`/api/rooms/${roomId}/uploads`, {
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
      task.uploadId = upload_id
      task.chunkSize = chunk_size
      task.totalChunks = total_chunks
      task.completedChunks = 0
      task.bytesUploaded = 0

      await uploadChunks(file, memberToken, upload_id, chunk_size, total_chunks, onProgress)

      const completeResp = await fetch(`/api/uploads/${upload_id}/complete`, {
        method: 'POST',
        headers: { 'X-Member-Token': memberToken },
      })

      if (!completeResp.ok) {
        const err = await completeResp.json()
        throw new Error(err.error || 'complete upload failed')
      }

      task.active = false
      uploading.value = false
      return upload_id
    } catch (e: any) {
      task.error = e.message || 'upload failed'
      task.active = false
      uploading.value = false
      return null
    }
  }

  function cancel() {
    aborted = true
    if (task.uploadId) {
      fetch(`/api/uploads/${task.uploadId}`, { method: 'DELETE' }).catch(() => {})
    }
  }

  function reset() {
    task.uploadId = ''
    task.filename = ''
    task.size = 0
    task.chunkSize = 0
    task.totalChunks = 0
    task.completedChunks = 0
    task.bytesUploaded = 0
    task.active = false
    task.error = ''
    progress.value = 0
    uploading.value = false
    aborted = false
  }

  return { task, progress, uploading, upload, uploadWithTask, cancel, reset }
}
