<template>
  <div class="player-area">
    <div v-if="!userStarted" class="click-to-start-overlay" @click="startPlaying">
      <n-space vertical align="center" :size="16">
        <n-icon :size="64" color="#fff">
          <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M8 5v14l11-7z"/>
          </svg>
        </n-icon>
        <n-text style="color: #fff; font-size: 18px;">点击开始播放</n-text>
      </n-space>
    </div>

    <video
      v-show="userStarted && mediaSource"
      ref="videoEl"
      class="video-element"
      :src="isDirect ? mediaSource : undefined"
      controls
      playsinline
    />

    <n-alert v-if="hlsError" type="error" :bordered="false" style="margin-top: 12px;">
      HLS 加载失败，可能是源站 CORS 限制。建议改用上传本地文件。
    </n-alert>

    <div v-if="ended" class="ended-overlay">
      <n-text style="color: #fff; font-size: 16px;">播放结束</n-text>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import Hls from 'hls.js'
import { usePlayer } from '../composables/usePlayer'
import type { MediaState, PlayerState, Message } from '../types/message'

const props = defineProps<{
  media: MediaState | null
  player: PlayerState
  syncDiff: number
  clockOffset: number
  serverNow: () => number
}>()

const emit = defineEmits<{
  (e: 'buffering', buffering: boolean): void
  (e: 'heartbeat', position: number, playing: boolean, duration?: number): void
  (e: 'seek-debounce', position: number): void
  (e: 'client-time', time: number): void
}>()

const videoEl = ref<HTMLVideoElement | null>(null)
const userStarted = ref(false)
const hlsError = ref(false)
const ended = ref(false)
let hls: Hls | null = null
let heartbeatTimer: ReturnType<typeof setInterval> | null = null
let seekDebounceTimer: ReturnType<typeof setTimeout> | null = null
let syncTimer: ReturnType<typeof requestAnimationFrame> | null = null
let programmaticSeek = false  // true 时 seeking 事件不发 WS

const { compute, reset } = usePlayer()

const isDirect = computed(() => {
  return props.media?.media_type === 'direct'
})

const mediaSource = computed(() => {
  if (!props.media) return ''
  return props.media.source_url
})

function startPlaying() {
  userStarted.value = true
  if (videoEl.value) {
    videoEl.value.muted = false
    videoEl.value.play().catch(() => {
      videoEl.value.muted = true
      videoEl.value.play()
    })
  }
}

function attachMedia(src: string, type: string) {
  destroyHls()
  hlsError.value = false
  ended.value = false
  reset()

  if (!videoEl.value) return

  if (type === 'hls') {
    if (Hls.isSupported()) {
      hls = new Hls({ enableWorker: true, lowLatencyMode: false })
      hls.loadSource(src)
      hls.attachMedia(videoEl.value)
      hls.on(Hls.Events.ERROR, (_event, data) => {
        if (data.fatal) {
          hlsError.value = true
          hls?.destroy()
        }
      })
    } else if (videoEl.value.canPlayType('application/vnd.apple.mpegurl')) {
      videoEl.value.src = src
    } else {
      hlsError.value = true
    }
  } else {
    videoEl.value.src = src
  }
}

function destroyHls() {
  if (hls) {
    hls.destroy()
    hls = null
  }
}

function videoPosition(): number {
  return videoEl.value ? videoEl.value.currentTime : 0
}

// Sync loop
function runSync() {
  if (!videoEl.value || !userStarted.value || !props.media) {
    syncTimer = requestAnimationFrame(runSync)
    return
  }

  const ve = videoEl.value
  const p = props.player

  if (ve.paused) {
    // When paused, reduce sync frequency to save CPU
    syncTimer = setTimeout(() => {
      syncTimer = requestAnimationFrame(runSync)
    }, 1000) as any
    return
  }

  // Compute expected position in real-time every frame
  let expectedPos: number
  if (p.playing) {
    expectedPos = p.position + (props.serverNow() - p.updated_at) / 1000
  } else {
    expectedPos = p.position
  }

  if (expectedPos < 0) {
    syncTimer = requestAnimationFrame(runSync)
    return
  }

  const localTime = ve.currentTime
  const diff = localTime - expectedPos
  const result = compute(diff)

  if (result.shouldSeek) {
    programmaticSeek = true
    ve.currentTime = expectedPos
    ve.playbackRate = 1.0
  } else {
    ve.playbackRate = result.rate
  }

  syncTimer = requestAnimationFrame(runSync)
}

