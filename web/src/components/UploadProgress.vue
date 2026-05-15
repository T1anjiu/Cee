<template>
  <div v-if="show" class="upload-progress-card">
    <div class="up-header">
      <div class="up-filename">{{ filename }}</div>
      <button v-if="active" class="up-cancel" @click="$emit('cancel')">
        <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>
      </button>
    </div>
    <div class="up-bar-wrapper">
      <div class="up-bar-track">
        <div class="up-bar-fill" :style="{ width: Math.min(100, percentage) + '%' }"></div>
      </div>
    </div>
    <div class="up-info">
      <span class="up-bytes">{{ uploadedStr }} / {{ totalStr }}</span>
      <span class="up-pct">{{ Math.round(percentage) }}%</span>
    </div>
    <div v-if="error" class="up-error">{{ error }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  show: boolean
  filename: string
  percentage: number
  uploaded: number
  total: number
  active: boolean
  error: string
}>()

defineEmits<{ (e: 'cancel'): void }>()

function formatBytes(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024) return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
  if (bytes >= 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  if (bytes >= 1024) return (bytes / 1024).toFixed(0) + ' KB'
  return bytes + ' B'
}

const uploadedStr = computed(() => formatBytes(props.uploaded))
const totalStr = computed(() => formatBytes(props.total))
</script>

<style scoped>
.upload-progress-card {
  background: rgba(22, 22, 42, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 10px;
  padding: 12px;
}

.up-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.up-filename {
  font-size: 13px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.8);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.up-cancel {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  border: none;
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.4);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.15s;
}

.up-cancel:hover {
  background: rgba(231, 76, 60, 0.2);
  color: #e74c3c;
}

.up-bar-wrapper {
  margin-bottom: 6px;
}

.up-bar-track {
  height: 6px;
  background: rgba(255, 255, 255, 0.06);
  border-radius: 3px;
  overflow: hidden;
}

.up-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #6c5ce7, #a29bfe);
  border-radius: 3px;
  transition: width 0.3s ease;
}

.up-info {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.3);
}

.up-bytes {
  font-variant-numeric: tabular-nums;
}

.up-pct {
  color: rgba(255, 255, 255, 0.4);
  font-weight: 600;
}

.up-error {
  margin-top: 6px;
  font-size: 12px;
  color: #e74c3c;
}
</style>
