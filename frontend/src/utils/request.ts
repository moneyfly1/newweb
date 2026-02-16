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

instance.interceptors.request.use((config) => {
  const userStore = useUserStore()
  if (userStore.token) {
    config.headers.Authorization = `Bearer ${userStore.token}`
  }
  return config
})

let isRefreshing = false
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

let isLoggingOut = false

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
        if (isRefreshing) {
          // Queue this request until refresh completes
          return new Promise((resolve, reject) => {
            pendingRequests.push({ resolve, reject })
          }).then(() => {
            originalRequest.headers.Authorization = `Bearer ${userStore.token}`
            return instance(originalRequest)
          })
        }

        isRefreshing = true
        originalRequest._retry = true

        try {
          const res = await axios.post('/api/v1/auth/refresh', { refresh_token: storedRefresh })
          const newToken = res.data?.data?.access_token
          const newRefresh = res.data?.data?.refresh_token
          if (newToken) {
            userStore.token = newToken
            if (newRefresh) userStore.refreshTokenVal = newRefresh
            localStorage.setItem('token', newToken)
            if (newRefresh) localStorage.setItem('refresh_token', newRefresh)
            processQueue(null, newToken)
            originalRequest.headers.Authorization = `Bearer ${newToken}`
            return instance(originalRequest)
          }
        } catch {
          processQueue(error, null)
        } finally {
          isRefreshing = false
        }
      }

      // Refresh failed or no refresh token — logout
      if (!isLoggingOut) {
        isLoggingOut = true
        userStore.logout(true)
        router.push('/login').finally(() => { isLoggingOut = false })
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
