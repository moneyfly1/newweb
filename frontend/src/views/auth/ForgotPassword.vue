<template>
  <AuthLayout title="高速稳定 · 全球节点">
    <template v-if="step === 1">
      <h2 class="auth-form-title">忘记密码</h2>
      <p class="auth-form-subtitle">输入你的邮箱地址，我们将发送验证码</p>
      <n-form ref="emailFormRef" :model="form" :rules="emailRules" label-placement="top" size="large">
        <n-form-item path="email" label="邮箱">
          <n-input v-model:value="form.email" placeholder="请输入邮箱" @keyup.enter="sendCode">
            <template #prefix><n-icon :component="MailOutline" /></template>
          </n-input>
        </n-form-item>
        <n-button type="primary" block size="large" :loading="sending" class="auth-submit-btn" @click="sendCode">
          发送验证码
        </n-button>
      </n-form>
    </template>
    <template v-else-if="step === 2">
      <h2 class="auth-form-title">重置密码</h2>
      <p class="auth-form-subtitle">验证码已发送至 {{ form.email }}</p>
      <n-form ref="resetFormRef" :model="form" :rules="resetRules" label-placement="top" size="large">
        <n-form-item path="code" label="验证码">
          <n-input v-model:value="form.code" placeholder="请输入验证码" maxlength="6" @keyup.enter="doReset">
            <template #prefix><n-icon :component="KeyOutline" /></template>
          </n-input>
        </n-form-item>
        <n-form-item path="password" label="新密码">
          <n-input v-model:value="form.password" type="password" show-password-on="click" placeholder="新密码（至少6位）">
            <template #prefix><n-icon :component="LockClosedOutline" /></template>
          </n-input>
        </n-form-item>
        <n-form-item path="confirmPassword" label="确认新密码">
          <n-input v-model:value="form.confirmPassword" type="password" show-password-on="click" placeholder="请再次输入新密码" @keyup.enter="doReset">
            <template #prefix><n-icon :component="LockClosedOutline" /></template>
          </n-input>
        </n-form-item>
        <n-button type="primary" block size="large" :loading="resetting" class="auth-submit-btn" @click="doReset">
          重置密码
        </n-button>
        <n-button text type="primary" size="small" style="margin-top: 12px;" @click="sendCode" :loading="sending" :disabled="countdown > 0">
          {{ countdown > 0 ? `${countdown}s 后重新发送` : '重新发送验证码' }}
        </n-button>
      </n-form>
    </template>
    <template v-else-if="step === 3">
      <h2 class="auth-form-title">密码已重置</h2>
      <p class="auth-form-subtitle">你的密码已成功重置，请使用新密码登录</p>
      <n-button type="primary" block size="large" class="auth-submit-btn" @click="goLogin">
        立即返回登录
      </n-button>
      <p class="auth-redirect-hint">{{ redirectCountdown }} 秒后自动跳转登录页…</p>
    </template>
    <div class="auth-form-footer">
      <router-link to="/login" class="auth-link">返回登录</router-link>
    </div>
  </AuthLayout>
</template>

<script setup lang="ts">
import { ref, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, type FormInst } from 'naive-ui'
import { MailOutline, LockClosedOutline, KeyOutline } from '@vicons/ionicons5'
import AuthLayout from '@/components/AuthLayout.vue'
import { forgotPassword, resetPassword } from '@/api/auth'
import { getErrorMessage } from '@/utils/error'

const router = useRouter()
const message = useMessage()
const emailFormRef = ref<FormInst | null>(null)
const resetFormRef = ref<FormInst | null>(null)
const step = ref(1)
const sending = ref(false)
const resetting = ref(false)
const countdown = ref(0)
// step3 成功后自动跳转登录页的倒计时
const redirectCountdown = ref(0)
let timer: ReturnType<typeof setInterval> | null = null
let redirectTimer: ReturnType<typeof setInterval> | null = null

const form = ref({ email: '', code: '', password: '', confirmPassword: '' })

// 修复：邮箱规则增加格式校验（type: 'email'）
const emailRules = {
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email' as const, message: '邮箱格式不正确', trigger: 'blur' },
  ],
}
const resetRules = {
  code: { required: true, message: '请输入验证码', trigger: 'blur' },
  password: { required: true, message: '请输入新密码', trigger: 'blur', min: 6 },
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: (_r: any, v: string) => v === form.value.password, message: '两次密码不一致', trigger: 'blur' },
  ],
}

function startCountdown() {
  countdown.value = 60
  timer = setInterval(() => {
    countdown.value--
    if (countdown.value <= 0 && timer) { clearInterval(timer); timer = null }
  }, 1000)
}

async function sendCode() {
  if (step.value === 1) await emailFormRef.value?.validate()
  sending.value = true
  try {
    await forgotPassword({ email: form.value.email })
    message.success('验证码已发送')
    step.value = 2
    startCountdown()
  } catch (e: any) {
    message.error(getErrorMessage(e, '发送失败'))
  } finally {
    sending.value = false
  }
}

async function doReset() {
  await resetFormRef.value?.validate()
  resetting.value = true
  try {
    await resetPassword({ email: form.value.email, code: form.value.code, password: form.value.password })
    message.success('密码重置成功')
    step.value = 3
    // 成功提示后 3 秒倒计时，自动跳转登录页
    startRedirectCountdown()
  } catch (e: any) {
    message.error(getErrorMessage(e, '重置失败'))
  } finally {
    resetting.value = false
  }
}

function startRedirectCountdown() {
  redirectCountdown.value = 3
  redirectTimer = setInterval(() => {
    redirectCountdown.value--
    if (redirectCountdown.value <= 0 && redirectTimer) {
      clearInterval(redirectTimer)
      redirectTimer = null
      router.push('/login')
    }
  }, 1000)
}

function goLogin() {
  if (redirectTimer) { clearInterval(redirectTimer); redirectTimer = null }
  router.push('/login')
}

onUnmounted(() => {
  if (timer) clearInterval(timer)
  if (redirectTimer) clearInterval(redirectTimer)
})
</script>
