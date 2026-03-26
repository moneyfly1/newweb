<template>
  <div class="settings-page">
    <!-- 统一的页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">系统设置</h2>
        <p class="page-subtitle">配置站点的核心运行参数、支付接口及安全策略</p>
      </div>
      <div class="header-right">
        <n-button type="primary" :loading="saving" @click="handleSave">
          <template #icon><n-icon><save-outline /></n-icon></template>
          保存所有更改
        </n-button>
      </div>
    </div>

    <!-- 设置容器：仿应用级布局 -->
    <n-card :bordered="false" content-style="padding: 0;" class="settings-card">
      <div class="settings-layout">
        <!-- 左侧导航区 -->
        <div class="settings-sidebar">
          <div 
            v-for="item in menuOptions" 
            :key="item.key"
            :class="['sidebar-item', activeTab === item.key ? 'active' : '']"
            @click="activeTab = item.key"
          >
            <n-icon :component="item.icon" size="20" />
            <span class="item-label">{{ item.label }}</span>
          </div>
        </div>

        <!-- 右侧内容区 -->
        <div class="settings-content">
          <n-spin :show="loading">
            <div class="content-inner">
              <transition name="fade-slide" mode="out-in">
                <!-- 基础设置 -->
                <div v-if="activeTab === 'basic'" key="basic">
                  <n-h3 prefix="bar">站点信息</n-h3>
                  <n-grid :cols="appStore.isMobile ? 1 : 2" :x-gap="32">
                    <n-form-item-gi label="站点名称"><n-input v-model:value="form.site_name" placeholder="网站显示的名称" /></n-form-item-gi>
                    <n-form-item-gi label="站点地址"><n-input v-model:value="form.site_url" placeholder="https://your-domain.com" /></n-form-item-gi>
                    <n-form-item-gi label="站点描述" span="2"><n-input v-model:value="form.site_description" type="textarea" :rows="2" /></n-form-item-gi>
                  </n-grid>
                  <n-divider />
                  <n-h3 prefix="bar">注册与访问</n-h3>
                  <n-grid :cols="appStore.isMobile ? 1 : 2" :x-gap="32" :y-gap="16">
                    <n-form-item-gi label="开放用户注册"><n-switch v-model:value="form.register_enabled" /></n-form-item-gi>
                    <n-form-item-gi label="强制邀请码注册"><n-switch v-model:value="form.register_invite_required" /></n-form-item-gi>
                    <n-form-item-gi label="开启邮箱验证 (OTP)"><n-switch v-model:value="form.register_email_verify" /></n-form-item-gi>
                    <n-form-item-gi label="Telegram 快捷登录"><n-switch v-model:value="form.telegram_login_enabled" /></n-form-item-gi>
                  </n-grid>
                </div>

                <!-- 运营设置 -->
                <div v-else-if="activeTab === 'operation'" key="operation">
                  <n-h3 prefix="bar">新用户激励</n-h3>
                  <n-grid :cols="appStore.isMobile ? 1 : 2" :x-gap="32">
                    <n-form-item-gi label="初始订阅天数"><n-input-number v-model:value="form.default_subscribe_days" style="width:100%" /></n-form-item-gi>
                    <n-form-item-gi label="默认设备限制"><n-input-number v-model:value="form.default_device_limit" style="width:100%" /></n-form-item-gi>
                    <n-form-item-gi label="邀请人奖励 (元)"><n-input-number v-model:value="form.invite_default_inviter_reward" :precision="2" style="width:100%" /></n-form-item-gi>
                    <n-form-item-gi label="受邀人奖励 (元)"><n-input-number v-model:value="form.invite_default_invitee_reward" :precision="2" style="width:100%" /></n-form-item-gi>
                  </n-grid>
                  <n-divider />
                  <n-h3 prefix="bar">每日签到</n-h3>
                  <n-grid :cols="appStore.isMobile ? 1 : 3" :x-gap="24">
                    <n-form-item-gi label="启用签到"><n-switch v-model:value="form.checkin_enabled" /></n-form-item-gi>
                    <n-form-item-gi label="最小奖励 (分)"><n-input-number v-model:value="form.checkin_min_reward" style="width:100%" /></n-form-item-gi>
                    <n-form-item-gi label="最大奖励 (分)"><n-input-number v-model:value="form.checkin_max_reward" style="width:100%" /></n-form-item-gi>
                  </n-grid>
                </div>

                <!-- 支付设置 -->
                <div v-else-if="activeTab === 'payment'" key="payment">
                  <n-alert type="warning" style="margin-bottom: 24px;">敏感密钥在保存时若未修改（显示为 ****）将不会被覆盖。</n-alert>
                  <n-collapse arrow-placement="right" :default-expanded-names="['alipay', 'stripe']">
                    <n-collapse-item title="支付宝 (Alipay)" name="alipay">
                      <n-grid :cols="appStore.isMobile ? 1 : 2" :x-gap="32">
                        <n-form-item-gi label="启用状态"><n-switch v-model:value="form.pay_alipay_enabled" /></n-form-item-gi>
                        <n-form-item-gi label="沙箱模式"><n-switch v-model:value="form.pay_alipay_sandbox" /></n-form-item-gi>
                        <n-form-item-gi label="应用 AppID"><n-input v-model:value="form.pay_alipay_app_id" /></n-form-item-gi>
                        <n-form-item-gi label="支付公网域名"><n-input v-model:value="form.payment_public_base_url" placeholder="https://pay.example.com" /></n-form-item-gi>
                        <n-form-item-gi label="应用私钥" span="2"><n-input v-model:value="form.pay_alipay_private_key" type="textarea" :rows="3" /></n-form-item-gi>
                        <n-form-item-gi label="支付宝公钥" span="2"><n-input v-model:value="form.pay_alipay_public_key" type="textarea" :rows="3" /></n-form-item-gi>
                        <n-form-item-gi label="异步通知地址" span="2"><n-input v-model:value="form.pay_alipay_notify_url" placeholder="留空则使用支付公网域名自动生成 /api/v1/payment/notify/alipay" /></n-form-item-gi>
                        <n-form-item-gi label="同步返回地址" span="2"><n-input v-model:value="form.pay_alipay_return_url" placeholder="留空则使用支付公网域名自动生成 /api/v1/payment/success" /></n-form-item-gi>
                      </n-grid>
                    </n-collapse-item>
                    <n-collapse-item title="易支付 (Epay)" name="epay">
                      <n-grid :cols="appStore.isMobile ? 1 : 2" :x-gap="32">
                        <n-form-item-gi label="网关地址" span="2"><n-input v-model:value="form.pay_epay_gateway" placeholder="https://epay.example.com" /></n-form-item-gi>
                        <n-form-item-gi label="商户号"><n-input v-model:value="form.pay_epay_merchant_id" /></n-form-item-gi>
                        <n-form-item-gi label="商户密钥"><n-input v-model:value="form.pay_epay_secret_key" type="password" show-password-on="click" /></n-form-item-gi>
                      </n-grid>
                    </n-collapse-item>
                    <n-collapse-item title="Stripe (信用卡)" name="stripe">
                      <n-grid :cols="appStore.isMobile ? 1 : 2" :x-gap="32">
                        <n-form-item-gi label="启用状态"><n-switch v-model:value="form.pay_stripe_enabled" /></n-form-item-gi>
                        <n-form-item-gi label="汇率 (1 USD = ? CNY)"><n-input-number v-model:value="form.pay_stripe_exchange_rate" :precision="2" style="width:100%" /></n-form-item-gi>
                        <n-form-item-gi label="Secret Key" span="2"><n-input v-model:value="form.pay_stripe_secret_key" type="password" show-password-on="click" /></n-form-item-gi>
                        <n-form-item-gi label="Webhook Secret" span="2"><n-input v-model:value="form.pay_stripe_webhook_secret" type="password" show-password-on="click" /></n-form-item-gi>
                      </n-grid>
                    </n-collapse-item>
                    <n-collapse-item title="内部余额支付" name="balance">
                      <n-form-item label="允许使用余额购买套餐"><n-switch v-model:value="form.pay_balance_enabled" /></n-form-item>
                    </n-collapse-item>
                  </n-collapse>
                </div>

                <!-- 邮件设置 -->
                <div v-else-if="activeTab === 'email'" key="email">
                  <n-h3 prefix="bar">SMTP 发信服务器</n-h3>
                  <n-grid :cols="appStore.isMobile ? 1 : 2" :x-gap="32">
                    <n-form-item-gi label="SMTP 主机"><n-input v-model:value="form.smtp_host" /></n-form-item-gi>
                    <n-form-item-gi label="端口"><n-input-number v-model:value="form.smtp_port" style="width:100%" /></n-form-item-gi>
                    <n-form-item-gi label="用户名"><n-input v-model:value="form.smtp_username" /></n-form-item-gi>
                    <n-form-item-gi label="密码"><n-input v-model:value="form.smtp_password" type="password" show-password-on="click" /></n-form-item-gi>
                    <n-form-item-gi label="加密方式"><n-select v-model:value="form.smtp_encryption" :options="encryptionOptions" /></n-form-item-gi>
                    <n-form-item-gi label="发件人地址"><n-input v-model:value="form.smtp_from_email" /></n-form-item-gi>
                  </n-grid>
                  <n-divider />
                  <n-h3 prefix="bar">连接测试</n-h3>
                  <n-input-group style="max-width: 400px;">
                    <n-input v-model:value="testEmail" placeholder="接收测试邮件的邮箱" />
                    <n-button type="info" :loading="sendingTest" @click="handleSendTestEmail">发送测试</n-button>
                  </n-input-group>
                </div>

                <!-- 通知设置 -->
                <div v-else-if="activeTab === 'notify'" key="notify">
                  <n-h3 prefix="bar">Telegram 机器人通知</n-h3>
                  <n-grid :cols="appStore.isMobile ? 1 : 2" :x-gap="32">
                    <n-form-item-gi label="启用通知"><n-switch v-model:value="form.notify_telegram_enabled" /></n-form-item-gi>
                    <n-form-item-gi label="管理员 ChatID"><n-input v-model:value="form.notify_telegram_chat_id" /></n-form-item-gi>
                    <n-form-item-gi label="Bot Token" span="2"><n-input v-model:value="form.notify_telegram_bot_token" type="password" show-password-on="click" /></n-form-item-gi>
                  </n-grid>
                  <n-divider />
                  <n-h3 prefix="bar">管理员订阅事件</n-h3>
                  <n-grid :cols="appStore.isMobile ? 2 : 4" :x-gap="24" :y-gap="12">
                    <n-form-item-gi label="新用户注册"><n-switch v-model:value="form.notify_new_user" /></n-form-item-gi>
                    <n-form-item-gi label="新订单创建"><n-switch v-model:value="form.notify_new_order" /></n-form-item-gi>
                    <n-form-item-gi label="支付成功"><n-switch v-model:value="form.notify_payment_success" /></n-form-item-gi>
                    <n-form-item-gi label="新工单提醒"><n-switch v-model:value="form.notify_new_ticket" /></n-form-item-gi>
                  </n-grid>
                </div>

                <!-- 安全设置 -->
                <div v-else-if="activeTab === 'security'" key="security">
                  <n-h3 prefix="bar">后台安全控制</n-h3>
                  <n-grid :cols="appStore.isMobile ? 1 : 2" :x-gap="32">
                    <n-form-item-gi label="最大登录失败次数"><n-input-number v-model:value="form.max_login_attempts" style="width:100%" /></n-form-item-gi>
                    <n-form-item-gi label="锁定时长 (分钟)"><n-input-number v-model:value="form.login_lockout_minutes" style="width:100%" /></n-form-item-gi>
                    <n-form-item-gi label="管理后台 IP 白名单" span="2">
                      <n-input v-model:value="form.ip_whitelist" type="textarea" :rows="3" placeholder="每行一个 IP 地址，留空则不限制" />
                    </n-form-item-gi>
                  </n-grid>
                  <n-divider />
                  <n-h3 prefix="bar">数据维护</n-h3>
                  <n-space>
                    <n-button type="primary" :loading="backupCreating" @click="handleCreateBackup">创建数据库备份</n-button>
                    <n-button secondary @click="handleUpdateGeoIP">更新 GeoIP 数据库</n-button>
                  </n-space>
                </div>
              </transition>
            </div>
          </n-spin>
        </div>
      </div>
    </n-card>

    <!-- 移动端悬浮保存按钮 -->
    <div class="mobile-fab" v-if="appStore.isMobile">
      <n-button circle type="primary" size="large" :loading="saving" @click="handleSave">
        <template #icon><n-icon><save-outline /></n-icon></template>
      </n-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { 
  SaveOutline, SettingsOutline, RocketOutline, CardOutline, 
  MailOutline, NotificationsOutline, ShieldCheckmarkOutline, RefreshOutline 
} from '@vicons/ionicons5'
import { getSettings, updateSettings, sendTestEmail, testTelegram, createBackup, listBackups, updateGeoIPFiles } from '@/api/admin'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()
const message = useMessage()

