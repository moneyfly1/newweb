import { ref } from 'vue'

const cache = new Map<string, { data: any; timestamp: number }>()
const CACHE_DURATION = 5 * 60 * 1000 // 5分钟

export function useApiCache() {
  const get = (key: string) => {
    const cached = cache.get(key)
    if (cached && Date.now() - cached.timestamp < CACHE_DURATION) {
      return cached.data
    }
    return null
  }

  const set = (key: string, data: any) => {
    cache.set(key, { data, timestamp: Date.now() })
  }

  const clear = (key?: string) => {
    if (key) {
      cache.delete(key)
    } else {
      cache.clear()
    }
  }

  return { get, set, clear }
}

export function debounce<T extends (...args: any[]) => any>(fn: T, delay = 300) {
  let timer: ReturnType<typeof setTimeout>
  return function (this: any, ...args: Parameters<T>) {
    clearTimeout(timer)
    timer = setTimeout(() => fn.apply(this, args), delay)
  }
}
