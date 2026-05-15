<template>
  <n-card v-if="show" :bordered="false" size="small">
    <n-space vertical :size="8">
      <n-text strong>{{ filename }}</n-text>
      <n-progress
        :percentage="percentage"
        :stroke-color="error ? '#e74c3c' : '#2080f0'"
        :height="16"
        :indicator-placement="'inside'"
      />
      <n-space justify="space-between">
        <n-text depth="3" style="font-size: 12px;">
          {{ uploadedStr }} / {{ totalStr }}
          {{ error ? ` - ${error}` : '' }}
        </n-text>
        <n-button v-if="active" size="tiny" @click="$emit('cancel')">取消</n-button>
      </n-space>
    </n-space>
  </n-card>
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
