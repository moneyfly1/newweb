<template>
  <div class="auth-page">
    <div class="auth-left">
      <div class="brand-area">
        <div class="brand-logo">C</div>
        <h1>CBoard</h1>
        <p class="brand-desc">高效、安全的代理订阅聚合管理平台</p>
      </div>
    </div>
    <div class="auth-right">
      <div class="auth-form-wrapper">
        <template v-if="step === 1">
          <h2>忘记密码</h2>
          <p class="auth-subtitle">输入你的邮箱地址，我们将发送验证码</p>
          <n-form ref="emailFormRef" :model="form" :rules="emailRules" label-placement="left" :show-label="false">
            <n-form-item path="email">
              <n-input v-model:value="form.email" placeholder="邮箱地址" size="large" @keyup.enter="sendCode">
                <template #prefix><n-icon :component="MailOutline" /></template>
              </n-input>
            </n-form-item>
            <n-button type="primary" block size="large" :loading="sending" @click="sendCode" style="border-radius: 8px; height: 44px;">
              发送验证码
            </n-button>
          </n-form>
        </template>
        <template v-if="step === 2">
          <h2>重置密码</h2>
          <p class="auth-subtitle">验证码已发送至 {{ form.email }}</p>
          <n-form ref="resetFormRef" :model="form" :rules="resetRules" label-placement="left" :show-label="false">
            <n-form-item path="code">
              <n-input v-model:value="form.code" placeholder="验证码" size="large" maxlength="6">
                <template #prefix><n-icon :component="KeyOutline" /></template>
              </n-input>
            </n-form-item>
            <n-form-item path="password">
              <n-input v-model:value="form.password" type="password" show-password-on="click" placeholder="新密码（至少6位）" size="large">
                <template #prefix><n-icon :component="LockClosedOutline" /></template>
              </n-input>
            </n-form-item>
            <n-form-item path="confirmPassword">
              <n-input v-model:value="form.confirmPassword" type="password" show-password-on="click" placeholder="确认新密码" size="large" @keyup.enter="doReset">
                <template #prefix><n-icon :component="LockClosedOutline" /></template>
              </n-input>
            </n-form-item>
            <n-button type="primary" block size="large" :loading="resetting" @click="doReset" style="border-radius: 8px; height: 44px;">
              重置密码
            </n-button>
            <n-button text type="primary" size="small" style="margin-top: 12px;" @click="sendCode" :loading="sending" :disabled="countdown > 0">
              {{ countdown > 0 ? `${countdown}s 后重新发送` : '重新发送验证码' }}
            </n-button>
          </n-form>
        </template>
        <template v-if="step === 3">
          <h2>密码已重置</h2>
          <p class="auth-subtitle">你的密码已成功重置，请使用新密码登录</p>
          <n-button type="primary" block size="large" @click="router.push('/login')" style="border-radius: 8px; height: 44px;">
            返回登录
          </n-button>
        </template>
        <div class="auth-footer">
          <router-link to="/login"><n-button text type="primary">返回登录</n-button></router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, type FormInst } from 'naive-ui'
import { MailOutline, LockClosedOutline, KeyOutline } from '@vicons/ionicons5'
import { forgotPassword, resetPassword } from '@/api/auth'

const router = useRouter()
const message = useMessage()
const emailFormRef = ref<FormInst | null>(null)
const resetFormRef = ref<FormInst | null>(null)
const step = ref(1)
const sending = ref(false)
const resetting = ref(false)
const countdown = ref(0)
let timer: ReturnType<typeof setInterval> | null = null

const form = ref({ email: '', code: '', password: '', confirmPassword: '' })

const emailRules = { email: { required: true, message: '请输入邮箱', trigger: 'blur' } }
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
    message.error(e.message || '发送失败')
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
  } catch (e: any) {
    message.error(e.message || '重置失败')
  } finally {
    resetting.value = false
  }
}

onUnmounted(() => { if (timer) clearInterval(timer) })
</script>

<style scoped>
.auth-page { height: 100vh; display: flex; }
.auth-left { flex: 1; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); display: flex; align-items: center; justify-content: center; color: #fff; padding: 48px; }
.brand-area { max-width: 400px; }
.brand-logo { width: 56px; height: 56px; background: rgba(255,255,255,0.2); border-radius: 14px; display: flex; align-items: center; justify-content: center; font-size: 28px; font-weight: bold; backdrop-filter: blur(10px); margin-bottom: 24px; }
.brand-area h1 { font-size: 36px; margin-bottom: 12px; font-weight: 700; }
.brand-desc { font-size: 16px; opacity: 0.85; line-height: 1.6; }
.auth-right { flex: 1; display: flex; align-items: center; justify-content: center; padding: 48px; }
.auth-form-wrapper { width: 100%; max-width: 400px; }
.auth-form-wrapper h2 { font-size: 28px; font-weight: 700; margin-bottom: 8px; }
.auth-subtitle { color: #999; margin-bottom: 32px; font-size: 15px; }
.auth-footer { text-align: center; margin-top: 24px; color: #999; font-size: 14px; }
@media (max-width: 768px) { .auth-left { display: none; } }
</style>
