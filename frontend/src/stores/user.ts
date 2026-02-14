import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi } from '@/api/auth'
import { getCurrentUser } from '@/api/user'

export interface UserInfo {
  id: number
  username: string
  email: string
  is_admin: boolean
  balance: number
  level: number
  is_active: boolean
}

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const refreshTokenVal = ref(localStorage.getItem('refresh_token') || '')
  const userInfo = ref<UserInfo | null>(null)

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => userInfo.value?.is_admin ?? false)

  async function login(email: string, password: string) {
    const res: any = await loginApi({ email, password })
    token.value = res.data.access_token
    refreshTokenVal.value = res.data.refresh_token || ''
    localStorage.setItem('token', token.value)
    localStorage.setItem('refresh_token', refreshTokenVal.value)
    // Login response includes user info, use it directly
    if (res.data.user) {
      userInfo.value = res.data.user
    } else {
      await fetchUser()
    }
  }

  async function fetchUser() {
    const res: any = await getCurrentUser()
    userInfo.value = res.data
  }

  function logout(skipApi = false) {
    const oldToken = token.value
    token.value = ''
    refreshTokenVal.value = ''
    userInfo.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('refresh_token')
    if (!skipApi && oldToken) {
      // Fire-and-forget with the old token
      try {
        import('@/utils/request').then(({ default: req }) => {
          req.post('/auth/logout', null, {
            headers: { Authorization: `Bearer ${oldToken}` }
          }).catch(() => {})
        })
      } catch {}
    }
  }

  return { token, refreshTokenVal, userInfo, isLoggedIn, isAdmin, login, fetchUser, logout }
})
