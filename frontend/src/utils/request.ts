import axios from 'axios'
import type { AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios'
import { useUserStore } from '@/stores/user'
import router from '@/router'

export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
  total?: number
}

const instance = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
})

const requestCache = new Map<string, { data: any; timestamp: number }>()
const CACHE_DURATION = 3 * 60 * 1000 // 3分钟

// 只缓存这些列表接口
const CACHEABLE_URLS = [
  '/admin/packages',
  '/admin/coupons',
  '/admin/levels',
  '/admin/announcements',
]

function shouldCache(url: string): boolean {
  return CACHEABLE_URLS.some(cacheable => url.includes(cacheable))
}

function getCacheKey(url: string, params?: any): string {
  return `${url}?${JSON.stringify(params || {})}`
}

function getCache(key: string) {
  const cached = requestCache.get(key)
  if (cached && Date.now() - cached.timestamp < CACHE_DURATION) {
    return cached.data
  }
  requestCache.delete(key)
  return null
}

function setCache(key: string, data: any) {
  requestCache.set(key, { data, timestamp: Date.now() })
  if (requestCache.size > 100) {
    const firstKey = requestCache.keys().next().value
    if (firstKey) requestCache.delete(firstKey)
  }
}

let isRefreshing = false
let csrfTokenCache = ''
let csrfTokenPromise: Promise<string> | null = null
let pendingRequests: Array<{
  resolve: (config: InternalAxiosRequestConfig) => void
  reject: (error: any) => void
}> = []

function processQueue(error: any, newToken: string | null) {
  pendingRequests.forEach(({ resolve, reject }) => {
    if (error) {
      reject(error)
    } else if (newToken) {
      resolve({} as InternalAxiosRequestConfig) // placeholder, actual retry below
    }
  })
  pendingRequests = []
}

export function clearRequestSessionCache() {
  csrfTokenCache = ''
  csrfTokenPromise = null
  requestCache.clear()
}

async function ensureCSRFToken(): Promise<string> {
  const userStore = useUserStore()
  if (!userStore.token) {
    return ''
  }
  if (csrfTokenCache) {
    return csrfTokenCache
  }
  if (csrfTokenPromise) {
    return csrfTokenPromise
  }

  csrfTokenPromise = instance.get('/csrf-token')
    .then((res: any) => {
      const token = res?.data?.csrf_token || ''
      csrfTokenCache = token
      return token
    })
    .catch(() => '')
    .finally(() => {
      csrfTokenPromise = null
    })

  return csrfTokenPromise
}

let isLoggingOut = false

instance.interceptors.request.use(async (config) => {
  const userStore = useUserStore()
  if (userStore.token) {
    config.headers.Authorization = `Bearer ${userStore.token}`
  }

  const method = (config.method || 'get').toUpperCase()
  const url = config.url || ''
  const isAuthEndpoint = url.startsWith('/auth/')
  const needsCSRF = ['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)
  if (needsCSRF && userStore.token && !isAuthEndpoint) {
    const csrfToken = await ensureCSRFToken()
    if (csrfToken) {
      config.headers = config.headers || {}
      ;(config.headers as any)['X-CSRF-Token'] = csrfToken
    }
  }

  return config
})

instance.interceptors.response.use(
  (response) => {
    if (response.config.responseType === 'blob') {
      return response
    }
    const data = response.data
    if (data.code !== 0) {
      return Promise.reject(new Error(data.message || '请求失败'))
    }
    return data
  },
  async (error) => {
    const originalRequest = error.config
    const url = originalRequest?.url || ''
    const isAuthEndpoint = url.startsWith('/auth/')

    // Attempt token refresh on 401 (skip for auth endpoints and retried requests)
    if (error.response?.status === 401 && !isAuthEndpoint && !originalRequest._retry) {
      const userStore = useUserStore()
      const storedRefresh = userStore.refreshTokenVal

      if (storedRefresh && !isRefreshing) {
        isRefreshing = true
        originalRequest._retry = true

        try {
          const res = await axios.post('/api/v1/auth/refresh', { refresh_token: storedRefresh })
          const newToken = res.data?.data?.access_token
          const newRefresh = res.data?.data?.refresh_token
          if (newToken) {
            userStore.token = newToken
            if (newRefresh) userStore.refreshTokenVal = newRefresh
            csrfTokenCache = ''
            localStorage.setItem('token', newToken)
            if (newRefresh) localStorage.setItem('refresh_token', newRefresh)
            originalRequest.headers.Authorization = `Bearer ${newToken}`
            isRefreshing = false
            return instance(originalRequest)
          }
        } catch (refreshError) {
          isRefreshing = false
        }
      }

      // Refresh failed — logout
      csrfTokenCache = ''
      userStore.logout(true)
      router.push('/login')
      return Promise.reject(new Error('登录已过期，请重新登录'))
    }

    // CSRF token may be expired; refresh once and retry.
    if (error.response?.status === 403 && originalRequest && !originalRequest._csrfRetry) {
      originalRequest._csrfRetry = true
      csrfTokenCache = ''
      const csrfToken = await ensureCSRFToken()
      if (csrfToken) {
        originalRequest.headers = originalRequest.headers || {}
        originalRequest.headers['X-CSRF-Token'] = csrfToken
        return instance(originalRequest)
      }
    }

    // For blob responses, try to parse the error body as JSON
    if (error.response?.data instanceof Blob) {
      return error.response.data.text().then((text: string) => {
        try {
          const json = JSON.parse(text)
          return Promise.reject(new Error(json.message || '请求失败'))
        } catch {
          return Promise.reject(new Error('请求失败'))
        }
      })
    }
    const serverMsg = error.response?.data?.message
    if (serverMsg) {
      return Promise.reject(new Error(serverMsg))
    }
    if (!error.response) {
      return Promise.reject(new Error('网络连接失败，请检查网络'))
    }
    return Promise.reject(error)
  }
)

const request = {
  get<T = any>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    // 只对特定列表接口使用缓存
    if (shouldCache(url)) {
      const cacheKey = getCacheKey(url, config?.params)
      const cached = getCache(cacheKey)
      if (cached) {
        return Promise.resolve(cached)
      }
      return instance.get(url, config).then((res: any) => {
        setCache(cacheKey, res)
        return res
      }) as any
    }
    // 其他接口不使用缓存
    return instance.get(url, config) as any
  },
  post<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return instance.post(url, data, config) as any
  },
  put<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return instance.put(url, data, config) as any
  },
  delete<T = any>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return instance.delete(url, config) as any
  },
}

export default request
