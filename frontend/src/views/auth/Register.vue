<template>
  <div class="auth-page">
    <div class="auth-left">
      <div class="brand-area">
        <div class="brand-logo">C</div>
        <h1>CBoard</h1>
        <p class="brand-desc">高效、安全的代理订阅聚合管理平台</p>
        <div class="feature-list">
          <div class="feature-item"><div class="feature-dot" /><span>多格式订阅聚合，一键导入</span></div>
          <div class="feature-item"><div class="feature-dot" /><span>智能设备管理，安全可控</span></div>
          <div class="feature-item"><div class="feature-dot" /><span>实时节点监控，稳定高速</span></div>
        </div>
      </div>
    </div>
    <div class="auth-right">
      <div class="auth-form-wrapper">
        <template v-if="registerDisabled">
          <h2>暂未开放注册</h2>
          <p class="auth-subtitle">管理员已关闭注册功能</p>
          <div class="auth-footer">
            已有账户？<router-link to="/login"><n-button text type="primary">立即登录</n-button></router-link>
          </div>
        </template>
        <template v-else>
          <h2>创建账户</h2>
          <p class="auth-subtitle">注册后即可开始使用</p>
          <n-form ref="formRef" :model="form" :rules="rules" label-placement="left" :show-label="false">
            <n-form-item path="username">
              <n-input v-model:value="form.username" placeholder="用户名" size="large">
                <template #prefix><n-icon :component="PersonOutline" /></template>
              </n-input>
            </n-form-item>
            <n-form-item path="email">
              <n-input v-model:value="form.email" placeholder="邮箱地址" size="large">
                <template #prefix><n-icon :component="MailOutline" /></template>
              </n-input>
            </n-form-item>
            <n-form-item v-if="emailVerifyRequired" path="verification_code">
              <n-input-group>
                <n-input v-model:value="form.verification_code" placeholder="邮箱验证码" size="large" style="flex: 1;">
                  <template #prefix><n-icon :component="ShieldCheckmarkOutline" /></template>
                </n-input>
                <n-button size="large" :loading="sendingCode" :disabled="codeCooldown > 0" @click="handleSendCode" style="width: 120px;">
                  {{ codeCooldown > 0 ? codeCooldown + 's' : '发送验证码' }}
                </n-button>
              </n-input-group>
            </n-form-item>
            <n-form-item path="password">
              <n-input v-model:value="form.password" type="password" show-password-on="click" placeholder="密码（至少6位）" size="large">
                <template #prefix><n-icon :component="LockClosedOutline" /></template>
              </n-input>
            </n-form-item>
            <n-form-item path="invite_code">
              <n-input-group>
                <n-input v-model:value="form.invite_code" :placeholder="inviteRequired ? '邀请码（必填）' : '邀请码（选填）'" size="large" style="flex: 1" @blur="autoValidateInvite">
                  <template #prefix><n-icon :component="GiftOutline" /></template>
                </n-input>
                <n-button size="large" :loading="validatingInvite" @click="handleValidateInvite" style="width: 80px">验证</n-button>
              </n-input-group>
            </n-form-item>
            <n-alert v-if="inviteValid === true" type="success" :bordered="false" size="small" style="margin-bottom: 16px">
              邀请码有效{{ inviteReward > 0 ? `，注册后可获得 ¥${inviteReward} 奖励` : '' }}
            </n-alert>
            <n-alert v-else-if="inviteValid === false" type="error" :bordered="false" size="small" style="margin-bottom: 16px">
              {{ inviteError }}
            </n-alert>
            <n-button type="primary" block size="large" :loading="loading" @click="handleRegister" style="border-radius: 8px; height: 44px;">
              注 册
            </n-button>
          </n-form>
          <div class="auth-footer">
            已有账户？<router-link to="/login"><n-button text type="primary">立即登录</n-button></router-link>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage, type FormInst } from 'naive-ui'
import { PersonOutline, MailOutline, LockClosedOutline, GiftOutline, ShieldCheckmarkOutline } from '@vicons/ionicons5'
import { register, sendVerificationCode } from '@/api/auth'
import { getPublicConfig, validateInviteCode } from '@/api/common'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const formRef = ref<FormInst | null>(null)
const loading = ref(false)
const sendingCode = ref(false)
const codeCooldown = ref(0)
const validatingInvite = ref(false)
const inviteValid = ref<boolean | null>(null)
const inviteReward = ref(0)
const inviteError = ref('')

