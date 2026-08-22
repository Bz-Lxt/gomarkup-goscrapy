<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NInput, NInputNumber, NSelect, NSwitch } from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import HeatmapCanvas from '@/components/HeatmapCanvas.vue'
import {
  createRule,
  createSnapshot,
  fetchSelectors,
  fetchSnapshotImage,
  getRule,
  previewRule,
  updateRule,
} from '@/api'
import type { ExtractorKind, RuleField, SelectorCandidate, Snapshot, SnapshotNode } from '@/api/types'
import { confirmDanger, toastError, toastSuccess } from '@/utils/feedback'
import { firstError, hasErrors, inRange, isHttpUrl, maxLen, required, type ErrorMap } from '@/utils/validate'
import { prettyJson } from '@/utils/format'

const route = useRoute()
const router = useRouter()

const isNew = computed(() => route.name === 'rule-new')
const ruleId = computed(() => Number(route.params.id))

const form = reactive({
  name: '',
  start_url: 'http://mock-target/list.html',
  item_selector: '',
  link_selector: '',
  respect_robots: true,
  qps: 2,
  fields: [] as RuleField[],
})
const errors = reactive<ErrorMap>({})
const fieldErrors = ref<ErrorMap[]>([])
const saving = ref(false)
const loadingRule = ref(false)

const snapUrl = ref('http://mock-target/list.html')
const snapLoading = ref(false)
const snapshot = ref<Snapshot | null>(null)
const imageSrc = ref('')
const selected = ref<SnapshotNode | null>(null)
const candidates = ref<SelectorCandidate[]>([])
const listHint = ref<{ item_selector: string; field_selector: string; hit_count: number } | null>(null)
const fieldNameDraft = ref('')
const previewText = ref('')

const kindOptions = [
  { label: 'CSS', value: 'css' },
  { label: 'XPath', value: 'xpath' },
  { label: 'Regex', value: 'regex' },
]

function emptyField(): RuleField {
  return { name: '', kind: 'css', expr: '', attr: 'text' }
}

function addField(seed?: Partial<RuleField>) {
  form.fields.push({ ...emptyField(), ...seed })
  fieldErrors.value.push({})
}

function removeField(idx: number) {
  form.fields.splice(idx, 1)
  fieldErrors.value.splice(idx, 1)
}

function validate(): boolean {
  errors.name = required(form.name, '规则名') || maxLen(form.name, 64, '规则名')
  errors.start_url = isHttpUrl(form.start_url, '起始 URL')
  errors.item_selector = required(form.item_selector, '列表选择器')
  errors.qps = inRange(Number(form.qps), 0.1, 100, 'QPS')
  if (form.fields.length === 0) {
    errors.fields = '至少添加一个提取字段'
  } else {
    errors.fields = undefined
  }
  fieldErrors.value = form.fields.map((f) => ({
    name: required(f.name, '字段名') || maxLen(f.name, 64, '字段名'),
    expr: required(f.expr, '表达式'),
    kind: required(f.kind, '类型'),
  }))
  const fieldBad = fieldErrors.value.some(hasErrors)
  return !hasErrors(errors) && !fieldBad
}

async function loadExisting() {
  if (isNew.value) {
    if (form.fields.length === 0) addField({ name: 'title', kind: 'css', expr: '.title', attr: 'text' })
    return
  }
  loadingRule.value = true
  try {
    const rule = await getRule(ruleId.value)
    form.name = rule.name
    form.start_url = rule.start_url
    form.item_selector = rule.item_selector
    form.link_selector = rule.link_selector
    form.respect_robots = rule.respect_robots
    form.qps = rule.qps
    form.fields = (rule.fields || []).map((f) => ({ ...f }))
    fieldErrors.value = form.fields.map(() => ({}))
    snapUrl.value = rule.start_url
  } finally {
    loadingRule.value = false
  }
}

async function onSave() {
  if (!validate()) {
    toastError(firstError(errors) || '请修正字段校验错误')
    return
  }
  saving.value = true
  try {
    const payload = {
      name: form.name.trim(),
      start_url: form.start_url.trim(),
      item_selector: form.item_selector.trim(),
      link_selector: form.link_selector.trim(),
      fields: form.fields.map((f) => ({
        name: f.name.trim(),
        kind: f.kind,
        expr: f.expr.trim(),
        attr: (f.attr || 'text').trim(),
      })),
      respect_robots: form.respect_robots,
      qps: Number(form.qps),
    }
    if (isNew.value) {
      const created = await createRule(payload)
      toastSuccess('规则已创建')
      await router.replace({ name: 'rule-edit', params: { id: String(created.id) } })
    } else {
      await updateRule(ruleId.value, payload)
      toastSuccess('规则已保存，版本 +1')
    }
  } finally {
    saving.value = false
  }
}

