import { http, unwrap, unwrapPage, asArray, resolveApiPath } from './http'
import type {
  ClusterNode,
  LoginData,
  ProxyItem,
  QueueStats,
  ResultItem,
  Rule,
  RulePayload,
  SelectorResult,
  Snapshot,
  Task,
  TaskPayload,
} from './types'

function num(v: unknown, fallback = 0): number {
  const n = Number(v)
  return Number.isFinite(n) ? n : fallback
}

function str(v: unknown, fallback = ''): string {
  return v == null ? fallback : String(v)
}

export async function login(username: string, password: string): Promise<LoginData> {
  const { data } = await http.post('/v1/auth/login', { username, password })
  return unwrap<LoginData>(data)
}

export async function fetchHealth(): Promise<unknown> {
  const { data } = await http.get('/v1/health')
  return unwrap(data)
}

export async function listRules(page: number, pageSize: number, keyword: string) {
  const { data } = await http.get('/v1/rules', {
    params: { page, page_size: pageSize, keyword },
  })
  return unwrapPage<Rule>(data, ['rules'])
}

export async function getRule(id: number): Promise<Rule> {
  const { data } = await http.get(`/v1/rules/${id}`)
  return unwrap<Rule>(data)
}

export async function createRule(payload: RulePayload): Promise<Rule> {
  const { data } = await http.post('/v1/rules', payload)
  return unwrap<Rule>(data)
}

export async function updateRule(id: number, payload: RulePayload): Promise<Rule> {
  const { data } = await http.patch(`/v1/rules/${id}`, payload)
  return unwrap<Rule>(data)
}

export async function deleteRule(id: number): Promise<void> {
  await http.delete(`/v1/rules/${id}`)
}

export async function previewRule(id: number, body: { html: string; url: string }): Promise<unknown> {
  const { data } = await http.post(`/v1/rules/${id}/preview`, body)
  return unwrap(data)
}

export async function listTasks(page: number, pageSize: number, status: string) {
  const { data } = await http.get('/v1/tasks', {
    params: { page, page_size: pageSize, status: status || undefined },
  })
  return unwrapPage<Task>(data, ['tasks'])
}

export async function getTask(id: number): Promise<Task> {
  const { data } = await http.get(`/v1/tasks/${id}`)
  return unwrap<Task>(data)
}

export async function createTask(payload: TaskPayload): Promise<Task> {
  const { data } = await http.post('/v1/tasks', payload)
  return unwrap<Task>(data)
}

export async function startTask(id: number): Promise<void> {
  await http.post(`/v1/tasks/${id}/start`)
}

export async function pauseTask(id: number): Promise<void> {
  await http.post(`/v1/tasks/${id}/pause`)
}

export async function cancelTask(id: number): Promise<void> {
  await http.post(`/v1/tasks/${id}/cancel`)
}

export async function listTaskResults(taskId: number, page: number, pageSize: number) {
  const { data } = await http.get(`/v1/tasks/${taskId}/results`, {
    params: { page, page_size: pageSize },
  })
  return unwrapPage<ResultItem>(data, ['results'])
}

export async function listResults(page: number, pageSize: number, taskId?: number) {
  const { data } = await http.get('/v1/results', {
    params: { page, page_size: pageSize, task_id: taskId || undefined },
  })
  return unwrapPage<ResultItem>(data, ['results'])
}

export async function createSnapshot(url: string): Promise<Snapshot> {
  const { data } = await http.post('/v1/snapshots', { url }, { timeout: 90000 })
  return unwrap<Snapshot>(data)
}

export async function fetchSnapshotImage(imageUrl: string): Promise<string> {
  const { data } = await http.get(resolveApiPath(imageUrl), { responseType: 'blob' })
  return URL.createObjectURL(data)
}

export async function fetchSelectors(snapshotId: string, nodeId: number): Promise<SelectorResult> {
  const { data } = await http.post(`/v1/snapshots/${snapshotId}/selectors`, { node_id: nodeId })
  return unwrap<SelectorResult>(data)
}

function normalizeNode(raw: Record<string, unknown>): ClusterNode {
  return {
    id: str(raw.id ?? raw.node_id ?? raw.name, 'node'),
    role: str(raw.role, 'worker'),
    status: str(raw.status, 'online'),
    cpu: num(raw.cpu ?? raw.cpu_pct ?? raw.cpu_percent),
    memory_mb: num(raw.memory_mb ?? raw.mem_mb ?? raw.memory),
    memory_total_mb: raw.memory_total_mb != null ? num(raw.memory_total_mb) : undefined,
    pages_per_min: num(raw.pages_per_min ?? raw.ppm ?? raw.rate),
    fail_rate: num(raw.fail_rate ?? raw.failure_rate),
  }
}

export async function fetchClusterNodes(): Promise<ClusterNode[]> {
  const { data } = await http.get('/v1/cluster/nodes')
  return asArray<Record<string, unknown>>(data, ['nodes']).map(normalizeNode)
}

export async function fetchClusterMetrics(): Promise<ClusterNode[]> {
  const { data } = await http.get('/v1/cluster/metrics')
  const payload = unwrap<unknown>(data)
  if (payload && typeof payload === 'object' && !Array.isArray(payload)) {
    const rec = payload as Record<string, unknown>
    if (Array.isArray(rec.nodes)) return rec.nodes.map((n) => normalizeNode(n as Record<string, unknown>))
  }
  return asArray<Record<string, unknown>>(data, ['metrics', 'nodes']).map(normalizeNode)
}

export async function fetchQueueStats(): Promise<QueueStats> {
  const { data } = await http.get('/v1/queue/stats')
  const rec = (unwrap<Record<string, unknown>>(data) ?? {}) as Record<string, unknown>
  return {
    pending: num(rec.pending ?? rec.ready ?? rec.queued),
    leased: num(rec.leased ?? rec.inflight ?? rec.running),
    delayed: num(rec.delayed ?? rec.retry),
    dead: num(rec.dead ?? rec.failed),
  }
}

function normalizeProxy(raw: Record<string, unknown>, idx: number): ProxyItem {
  const host = str(raw.address ?? raw.addr ?? raw.host ?? raw.url)
  const port = raw.port != null ? `:${raw.port}` : ''
  return {
    id: str(raw.id ?? raw.address ?? idx),
    address: host.includes(':') || !port ? host : `${host}${port}`,
    status: str(raw.status ?? (raw.healthy === true ? 'healthy' : raw.healthy === false ? 'down' : 'unknown'), 'unknown'),
    hits: num(raw.hits ?? raw.hit_count ?? raw.success),
    evictions: num(raw.evictions ?? raw.eviction_count ?? raw.ejected),
    latency_ms: num(raw.latency_ms ?? raw.rtt_ms),
    last_check: str(raw.last_check ?? raw.last_check_at ?? raw.checked_at),
  }
}

export async function fetchProxies(): Promise<ProxyItem[]> {
  const { data } = await http.get('/v1/proxies')
  return asArray<Record<string, unknown>>(data, ['proxies']).map(normalizeProxy)
}