const siteConfig = ref<Record<string, string>>({})
const registerDisabled = computed(() => {
  const v = siteConfig.value['register_enabled']
  return v === 'false' || v === '0'
})
const emailVerifyRequired = computed(() => {
  const v = siteConfig.value['register_email_verify']
  return v === 'true' || v === '1'
})
const inviteRequired = computed(() => {
  const v = siteConfig.value['register_invite_required']
  return v === 'true' || v === '1'
})

const form = ref({ username: '', email: '', password: '', invite_code: '', verification_code: '' })
const rules = computed(() => ({
  username: { required: true, message: '请输入用户名', trigger: 'blur' },
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email' as const, message: '邮箱格式不正确', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码至少6位', trigger: 'blur' },
  ],
  invite_code: inviteRequired.value
    ? [{ required: true, message: '请输入邀请码', trigger: 'blur' }]
    : [],
  verification_code: emailVerifyRequired.value
    ? [{ required: true, message: '请输入验证码', trigger: 'blur' }]
    : [],
}))

const handleSendCode = async () => {
  if (!form.value.email) {
    message.warning('请先输入邮箱')
    return
  }
  sendingCode.value = true
  try {
    await sendVerificationCode({ email: form.value.email, purpose: 'register' })
    message.success('验证码已发送')
    codeCooldown.value = 60
    const timer = setInterval(() => {
      codeCooldown.value--
      if (codeCooldown.value <= 0) clearInterval(timer)
    }, 1000)
  } catch (e: any) {
    message.error(e.message || '发送失败')
  } finally {
    sendingCode.value = false
  }
}

async function handleRegister() {
  await formRef.value?.validate()
  loading.value = true
  try {
    await register(form.value)
    message.success('注册成功，请登录')
    router.push('/login')
  } catch (e: any) {
    message.error(e.message || '注册失败')
  } finally {
    loading.value = false
  }
}

const handleValidateInvite = async () => {
  const code = form.value.invite_code.trim()
  if (!code) { inviteValid.value = null; return }
  validatingInvite.value = true
  try {
    const res = await validateInviteCode(code)
    inviteValid.value = true
    inviteReward.value = res.data?.invitee_reward || 0
    inviteError.value = ''
  } catch (e: any) {
    inviteValid.value = false
    inviteError.value = e.message || '邀请码无效'
    inviteReward.value = 0
  } finally {
    validatingInvite.value = false
  }
}

const autoValidateInvite = () => {
  if (form.value.invite_code.trim() && inviteValid.value === null) {
    handleValidateInvite()
  }
}

onMounted(async () => {
  try {
    const res = await getPublicConfig()
    siteConfig.value = res.data || {}
  } catch {}
  // Read invite code from URL
  const code = route.query.code as string
  if (code) {
    form.value.invite_code = code
    handleValidateInvite()
  }
})
</script>

<style scoped>
.auth-page { height: 100vh; display: flex; }
.auth-left {
  flex: 1; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex; align-items: center; justify-content: center; color: #fff; padding: 48px;
}
.brand-area { max-width: 400px; }
.brand-logo {
  width: 56px; height: 56px; background: rgba(255,255,255,0.2); border-radius: 14px;
  display: flex; align-items: center; justify-content: center; font-size: 28px; font-weight: bold;
  backdrop-filter: blur(10px); margin-bottom: 24px;
}
.brand-area h1 { font-size: 36px; margin-bottom: 12px; font-weight: 700; }
.brand-desc { font-size: 16px; opacity: 0.85; margin-bottom: 40px; line-height: 1.6; }
.feature-list { display: flex; flex-direction: column; gap: 16px; }
.feature-item { display: flex; align-items: center; gap: 12px; font-size: 15px; opacity: 0.9; }
.feature-dot { width: 8px; height: 8px; border-radius: 50%; background: rgba(255,255,255,0.7); flex-shrink: 0; }
.auth-right {
  flex: 1; display: flex; align-items: center; justify-content: center; padding: 48px; background: var(--bg-color, #fff);
}
.auth-form-wrapper { width: 100%; max-width: 400px; }
.auth-form-wrapper h2 { font-size: 28px; font-weight: 700; margin-bottom: 8px; }
.auth-subtitle { color: var(--text-color-secondary, #999); margin-bottom: 32px; font-size: 15px; }
.auth-footer { text-align: center; margin-top: 24px; color: var(--text-color-secondary, #999); font-size: 14px; }
@media (max-width: 768px) { .auth-left { display: none; } }
</style>