async function onSnapshot() {
  const err = isHttpUrl(snapUrl.value, '快照 URL')
  if (err) {
    toastError(err)
    return
  }
  snapLoading.value = true
  selected.value = null
  candidates.value = []
  listHint.value = null
  if (imageSrc.value) URL.revokeObjectURL(imageSrc.value)
  imageSrc.value = ''
  try {
    const snap = await createSnapshot(snapUrl.value.trim())
    snapshot.value = snap
    imageSrc.value = await fetchSnapshotImage(snap.image_url)
    form.start_url = snapUrl.value.trim()
    toastSuccess(`快照完成，热区 ${snap.nodes?.length ?? 0} 个`)
  } finally {
    snapLoading.value = false
  }
}

async function onSelectNode(node: SnapshotNode) {
  selected.value = node
  if (!snapshot.value) return
  const res = await fetchSelectors(snapshot.value.snapshot_id, node.node_id)
  candidates.value = res.candidates || []
  listHint.value = res.list_rule ?? null
  if (!fieldNameDraft.value) {
    fieldNameDraft.value = (node.text || node.tag || 'field').slice(0, 24)
  }
}

function applyCandidate(c: SelectorCandidate, target: 'item' | 'link' | 'field') {
  if (target === 'item') {
    form.item_selector = c.expr
    toastSuccess('已写入列表选择器')
    return
  }
  if (target === 'link') {
    form.link_selector = c.expr
    toastSuccess('已写入链接选择器')
    return
  }
  const name = fieldNameDraft.value.trim() || selected.value?.tag || 'field'
  addField({ name, kind: c.kind as ExtractorKind, expr: c.expr, attr: 'text' })
  toastSuccess(`已添加字段 ${name}`)
}

function applyListHint() {
  if (!listHint.value) return
  form.item_selector = listHint.value.item_selector
  const name = fieldNameDraft.value.trim() || 'title'
  addField({ name, kind: 'css', expr: listHint.value.field_selector, attr: 'text' })
  toastSuccess(`已应用列表规则，命中 ${listHint.value.hit_count} 条`)
}

async function onPreview() {
  if (isNew.value) {
    toastError('请先保存规则再预览')
    return
  }
  const data = await previewRule(ruleId.value, { html: '', url: form.start_url })
  previewText.value = prettyJson(data)
}

async function onLeave() {
  const ok = await confirmDanger({
    title: '返回列表',
    content: '未保存的修改将丢失，确认离开？',
  })
  if (ok) void router.push({ name: 'rules' })
}

onMounted(loadExisting)
</script>

