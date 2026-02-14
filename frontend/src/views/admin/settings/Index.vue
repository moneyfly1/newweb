<template>
  <div class="settings-container">
    <n-card title="系统设置" :bordered="false">
      <n-spin :show="loading">
        <n-tabs type="line" animated>
          <!-- Tab 1: 基本设置 -->
          <n-tab-pane name="basic" tab="基本设置">
            <n-form label-placement="left" label-width="140" :model="form">
              <n-form-item label="站点名称">
                <n-input v-model:value="form.site_name" placeholder="请输入站点名称" />
              </n-form-item>
              <n-form-item label="站点描述">
                <n-input v-model:value="form.site_description" type="textarea" placeholder="请输入站点描述" :rows="3" />
              </n-form-item>
              <n-form-item label="站点地址">
                <n-input v-model:value="form.site_url" placeholder="https://example.com" />
              </n-form-item>
              <n-form-item label="客服邮箱">
                <n-input v-model:value="form.support_email" placeholder="support@example.com" />
              </n-form-item>
              <n-form-item label="客服 QQ">
                <n-input v-model:value="form.support_qq" placeholder="请输入客服 QQ" />
              </n-form-item>
              <n-form-item label="客服 Telegram">
                <n-input v-model:value="form.support_telegram" placeholder="请输入 Telegram 用户名" />
              </n-form-item>
              <n-space justify="center" style="margin-top: 24px">
                <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
              </n-space>
            </n-form>
          </n-tab-pane>

          <!-- Tab 2: 注册设置 -->
          <n-tab-pane name="register" tab="注册设置">
            <n-form label-placement="left" label-width="200" :model="form">
              <n-form-item label="开放注册">
                <n-switch v-model:value="form.register_enabled" />
              </n-form-item>
              <n-form-item label="注册邮箱验证">
                <n-switch v-model:value="form.register_email_verify" />
              </n-form-item>
              <n-form-item label="注册需要邀请码">
                <n-switch v-model:value="form.register_invite_required" />
              </n-form-item>
              <n-form-item label="新用户默认设备数限制">
                <n-input-number v-model:value="form.default_device_limit" :min="1" :max="100" />
              </n-form-item>
              <n-form-item label="新用户默认订阅天数">
                <n-input-number v-model:value="form.default_subscribe_days" :min="0" :max="3650" />
              </n-form-item>
              <n-form-item label="最小密码长度">
                <n-input-number v-model:value="form.min_password_length" :min="6" :max="32" />
              </n-form-item>
              <n-space justify="center" style="margin-top: 24px">
                <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
              </n-space>
            </n-form>
          </n-tab-pane>

          <!-- Tab 3: 邮件设置 -->
          <n-tab-pane name="email" tab="邮件设置">
            <n-form label-placement="left" label-width="140" :model="form">
              <n-form-item label="SMTP 主机">
                <n-input v-model:value="form.smtp_host" placeholder="smtp.example.com" />
              </n-form-item>
              <n-form-item label="SMTP 端口">
                <n-input-number v-model:value="form.smtp_port" :min="1" :max="65535" />
              </n-form-item>
              <n-form-item label="SMTP 用户名">
                <n-input v-model:value="form.smtp_username" placeholder="请输入 SMTP 用户名" />
              </n-form-item>
              <n-form-item label="SMTP 密码">
                <n-input v-model:value="form.smtp_password" type="password" show-password-on="click" placeholder="请输入 SMTP 密码" />
              </n-form-item>
              <n-form-item label="加密方式">
                <n-select v-model:value="form.smtp_encryption" :options="encryptionOptions" placeholder="请选择加密方式" />
              </n-form-item>
              <n-form-item label="发件人邮箱">
                <n-input v-model:value="form.smtp_from_email" placeholder="noreply@example.com" />
              </n-form-item>
              <n-form-item label="发件人名称">
                <n-input v-model:value="form.smtp_from_name" placeholder="请输入发件人名称" />
              </n-form-item>
              <n-divider />
              <n-form-item label="测试邮件">
                <n-space>
                  <n-input v-model:value="testEmail" placeholder="输入测试邮箱地址" style="width: 280px" />
                  <n-button type="info" :loading="sendingTest" @click="handleSendTestEmail">发送测试邮件</n-button>
                </n-space>
              </n-form-item>
              <n-space justify="center" style="margin-top: 24px">
                <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
              </n-space>
            </n-form>
          </n-tab-pane>

          <!-- Tab 4: 支付设置 -->
          <n-tab-pane name="payment" tab="支付设置">
            <n-space vertical :size="16">
              <n-card title="余额支付" size="small" :bordered="true">
                <n-form label-placement="left" label-width="140" :model="form">
                  <n-form-item label="启用余额支付">
                    <n-switch v-model:value="form.pay_balance_enabled" />
                  </n-form-item>
                </n-form>
              </n-card>

              <n-card title="支付宝" size="small" :bordered="true" :collapsible="true">
                <n-form label-placement="left" label-width="140" :model="form">
                  <n-form-item label="启用支付宝">
                    <n-switch v-model:value="form.pay_alipay_enabled" />
                  </n-form-item>
                  <n-form-item label="App ID">
                    <n-input v-model:value="form.pay_alipay_app_id" placeholder="请输入支付宝 App ID" />
                  </n-form-item>
                  <n-form-item label="应用私钥">
                    <n-input v-model:value="form.pay_alipay_private_key" type="textarea" placeholder="请输入应用私钥（PKCS1 或 PKCS8 格式）" :rows="3" />
                  </n-form-item>
                  <n-form-item label="支付宝公钥">
                    <n-input v-model:value="form.pay_alipay_public_key" type="textarea" placeholder="请输入支付宝公钥（用于验证回调签名）" :rows="3" />
                  </n-form-item>
                  <n-form-item label="异步回调地址">
                    <n-input v-model:value="form.pay_alipay_notify_url" :placeholder="alipayNotifyUrlHint" />
                    <template #feedback>
                      <span style="font-size: 12px; color: #999">支付宝服务器通知地址，留空则自动使用: {{ alipayNotifyUrlHint }}</span>
                    </template>
                  </n-form-item>
                  <n-form-item label="同步回调地址">
                    <n-input v-model:value="form.pay_alipay_return_url" :placeholder="alipayReturnUrlHint" />
                    <template #feedback>
                      <span style="font-size: 12px; color: #999">支付完成后跳转地址，留空则自动使用: {{ alipayReturnUrlHint }}</span>
                    </template>
                  </n-form-item>
                  <n-form-item label="沙箱模式">
                    <n-switch v-model:value="form.pay_alipay_sandbox" />
                    <span style="margin-left: 8px; font-size: 12px; color: #999">开启后使用支付宝沙箱环境测试</span>
                  </n-form-item>
                </n-form>
              </n-card>

              <n-card title="微信支付" size="small" :bordered="true" :collapsible="true">
                <n-form label-placement="left" label-width="140" :model="form">
                  <n-form-item label="启用微信支付">
                    <n-switch v-model:value="form.pay_wechat_enabled" />
                  </n-form-item>
                  <n-form-item label="App ID">
                    <n-input v-model:value="form.pay_wechat_app_id" placeholder="请输入微信 App ID" />
                  </n-form-item>
                  <n-form-item label="MCH ID">
                    <n-input v-model:value="form.pay_wechat_mch_id" placeholder="请输入商户号" />
                  </n-form-item>
                  <n-form-item label="API Key">
                    <n-input v-model:value="form.pay_wechat_api_key" type="password" show-password-on="click" placeholder="请输入 API 密钥" />
                  </n-form-item>
                </n-form>
              </n-card>

              <n-card title="易支付" size="small" :bordered="true" :collapsible="true">
                <n-form label-placement="left" label-width="140" :model="form">
                  <n-form-item label="启用易支付">
                    <n-switch v-model:value="form.pay_epay_enabled" />
                  </n-form-item>
                  <n-form-item label="网关地址">
                    <n-input v-model:value="form.pay_epay_gateway" placeholder="https://pay.example.com" />
                  </n-form-item>
                  <n-form-item label="商户 ID">
                    <n-input v-model:value="form.pay_epay_merchant_id" placeholder="请输入商户 ID" />
                  </n-form-item>
                  <n-form-item label="商户密钥">
                    <n-input v-model:value="form.pay_epay_secret_key" type="password" show-password-on="click" placeholder="请输入商户密钥" />
                  </n-form-item>
                </n-form>
              </n-card>

              <n-space justify="center" style="margin-top: 24px">
                <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
              </n-space>
            </n-space>
          </n-tab-pane>

          <!-- Tab 5: 通知设置 -->
          <n-tab-pane name="notification" tab="通知设置">
            <n-form label-placement="left" label-width="200" :model="form">
              <n-form-item label="管理员通知邮箱">
                <n-input v-model:value="form.notify_admin_email" placeholder="admin@example.com" />
              </n-form-item>
              <n-form-item label="Telegram Bot Token">
                <n-input v-model:value="form.notify_telegram_bot_token" placeholder="请输入 Bot Token" />
              </n-form-item>
              <n-form-item label="Telegram Chat ID">
                <n-input v-model:value="form.notify_telegram_chat_id" placeholder="请输入 Chat ID" />
              </n-form-item>
              <n-form-item label="Bark 服务器地址">
                <n-input v-model:value="form.notify_bark_server" placeholder="https://api.day.app" />
              </n-form-item>
              <n-form-item label="Bark Device Key">
                <n-input v-model:value="form.notify_bark_device_key" placeholder="请输入 Device Key" />
              </n-form-item>
              <n-divider />
              <n-form-item label="新订单通知">
                <n-switch v-model:value="form.notify_new_order" />
              </n-form-item>
              <n-form-item label="新工单通知">
                <n-switch v-model:value="form.notify_new_ticket" />
              </n-form-item>
              <n-form-item label="新用户注册通知">
                <n-switch v-model:value="form.notify_new_user" />
              </n-form-item>
              <n-form-item label="订阅到期提醒">
                <n-switch v-model:value="form.notify_expiry_reminder" />
              </n-form-item>
              <n-space justify="center" style="margin-top: 24px">
                <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
              </n-space>
            </n-form>
          </n-tab-pane>

          <!-- Tab 6: 安全设置 -->
          <n-tab-pane name="security" tab="安全设置">
            <n-form label-placement="left" label-width="200" :model="form">
              <n-form-item label="最大登录尝试次数">
                <n-input-number v-model:value="form.max_login_attempts" :min="1" :max="100" />
              </n-form-item>
              <n-form-item label="登录锁定时长 (分钟)">
                <n-input-number v-model:value="form.login_lockout_minutes" :min="1" :max="1440" />
              </n-form-item>
              <n-form-item label="会话超时 (分钟)">
                <n-input-number v-model:value="form.session_timeout_minutes" :min="5" :max="10080" />
              </n-form-item>
              <n-form-item label="异常登录提醒">
                <n-switch v-model:value="form.abnormal_login_alert" />
              </n-form-item>
              <n-form-item label="IP 白名单">
                <n-input v-model:value="form.ip_whitelist" type="textarea" placeholder="每行一个 IP 地址" :rows="5" />
              </n-form-item>
              <n-space justify="center" style="margin-top: 24px">
                <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
              </n-space>
            </n-form>
          </n-tab-pane>
          <!-- Tab 7: 备份与恢复 -->
          <n-tab-pane name="backup" tab="备份与恢复">
            <n-space vertical :size="16">
              <n-card title="数据库备份" size="small" :bordered="true">
                <n-space vertical :size="12">
                  <n-text>创建当前数据库的完整备份。备份文件将保存在服务器上。</n-text>
                  <n-space>
                    <n-button type="primary" :loading="backupCreating" @click="handleCreateBackup">创建备份</n-button>
                    <n-button @click="loadBackups" :loading="backupLoading">刷新列表</n-button>
                  </n-space>
                </n-space>
              </n-card>

              <n-card title="GitHub 备份设置" size="small" :bordered="true">
                <n-form label-placement="left" label-width="160" :model="form">
                  <n-form-item label="启用 GitHub 备份">
                    <n-switch v-model:value="form.backup_github_enabled" />
                  </n-form-item>
                  <n-form-item label="GitHub Token">
                    <n-input v-model:value="form.backup_github_token" type="password" show-password-on="click" placeholder="ghp_xxxxxxxxxxxx" />
                  </n-form-item>
                  <n-form-item label="仓库地址">
                    <n-input v-model:value="form.backup_github_repo" placeholder="username/repo-name" />
                  </n-form-item>
                  <n-form-item label="自动备份间隔 (小时)">
                    <n-input-number v-model:value="form.backup_interval_hours" :min="0" :max="720" />
                  </n-form-item>
                  <n-space justify="center" style="margin-top: 16px">
                    <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
                  </n-space>
                </n-form>
              </n-card>

              <n-card title="备份记录" size="small" :bordered="true">
                <n-data-table
                  :columns="backupColumns"
                  :data="backups"
                  :loading="backupLoading"
                  :bordered="false"
                  size="small"
                />
                <n-empty v-if="!backupLoading && backups.length === 0" description="暂无备份记录" style="padding: 40px 0" />
              </n-card>
            </n-space>
          </n-tab-pane>
        </n-tabs>
      </n-spin>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { getSettings, updateSettings, sendTestEmail, createBackup, listBackups } from '@/api/admin'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const sendingTest = ref(false)
