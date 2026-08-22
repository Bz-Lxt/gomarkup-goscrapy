import axios, { type AxiosError } from 'axios'
import type { ApiEnvelope } from './types'
import { TOKEN_KEY } from '@/constants'
import { toastError } from '@/utils/feedback'
import { logger } from '@/utils/logger'

export { TOKEN_KEY }

const baseURL = import.meta.env.VITE_API_BASE || '/api'

export const http = axios.create({
  baseURL,
  timeout: 60000,
})

export function resolveApiPath(path: string): string {
  if (/^https?:\/\//.test(path)) return path
  const base = (import.meta.env.VITE_API_BASE || '/api').replace(/\/$/, '')
  if (path.startsWith(base + '/')) return path.slice(base.length)
  if (path.startsWith('/api/')) return path.slice(4)
  return path
}

function isEnvelope(value: unknown): value is ApiEnvelope<unknown> {
  return !!value && typeof value === 'object' && 'code' in value && 'message' in value
}

http.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) {
    config.headers = config.headers ?? {}
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (res) => {
    if (res.config.responseType === 'blob') return res
    const body = res.data
    if (isEnvelope(body) && body.code !== 0) {
      const err = new Error(body.message || '请求失败')
      toastError(err.message)
      return Promise.reject(err)
    }
    return res
  },
  async (error: AxiosError<ApiEnvelope<unknown>>) => {
    const status = error.response?.status
    const msg = error.response?.data?.message || error.message || '网络异常'
    if (status === 401) {
      const { useAuthStore } = await import('@/stores/auth')
      const { default: router } = await import('@/router')
      useAuthStore().clearSession()
      if (router.currentRoute.value.name !== 'login') {
        void router.push({
          name: 'login',
          query: { redirect: router.currentRoute.value.fullPath },
        })
      }
    }
    logger.warn('http error', status, msg)
    toastError(msg)
    return Promise.reject(error)
  },
)

export function unwrap<T>(data: unknown): T {
  if (isEnvelope(data)) return data.data as T
  return data as T
}

export function unwrapPage<T>(data: unknown, fallbackKeys: string[] = []): {
  items: T[]
  total: number
  page: number
  page_size: number
} {
  const payload = unwrap<unknown>(data)
  if (Array.isArray(payload)) {
    return { items: payload as T[], total: payload.length, page: 1, page_size: payload.length }
  }
  if (!payload || typeof payload !== 'object') {
    return { items: [], total: 0, page: 1, page_size: 20 }
  }
  const rec = payload as Record<string, unknown>
  let items: unknown = rec.items ?? rec.list ?? rec.records
  if (!Array.isArray(items)) {
    for (const key of fallbackKeys) {
      if (Array.isArray(rec[key])) {
        items = rec[key]
        break
      }
    }
  }
  const list = Array.isArray(items) ? (items as T[]) : []
  return {
    items: list,
    total: Number(rec.total ?? rec.count ?? list.length),
    page: Number(rec.page ?? 1),
    page_size: Number(rec.page_size ?? rec.pageSize ?? 20),
  }
}

export function asArray<T>(data: unknown, keys: string[] = []): T[] {
  const payload = unwrap<unknown>(data)
  if (Array.isArray(payload)) return payload as T[]
  if (!payload || typeof payload !== 'object') return []
  const rec = payload as Record<string, unknown>
  for (const key of ['items', 'list', ...keys]) {
    if (Array.isArray(rec[key])) return rec[key] as T[]
  }
  return []
}
