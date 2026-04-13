import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi, telegramLogin as telegramLoginApi } from '@/api/auth'
import { getCurrentUser } from '@/api/user'
import request, { clearRequestSessionCache, prefetchCSRFToken } from '@/utils/request'

export interface UserInfo {
  id: number
  username: string
  email: string
  is_admin: boolean
  balance: number
  level: number
  is_active: boolean
  telegram_username?: string
}

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const refreshTokenVal = ref(localStorage.getItem('refresh_token') || '')
  const userInfo = ref<UserInfo | null>(null)

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => userInfo.value?.is_admin ?? false)

  async function login(email: string, password: string) {
    const res: any = await loginApi({ email, password })
    clearRequestSessionCache()
    token.value = res.data.access_token
    refreshTokenVal.value = res.data.refresh_token || ''
    localStorage.setItem('token', token.value)
    localStorage.setItem('refresh_token', refreshTokenVal.value)
    prefetchCSRFToken()
    // Login response includes user info, use it directly
    if (res.data.user) {
      userInfo.value = res.data.user
    } else {
      await fetchUser()
    }
  }

  let fetchingUser: Promise<void> | null = null

  async function fetchUser() {
    // Deduplicate concurrent fetchUser calls
    if (fetchingUser) return fetchingUser
    fetchingUser = (async () => {
      const res: any = await getCurrentUser()
      userInfo.value = res.data
    })()
    try {
      await fetchingUser
    } finally {
      fetchingUser = null
    }
  }

  async function loginWithTelegram(data: any) {
    const res: any = await telegramLoginApi(data)
    clearRequestSessionCache()
    token.value = res.data.access_token
    refreshTokenVal.value = res.data.refresh_token || ''
    localStorage.setItem('token', token.value)
    localStorage.setItem('refresh_token', refreshTokenVal.value)
    prefetchCSRFToken()
    if (res.data.user) {
      userInfo.value = res.data.user
    } else {
      await fetchUser()
    }
  }

  function logout(skipApi = false) {
    clearRequestSessionCache()
    const oldToken = token.value
    const oldRefreshToken = refreshTokenVal.value
    token.value = ''
    refreshTokenVal.value = ''
    userInfo.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('refresh_token')
    localStorage.removeItem('admin_token')
    localStorage.removeItem('admin_user')
    localStorage.removeItem('admin_session')
    if (!skipApi && oldToken) {
      try {
        request.post('/auth/logout', { refresh_token: oldRefreshToken || undefined }, {
          headers: { Authorization: `Bearer ${oldToken}` },
        }).catch(() => {})
      } catch {}
    }
  }

  return { token, refreshTokenVal, userInfo, isLoggedIn, isAdmin, login, loginWithTelegram, fetchUser, logout }
})