const testEmail = ref('')
const backupLoading = ref(false)
const backupCreating = ref(false)
const backups = ref<any[]>([])

// Computed URL hints based on site_url
const siteBase = computed(() => {
  const url = form.value.site_url || ''
  if (!url) return window.location.origin
  return url.startsWith('http') ? url.replace(/\/+$/, '') : 'https://' + url.replace(/\/+$/, '')
})
const alipayNotifyUrlHint = computed(() => siteBase.value + '/api/v1/payment/notify/alipay')
const alipayReturnUrlHint = computed(() => siteBase.value + '/payment/return')

const encryptionOptions = [
  { label: '无', value: 'none' },
  { label: 'TLS', value: 'tls' },
  { label: 'SSL', value: 'ssl' }
]

// Boolean keys that should be stored as switches
const booleanKeys = [
  'register_enabled', 'register_email_verify', 'register_invite_required',
  'pay_balance_enabled', 'pay_alipay_enabled', 'pay_alipay_sandbox', 'pay_wechat_enabled', 'pay_epay_enabled',
  'notify_new_order', 'notify_new_ticket', 'notify_new_user', 'notify_expiry_reminder',
  'abnormal_login_alert', 'backup_github_enabled'
]

// Number keys that should be stored as numbers
const numberKeys = [
  'default_device_limit', 'default_subscribe_days', 'min_password_length',
  'smtp_port', 'max_login_attempts', 'login_lockout_minutes', 'session_timeout_minutes',
  'backup_interval_hours'
]

