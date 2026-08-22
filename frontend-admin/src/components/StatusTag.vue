<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'

const props = defineProps<{ value: string }>()

const tone = computed(() => {
  const v = (props.value || '').toLowerCase()
  if (['running', 'healthy', 'up', 'online', 'succeeded', 'ok'].includes(v)) {
    return { type: 'success' as const, label: props.value }
  }
  if (['paused', 'unknown', 'degraded', 'created'].includes(v)) {
    return { type: 'warning' as const, label: props.value }
  }
  if (['failed', 'down', 'dead', 'cancelled', 'error'].includes(v)) {
    return { type: 'error' as const, label: props.value }
  }
  return { type: 'default' as const, label: props.value || '—' }
})
</script>

<template>
  <n-tag :type="tone.type" size="small" round :bordered="false">
    {{ tone.label }}
  </n-tag>
</template>
