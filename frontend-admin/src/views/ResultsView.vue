<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NDataTable, NInputNumber, NPagination } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import { listResults } from '@/api'
import type { ResultItem } from '@/api/types'
import { formatDateTime, prettyJson } from '@/utils/format'

const router = useRouter()
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const rows = ref<ResultItem[]>([])
const taskId = ref<number | null>(null)

const columns: DataTableColumns<ResultItem> = [
  { title: 'ID', key: 'id', width: 80 },
  {
    title: '任务',
    key: 'task_id',
    width: 90,
    render: (row) =>
      h(
        NButton,
        { text: true, type: 'primary', onClick: () => router.push({ name: 'task-detail', params: { id: String(row.task_id) } }) },
        () => `#${row.task_id}`,
      ),
  },
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

async function load() {
  loading.value = true
  try {
    const res = await listResults(page.value, pageSize.value, taskId.value ?? undefined)
    rows.value = res.items
    total.value = res.total
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  void load()
}

onMounted(load)
</script>

<template>
  <div class="w-full">
    <PageHeader title="全局结果" subtitle="跨任务浏览已落库的结构化抓取结果。">
      <n-input-number v-model:value="taskId" placeholder="按 task_id 过滤" :min="1" class="w-[180px]" clearable />
      <n-button class="sonar-btn" secondary @click="search">筛选</n-button>
      <n-button class="sonar-btn" @click="() => { taskId = null; search() }">清除过滤</n-button>
    </PageHeader>

    <div class="table-scroll sonar-card p-3">
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
          @update:page="load"
          @update:page-size="load"
        />
      </div>
    </div>
  </div>
</template>