const activeTab = ref('basic')
const loading = ref(false)
const saving = ref(false)
const sendingTest = ref(false)
const backupCreating = ref(false)
const testEmail = ref('')

const menuOptions = [
  { label: '基础设置', key: 'basic', icon: SettingsOutline },
  { label: '运营参数', key: 'operation', icon: RocketOutline },
  { label: '支付网关', key: 'payment', icon: CardOutline },
  { label: '邮件服务', key: 'email', icon: MailOutline },
  { label: '通知监控', key: 'notify', icon: NotificationsOutline },
  { label: '安全维护', key: 'security', icon: ShieldCheckmarkOutline },
]

const encryptionOptions = [
  { label: '无 (Plain)', value: 'none' },
  { label: 'STARTTLS (587)', value: 'tls' },
  { label: 'SSL/TLS (465)', value: 'ssl' }
]

const form = ref<Record<string, any>>({
  site_name: '', site_description: '', site_url: '',
  register_enabled: true, register_email_verify: false, register_invite_required: false,
  invite_default_inviter_reward: 0, invite_default_invitee_reward: 0,
  default_subscribe_days: 0, default_device_limit: 3,
  telegram_login_enabled: false,
  smtp_host: '', smtp_port: 465, smtp_username: '', smtp_password: '', smtp_encryption: 'ssl', smtp_from_email: '',
  pay_alipay_enabled: false, pay_alipay_sandbox: false, pay_alipay_app_id: '', pay_alipay_private_key: '', pay_alipay_public_key: '',
  payment_public_base_url: '', pay_alipay_notify_url: '', pay_alipay_return_url: '',
  pay_epay_gateway: '', pay_epay_merchant_id: '', pay_epay_secret_key: '',
  pay_stripe_enabled: false, pay_stripe_secret_key: '', pay_stripe_webhook_secret: '', pay_stripe_exchange_rate: 7.2,
  pay_balance_enabled: true,
  notify_telegram_enabled: false, notify_telegram_bot_token: '', notify_telegram_chat_id: '',
  notify_new_user: false, notify_new_order: false, notify_payment_success: false, notify_new_ticket: false,
  max_login_attempts: 5, login_lockout_minutes: 30, ip_whitelist: '',
  checkin_enabled: true, checkin_min_reward: 10, checkin_max_reward: 50
})

