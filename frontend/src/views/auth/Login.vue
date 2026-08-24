<template>
  <AuthLayout title="高速稳定 · 全球节点" :subtitle="brandFeatures">
    <h2 class="auth-form-title">欢迎回来</h2>
    <p class="auth-form-subtitle">登录你的账户继续使用</p>

    <n-form ref="formRef" :model="form" :rules="rules" label-placement="top" size="large">
      <n-form-item path="email" label="邮箱">
        <n-input v-model:value="form.email" placeholder="请输入邮箱" :input-props="{ autocomplete: 'email' }">
          <template #prefix><n-icon :component="MailOutline" /></template>
        </n-input>
      </n-form-item>
      <n-form-item path="password" label="密码">
        <n-input v-model:value="form.password" type="password" show-password-on="click" placeholder="请输入密码" :input-props="{ autocomplete: 'current-password' }" @keyup.enter="handleLogin">
          <template #prefix><n-icon :component="LockClosedOutline" /></template>
        </n-input>
      </n-form-item>

      <div class="auth-form-extra">
        <n-checkbox v-model:checked="rememberMe">记住我</n-checkbox>
        <router-link to="/forgot-password" class="auth-link">忘记密码？</router-link>
      </div>

      <n-button type="primary" block size="large" :loading="loading" class="auth-submit-btn" @click="handleLogin">
        登 录
      </n-button>
    </n-form>

    <div class="auth-form-footer">
      还没有账户？<router-link to="/register" class="auth-link">立即注册</router-link>
    </div>

    <!-- Telegram Login -->
    <div v-if="telegramEnabled" class="telegram-login-section">
      <n-divider>Telegram 登录</n-divider>
      <div ref="telegramWidgetRef" class="telegram-widget-container"></div>
    </div>
  </AuthLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage, type FormInst } from 'naive-ui'
import { MailOutline, LockClosedOutline } from '@vicons/ionicons5'
import AuthLayout from '@/components/AuthLayout.vue'
import { useUserStore } from '@/stores/user'
import { getPublicConfig } from '@/api/common'
import { getErrorMessage, silentCatch } from '@/utils/error'

const router = useRouter()
const route = useRoute()

// 登录成功后跳回原始目标（若带 redirect 参数）
const resolvePostLoginPath = () => {
  const r = route.query.redirect
  if (typeof r === 'string' && r.startsWith('/') && !r.startsWith('//')) {
    try { return decodeURIComponent(r) } catch { return r }
  }
  return userStore.isAdmin ? '/admin' : '/'
}

const message = useMessage()
const userStore = useUserStore()
const formRef = ref<FormInst | null>(null)
const loading = ref(false)
const rememberMe = ref(false)
const telegramEnabled = ref(false)
const telegramBotUsername = ref('')
const telegramWidgetRef = ref<HTMLElement | null>(null)

const brandFeatures = ['多格式订阅聚合，一键导入', '智能设备管理，安全可控', '实时节点监控，稳定高速']

const form = ref({ email: '', password: '' })
// 校验错误由 n-form-item 内联 feedback 展示（非仅 toast）
const rules = {
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email' as const, message: '邮箱格式不正确', trigger: 'blur' },
  ],
  password: { required: true, message: '请输入密码', trigger: 'blur' },
}

async function handleLogin() {
  await formRef.value?.validate()
  loading.value = true
  try {
    await userStore.login(form.value.email, form.value.password)
    applyRememberMe()
    message.success('登录成功')
    // 成功过渡：loading 保持 300ms，短暂停留后跳转
    await new Promise((resolve) => setTimeout(resolve, 300))
    router.push(resolvePostLoginPath())
  } catch (e: any) {
    message.error(getErrorMessage(e, '登录失败'))
  } finally {
    loading.value = false
  }
}

/**
 * 「记住我」落地：
 * stores/user.ts 的 login() 写死 localStorage（任务约束：不改 store），因此：
 * - 勾选「记住我」：保持 store 默认行为，token 持久化到 localStorage（关闭浏览器后仍保持登录）；
 * - 未勾选：登录成功后把 token 从 localStorage 挪到 sessionStorage（关闭标签页/浏览器即失效）。
 *
 * 已知边界（store 写死 localStorage，此处不越权修改 store）：
 * 1. 页面刷新时 store 仅从 localStorage 初始化，sessionStorage 中的 token 不会恢复 ——
 *    正好符合"不记住"的会话级语义；
 * 2. 401 自动刷新拦截器（utils/request.ts）刷新 token 时会写回 localStorage，
 *    属 store/拦截器固定行为，本页无法拦截；
 * 3. logout() 只清理 localStorage，sessionStorage 随标签页关闭自动清理。
 */
function applyRememberMe() {
  if (rememberMe.value) return
  try {
    const token = userStore.token
    const refreshToken = userStore.refreshTokenVal
    if (token) sessionStorage.setItem('token', token)
    if (refreshToken) sessionStorage.setItem('refresh_token', refreshToken)
    localStorage.removeItem('token')
    localStorage.removeItem('refresh_token')
  } catch {
    // storage 不可用（如隐私模式）时忽略，保持默认行为
  }
}

function loadTelegramWidget() {
  if (!telegramWidgetRef.value || !telegramBotUsername.value) return
  // Define global callback with nonce to prevent spoofing
  const callbackNonce = Math.random().toString(36).substring(2)
  ;(window as any).__telegramAuthNonce = callbackNonce
  ;(window as any).onTelegramAuth = async (user: any) => {
    // 验证调用来源的 nonce
    if ((window as any).__telegramAuthNonce !== callbackNonce) return
    delete (window as any).__telegramAuthNonce
    try {
      await userStore.loginWithTelegram(user)
      message.success('登录成功')
      router.push(resolvePostLoginPath())
    } catch (e: any) {
      message.error(getErrorMessage(e, 'Telegram 登录失败'))
    }
  }
  const script = document.createElement('script')
  script.src = 'https://telegram.org/js/telegram-widget.js?22'
  script.setAttribute('data-telegram-login', telegramBotUsername.value)
  script.setAttribute('data-size', 'large')
  script.setAttribute('data-radius', '8')
  script.setAttribute('data-onauth', 'onTelegramAuth(user)')
  script.setAttribute('data-request-access', 'write')
  script.async = true
  while (telegramWidgetRef.value.firstChild) {
    telegramWidgetRef.value.removeChild(telegramWidgetRef.value.firstChild)
  }
  telegramWidgetRef.value.appendChild(script)
}

onMounted(async () => {
  try {
    const res: any = await getPublicConfig()
    if (res.data) {
      const enabled = res.data.telegram_login_enabled
      telegramEnabled.value = enabled === 'true' || enabled === '1'
      telegramBotUsername.value = res.data.telegram_bot_username || ''
      if (telegramEnabled.value && telegramBotUsername.value) {
        setTimeout(loadTelegramWidget, 100)
      }
    }
  } catch (e) {
    silentCatch(e, 'loadTelegramConfig')
  }
})
</script>

<style scoped>
.telegram-login-section { margin-top: 16px; }
.telegram-widget-container { display: flex; justify-content: center; min-height: 40px; }
</style>
