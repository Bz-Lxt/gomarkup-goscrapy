<script setup lang="ts">
import { computed } from 'vue'
import type { ClusterNode } from '@/api/types'
import { failTone, formatNumber, formatPct } from '@/utils/format'

const props = defineProps<{ node: ClusterNode }>()

const tone = computed(() => failTone(props.node.fail_rate))
const memPct = computed(() => {
  const total = props.node.memory_total_mb
  if (total && total > 0) return Math.min(100, (props.node.memory_mb / total) * 100)
  return Math.min(100, props.node.memory_mb / 10)
})
const cpuPct = computed(() => Math.min(100, props.node.cpu))
</script>

<template>
  <article class="node-card" :class="`tone-${tone}`">
    <div class="flex items-start justify-between">
      <div>
        <div class="font-display text-lg tracking-wide">{{ node.id }}</div>
        <div class="mt-1 font-mono text-[11px] uppercase tracking-[0.2em] text-muted">
          {{ node.role || 'worker' }} · {{ node.status || 'online' }}
        </div>
      </div>
      <div class="text-right">
        <div class="font-display text-2xl text-cyan">{{ formatNumber(node.pages_per_min, 0) }}</div>
        <div class="font-mono text-[10px] uppercase tracking-widest text-muted">Pages/Min</div>
      </div>
    </div>

    <div class="mt-5 space-y-3">
      <div>
        <div class="mb-1 flex justify-between font-mono text-[11px] text-muted">
          <span>CPU</span><span>{{ formatNumber(node.cpu, 1) }}%</span>
        </div>
        <div class="bar"><i :style="{ width: cpuPct + '%' }" /></div>
      </div>
      <div>
        <div class="mb-1 flex justify-between font-mono text-[11px] text-muted">
          <span>内存</span><span>{{ formatNumber(node.memory_mb, 0) }} MB</span>
        </div>
        <div class="bar"><i :style="{ width: memPct + '%' }" /></div>
      </div>
      <div class="flex items-center justify-between pt-1">
        <span class="font-mono text-[11px] uppercase tracking-widest text-muted">失败率</span>
        <span class="fail font-mono text-sm">{{ formatPct(node.fail_rate) }}</span>
      </div>
    </div>
  </article>
</template>

<style scoped>
.node-card {
  background: var(--bg-1);
  border: 1px solid var(--line);
  border-radius: 14px;
  padding: 18px 18px 16px;
  box-shadow: 0 0 0 1px rgba(62, 224, 197, 0.08), 0 0 24px rgba(62, 224, 197, 0.05);
}
.node-card.tone-warn {
  border-color: #f5b942;
  box-shadow: 0 0 0 1px rgba(245, 185, 66, 0.18), 0 0 22px rgba(245, 185, 66, 0.08);
}
.node-card.tone-bad {
  border-color: #ff6b7a;
  box-shadow: 0 0 0 1px rgba(255, 107, 122, 0.2), 0 0 22px rgba(255, 107, 122, 0.08);
}
.bar {
  height: 6px;
  background: #122632;
  border-radius: 99px;
  overflow: hidden;
}
.bar i {
  display: block;
  height: 100%;
  background: linear-gradient(90deg, #27c4ab, #3ee0c5);
  box-shadow: 0 0 8px rgba(62, 224, 197, 0.6);
}
.tone-warn .fail {
  color: #f5b942;
}
.tone-bad .fail {
  color: #ff6b7a;
}
.tone-ok .fail {
  color: #3ee0c5;
}
</style>