const maskedFields = ref<Set<string>>(new Set())
const sensitiveKeys = ['smtp_password', 'pay_alipay_private_key', 'pay_alipay_public_key', 'pay_epay_secret_key', 'pay_stripe_secret_key', 'pay_stripe_webhook_secret', 'notify_telegram_bot_token']

const loadSettings = async () => {
  loading.value = true
  try {
    const res = await getSettings()
    if (res.code === 0 && res.data) {
      const data = res.data as Record<string, any>
      maskedFields.value.clear()
      for (const key of Object.keys(form.value)) {
        if (key in data) {
          if (typeof form.value[key] === 'boolean') {
            form.value[key] = data[key] === true || data[key] === 'true' || data[key] === '1'
          } else if (typeof form.value[key] === 'number') {
            form.value[key] = Number(data[key]) || 0
          } else {
            form.value[key] = data[key]
            if (sensitiveKeys.includes(key) && data[key] === '****') maskedFields.value.add(key)
          }
        }
      }
    }
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  saving.value = true
  try {
    const dataToSave: Record<string, any> = {}
    for (const key of Object.keys(form.value)) {
      const val = form.value[key]
      if (sensitiveKeys.includes(key) && maskedFields.value.has(key) && val === '****') continue
      dataToSave[key] = val
    }
    const res = await updateSettings(dataToSave)
    if (res.code === 0) {
      message.success('系统配置已持久化保存')
      await loadSettings()
    }
  } finally {
    saving.value = false
  }
}

const handleSendTestEmail = async () => {
  if (!testEmail.value) return message.warning('请输入测试邮箱')
  sendingTest.value = true
  try {
    await sendTestEmail({ email: testEmail.value })
    message.success('测试邮件已发出')
  } finally { sendingTest.value = false }
}

const handleCreateBackup = async () => {
  backupCreating.value = true
  try {
    await createBackup()
    message.success('数据库备份已生成在服务器 backups 目录')
  } finally { backupCreating.value = false }
}

const handleUpdateGeoIP = async () => {
  try {
    await updateGeoIPFiles()
    message.success('GeoIP 库更新任务已启动')
  } catch {}
}

onMounted(() => loadSettings())
</script>

<style scoped>
.settings-page {
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-title {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  color: var(--n-title-text-color);
}

.page-subtitle {
  margin: 4px 0 0 0;
  color: #888;
  font-size: 14px;
}

.settings-card {
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0,0,0,0.05);
  min-height: 700px;
}

.settings-layout {
  display: flex;
  min-height: 700px;
}

/* 侧边导航区 */
.settings-sidebar {
  width: 240px;
  background: #f9fafb;
  border-right: 1px solid #efeff5;
  padding: 16px 0;
  flex-shrink: 0;
}

.sidebar-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 24px;
  cursor: pointer;
  transition: all 0.2s ease;
  color: #666;
}

