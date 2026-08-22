const BEIJING_OFFSET_MS = 8 * 60 * 60 * 1000
const DISPLAY_RE = /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/

function pad(n: number): string {
  return String(n).padStart(2, '0')
}

function formatParts(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

/** 展示层统一 yyyy-MM-dd HH:mm:ss（GMT+8） */
export function formatDateTime(value: unknown): string {
  if (value == null || value === '') return '—'
  if (typeof value === 'string' && DISPLAY_RE.test(value.trim())) return value.trim()

  if (typeof value === 'string' || typeof value === 'number' || value instanceof Date) {
    const raw = value instanceof Date ? value : new Date(value)
    if (Number.isNaN(raw.getTime())) return String(value)
    if (typeof value === 'string' && /Z$|[+-]\d{2}:\d{2}$/.test(value)) {
      const bj = new Date(raw.getTime() + BEIJING_OFFSET_MS)
      return `${bj.getUTCFullYear()}-${pad(bj.getUTCMonth() + 1)}-${pad(bj.getUTCDate())} ${pad(bj.getUTCHours())}:${pad(bj.getUTCMinutes())}:${pad(bj.getUTCSeconds())}`
    }
    return formatParts(raw)
  }
  return String(value)
}

export function formatPct(rate: number | undefined | null, digits = 1): string {
  const n = Number(rate ?? 0)
  if (Number.isNaN(n)) return '0.0%'
  const pct = n <= 1 ? n * 100 : n
  return `${pct.toFixed(digits)}%`
}

export function formatNumber(n: number | undefined | null, digits = 1): string {
  const v = Number(n ?? 0)
  if (Number.isNaN(v)) return '0'
  return v.toFixed(digits)
}

export function prettyJson(value: unknown): string {
  if (value == null) return '—'
  if (typeof value === 'string') {
    try {
      return JSON.stringify(JSON.parse(value), null, 2)
    } catch {
      return value
    }
  }
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

export function failTone(rate: number | undefined | null): 'ok' | 'warn' | 'bad' {
  const n = Number(rate ?? 0)
  const pct = n <= 1 ? n * 100 : n
  if (pct > 10) return 'bad'
  if (pct > 5) return 'warn'
  return 'ok'
}
