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

// 重试配置
const MAX_RETRIES = 2
const RETRY_DELAY = 1000

instance.interceptors.request.use((config: any) => {
  config.retryCount = config.retryCount || 0
  return config
})

const requestCache = new Map<string, { data: any; timestamp: number }>()
const CACHE_DURATION = 10 * 1000 // 10秒：防重复请求尖峰，又保证列表操作后基本实时可见

// 只缓存这些列表接口（精确路径匹配，避免 /packages 误配 /admin/packages、/settings 误配 /notification-settings 等）
const CACHEABLE_URLS = [
  '/admin/packages',
  '/admin/coupons',
  '/admin/user-levels',
  '/admin/announcements',
  '/config',
  '/packages',
  '/announcements',
  '/nodes',
  '/settings',
]

function shouldCache(url: string): boolean {
  const path = url.split('?')[0]
  return CACHEABLE_URLS.some(cacheable => path === cacheable)
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

function clearResponseCache() {
  requestCache.clear()
}

let isRefreshing = false
let csrfTokenCache = ''
let csrfTokenPromise: Promise<string> | null = null
let pendingRequests: Array<{
  resolve: (newToken: string) => void
  reject: (error: any) => void
}> = []

// 刷新完成后统一放行挂起的请求（成功传新 token，失败传 error）
// 边界兜底：error 与 newToken 都为空时视为失败，避免队列永久挂起
function processQueue(error: any, newToken: string | null) {
  const finalError = error || new Error('登录已过期，请重新登录')
  pendingRequests.forEach(({ resolve, reject }) => {
    if (newToken) {
      resolve(newToken)
    } else {
      reject(finalError)
    }
  })
  pendingRequests = []
}

export function clearRequestSessionCache() {
  csrfTokenCache = ''
  csrfTokenPromise = null
  requestCache.clear()
}

// 预取 CSRF token，登录后调用以消除首次 mutation 请求的阻塞
export function prefetchCSRFToken() {
  ensureCSRFToken()
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

      if (storedRefresh) {
        // 已有刷新在进行中：挂入队列，刷新完成后用新 token 统一重放，
        // 避免并发 401 直接走登出（误登出 bug）
        if (isRefreshing) {
          return new Promise<string>((resolve, reject) => {
            pendingRequests.push({ resolve, reject })
          }).then((newToken) => {
            originalRequest.headers = originalRequest.headers || {}
            originalRequest.headers.Authorization = `Bearer ${newToken}`
            return instance(originalRequest)
          }).catch((e) => Promise.reject(e))
        }

        isRefreshing = true
        originalRequest._retry = true

        try {
          // 裸 axios 加 10s 超时，防止刷新请求挂起导致 isRefreshing 永久为 true、全部请求卡死
          const res = await axios.post('/api/v1/auth/refresh', { refresh_token: storedRefresh }, { timeout: 10000 })
          const newToken = res.data?.data?.access_token
          const newRefresh = res.data?.data?.refresh_token
          if (newToken) {
            userStore.token = newToken
            if (newRefresh) userStore.refreshTokenVal = newRefresh
            csrfTokenCache = ''
            localStorage.setItem('token', newToken)
            if (newRefresh) localStorage.setItem('refresh_token', newRefresh)
            processQueue(null, newToken)
            isRefreshing = false
            originalRequest.headers.Authorization = `Bearer ${newToken}`
            return instance(originalRequest)
          }
          processQueue(new Error('刷新失败'), null)
          isRefreshing = false
        } catch (refreshError) {
          processQueue(refreshError, null)
          isRefreshing = false
        }
      }

      // 刷新失败或没有 refresh token — 登出
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
      // 保留 response 引用（status/data），供调用方用 e?.response?.status 判断业务分支（如 404 空态）
      const err: any = new Error(serverMsg)
      err.response = error.response
      err.status = error.response?.status
      return Promise.reject(err)
    }
    // 网络错误重试：仅对幂等的 GET/HEAD 自动重试。
    // 写请求（POST/PUT/DELETE）不重试，防止超时/断网后自动重发导致重复下单/重复扣款。
    const retryMethod = (originalRequest?.method || 'get').toUpperCase()
    if (!error.response && originalRequest && ['GET', 'HEAD'].includes(retryMethod) && originalRequest.retryCount < MAX_RETRIES) {
      originalRequest.retryCount++
      await new Promise(resolve => setTimeout(resolve, RETRY_DELAY))
      return instance(originalRequest)
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
    return instance.post(url, data, config).then((res: any) => {
      clearResponseCache()
      return res
    }) as any
  },
  put<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return instance.put(url, data, config).then((res: any) => {
      clearResponseCache()
      return res
    }) as any
  },
  delete<T = any>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return instance.delete(url, config).then((res: any) => {
      clearResponseCache()
      return res
    }) as any
  },
}

export default request
