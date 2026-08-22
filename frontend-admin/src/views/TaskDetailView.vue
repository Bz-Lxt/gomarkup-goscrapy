<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NDataTable, NPagination } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import StatusTag from '@/components/StatusTag.vue'
import { cancelTask, getTask, listTaskResults, pauseTask, startTask } from '@/api'
import type { ResultItem, Task } from '@/api/types'
import { formatDateTime, prettyJson } from '@/utils/format'
import { confirmDanger, toastSuccess } from '@/utils/feedback'

const route = useRoute()
const router = useRouter()
const id = computed(() => Number(route.params.id))
const task = ref<Task | null>(null)
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const rows = ref<ResultItem[]>([])

const columns: DataTableColumns<ResultItem> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: 'URL', key: 'url', ellipsis: { tooltip: true } },
  {
    title: 'PAYLOAD',
    key: 'payload',
    render: (row) => h('pre', { class: 'font-mono whitespace-pre-wrap text-xs' }, prettyJson(row.payload)),
  },
  {
    title: '抓取时间',
    key: 'created_at',
    width: 190,
    render: (row) => formatDateTime(row.created_at),
  },
]

async function loadTask() {
  task.value = await getTask(id.value)
}

async function loadResults() {
  loading.value = true
  try {
    const res = await listTaskResults(id.value, page.value, pageSize.value)
    rows.value = res.items
    total.value = res.total
  } finally {
    loading.value = false
  }
}

async function reload() {
  await Promise.all([loadTask(), loadResults()])
}

async function onStart() {
  await startTask(id.value)
  toastSuccess('任务已启动')
  await reload()
}

async function onPause() {
  await pauseTask(id.value)
  toastSuccess('任务已暂停')
  await reload()
}

async function onCancel() {
  const ok = await confirmDanger({
    title: '取消任务',
    content: `确认取消任务 #${id.value}？`,
    positiveText: '取消任务',
  })
  if (!ok) return
  await cancelTask(id.value)
  toastSuccess('任务已取消')
  await reload()
}

onMounted(reload)
</script>

<template>
  <div class="w-full">
    <PageHeader :title="task ? task.name : `任务 #${id}`" :subtitle="`任务详情与结果分页 · #${id}`">
      <n-button class="sonar-btn" secondary @click="router.push({ name: 'tasks' })">返回列表</n-button>
      <n-button class="sonar-btn" secondary @click="reload">刷新</n-button>
      <n-button
        v-if="task && (task.status === 'created' || task.status === 'paused')"
        class="sonar-btn"
        type="primary"
        @click="onStart"
      >
        启动
      </n-button>
      <n-button v-if="task?.status === 'running'" class="sonar-btn" secondary @click="onPause">暂停</n-button>
      <n-button
        v-if="task && ['created', 'running', 'paused'].includes(task.status)"
        class="sonar-btn-danger"
        type="error"
        ghost
        @click="onCancel"
      >
        取消
      </n-button>
    </PageHeader>

    <section v-if="task" class="mb-5 grid w-full grid-cols-2 gap-3 md:grid-cols-4">
      <div class="sonar-card px-4 py-3">
        <div class="font-mono text-[10px] uppercase tracking-widest text-muted">状态</div>
        <div class="mt-2"><StatusTag :value="task.status" /></div>
      </div>
      <div class="sonar-card px-4 py-3">
        <div class="font-mono text-[10px] uppercase tracking-widest text-muted">规则</div>
        <div class="mt-2 font-mono">#{{ task.rule_id }} {{ task.rule_name || '' }}</div>
      </div>
      <div class="sonar-card px-4 py-3">
        <div class="font-mono text-[10px] uppercase tracking-widest text-muted">已抓 / 失败</div>
        <div class="mt-2 font-display text-xl">{{ task.pages_crawled ?? 0 }} / {{ task.pages_failed ?? 0 }}</div>
      </div>
      <div class="sonar-card px-4 py-3">
        <div class="font-mono text-[10px] uppercase tracking-widest text-muted">更新时间</div>
        <div class="mt-2 font-mono text-sm">{{ formatDateTime(task.updated_at) }}</div>
      </div>
    </section>

    <section v-if="task" class="sonar-card mb-5 p-4">
      <div class="font-display mb-2">种子 URL</div>
      <div class="font-mono break-all text-xs text-muted">{{ (task.seed_urls || []).join('  ·  ') }}</div>
      <div class="mt-3 font-mono text-xs text-muted">
        深度 {{ task.max_depth }} · 并发 {{ task.concurrency }} · 创建 {{ formatDateTime(task.created_at) }}
      </div>
    </section>

    <section class="table-scroll sonar-card p-3">
      <div class="mb-3 px-1 font-display">抓取结果</div>
      <n-data-table
        :columns="columns"
        :data="rows"
        :loading="loading"
        :bordered="false"
        striped
        :row-key="(r: ResultItem) => r.id"
      />
      <div class="mt-4 flex justify-end">
        <n-pagination
          v-model:page="page"
          v-model:page-size="pageSize"
          :item-count="total"
          :page-sizes="[10, 20, 50]"
          show-size-picker
          @update:page="loadResults"
          @update:page-size="loadResults"
        />
      </div>
    </section>
  </div>
</template>
