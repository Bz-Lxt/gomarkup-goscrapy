import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { fetchClusterMetrics, fetchClusterNodes, fetchQueueStats } from '@/api'
import type { ClusterNode, MetricPoint, QueueStats } from '@/api/types'
import { TOKEN_KEY } from '@/constants'
import { logger } from '@/utils/logger'

const MAX_POINTS = 40

export const useMetricsStore = defineStore('metrics', () => {
  const nodes = ref<ClusterNode[]>([])
  const series = ref<MetricPoint[]>([])
  const queue = ref<QueueStats>({ pending: 0, leased: 0, delayed: 0, dead: 0 })
  const connected = ref(false)
  const lastTs = ref('')

  let ws: WebSocket | null = null
  let retry = 0
  let closed = false
  let timer: number | null = null

  const totals = computed(() => {
    const ppm = nodes.value.reduce((s, n) => s + (n.pages_per_min || 0), 0)
    const fail = nodes.value.length
      ? nodes.value.reduce((s, n) => s + (n.fail_rate || 0), 0) / nodes.value.length
      : 0
    return { ppm, fail }
  })

  function applyNodes(list: ClusterNode[], ts?: string) {
    if (!list.length) return
    nodes.value = list
    const stamp = ts || lastTs.value
    lastTs.value = stamp
    series.value = [
      ...series.value,
      {
        ts: stamp,
        pages_per_min: list.reduce((s, n) => s + (n.pages_per_min || 0), 0),
        fail_rate: list.reduce((s, n) => s + (n.fail_rate || 0), 0) / list.length,
      },
    ].slice(-MAX_POINTS)
  }

  async function hydrate() {
    try {
      const [fromNodes, fromMetrics, qs] = await Promise.all([
        fetchClusterNodes().catch(() => [] as ClusterNode[]),
        fetchClusterMetrics().catch(() => [] as ClusterNode[]),
        fetchQueueStats().catch(() => queue.value),
      ])
      queue.value = qs
      applyNodes(fromMetrics.length ? fromMetrics : fromNodes, lastTs.value)
    } catch (err) {
      logger.warn('hydrate metrics failed', err)
    }
  }

  function wsUrl(): string {
    const token = localStorage.getItem(TOKEN_KEY) ?? ''
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${location.host}/api/v1/ws/metrics?token=${encodeURIComponent(token)}`
  }

  function connect() {
    closed = false
    disconnectSocket()
    try {
      ws = new WebSocket(wsUrl())
    } catch (err) {
      logger.warn('ws construct failed', err)
      scheduleReconnect()
      return
    }
    ws.onopen = () => {
      connected.value = true
      retry = 0
      logger.info('metrics ws open')
    }
    ws.onmessage = (ev) => {
      try {
        const frame = JSON.parse(String(ev.data)) as {
          type?: string
          ts?: string
          nodes?: ClusterNode[]
        }
        if (frame.nodes) applyNodes(frame.nodes, frame.ts)
      } catch (err) {
        logger.warn('ws frame parse failed', err)
      }
    }
    ws.onerror = () => {
      connected.value = false
    }
    ws.onclose = () => {
      connected.value = false
      if (!closed) scheduleReconnect()
    }
  }

  function scheduleReconnect() {
    if (closed) return
    const wait = Math.min(8000, 800 * 2 ** retry)
    retry += 1
    window.setTimeout(() => {
      if (!closed) connect()
    }, wait)
  }

  function startPolling() {
    stopPolling()
    timer = window.setInterval(() => {
      if (!connected.value) void hydrate()
    }, 5000)
  }

  function stopPolling() {
    if (timer != null) {
      window.clearInterval(timer)
      timer = null
    }
  }

  function disconnectSocket() {
    if (ws) {
      ws.onclose = null
      ws.close()
      ws = null
    }
    connected.value = false
  }

  function teardown() {
    closed = true
    stopPolling()
    disconnectSocket()
  }

  return {
    nodes,
    series,
    queue,
    connected,
    lastTs,
    totals,
    hydrate,
    connect,
    startPolling,
    teardown,
  }
})
