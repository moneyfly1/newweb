<template>
  <div class="auth-page">
    <div class="auth-left">
      <div class="brand-area">
        <div class="brand-logo">C</div>
        <h1>CBoard</h1>
        <p class="brand-desc">高效、安全的代理订阅聚合管理平台</p>
        <div class="feature-list">
          <div class="feature-item">
            <div class="feature-dot" />
            <span>多格式订阅聚合，一键导入</span>
          </div>
          <div class="feature-item">
            <div class="feature-dot" />
            <span>智能设备管理，安全可控</span>
          </div>
          <div class="feature-item">
            <div class="feature-dot" />
            <span>实时节点监控，稳定高速</span>
          </div>
        </div>
      </div>
    </div>
    <div class="auth-right">
      <div class="auth-form-wrapper">
        <h2>欢迎回来</h2>
        <p class="auth-subtitle">登录你的账户继续使用</p>
        <n-form ref="formRef" :model="form" :rules="rules" label-placement="left" :show-label="false">
          <n-form-item path="email">
            <n-input v-model:value="form.email" placeholder="邮箱地址" size="large" :input-props="{ autocomplete: 'email' }">
              <template #prefix><n-icon :component="MailOutline" /></template>
            </n-input>
          </n-form-item>
          <n-form-item path="password">
            <n-input v-model:value="form.password" type="password" show-password-on="click" placeholder="密码" size="large" @keyup.enter="handleLogin">
              <template #prefix><n-icon :component="LockClosedOutline" /></template>
            </n-input>
          </n-form-item>
          <div class="form-extra">
            <n-checkbox v-model:checked="rememberMe">记住我</n-checkbox>
            <router-link to="/forgot-password"><n-button text type="primary" size="small">忘记密码？</n-button></router-link>
          </div>
          <n-button type="primary" block size="large" :loading="loading" @click="handleLogin" style="margin-top: 8px; border-radius: 8px; height: 44px;">
            登 录
          </n-button>
        </n-form>
        <div class="auth-footer">
          还没有账户？<router-link to="/register"><n-button text type="primary">立即注册</n-button></router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, type FormInst } from 'naive-ui'
import { MailOutline, LockClosedOutline } from '@vicons/ionicons5'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const message = useMessage()
const userStore = useUserStore()
const formRef = ref<FormInst | null>(null)
const loading = ref(false)
const rememberMe = ref(false)

const form = ref({ email: '', password: '' })
const rules = {
  email: { required: true, message: '请输入邮箱', trigger: 'blur' },
  password: { required: true, message: '请输入密码', trigger: 'blur' },
}

async function handleLogin() {
  await formRef.value?.validate()
  loading.value = true
  try {
    await userStore.login(form.value.email, form.value.password)
    message.success('登录成功')
    router.push(userStore.isAdmin ? '/admin' : '/')
  } catch (e: any) {
    message.error(e.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page {
  height: 100vh;
  display: flex;
}
.auth-left {
  flex: 1;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  padding: 48px;
}
.brand-area { max-width: 400px; }
.brand-logo {
  width: 56px; height: 56px;
  background: rgba(255,255,255,0.2);
  border-radius: 14px;
  display: flex; align-items: center; justify-content: center;
  font-size: 28px; font-weight: bold;
  backdrop-filter: blur(10px);
  margin-bottom: 24px;
}
.brand-area h1 { font-size: 36px; margin-bottom: 12px; font-weight: 700; }
.brand-desc { font-size: 16px; opacity: 0.85; margin-bottom: 40px; line-height: 1.6; }
.feature-list { display: flex; flex-direction: column; gap: 16px; }
.feature-item { display: flex; align-items: center; gap: 12px; font-size: 15px; opacity: 0.9; }
.feature-dot { width: 8px; height: 8px; border-radius: 50%; background: rgba(255,255,255,0.7); flex-shrink: 0; }
.auth-right {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px;
  background: var(--n-color);
}
.auth-form-wrapper { width: 100%; max-width: 400px; }
.auth-form-wrapper h2 { font-size: 28px; font-weight: 700; margin-bottom: 8px; }
.auth-subtitle { color: #999; margin-bottom: 32px; font-size: 15px; }
.form-extra { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.auth-footer { text-align: center; margin-top: 24px; color: #999; font-size: 14px; }

@media (max-width: 768px) {
  .auth-left { display: none; }
  .auth-right { flex: 1; }
}
</style>
