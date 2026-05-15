<template>
  <div class="player-area">
    <div v-if="!userStarted" class="click-to-start-overlay" @click="startPlaying">
      <div class="play-circle">
        <svg viewBox="0 0 24 24" fill="currentColor" width="48" height="48">
          <path d="M8 5v14l11-7z"/>
        </svg>
      </div>
      <div class="play-hint">点击开始播放</div>
    </div>

    <video
      v-show="userStarted && mediaSource"
      ref="videoEl"
      class="video-element"
      :src="isDirect ? mediaSource : undefined"
      controls
      playsinline
    />

    <div v-if="hlsError" class="hls-error">
      <svg viewBox="0 0 24 24" fill="currentColor" width="18" height="18"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>
      <span>HLS 加载失败，可能是跨域限制。建议改用上传。</span>
    </div>

    <div v-if="ended" class="ended-overlay">
      <div class="ended-content">
        <svg viewBox="0 0 24 24" fill="currentColor" width="40" height="40"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/></svg>
        <span>播放结束</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import Hls from 'hls.js'
import { usePlayer } from '../composables/usePlayer'
import type { MediaState, PlayerState } from '../types/message'

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
}>()

const videoEl = ref<HTMLVideoElement | null>(null)
const userStarted = ref(false)
const hlsError = ref(false)
const ended = ref(false)
let hls: Hls | null = null
let heartbeatTimer: ReturnType<typeof setInterval> | null = null
let seekDebounceTimer: ReturnType<typeof setTimeout> | null = null
let syncTimer: number | null = null
let programmaticSeek = false

const { compute, reset } = usePlayer()

const isDirect = computed(() => props.media?.media_type === 'direct')

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
  if (hls) { hls.destroy(); hls = null }
}

function videoPosition(): number {
  return videoEl.value ? videoEl.value.currentTime : 0
}

function runSync() {
  if (!videoEl.value || !userStarted.value || !props.media) {
    syncTimer = requestAnimationFrame(runSync)
    return
  }

  const ve = videoEl.value
  const p = props.player

  if (ve.paused) {
    syncTimer = setTimeout(() => {
      syncTimer = requestAnimationFrame(runSync)
    }, 1000) as any
    return
  }

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

  if (p.reason === 'ended') { ended.value = true; ve.pause(); return }
  ended.value = false

  if (p.playing) {
    const expectedPos = p.position + (props.serverNow() - p.updated_at) / 1000
    if (expectedPos >= 0 && Math.abs(ve.currentTime - expectedPos) > 0.3) {
      programmaticSeek = true
      ve.currentTime = expectedPos
    }
    ve.play().catch(() => {})
  } else {
    ve.pause()
    if (Math.abs(ve.currentTime - p.position) > 0.5) {
      programmaticSeek = true
      ve.currentTime = p.position
    }
  }
}

function onVideoSeeking() {
  if (programmaticSeek) {
    programmaticSeek = false
    if (seekDebounceTimer) { clearTimeout(seekDebounceTimer); seekDebounceTimer = null }
    return
  }
  if (seekDebounceTimer) clearTimeout(seekDebounceTimer)
  seekDebounceTimer = setTimeout(() => {
    if (videoEl.value) emit('seek-debounce', videoEl.value.currentTime)
  }, 200)
}

function onVideoPlaying() { emit('buffering', false) }
function onVideoWaiting() { emit('buffering', true) }
function onVideoEnded() { ended.value = true }

watch(() => props.player, handlePlayerUpdate, { deep: true })
watch(() => props.media, (newMedia) => {
  if (newMedia && userStarted.value) attachMedia(newMedia.source_url, newMedia.media_type)
}, { deep: true })

onMounted(() => {
  const ve = videoEl.value
  if (!ve) return
  ve.addEventListener('seeking', onVideoSeeking)
  ve.addEventListener('playing', onVideoPlaying)
  ve.addEventListener('waiting', onVideoWaiting)
  ve.addEventListener('ended', onVideoEnded)
  ve.preservesPitch = true
  try { (ve as any).webkitPreservesPitch = true } catch {}

  if (props.media && userStarted.value) attachMedia(props.media.source_url, props.media.media_type)

  heartbeatTimer = setInterval(() => {
    if (videoEl.value) {
      const dur = videoEl.value.duration && isFinite(videoEl.value.duration) ? videoEl.value.duration : undefined
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
  border-radius: 14px;
  overflow: hidden;
  aspect-ratio: 16 / 9;
  box-shadow: 0 4px 30px rgba(0,0,0,0.5);
}

.click-to-start-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  background: rgba(0, 0, 0, 0.85);
  cursor: pointer;
  z-index: 10;
  transition: background 0.3s;
}

.click-to-start-overlay:hover {
  background: rgba(0, 0, 0, 0.7);
}

.play-circle {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: linear-gradient(135deg, #6c5ce7, #7d6ef0);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  transition: transform 0.2s, box-shadow 0.2s;
  box-shadow: 0 8px 30px rgba(108, 92, 231, 0.3);
}

.click-to-start-overlay:hover .play-circle {
  transform: scale(1.08);
  box-shadow: 0 12px 40px rgba(108, 92, 231, 0.4);
}

.play-hint {
  color: rgba(255, 255, 255, 0.7);
  font-size: 16px;
  font-weight: 500;
  letter-spacing: 1px;
}

.video-element {
  width: 100%;
  height: 100%;
  display: block;
}

.hls-error {
  position: absolute;
  top: 12px;
  left: 12px;
  right: 12px;
  padding: 10px 14px;
  background: rgba(231, 76, 60, 0.15);
  border: 1px solid rgba(231, 76, 60, 0.3);
  border-radius: 8px;
  color: #e74c3c;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 8px;
  z-index: 5;
  backdrop-filter: blur(8px);
}

.ended-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.7);
  z-index: 5;
}

.ended-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: rgba(255, 255, 255, 0.6);
  font-size: 18px;
}
</style>
