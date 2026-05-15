<template>
  <div class="player-wrap">
    <div v-if="!userStarted" class="start-overlay" @click="startPlaying">
      <div class="start-btn">
        <n-icon :size="28" color="#fff"><svg viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg></n-icon>
      </div>
      <span class="start-hint">点击开始播放</span>
    </div>

    <video v-show="userStarted && mediaSource" ref="videoEl" class="video-el" :src="isDirect ? mediaSource : undefined" controls playsinline />

    <n-alert v-if="hlsError" type="error" :bordered="false" closable style="position:absolute;top:12px;left:12px;right:12px;z-index:5;max-width:calc(100% - 24px);font-size:12px;">
      <template #icon><n-icon><svg viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg></n-icon></template>
      HLS 加载失败，可能是跨域限制
    </n-alert>

    <div v-if="ended" class="start-overlay" style="background:rgba(0,0,0,0.65);">
      <n-icon :size="32" color="rgba(255,255,255,0.4)"><svg viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/></svg></n-icon>
      <span style="color:rgba(255,255,255,0.4);font-size:15px;margin-top:6px;">播放结束</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import Hls from 'hls.js'
import { usePlayer } from '../composables/usePlayer'
import type { MediaState, PlayerState } from '../types/message'

const props = defineProps<{ media: MediaState | null; player: PlayerState; syncDiff: number; clockOffset: number; serverNow: () => number }>()
const emit = defineEmits<{ (e: 'buffering', buffering: boolean): void; (e: 'heartbeat', position: number, playing: boolean, duration?: number): void; (e: 'seek-debounce', position: number): void }>()

const videoEl = ref<HTMLVideoElement | null>(null)
const userStarted = ref(false); const hlsError = ref(false); const ended = ref(false)
let hls: Hls | null = null; let heartbeatTimer: ReturnType<typeof setInterval> | null = null; let seekDebounceTimer: ReturnType<typeof setTimeout> | null = null; let syncTimer: number | null = null; let programmaticSeek = false
const { compute, reset } = usePlayer()
const isDirect = computed(() => props.media?.media_type === 'direct')
const mediaSource = computed(() => props.media?.source_url || '')

function startPlaying() {
  userStarted.value = true; if (!videoEl.value) return
  videoEl.value.muted = false; videoEl.value.play().catch(() => { videoEl.value!.muted = true; videoEl.value!.play() })
}

function attachMedia(src: string, type: string) {
  destroyHls(); hlsError.value = false; ended.value = false; reset()
  if (!videoEl.value) return
  if (type === 'hls') {
    if (Hls.isSupported()) { hls = new Hls({ enableWorker: true, lowLatencyMode: false }); hls.loadSource(src); hls.attachMedia(videoEl.value); hls.on(Hls.Events.ERROR, (_e, d) => { if (d.fatal) { hlsError.value = true; hls?.destroy() } }) }
    else if (videoEl.value.canPlayType('application/vnd.apple.mpegurl')) { videoEl.value.src = src } else { hlsError.value = true }
  } else { videoEl.value.src = src }
}
function destroyHls() { if (hls) { hls.destroy(); hls = null } }

function runSync() {
  if (!videoEl.value || !userStarted.value || !props.media) { syncTimer = requestAnimationFrame(runSync); return }
  const ve = videoEl.value
  if (ve.paused) { syncTimer = setTimeout(() => { syncTimer = requestAnimationFrame(runSync) }, 1000) as any; return }
  const ep = props.player.playing ? props.player.position + (props.serverNow() - props.player.updated_at) / 1000 : props.player.position
  if (ep < 0) { syncTimer = requestAnimationFrame(runSync); return }
  const r = compute(ve.currentTime - ep)
  if (r.shouldSeek) { programmaticSeek = true; ve.currentTime = ep; ve.playbackRate = 1.0 } else ve.playbackRate = r.rate
  syncTimer = requestAnimationFrame(runSync)
}

