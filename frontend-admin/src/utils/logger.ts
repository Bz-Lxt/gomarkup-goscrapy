type Level = 'debug' | 'info' | 'warn' | 'error'

const rank: Record<Level, number> = { debug: 10, info: 20, warn: 30, error: 40 }
const isProd = import.meta.env.PROD
const min = isProd ? rank.info : rank.debug

function emit(level: Level, args: unknown[]) {
  if (rank[level] < min) return
  const prefix = `[goscrapy:${level}]`
  if (level === 'error') console.error(prefix, ...args)
  else if (level === 'warn') console.warn(prefix, ...args)
  else if (level === 'info') console.info(prefix, ...args)
}

export const logger = {
  debug: (...args: unknown[]) => emit('debug', args),
  info: (...args: unknown[]) => emit('info', args),
  warn: (...args: unknown[]) => emit('warn', args),
  error: (...args: unknown[]) => emit('error', args),
}
