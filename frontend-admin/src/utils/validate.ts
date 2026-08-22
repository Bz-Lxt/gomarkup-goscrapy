export type FieldError = string | undefined
export type ErrorMap = Record<string, FieldError>

export function required(value: unknown, label: string): FieldError {
  if (value == null) return `${label}不能为空`
  if (typeof value === 'string' && !value.trim()) return `${label}不能为空`
  if (Array.isArray(value) && value.length === 0) return `${label}不能为空`
  return undefined
}

export function isHttpUrl(value: string, label: string): FieldError {
  const empty = required(value, label)
  if (empty) return empty
  try {
    const u = new URL(value.trim())
    if (u.protocol !== 'http:' && u.protocol !== 'https:') {
      return `${label}必须是 http(s) 地址`
    }
  } catch {
    return `${label}格式无效，示例 https://example.com/list`
  }
  return undefined
}

export function inRange(value: number, min: number, max: number, label: string): FieldError {
  if (Number.isNaN(value) || value < min || value > max) {
    return `${label}须在 ${min}–${max} 之间`
  }
  return undefined
}

export function maxLen(value: string, max: number, label: string): FieldError {
  if (value.length > max) return `${label}最多 ${max} 个字符`
  return undefined
}

export function firstError(map: ErrorMap): string | undefined {
  return Object.values(map).find((v) => !!v)
}

export function hasErrors(map: ErrorMap): boolean {
  return Object.values(map).some((v) => !!v)
}