.sidebar-item:hover {
  background: #f3f4f6;
  color: var(--n-primary-color);
}

.sidebar-item.active {
  background: white;
  color: var(--n-primary-color);
  font-weight: 600;
  border-right: 3px solid var(--n-primary-color);
  margin-right: -1px;
}

.item-label {
  font-size: 14px;
}

/* 内容展示区 */
.settings-content {
  flex: 1;
  background: white;
  padding: 32px 48px;
  overflow-y: auto;
}

.content-inner {
  max-width: 900px;
}

.mobile-fab {
  position: fixed;
  right: 24px;
  bottom: 80px;
  z-index: 100;
}

@media (max-width: 992px) {
  .settings-sidebar { width: 180px; }
  .settings-content { padding: 24px 32px; }
}

@media (max-width: 767px) {
  .settings-page { padding: 12px; }
  .settings-layout { flex-direction: column; }
  .settings-sidebar {
    width: 100%;
    display: flex;
    overflow-x: auto;
    padding: 8px;
    border-right: none;
    border-bottom: 1px solid #efeff5;
  }
  .sidebar-item {
    padding: 8px 16px;
    flex-shrink: 0;
    border-right: none !important;
    border-bottom: 2px solid transparent;
  }
  .sidebar-item.active {
    border-bottom: 2px solid var(--n-primary-color);
  }
  .settings-content { padding: 20px 16px; }
  .page-header { flex-direction: column; align-items: flex-start; gap: 16px; }
  .header-right { width: 100%; }
  .header-right .n-button { width: 100%; }
}

/* 切换动画 */
.fade-slide-enter-active, .fade-slide-leave-active {
  transition: all 0.25s ease;
}
.fade-slide-enter-from {
  opacity: 0;
  transform: translateX(10px);
}
.fade-slide-leave-to {
  opacity: 0;
  transform: translateX(-10px);
}
</style>
