<script setup lang="ts">
import { h, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NDataTable, NInput, NInputNumber, NModal, NPagination, NSelect, NSpace } from 'naive-ui'
import type { DataTableColumns, SelectOption } from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import StatusTag from '@/components/StatusTag.vue'
import { cancelTask, createTask, listRules, listTasks, pauseTask, startTask } from '@/api'
import type { Task } from '@/api/types'
import { formatDateTime } from '@/utils/format'
import { confirmDanger, toastError, toastSuccess } from '@/utils/feedback'
import { firstError, hasErrors, inRange, isHttpUrl, required, type ErrorMap } from '@/utils/validate'

const router = useRouter()
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const rows = ref<Task[]>([])
const statusFilter = ref('')
const showCreate = ref(false)
const submitting = ref(false)
const ruleOptions = ref<SelectOption[]>([])

const form = reactive({
  name: '',
  rule_id: null as number | null,
  seed_text: 'http://mock-target/list.html',
  max_depth: 2,
  concurrency: 4,
})
const errors = reactive<ErrorMap>({})

const statusOptions = [
  { label: '全部状态', value: '' },
  { label: 'created', value: 'created' },
  { label: 'running', value: 'running' },
  { label: 'paused', value: 'paused' },
  { label: 'succeeded', value: 'succeeded' },
  { label: 'failed', value: 'failed' },
  { label: 'cancelled', value: 'cancelled' },
]

function actionsFor(row: Task) {
  const buttons = [
    h(NButton, { size: 'tiny', class: 'sonar-btn', onClick: () => goDetail(row.id) }, () => '详情'),
  ]
  if (row.status === 'created' || row.status === 'paused') {
    buttons.push(
      h(NButton, { size: 'tiny', type: 'primary', class: 'sonar-btn', onClick: () => onStart(row) }, () => '启动'),
    )
  }
  if (row.status === 'running') {
    buttons.push(
      h(NButton, { size: 'tiny', secondary: true, class: 'sonar-btn', onClick: () => onPause(row) }, () => '暂停'),
    )
  }
  if (row.status === 'created' || row.status === 'running' || row.status === 'paused') {
    buttons.push(
      h(
        NButton,
        { size: 'tiny', type: 'error', ghost: true, class: 'sonar-btn-danger', onClick: () => onCancel(row) },
        () => '取消',
      ),
    )
  }
  return h(NSpace, { size: 6 }, () => buttons)
}

const columns: DataTableColumns<Task> = [
  { title: 'ID', key: 'id', width: 70 },
  { title: '名称', key: 'name', ellipsis: { tooltip: true } },
  { title: '规则', key: 'rule_id', width: 80 },
  {
    title: '状态',
    key: 'status',
    width: 120,
    render: (row) => h(StatusTag, { value: row.status }),
  },
  { title: '深度', key: 'max_depth', width: 70 },
  { title: '并发', key: 'concurrency', width: 70 },
  {
    title: '更新时间',
    key: 'updated_at',
    width: 190,
    render: (row) => formatDateTime(row.updated_at),
  },
  { title: '操作', key: 'actions', width: 240, render: (row) => actionsFor(row) },
]

async function load() {
  loading.value = true
  try {
    const res = await listTasks(page.value, pageSize.value, statusFilter.value)
    rows.value = res.items
    total.value = res.total
  } finally {
    loading.value = false
  }
}

async function loadRules() {
  const res = await listRules(1, 100, '')
  ruleOptions.value = res.items.map((r) => ({ label: `${r.id} · ${r.name}`, value: r.id }))
}

function goDetail(id: number) {
  void router.push({ name: 'task-detail', params: { id: String(id) } })
}

async function onStart(row: Task) {
  await startTask(row.id)
  toastSuccess(`任务 #${row.id} 已启动`)
  await load()
}

async function onPause(row: Task) {
  await pauseTask(row.id)
  toastSuccess(`任务 #${row.id} 已暂停`)
  await load()
}

async function onCancel(row: Task) {
  const ok = await confirmDanger({
    title: '取消任务',
    content: `确认取消任务「${row.name}」？进行中的租约会被回收。`,
    positiveText: '取消任务',
  })
  if (!ok) return
  await cancelTask(row.id)
  toastSuccess('任务已取消')
  await load()
}

