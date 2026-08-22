<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { NButton } from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import NodeCard from '@/components/NodeCard.vue'
import { useMetricsStore } from '@/stores/metrics'
import { formatNumber, formatPct } from '@/utils/format'
import { toastSuccess } from '@/utils/feedback'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent])

const metrics = useMetricsStore()

const option = computed(() => ({
  backgroundColor: 'transparent',
  textStyle: { color: '#7FA3AE', fontFamily: 'IBM Plex Sans' },
  tooltip: { trigger: 'axis' },
  legend: {
    data: ['Pages/Min', '失败率 %'],
    textStyle: { color: '#7FA3AE' },
    top: 4,
  },
  grid: { left: 48, right: 48, top: 48, bottom: 32 },
  xAxis: {
    type: 'category',
    data: metrics.series.map((p) => p.ts.slice(11)),
    axisLine: { lineStyle: { color: '#1E3A47' } },
    axisLabel: { color: '#7FA3AE', fontFamily: 'IBM Plex Mono' },
  },
  yAxis: [
    {
      type: 'value',
      name: 'ppm',
      splitLine: { lineStyle: { color: '#1E3A47' } },
      axisLabel: { color: '#7FA3AE' },
    },
    {
      type: 'value',
      name: '%',
      splitLine: { show: false },
      axisLabel: { color: '#7FA3AE' },
    },
  ],
  series: [
    {
      name: 'Pages/Min',
      type: 'line',
      smooth: true,
      symbol: 'circle',
      symbolSize: 6,
      data: metrics.series.map((p) => Number(p.pages_per_min.toFixed(1))),
      lineStyle: { color: '#3EE0C5', width: 2 },
      itemStyle: { color: '#3EE0C5' },
      areaStyle: { color: 'rgba(62,224,197,0.12)' },
    },
    {
      name: '失败率 %',
      type: 'line',
      yAxisIndex: 1,
      smooth: true,
      data: metrics.series.map((p) => {
        const n = p.fail_rate
        return Number(((n <= 1 ? n * 100 : n)).toFixed(2))
      }),
      lineStyle: { color: '#F5B942', width: 2 },
      itemStyle: { color: '#F5B942' },
    },
  ],
}))

onMounted(async () => {
  await metrics.hydrate()
  metrics.connect()
  metrics.startPolling()
})

onUnmounted(() => {
  metrics.teardown()
})

async function refresh() {
  await metrics.hydrate()
  toastSuccess('指标已刷新')
}
</script>

<template>
  <div class="w-full">
    <PageHeader title="集群声呐" subtitle="节点负载、抓取速率与失败率，WebSocket ≤3s 推送。">
      <n-button class="sonar-btn" secondary @click="refresh">手动刷新</n-button>
      <span class="font-mono text-xs" :class="metrics.connected ? 'text-cyan' : 'text-amber'">
        {{ metrics.connected ? 'WS 在线' : 'WS 重连 / 轮询' }}
      </span>
    </PageHeader>

    <section class="mb-5 grid w-full grid-cols-2 gap-3 md:grid-cols-4">
      <div class="sonar-card px-4 py-3">
        <div class="font-mono text-[10px] uppercase tracking-widest text-muted">队列待领取</div>
        <div class="font-display mt-1 text-2xl">{{ metrics.queue.pending }}</div>
      </div>
      <div class="sonar-card px-4 py-3">
        <div class="font-mono text-[10px] uppercase tracking-widest text-muted">租约中</div>
        <div class="font-display mt-1 text-2xl">{{ metrics.queue.leased }}</div>
      </div>
      <div class="sonar-card px-4 py-3">
        <div class="font-mono text-[10px] uppercase tracking-widest text-muted">合计 Pages/Min</div>
        <div class="font-display mt-1 text-2xl text-cyan">{{ formatNumber(metrics.totals.ppm, 0) }}</div>
      </div>
      <div class="sonar-card px-4 py-3">
        <div class="font-mono text-[10px] uppercase tracking-widest text-muted">平均失败率</div>
        <div class="font-display mt-1 text-2xl">{{ formatPct(metrics.totals.fail) }}</div>
      </div>
    </section>

    <section class="mb-5 grid w-full grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      <NodeCard v-for="node in metrics.nodes" :key="node.id" :node="node" />
      <div v-if="!metrics.nodes.length" class="sonar-card px-5 py-8 text-sm text-muted">
        尚未收到节点心跳。确认 Worker 已接入 Master 控制面。
      </div>
    </section>

    <section class="sonar-card w-full p-4">
      <div class="mb-2 font-display tracking-wide">速率回波</div>
      <v-chart class="rate-chart" :option="option" autoresize />
    </section>
  </div>
</template>

<style scoped>
.rate-chart {
  width: 100%;
  height: 340px;
}
@media (max-width: 480px) {
  .rate-chart {
    height: 240px;
  }
}
</style>
