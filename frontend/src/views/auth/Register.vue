<template>
  <AuthLayout title="高速稳定 · 全球节点" :subtitle="brandFeatures">
    <template v-if="registerDisabled">
      <h2 class="auth-form-title">暂未开放注册</h2>
      <p class="auth-form-subtitle">管理员已关闭注册功能</p>
      <div class="auth-form-footer">
        已有账户？<router-link to="/login" class="auth-link">立即登录</router-link>
      </div>
    </template>
    <template v-else>
      <h2 class="auth-form-title">创建账户</h2>
      <p class="auth-form-subtitle">注册后即可开始使用</p>

      <n-form ref="formRef" :model="form" :rules="rules" label-placement="top" size="large">
        <n-form-item path="username" label="用户名">
          <n-input v-model:value="form.username" placeholder="请输入用户名">
            <template #prefix><n-icon :component="PersonOutline" /></template>
          </n-input>
        </n-form-item>
        <n-form-item path="email" label="邮箱">
          <n-input v-model:value="form.email" placeholder="请输入邮箱">
            <template #prefix><n-icon :component="MailOutline" /></template>
          </n-input>
        </n-form-item>
        <n-form-item v-if="emailVerifyRequired" path="verification_code" label="邮箱验证码">
          <n-input-group>
            <n-input v-model:value="form.verification_code" placeholder="请输入验证码" style="flex: 1;">
              <template #prefix><n-icon :component="ShieldCheckmarkOutline" /></template>
            </n-input>
            <n-button size="large" :loading="sendingCode" :disabled="codeCooldown > 0" @click="handleSendCode" style="width: 120px;">
              {{ codeCooldown > 0 ? codeCooldown + 's' : '发送验证码' }}
            </n-button>
          </n-input-group>
        </n-form-item>
        <n-form-item path="password" label="密码">
          <n-input v-model:value="form.password" type="password" show-password-on="click" placeholder="密码（至少8位，含字母和数字）">
            <template #prefix><n-icon :component="LockClosedOutline" /></template>
          </n-input>
        </n-form-item>
        <n-form-item path="confirmPassword" label="确认密码">
          <n-input v-model:value="form.confirmPassword" type="password" show-password-on="click" placeholder="请再次输入密码" @keyup.enter="handleRegister">
            <template #prefix><n-icon :component="LockClosedOutline" /></template>
          </n-input>
        </n-form-item>
        <n-form-item v-if="inviteEnabled" path="invite_code" label="邀请码">
          <n-input-group>
            <n-input v-model:value="form.invite_code" :placeholder="inviteRequired ? '邀请码（必填）' : '邀请码（选填）'" style="flex: 1" @blur="autoValidateInvite">
              <template #prefix><n-icon :component="GiftOutline" /></template>
            </n-input>
            <n-button size="large" :loading="validatingInvite" @click="handleValidateInvite" style="width: 80px">验证</n-button>
          </n-input-group>
        </n-form-item>
        <n-alert v-if="inviteEnabled && inviteValid === true" type="success" :bordered="false" size="small" style="margin-bottom: 16px">
          邀请码有效{{ inviteReward > 0 ? `，注册后可获得 ¥${inviteReward} 奖励` : '' }}
        </n-alert>
        <n-alert v-else-if="inviteEnabled && inviteValid === false" type="error" :bordered="false" size="small" style="margin-bottom: 16px">
          {{ inviteError }}
        </n-alert>

        <div class="auth-terms-row">
          <n-checkbox v-model:checked="agreedTerms">
            我已阅读并同意
            <router-link to="/terms" class="auth-link" @click.stop>《服务条款》</router-link>
          </n-checkbox>
        </div>

        <n-button type="primary" block size="large" :loading="loading" :disabled="!agreedTerms" class="auth-submit-btn" @click="handleRegister">
          注 册
        </n-button>
      </n-form>
      <div class="auth-form-footer">
        已有账户？<router-link to="/login" class="auth-link">立即登录</router-link>
      </div>
    </template>
  </AuthLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage, type FormInst } from 'naive-ui'
import { PersonOutline, MailOutline, LockClosedOutline, GiftOutline, ShieldCheckmarkOutline } from '@vicons/ionicons5'
import AuthLayout from '@/components/AuthLayout.vue'
import { register, sendVerificationCode } from '@/api/auth'
import { getPublicConfig, validateInviteCode } from '@/api/common'
import { getErrorMessage, silentCatch } from '@/utils/error'

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
// 是否已同意《服务条款》
const agreedTerms = ref(false)
let codeTimer: ReturnType<typeof setInterval> | null = null

const brandFeatures = ['多格式订阅聚合，一键导入', '智能设备管理，安全可控', '实时节点监控，稳定高速']

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
const inviteEnabled = computed(() => inviteRequired.value)

const form = ref({ username: '', email: '', password: '', confirmPassword: '', invite_code: '', verification_code: '' })
const rules = computed(() => ({
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 32, message: '用户名长度需在3-32个字符之间', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_\u4e00-\u9fa5]+$/, message: '用户名只能包含字母、数字、下划线和中文', trigger: 'blur' },
  ],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email' as const, message: '邮箱格式不正确', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { validator: (_r: any, v: string) => !v || (v.length >= 8 && /[a-zA-Z]/.test(v) && /[0-9]/.test(v)), message: '密码至少8位，需包含字母和数字', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    { validator: (_r: any, v: string) => !v || v === form.value.password, message: '两次输入的密码不一致', trigger: 'blur' },
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
    codeTimer = setInterval(() => {
      codeCooldown.value--
      if (codeCooldown.value <= 0 && codeTimer) { clearInterval(codeTimer); codeTimer = null }
    }, 1000)
  } catch (e: any) {
    message.error(getErrorMessage(e, '发送失败'))
  } finally {
    sendingCode.value = false
  }
}

async function handleRegister() {
  // 未勾选《服务条款》禁止提交（按钮同时置灰，双保险）
  if (!agreedTerms.value) {
    message.warning('请先阅读并同意《服务条款》')
    return
  }
  await formRef.value?.validate()
  loading.value = true
  try {
    await register(form.value)
    message.success('注册成功，请登录')
    // 成功过渡：短暂停留 300ms 后跳转登录页
    await new Promise((resolve) => setTimeout(resolve, 300))
    router.push('/login')
  } catch (e: any) {
    message.error(getErrorMessage(e, '注册失败'))
  } finally {
    loading.value = false
  }
}

const handleValidateInvite = async () => {
  if (!inviteEnabled.value) {
    inviteValid.value = null
    inviteReward.value = 0
    inviteError.value = ''
    return
  }
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
    inviteError.value = getErrorMessage(e, '邀请码无效')
    inviteReward.value = 0
  } finally {
    validatingInvite.value = false
  }
}

const autoValidateInvite = () => {
  if (!inviteEnabled.value) return
  if (form.value.invite_code.trim() && inviteValid.value === null) {
    handleValidateInvite()
  }
}

onMounted(async () => {
  try {
    const res = await getPublicConfig()
    siteConfig.value = res.data || {}
  } catch (e) {
    silentCatch(e, 'loadSiteConfig')
  }
  // Read invite code from URL
  const code = route.query.code as string
  if (code && inviteEnabled.value) {
    form.value.invite_code = code
    handleValidateInvite()
  }
})

onUnmounted(() => { if (codeTimer) clearInterval(codeTimer) })
</script>
