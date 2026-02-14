<template>
  <div>
    <n-tabs type="line" animated>
      <n-tab-pane name="profile" tab="个人资料">
        <n-card :bordered="false">
          <n-form ref="profileFormRef" :model="profileForm" label-placement="left" label-width="100">
            <n-form-item label="用户名">
              <n-input v-model:value="profileForm.username" placeholder="用户名" />
            </n-form-item>
            <n-form-item label="昵称">
              <n-input v-model:value="profileForm.nickname" placeholder="昵称（可选）" />
            </n-form-item>
            <n-form-item label="邮箱">
              <n-input :value="userStore.userInfo?.email" disabled />
            </n-form-item>
            <n-form-item label="主题">
              <n-select v-model:value="profileForm.theme" :options="themeOptions" />
            </n-form-item>
            <n-form-item label="语言">
              <n-select v-model:value="profileForm.language" :options="langOptions" />
            </n-form-item>
            <n-form-item label="时区">
              <n-select v-model:value="profileForm.timezone" :options="tzOptions" filterable />
            </n-form-item>
            <n-form-item>
              <n-button type="primary" :loading="savingProfile" @click="saveProfile">保存</n-button>
            </n-form-item>
          </n-form>
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="password" tab="修改密码">
        <n-card :bordered="false">
          <n-form ref="pwFormRef" :model="pwForm" :rules="pwRules" label-placement="left" label-width="100">
            <n-form-item label="当前密码" path="old_password">
              <n-input v-model:value="pwForm.old_password" type="password" show-password-on="click" />
            </n-form-item>
            <n-form-item label="新密码" path="new_password">
              <n-input v-model:value="pwForm.new_password" type="password" show-password-on="click" />
            </n-form-item>
            <n-form-item label="确认密码" path="confirm_password">
              <n-input v-model:value="pwForm.confirm_password" type="password" show-password-on="click" />
            </n-form-item>
            <n-form-item>
              <n-button type="primary" :loading="savingPw" @click="savePw">修改密码</n-button>
            </n-form-item>
          </n-form>
        </n-card>
      </n-tab-pane>
      <n-tab-pane name="notification" tab="通知设置">
        <n-card :bordered="false">
          <n-form label-placement="left" label-width="140">
            <n-form-item label="邮件通知">
              <n-switch v-model:value="notifForm.email_notifications" @update:value="saveNotif" />
            </n-form-item>
            <n-form-item label="异常登录提醒">
              <n-switch v-model:value="notifForm.abnormal_login_alert_enabled" @update:value="saveNotif" />
            </n-form-item>
            <n-form-item label="推送通知">
              <n-switch v-model:value="notifForm.push_notifications" @update:value="saveNotif" />
            </n-form-item>
          </n-form>
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="privacy" tab="隐私设置">
        <n-card :bordered="false">
          <n-form label-placement="left" label-width="140">
            <n-form-item label="数据共享">
              <n-switch v-model:value="privacyForm.data_sharing" @update:value="savePrivacy" />
            </n-form-item>
            <n-form-item label="使用分析">
              <n-switch v-model:value="privacyForm.analytics" @update:value="savePrivacy" />
            </n-form-item>
          </n-form>
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="login-history" tab="登录历史">
        <n-card :bordered="false">
          <n-data-table :columns="historyColumns" :data="loginHistory" :loading="loadingHistory" :pagination="{ pageSize: 10 }" />
        </n-card>
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useMessage, NTag, type FormInst } from 'naive-ui'
import { useUserStore } from '@/stores/user'
import { useAppStore } from '@/stores/app'
import { updateProfile, changePassword, getNotificationSettings, updateNotificationSettings, getPrivacySettings, updatePrivacySettings, getLoginHistory } from '@/api/user'

const message = useMessage()
const userStore = useUserStore()
const appStore = useAppStore()

const profileFormRef = ref<FormInst | null>(null)
const pwFormRef = ref<FormInst | null>(null)
const savingProfile = ref(false)
const savingPw = ref(false)
const loadingHistory = ref(false)

const profileForm = ref({
  username: userStore.userInfo?.username || '',
  nickname: '',
  theme: appStore.currentTheme,
  language: 'zh-CN',
  timezone: 'Asia/Shanghai',
})

const pwForm = ref({ old_password: '', new_password: '', confirm_password: '' })
const pwRules = {
  old_password: { required: true, message: '请输入当前密码', trigger: 'blur' },
  new_password: { required: true, message: '请输入新密码', trigger: 'blur', min: 6 },
  confirm_password: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: (_r: any, v: string) => v === pwForm.value.new_password, message: '两次密码不一致', trigger: 'blur' },
  ],
}

const notifForm = ref({ email_notifications: true, abnormal_login_alert_enabled: true, push_notifications: true })
const privacyForm = ref({ data_sharing: true, analytics: true })
const loginHistory = ref<any[]>([])

const themeOptions = appStore.availableThemes.map((t: any) => ({ label: t.label, value: t.value }))
const langOptions = [{ label: '简体中文', value: 'zh-CN' }, { label: 'English', value: 'en' }]
const tzOptions = [
  { label: 'Asia/Shanghai (UTC+8)', value: 'Asia/Shanghai' },
  { label: 'Asia/Tokyo (UTC+9)', value: 'Asia/Tokyo' },
  { label: 'America/New_York (UTC-5)', value: 'America/New_York' },
  { label: 'Europe/London (UTC+0)', value: 'Europe/London' },
]

const historyColumns = [
  { title: '登录时间', key: 'login_time', render: (row: any) => new Date(row.login_time).toLocaleString() },
  { title: 'IP 地址', key: 'ip_address' },
  { title: '地区', key: 'location', render: (row: any) => row.location || '-' },
  { title: '状态', key: 'login_status', render: (row: any) => h(NTag, { type: row.login_status === 'success' ? 'success' : 'error', size: 'small', bordered: false }, () => row.login_status === 'success' ? '成功' : '失败') },
]

async function saveProfile() {
  savingProfile.value = true
  try {
    await updateProfile(profileForm.value)
    appStore.setTheme(profileForm.value.theme)
    message.success('保存成功')
    await userStore.fetchUser()
  } catch (e: any) { message.error(e.message || '保存失败') }
  finally { savingProfile.value = false }
}

async function savePw() {
  await pwFormRef.value?.validate()
  savingPw.value = true
  try {
    await changePassword({ old_password: pwForm.value.old_password, new_password: pwForm.value.new_password })
    message.success('密码修改成功')
    pwForm.value = { old_password: '', new_password: '', confirm_password: '' }
  } catch (e: any) { message.error(e.message || '修改失败') }
  finally { savingPw.value = false }
}

async function saveNotif() {
  try { await updateNotificationSettings(notifForm.value) } catch {}
}
async function savePrivacy() {
  try { await updatePrivacySettings(privacyForm.value) } catch {}
}

onMounted(async () => {
  try {
    const res: any = await getNotificationSettings()
    if (res.data) Object.assign(notifForm.value, res.data)
  } catch {}
  try {
    const res: any = await getPrivacySettings()
    if (res.data) Object.assign(privacyForm.value, res.data)
  } catch {}
  loadingHistory.value = true
  try {
    const res: any = await getLoginHistory()
    loginHistory.value = res.data || []
  } catch {}
  finally { loadingHistory.value = false }
})
</script>

<style scoped>
@media (max-width: 767px) {
  .n-card { border-radius: 10px; }
}
</style>