function handlePlayerUpdate() {
  if (!videoEl.value || !userStarted.value) return; const ve = videoEl.value; const p = props.player
  if (p.reason === 'ended') { ended.value = true; ve.pause(); return }
  ended.value = false
  if (p.playing) {
    const ep = p.position + (props.serverNow() - p.updated_at) / 1000
    if (ep >= 0 && Math.abs(ve.currentTime - ep) > 0.3) { programmaticSeek = true; ve.currentTime = ep }
    ve.play().catch(() => {})
  } else { ve.pause(); if (Math.abs(ve.currentTime - p.position) > 0.5) { programmaticSeek = true; ve.currentTime = p.position } }
}

function onVideoSeeking() {
  if (programmaticSeek) { programmaticSeek = false; if (seekDebounceTimer) { clearTimeout(seekDebounceTimer); seekDebounceTimer = null } return }
  if (seekDebounceTimer) clearTimeout(seekDebounceTimer)
  seekDebounceTimer = setTimeout(() => { if (videoEl.value) emit('seek-debounce', videoEl.value.currentTime) }, 200)
}
function onVideoPlaying() { emit('buffering', false) }
function onVideoWaiting() { emit('buffering', true) }
function onVideoEnded() { ended.value = true }

watch(() => props.player, handlePlayerUpdate, { deep: true })
watch(() => props.media, (m) => { if (m && userStarted.value) attachMedia(m.source_url, m.media_type) }, { deep: true })

onMounted(() => {
  const ve = videoEl.value; if (!ve) return
  ve.addEventListener('seeking', onVideoSeeking); ve.addEventListener('playing', onVideoPlaying)
  ve.addEventListener('waiting', onVideoWaiting); ve.addEventListener('ended', onVideoEnded)
  ve.preservesPitch = true; try { (ve as any).webkitPreservesPitch = true } catch {}
  if (props.media && userStarted.value) attachMedia(props.media.source_url, props.media.media_type)
  heartbeatTimer = setInterval(() => { if (videoEl.value) emit('heartbeat', videoEl.value.currentTime, !videoEl.value.paused, videoEl.value.duration && isFinite(videoEl.value.duration) ? videoEl.value.duration : undefined) }, 5000)
  syncTimer = requestAnimationFrame(runSync)
})

onUnmounted(() => {
  destroyHls(); if (heartbeatTimer) clearInterval(heartbeatTimer); if (seekDebounceTimer) clearTimeout(seekDebounceTimer); if (syncTimer) cancelAnimationFrame(syncTimer)
  const ve = videoEl.value; if (ve) { ve.removeEventListener('seeking', onVideoSeeking); ve.removeEventListener('playing', onVideoPlaying); ve.removeEventListener('waiting', onVideoWaiting); ve.removeEventListener('ended', onVideoEnded) }
})
</script>

<style scoped>
.player-wrap {
  position: relative; width: 100%; background: #000;
  border-radius: 10px; overflow: hidden; aspect-ratio: 16 / 9;
  box-shadow: 0 4px 20px rgba(0,0,0,0.4);
}
.start-overlay {
  position: absolute; inset: 0; display: flex; flex-direction: column;
  align-items: center; justify-content: center; gap: 10px;
  background: rgba(0,0,0,0.78); cursor: pointer; z-index: 10;
  transition: background 0.25s;
}
.start-overlay:hover { background: rgba(0,0,0,0.6); }
.start-btn {
  width: 64px; height: 64px; border-radius: 50%;
  background: linear-gradient(135deg, #6366f1, #818cf8);
  display: flex; align-items: center; justify-content: center;
  box-shadow: 0 6px 24px rgba(99,102,241,0.25);
  transition: transform 0.2s, box-shadow 0.2s;
}
.start-overlay:hover .start-btn { transform: scale(1.06); box-shadow: 0 8px 32px rgba(99,102,241,0.35); }
.start-hint { color: rgba(255,255,255,0.6); font-size: 14px; font-weight: 500; letter-spacing: 0.5px; }
.video-el { width: 100%; height: 100%; display: block; }
</style>
