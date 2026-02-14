<template>
  <div class="admin-login-page">
    <div class="login-card">
      <div class="card-header">
        <div class="admin-logo">A</div>
        <h1>CBoard Admin</h1>
        <p class="subtitle">管理员登录</p>
      </div>

      <n-form ref="formRef" :model="form" :rules="rules" label-placement="left" :show-label="false">
        <n-form-item path="email">
          <n-input
            v-model:value="form.email"
            placeholder="用户名/邮箱"
            size="large"
            :input-props="{ autocomplete: 'email' }"
          >
            <template #prefix><n-icon :component="PersonOutline" /></template>
          </n-input>
        </n-form-item>

        <n-form-item path="password">
          <n-input
            v-model:value="form.password"
            type="password"
            show-password-on="click"
            placeholder="密码"
            size="large"
            @keyup.enter="handleLogin"
          >
            <template #prefix><n-icon :component="LockClosedOutline" /></template>
          </n-input>
        </n-form-item>

        <n-button
          type="primary"
          block
          size="large"
          :loading="loading"
          @click="handleLogin"
          style="margin-top: 16px; border-radius: 8px; height: 44px;"
        >
          登 录
        </n-button>
      </n-form>

      <div class="login-footer">
        <router-link to="/login">
          <n-button text type="primary">
            <template #icon><n-icon :component="ArrowBackOutline" /></template>
            返回用户登录
          </n-button>
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, type FormInst } from 'naive-ui'
import { PersonOutline, LockClosedOutline, ArrowBackOutline } from '@vicons/ionicons5'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const message = useMessage()
const userStore = useUserStore()
const formRef = ref<FormInst | null>(null)
const loading = ref(false)

const form = ref({ email: '', password: '' })
const rules = {
  email: { required: true, message: '请输入用户名或邮箱', trigger: 'blur' },
  password: { required: true, message: '请输入密码', trigger: 'blur' },
}

async function handleLogin() {
  await formRef.value?.validate()
  loading.value = true
  try {
    await userStore.login(form.value.email, form.value.password)

    if (!userStore.isAdmin) {
      message.error('您没有管理员权限，无法访问管理后台')
      userStore.logout()
      return
    }

    message.success('登录成功')
    router.push('/admin')
  } catch (e: any) {
    message.error(e.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.admin-login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1e3c72 0%, #2a5298 50%, #7e22ce 100%);
  padding: 24px;
}

.login-card {
  width: 100%;
  max-width: 420px;
  background: var(--n-color);
  border-radius: 16px;
  padding: 48px 40px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.card-header {
  text-align: center;
  margin-bottom: 40px;
}

.admin-logo {
  width: 64px;
  height: 64px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  font-weight: bold;
  color: #fff;
  margin-bottom: 20px;
  box-shadow: 0 8px 24px rgba(102, 126, 234, 0.4);
}

.card-header h1 {
  font-size: 28px;
  font-weight: 700;
  margin-bottom: 8px;
}

.subtitle {
  color: #999;
  font-size: 15px;
}

.login-footer {
  text-align: center;
  margin-top: 24px;
}

@media (max-width: 480px) {
  .login-card {
    padding: 36px 28px;
  }
}
</style>
