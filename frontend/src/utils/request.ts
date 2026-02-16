import axios from 'axios'
import type { AxiosRequestConfig } from 'axios'
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

let isLoggingOut = false

instance.interceptors.response.use(
  (response) => {
    // Skip JSON code check for blob responses (e.g. CSV export)
    if (response.config.responseType === 'blob') {
      return response
    }
    const data = response.data
    if (data.code !== 0) {
      return Promise.reject(new Error(data.message || '请求失败'))
    }
    return data
  },
  (error) => {
    const url = error.config?.url || ''
    const isAuthEndpoint = url.startsWith('/auth/')
    if (error.response?.status === 401 && !isLoggingOut && !isAuthEndpoint) {
      isLoggingOut = true
      const userStore = useUserStore()
      userStore.logout(true)
      router.push('/login').finally(() => {
        isLoggingOut = false
      })
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
    // Extract server error message if available
    const serverMsg = error.response?.data?.message
    if (serverMsg) {
      return Promise.reject(new Error(serverMsg))
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
