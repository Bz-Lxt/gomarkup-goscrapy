export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
}

export interface Page<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export type ExtractorKind = 'xpath' | 'css' | 'regex'
export type TaskStatus = 'created' | 'running' | 'paused' | 'succeeded' | 'failed' | 'cancelled'

export interface RuleField {
  name: string
  kind: ExtractorKind
  expr: string
  attr: string
}

export interface Rule {
  id: number
  name: string
  start_url: string
  item_selector: string
  link_selector: string
  fields: RuleField[]
  respect_robots: boolean
  qps: number
  version: number
  created_at: string
  updated_at: string
}

export interface RulePayload {
  name: string
  start_url: string
  item_selector: string
  link_selector: string
  fields: RuleField[]
  respect_robots: boolean
  qps: number
}

export interface Task {
  id: number
  name: string
  rule_id: number
  rule_name?: string
  seed_urls: string[]
  max_depth: number
  concurrency: number
  status: TaskStatus
  pages_crawled?: number
  pages_failed?: number
  created_at: string
  updated_at: string
  started_at?: string
  finished_at?: string
}

export interface TaskPayload {
  name: string
  rule_id: number
  seed_urls: string[]
  max_depth: number
  concurrency: number
}

export interface ResultItem {
  id: number
  task_id: number
  url: string
  payload: unknown
  created_at: string
}

export interface ClusterNode {
  id: string
  role?: string
  status?: string
  cpu: number
  memory_mb: number
  memory_total_mb?: number
  pages_per_min: number
  fail_rate: number
}

export interface MetricPoint {
  ts: string
  pages_per_min: number
  fail_rate: number
}

export interface QueueStats {
  pending: number
  leased: number
  delayed: number
  dead: number
}

export interface ProxyItem {
  id: string
  address: string
  status: string
  hits: number
  evictions: number
  latency_ms: number
  last_check: string
}

export interface BoxModel {
  x: number
  y: number
  w: number
  h: number
}

export interface SnapshotNode {
  node_id: number
  tag: string
  text: string
  box: BoxModel
}

export interface Snapshot {
  snapshot_id: string
  width: number
  height: number
  image_url: string
  nodes: SnapshotNode[]
}

export interface SelectorCandidate {
  kind: ExtractorKind
  expr: string
  unique: boolean
  score: number
}

export interface ListRuleHint {
  item_selector: string
  field_selector: string
  hit_count: number
}

export interface SelectorResult {
  candidates: SelectorCandidate[]
  list_rule?: ListRuleHint
}

export interface LoginData {
  token: string
  expires_in: number
  username: string
}

export interface WsMetricsFrame {
  type: string
  ts: string
  nodes: ClusterNode[]
}
