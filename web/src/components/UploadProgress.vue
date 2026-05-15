<template>
  <div v-if="show" class="up-card">
    <n-space justify="space-between" align="center" style="margin-bottom:6px;">
      <n-text style="font-size:12px;font-weight:600;color:rgba(255,255,255,0.8);overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">{{ filename }}</n-text>
      <n-button v-if="active" size="tiny" quaternary circle @click="$emit('cancel')">
        <template #icon><n-icon :size="12"><svg viewBox="0 0 24 24"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg></n-icon></template>
      </n-button>
    </n-space>
    <n-progress type="line" :percentage="Math.min(100, percentage)" :height="4" :border-radius="2" :status="error ? 'error' : 'default'" :processing="active && !error" :show-indicator="false" />
    <n-space justify="space-between" style="margin-top:4px;">
      <n-text depth="3" style="font-size:10px;">{{ uploadedStr }} / {{ totalStr }}</n-text>
      <n-text depth="3" style="font-size:10px;font-weight:600;">{{ Math.round(percentage) }}%</n-text>
    </n-space>
    <n-text v-if="error" type="error" style="font-size:11px;margin-top:4px;display:block;">{{ error }}</n-text>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
const props = defineProps<{ show: boolean; filename: string; percentage: number; uploaded: number; total: number; active: boolean; error: string }>()
defineEmits<{ (e: 'cancel'): void }>()
const fmt = (b: number) => b >= 1073741824 ? (b / 1073741824).toFixed(1) + ' GB' : b >= 1048576 ? (b / 1048576).toFixed(1) + ' MB' : b >= 1024 ? (b / 1024).toFixed(0) + ' KB' : b + ' B'
const uploadedStr = computed(() => fmt(props.uploaded))
const totalStr = computed(() => fmt(props.total))
</script>

<style scoped>
.up-card { background: rgba(20,20,31,0.6); border: 1px solid rgba(255,255,255,0.04); border-radius: 8px; padding: 10px 12px; }
</style>