<template>
  <div class="w-full">
    <PageHeader
      :title="isNew ? '新建规则' : `规则 #${ruleId}`"
      subtitle="输入 URL 拍摄快照，点击热区生成选择器候选。"
    >
      <n-button class="sonar-btn" secondary @click="onLeave">返回列表</n-button>
      <n-button v-if="!isNew" class="sonar-btn" secondary :disabled="saving" @click="onPreview">试运行</n-button>
      <n-button class="sonar-btn" type="primary" :loading="saving" @click="onSave">保存规则</n-button>
    </PageHeader>

    <div class="grid w-full grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1.4fr)_minmax(320px,1fr)]">
      <section class="sonar-card p-4">
        <div class="mb-3 flex flex-col gap-2 md:flex-row">
          <n-input v-model:value="snapUrl" class="flex-1" placeholder="http://mock-target/list.html" />
          <n-button class="sonar-btn" type="primary" :loading="snapLoading" @click="onSnapshot">拍摄快照</n-button>
        </div>
        <HeatmapCanvas
          v-if="snapshot && imageSrc"
          :snapshot="snapshot"
          :image-src="imageSrc"
          :selected-id="selected?.node_id"
          @select="onSelectNode"
        />
        <div v-else class="px-2 py-16 text-center text-sm text-muted">
          {{ loadingRule ? '正在载入规则…' : snapLoading ? '渲染器正在成像…' : '输入目标 URL 并拍摄快照，热区将叠在截图上。' }}
        </div>
      </section>

      <section class="space-y-4">
        <div class="sonar-card p-4">
          <div class="font-display mb-3">规则字段</div>
          <label class="mb-1 block text-xs text-muted">规则名 *</label>
          <n-input v-model:value="form.name" placeholder="mock-shop-list" />
          <div v-if="errors.name" class="field-error">{{ errors.name }}</div>

          <label class="mb-1 mt-3 block text-xs text-muted">起始 URL *</label>
          <n-input v-model:value="form.start_url" />
          <div v-if="errors.start_url" class="field-error">{{ errors.start_url }}</div>

          <label class="mb-1 mt-3 block text-xs text-muted">列表选择器 item_selector *</label>
          <n-input v-model:value="form.item_selector" class="font-mono" placeholder=".product-card" />
          <div v-if="errors.item_selector" class="field-error">{{ errors.item_selector }}</div>

          <label class="mb-1 mt-3 block text-xs text-muted">链接选择器 link_selector</label>
          <n-input v-model:value="form.link_selector" class="font-mono" placeholder="a.product-link" />

          <div class="mt-3 flex items-center justify-between gap-3">
            <div>
              <div class="text-xs text-muted">遵守 robots.txt</div>
              <n-switch v-model:value="form.respect_robots" />
            </div>
            <div class="flex-1">
              <div class="text-xs text-muted">QPS（0.1–100）</div>
              <n-input-number v-model:value="form.qps" :min="0.1" :max="100" :step="0.1" class="w-full" />
              <div v-if="errors.qps" class="field-error">{{ errors.qps }}</div>
            </div>
          </div>
        </div>

        <div class="sonar-card p-4">
          <div class="mb-3 flex items-center justify-between">
            <div class="font-display">提取字段</div>
            <n-button size="tiny" class="sonar-btn" secondary @click="addField()">添加字段</n-button>
          </div>
          <div v-if="errors.fields" class="field-error mb-2">{{ errors.fields }}</div>
          <div v-for="(f, idx) in form.fields" :key="idx" class="mb-3 rounded-lg border border-line p-3">
            <div class="grid grid-cols-1 gap-2 phone:grid-cols-1 md:grid-cols-2">
              <div>
                <n-input v-model:value="f.name" placeholder="字段名" />
                <div v-if="fieldErrors[idx]?.name" class="field-error">{{ fieldErrors[idx].name }}</div>
              </div>
              <n-select v-model:value="f.kind" :options="kindOptions" />
              <n-input v-model:value="f.expr" class="md:col-span-2" placeholder="选择器 / 正则" />
              <div v-if="fieldErrors[idx]?.expr" class="field-error md:col-span-2">{{ fieldErrors[idx].expr }}</div>
              <n-input v-model:value="f.attr" placeholder="attr，默认 text" />
              <n-button size="tiny" type="error" ghost class="sonar-btn-danger" @click="removeField(idx)">
                移除字段
              </n-button>
            </div>
          </div>
        </div>

        <div class="sonar-card p-4">
          <div class="font-display mb-2">候选选择器</div>
          <div v-if="selected" class="mb-2 font-mono text-xs text-muted">
            #{{ selected.node_id }} {{ selected.tag }} · {{ selected.text }}
          </div>
          <n-input v-model:value="fieldNameDraft" class="mb-3" placeholder="写入字段时的名称" />
          <div v-if="listHint" class="mb-3 rounded-md border border-line p-2 text-xs">
            列表泛化：{{ listHint.item_selector }} → {{ listHint.field_selector }}（命中 {{ listHint.hit_count }}）
            <n-button size="tiny" class="sonar-btn ml-2" type="primary" @click="applyListHint">应用列表规则</n-button>
          </div>
          <div v-if="!candidates.length" class="text-sm text-muted">点击左侧热区后，候选会填到这里。</div>
          <div v-for="(c, i) in candidates" :key="i" class="mb-2 rounded-md border border-line p-2">
            <div class="font-mono text-xs text-cyan">{{ c.kind }} · score {{ c.score.toFixed(2) }} · {{ c.unique ? 'unique' : 'shared' }}</div>
            <div class="font-mono mt-1 break-all text-xs">{{ c.expr }}</div>
            <div class="mt-2 flex flex-wrap gap-2">
              <n-button size="tiny" class="sonar-btn" @click="applyCandidate(c, 'item')">填入列表</n-button>
              <n-button size="tiny" class="sonar-btn" secondary @click="applyCandidate(c, 'link')">填入链接</n-button>
              <n-button size="tiny" class="sonar-btn" type="primary" @click="applyCandidate(c, 'field')">添加为字段</n-button>
            </div>
          </div>
        </div>

        <div v-if="previewText" class="sonar-card p-4">
          <div class="font-display mb-2">试运行结果</div>
          <pre class="font-mono whitespace-pre-wrap text-xs">{{ previewText }}</pre>
        </div>
      </section>
    </div>
  </div>
</template>
