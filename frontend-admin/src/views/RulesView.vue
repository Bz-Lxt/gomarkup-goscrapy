<script setup lang="ts">
import { h, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NDataTable, NInput, NPagination, NSpace } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import { deleteRule, listRules, previewRule } from '@/api'
import type { Rule } from '@/api/types'
import { formatDateTime } from '@/utils/format'
import { confirmDanger, toastError, toastSuccess } from '@/utils/feedback'

const router = useRouter()
const loading = ref(false)
const keyword = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const rows = ref<Rule[]>([])

const columns: DataTableColumns<Rule> = [
  { title: 'ID', key: 'id', width: 70 },
  { title: '名称', key: 'name', ellipsis: { tooltip: true } },
  { title: '起始 URL', key: 'start_url', ellipsis: { tooltip: true } },
  { title: 'ITEM', key: 'item_selector', width: 160, ellipsis: { tooltip: true } },
  { title: 'QPS', key: 'qps', width: 70 },
  { title: '版本', key: 'version', width: 70 },
  {
    title: '更新时间',
    key: 'updated_at',
    width: 190,
    render: (row) => formatDateTime(row.updated_at),
  },
  {
    title: '操作',
    key: 'actions',
    width: 280,
    render: (row) =>
      h(NSpace, { size: 6 }, () => [
        h(NButton, { size: 'tiny', class: 'sonar-btn', onClick: () => goEdit(row.id) }, () => '配置'),
        h(NButton, { size: 'tiny', secondary: true, class: 'sonar-btn', onClick: () => onPreview(row) }, () => '预览'),
        h(
          NButton,
          { size: 'tiny', type: 'error', ghost: true, class: 'sonar-btn-danger', onClick: () => onDelete(row) },
          () => '删除',
        ),
      ]),
  },
]

const preview = reactive({ open: false, text: '' })

async function load() {
  loading.value = true
  try {
    const res = await listRules(page.value, pageSize.value, keyword.value.trim())
    rows.value = res.items
    total.value = res.total
  } finally {
    loading.value = false
  }
}

function goNew() {
  void router.push({ name: 'rule-new' })
}

function goEdit(id: number) {
  void router.push({ name: 'rule-edit', params: { id: String(id) } })
}

async function onDelete(row: Rule) {
  const ok = await confirmDanger({
    title: '删除规则',
    content: `确认删除规则「${row.name}」？已绑定任务不会自动取消。`,
    positiveText: '删除',
  })
  if (!ok) return
  await deleteRule(row.id)
  toastSuccess('规则已删除')
  await load()
}

async function onPreview(row: Rule) {
  try {
    const data = await previewRule(row.id, { html: '', url: row.start_url })
    preview.text = JSON.stringify(data, null, 2)
    preview.open = true
    toastSuccess('预览已返回')
  } catch {
    toastError('预览失败')
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
    <PageHeader title="规则工坊" subtitle="维护 XPath / CSS / Regex 提取规则，进入可视化配置器点选元素。">
      <n-input
        v-model:value="keyword"
        class="w-[220px]"
        placeholder="按名称搜索"
        @keydown.enter="search"
      />
      <n-button class="sonar-btn" secondary @click="search">搜索</n-button>
      <n-button class="sonar-btn" type="primary" @click="goNew">新建规则</n-button>
    </PageHeader>

    <div class="table-scroll sonar-card p-3">
      <n-data-table
        :columns="columns"
        :data="rows"
        :loading="loading"
        :bordered="false"
        striped
        :row-key="(r: Rule) => r.id"
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

    <div v-if="preview.open" class="sonar-card mt-4 p-4">
      <div class="mb-2 flex items-center justify-between">
        <div class="font-display">规则预览</div>
        <n-button size="tiny" secondary class="sonar-btn" @click="preview.open = false">关闭</n-button>
      </div>
      <pre class="font-mono whitespace-pre-wrap text-xs text-ink">{{ preview.text }}</pre>
    </div>
  </div>
</template>
