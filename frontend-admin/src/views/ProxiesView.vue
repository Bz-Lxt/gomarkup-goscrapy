<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { NButton, NDataTable } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import StatusTag from '@/components/StatusTag.vue'
import { fetchProxies } from '@/api'
import type { ProxyItem } from '@/api/types'
import { formatDateTime, formatNumber } from '@/utils/format'
import { toastSuccess } from '@/utils/feedback'

const loading = ref(false)
const rows = ref<ProxyItem[]>([])

const columns: DataTableColumns<ProxyItem> = [
  { title: '地址', key: 'address', ellipsis: { tooltip: true } },
  {
    title: '状态',
    key: 'status',
    width: 120,
    render: (row) => h(StatusTag, { value: row.status }),
  },
  { title: '命中', key: 'hits', width: 90 },
  { title: '驱逐', key: 'evictions', width: 90 },
  {
    title: '延迟',
    key: 'latency_ms',
    width: 110,
    render: (row) => `${formatNumber(row.latency_ms, 0)} ms`,
  },
  {
    title: '最近检查',
    key: 'last_check',
    width: 200,
    render: (row) => formatDateTime(row.last_check),
  },
]

const totals = () => {
  const hits = rows.value.reduce((s, r) => s + (r.hits || 0), 0)
  const evictions = rows.value.reduce((s, r) => s + (r.evictions || 0), 0)
  const healthy = rows.value.filter((r) => ['healthy', 'up', 'online', 'ok'].includes(r.status.toLowerCase())).length
  return { hits, evictions, healthy }
}

async function load() {
  loading.value = true
  try {
    rows.value = await fetchProxies()
    toastSuccess(`代理池 ${rows.value.length} 条`)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="w-full">
    <PageHeader title="代理池" subtitle="轮询命中、健康检查与失败驱逐计数。模式由 PROXY_POOL_MODE 切换。">
      <n-button class="sonar-btn" type="primary" :loading="loading" @click="load">刷新状态</n-button>
    </PageHeader>

    <section class="mb-5 grid w-full grid-cols-1 gap-3 phone:grid-cols-1 md:grid-cols-3">
      <div class="sonar-card px-4 py-3">
        <div class="font-mono text-[10px] uppercase tracking-widest text-muted">健康节点</div>
        <div class="font-display mt-1 text-2xl text-cyan">{{ totals().healthy }} / {{ rows.length }}</div>
      </div>
      <div class="sonar-card px-4 py-3">
        <div class="font-mono text-[10px] uppercase tracking-widest text-muted">累计命中</div>
        <div class="font-display mt-1 text-2xl">{{ totals().hits }}</div>
      </div>
      <div class="sonar-card px-4 py-3">
        <div class="font-mono text-[10px] uppercase tracking-widest text-muted">驱逐计数</div>
        <div class="font-display mt-1 text-2xl">{{ totals().evictions }}</div>
      </div>
    </section>

    <div class="table-scroll sonar-card p-3">
      <n-data-table
        :columns="columns"
        :data="rows"
        :loading="loading"
        :bordered="false"
        striped
        :row-key="(r: ProxyItem) => r.id || r.address"
      />
    </div>
  </div>
</template>