function handlePlayerUpdate() {
  if (!videoEl.value || !userStarted.value) return

  const ve = videoEl.value
  const p = props.player

  if (p.reason === 'ended') {
    ended.value = true
    ve.pause()
    return
  }

  ended.value = false

  if (p.playing) {
    // Compute expected position in real-time
    const expectedPos = p.position + (props.serverNow() - p.updated_at) / 1000
    if (expectedPos >= 0 && Math.abs(ve.currentTime - expectedPos) > 0.3) {
      programmaticSeek = true
      ve.currentTime = expectedPos
    }
    ve.play().catch(() => {})
  } else {
    ve.pause()
    // Align to the paused position
    if (Math.abs(ve.currentTime - p.position) > 0.5) {
      programmaticSeek = true
      ve.currentTime = p.position
    }
  }
}

function onVideoSeeking() {
  if (programmaticSeek) {
    // Internal seek (sync loop or handlePlayerUpdate) — don't send WS
    programmaticSeek = false
    if (seekDebounceTimer) {
      clearTimeout(seekDebounceTimer)
      seekDebounceTimer = null
    }
    return
  }
  if (seekDebounceTimer) {
    clearTimeout(seekDebounceTimer)
  }
  seekDebounceTimer = setTimeout(() => {
    if (videoEl.value) {
      emit('seek-debounce', videoEl.value.currentTime)
    }
  }, 200)
}

function onVideoPlaying() {
  emit('buffering', false)
}

function onVideoWaiting() {
  emit('buffering', true)
}

function onVideoEnded() {
  ended.value = true
}

watch(() => props.player, handlePlayerUpdate, { deep: true })

watch(() => props.media, (newMedia) => {
  if (newMedia && userStarted.value) {
    attachMedia(newMedia.source_url, newMedia.media_type)
  }
}, { deep: true })

onMounted(() => {
  const ve = videoEl.value
  if (!ve) return

  ve.addEventListener('seeking', onVideoSeeking)
  ve.addEventListener('playing', onVideoPlaying)
  ve.addEventListener('waiting', onVideoWaiting)
  ve.addEventListener('ended', onVideoEnded)

  // preservesPitch for all browsers
  ve.preservesPitch = true
  try { (ve as any).webkitPreservesPitch = true } catch {}

  if (props.media && userStarted.value) {
    attachMedia(props.media.source_url, props.media.media_type)
  }

  heartbeatTimer = setInterval(() => {
    if (videoEl.value) {
      const dur = videoEl.value.duration && isFinite(videoEl.value.duration)
        ? videoEl.value.duration
        : undefined
      emit('heartbeat', videoPosition(), !videoEl.value.paused, dur)
    }
  }, 5000)

  syncTimer = requestAnimationFrame(runSync)
})

onUnmounted(() => {
  destroyHls()
  if (heartbeatTimer) clearInterval(heartbeatTimer)
  if (seekDebounceTimer) clearTimeout(seekDebounceTimer)
  if (syncTimer) cancelAnimationFrame(syncTimer)
  const ve = videoEl.value
  if (ve) {
    ve.removeEventListener('seeking', onVideoSeeking)
    ve.removeEventListener('playing', onVideoPlaying)
    ve.removeEventListener('waiting', onVideoWaiting)
    ve.removeEventListener('ended', onVideoEnded)
  }
})
</script>

<style scoped>
.player-area {
  position: relative;
  width: 100%;
  background: #000;
  border-radius: 8px;
  overflow: hidden;
  aspect-ratio: 16 / 9;
}

.click-to-start-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.85);
  cursor: pointer;
  z-index: 10;
  transition: background 0.2s;
}

.click-to-start-overlay:hover {
  background: rgba(0, 0, 0, 0.7);
}

.video-element {
  width: 100%;
  height: 100%;
  display: block;
}

.ended-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.6);
  z-index: 5;
}
</style>