function parseSeeds(): string[] {
  return form.seed_text
    .split(/[\n,]/)
    .map((s) => s.trim())
    .filter(Boolean)
}

function validate(): boolean {
  errors.name = required(form.name, '任务名')
  errors.rule_id = form.rule_id ? undefined : '必须选择规则'
  const seeds = parseSeeds()
  errors.seed_text = seeds.length ? undefined : '至少一条种子 URL'
  if (!errors.seed_text) {
    errors.seed_text = seeds.map((u) => isHttpUrl(u, '种子 URL')).find(Boolean)
  }
  errors.max_depth = inRange(Number(form.max_depth), 0, 20, '最大深度')
  errors.concurrency = inRange(Number(form.concurrency), 1, 32, '并发')
  return !hasErrors(errors)
}

async function onCreate() {
  if (!validate()) {
    toastError(firstError(errors) || '请修正任务表单')
    return
  }
  submitting.value = true
  try {
    const created = await createTask({
      name: form.name.trim(),
      rule_id: Number(form.rule_id),
      seed_urls: parseSeeds(),
      max_depth: Number(form.max_depth),
      concurrency: Number(form.concurrency),
    })
    toastSuccess(`任务已创建 #${created.id ?? ''}`)
    showCreate.value = false
    await load()
    if (created.id) goDetail(created.id)
  } finally {
    submitting.value = false
  }
}

function openCreate() {
  showCreate.value = true
  void loadRules()
}

onMounted(load)
</script>

<template>
  <div class="w-full">
    <PageHeader title="任务队列" subtitle="创建任务并驱动状态机：created → running → paused | succeeded | failed | cancelled。">
      <n-select
        v-model:value="statusFilter"
        class="w-[160px]"
        :options="statusOptions"
        @update:value="() => { page = 1; load() }"
      />
      <n-button class="sonar-btn" secondary @click="load">刷新</n-button>
      <n-button class="sonar-btn" type="primary" @click="openCreate">创建任务</n-button>
    </PageHeader>

    <div class="table-scroll sonar-card p-3">
      <n-data-table :columns="columns" :data="rows" :loading="loading" :bordered="false" striped :row-key="(r: Task) => r.id" />
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

    <n-modal v-model:show="showCreate" preset="card" title="创建抓取任务" style="width: min(560px, 92vw)">
      <label class="mb-1 block text-xs text-muted">任务名 *</label>
      <n-input v-model:value="form.name" placeholder="demo-crawl" />
      <div v-if="errors.name" class="field-error">{{ errors.name }}</div>

      <label class="mb-1 mt-3 block text-xs text-muted">绑定规则 *</label>
      <n-select v-model:value="form.rule_id" :options="ruleOptions" placeholder="选择规则" />
      <div v-if="errors.rule_id" class="field-error">{{ errors.rule_id }}</div>

      <label class="mb-1 mt-3 block text-xs text-muted">种子 URL *（换行或逗号分隔）</label>
      <n-input v-model:value="form.seed_text" type="textarea" :rows="3" />
      <div v-if="errors.seed_text" class="field-error">{{ errors.seed_text }}</div>

      <div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
        <div>
          <div class="mb-1 text-xs text-muted">最大深度 0–20</div>
          <n-input-number v-model:value="form.max_depth" :min="0" :max="20" class="w-full" />
          <div v-if="errors.max_depth" class="field-error">{{ errors.max_depth }}</div>
        </div>
        <div>
          <div class="mb-1 text-xs text-muted">并发 1–32</div>
          <n-input-number v-model:value="form.concurrency" :min="1" :max="32" class="w-full" />
          <div v-if="errors.concurrency" class="field-error">{{ errors.concurrency }}</div>
        </div>
      </div>

      <div class="mt-5 flex justify-end gap-2">
        <n-button class="sonar-btn" secondary @click="showCreate = false">关闭</n-button>
        <n-button class="sonar-btn" type="primary" :loading="submitting" @click="onCreate">提交创建</n-button>
      </div>
    </n-modal>
  </div>
</template>