const form = ref<Record<string, any>>({
  // Basic
  site_name: '',
  site_description: '',
  site_url: '',
  support_email: '',
  support_qq: '',
  support_telegram: '',
  // Registration
  register_enabled: true,
  register_email_verify: false,
  register_invite_required: false,
  default_device_limit: 3,
  default_subscribe_days: 0,
  min_password_length: 8,
  // Email
  smtp_host: '',
  smtp_port: 465,
  smtp_username: '',
  smtp_password: '',
  smtp_encryption: 'ssl',
  smtp_from_email: '',
  smtp_from_name: '',
  // Payment
  pay_balance_enabled: true,
  pay_alipay_enabled: false,
  pay_alipay_app_id: '',
  pay_alipay_private_key: '',
  pay_alipay_public_key: '',
  pay_alipay_notify_url: '',
  pay_alipay_return_url: '',
  pay_alipay_sandbox: false,
  pay_wechat_enabled: false,
  pay_wechat_app_id: '',
  pay_wechat_mch_id: '',
  pay_wechat_api_key: '',
  pay_epay_enabled: false,
  pay_epay_gateway: '',
  pay_epay_merchant_id: '',
  pay_epay_secret_key: '',
  // Notification
  notify_admin_email: '',
  notify_telegram_bot_token: '',
  notify_telegram_chat_id: '',
  notify_bark_server: '',
  notify_bark_device_key: '',
  notify_new_order: false,
  notify_new_ticket: false,
  notify_new_user: false,
  notify_expiry_reminder: false,
  // Security
  max_login_attempts: 5,
  login_lockout_minutes: 30,
  session_timeout_minutes: 120,
  abnormal_login_alert: true,
  ip_whitelist: '',
  // Backup
  backup_github_enabled: false,
  backup_github_token: '',
  backup_github_repo: '',
  backup_interval_hours: 24,
})

const loadSettings = async () => {
  loading.value = true
  try {
    const res = await getSettings()
    if (res.code === 0 && res.data) {
      const data = res.data as Record<string, any>
      for (const key of Object.keys(form.value)) {
        if (key in data) {
          if (booleanKeys.includes(key)) {
            form.value[key] = data[key] === true || data[key] === 'true' || data[key] === '1'
          } else if (numberKeys.includes(key)) {
            form.value[key] = Number(data[key]) || form.value[key]
          } else {
            form.value[key] = data[key]
          }
        }
      }
    } else {
      message.error(res.message || '加载设置失败')
    }
  } catch (error: any) {
    message.error(error.message || '加载设置失败')
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  saving.value = true
  try {
    const res = await updateSettings(form.value)
    if (res.code === 0) {
      message.success('保存成功')
    } else {
      message.error(res.message || '保存失败')
    }
  } catch (error: any) {
    message.error(error.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const handleSendTestEmail = async () => {
  if (!testEmail.value) {
    message.warning('请输入测试邮箱地址')
    return
  }
  sendingTest.value = true
  try {
    const res = await sendTestEmail({ email: testEmail.value })
    if (res.code === 0) {
      message.success('测试邮件已发送')
    } else {
      message.error(res.message || '发送失败')
    }
  } catch (error: any) {
    message.error(error.message || '发送失败')
  } finally {
    sendingTest.value = false
  }
}

const backupColumns = [
  { title: '文件名', key: 'filename', ellipsis: { tooltip: true } },
  { title: '大小', key: 'size' },
  { title: '创建时间', key: 'created_at', width: 180 },
]

const loadBackups = async () => {
  backupLoading.value = true
  try {
    const res = await listBackups()
    backups.value = res.data || []
  } catch (error: any) {
    message.error(error.message || '加载备份列表失败')
  } finally {
    backupLoading.value = false
  }
}

const handleCreateBackup = async () => {
  backupCreating.value = true
  try {
    const res = await createBackup()
    message.success(res.message || '备份已创建')
    loadBackups()
  } catch (error: any) {
    message.error(error.message || '创建备份失败')
  } finally {
    backupCreating.value = false
  }
}

onMounted(() => {
  loadSettings()
})
</script>

<style scoped>
.settings-container {
  padding: 20px;
}

:deep(.n-tabs-pane-wrapper) {
  padding-top: 20px;
}

:deep(.n-card.n-card--bordered) {
  margin-bottom: 0;
}

@media (max-width: 767px) {
  .settings-container { padding: 8px; }
  :deep(.n-tabs-pane-wrapper) { padding-top: 12px; }
  :deep(.n-form-item .n-form-item-label) { min-width: 80px; }
}
</style>
