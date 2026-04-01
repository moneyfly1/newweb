<template>
  <div class="subscription-page">
    <n-spin :show="loading">
      <div v-if="!subscription" class="empty-state">
        <n-empty description="您还没有订阅">
          <template #extra>
            <n-button type="primary" size="large" @click="$router.push('/shop')">购买套餐</n-button>
          </template>
        </n-empty>
      </div>

      <template v-else>
        <!-- Modern Hero Card -->
        <div class="modern-hero">
          <div class="hero-header">
            <div class="hero-title-section">
              <div class="status-badge" :class="statusClass">
                <n-icon :size="16" :component="statusIcon" />
                <span>{{ statusText }}</span>
              </div>
              <h1 class="package-title">{{ subscription.package_name || '当前套餐' }}</h1>
            </div>
            <n-button type="primary" size="medium" @click="showUpgradeModal = true">
              <template #icon><n-icon :component="ArrowUpCircleOutline" /></template>
              升级套餐
            </n-button>
          </div>

          <div class="stats-grid">
            <div class="stat-item">
              <div class="stat-icon balance-icon">
                <n-icon :size="24" :component="WalletOutline" />
              </div>
              <div class="stat-content">
                <span class="stat-label">账户余额</span>
                <span class="stat-value">¥{{ (userBalance ?? 0).toFixed(2) }}</span>
              </div>
            </div>
            <div class="stat-item">
              <div class="stat-icon days-icon">
                <n-icon :size="24" :component="TimeOutline" />
              </div>
              <div class="stat-content">
                <span class="stat-label">剩余天数</span>
                <span class="stat-value">{{ remainingDays }} 天</span>
              </div>
            </div>
            <div class="stat-item">
              <div class="stat-icon device-icon">
                <n-icon :size="24" :component="PhonePortraitOutline" />
              </div>
              <div class="stat-content">
                <span class="stat-label">设备使用</span>
                <span class="stat-value">{{ devices.length }}/{{ subscription.device_limit || 0 }}</span>
              </div>
            </div>
            <div class="stat-item">
              <div class="stat-icon expire-icon">
                <n-icon :size="24" :component="CalendarOutline" />
              </div>
              <div class="stat-content">
                <span class="stat-label">到期时间</span>
                <span class="stat-value stat-date">{{ formatDate(subscription.expire_time) }}</span>
              </div>
            </div>
          </div>

          <div class="hero-footer">
            <n-button
              text
              size="small"
              :disabled="!canConvert"
              @click="showConvertModal = true"
            >
              转换剩余天数为余额
            </n-button>
          </div>
        </div>

        <!-- Subscription URLs Card -->
        <div class="url-card">
          <div class="card-header">
            <h3 class="card-title">订阅地址</h3>
            <n-space :size="8">
              <n-button size="small" @click="handleSendEmail" :loading="sendingEmail">
                <template #icon><n-icon :component="MailOutline" /></template>
                发送邮箱
              </n-button>
              <n-button size="small" type="warning" @click="showResetModal = true">
                <template #icon><n-icon :component="RefreshOutline" /></template>
                重置
              </n-button>
            </n-space>
          </div>

          <div class="url-list">
            <div class="url-item">
              <div class="url-header">
                <span class="url-type">通用订阅</span>
                <n-tag size="small" type="info">Universal / V2Ray / Shadowrocket</n-tag>
              </div>
              <div class="url-actions">
                <n-input class="url-input" :value="showSubUrls ? subscriptionUrl : maskUrl(subscriptionUrl)" readonly size="small" />
                <div class="url-buttons">
                  <n-button size="small" @click="showSubUrls = !showSubUrls">
                    <template #icon><n-icon :component="showSubUrls ? EyeOffOutline : EyeOutline" /></template>
                    {{ showSubUrls ? '隐藏' : '显示' }}
                  </n-button>
                  <n-button type="primary" size="small" @click="copyToClipboard(subscriptionUrl, '通用订阅地址')">
                    <template #icon><n-icon :component="CopyOutline" /></template>
                    复制
                  </n-button>
                  <n-button size="small" @click="showQrCode(subscriptionUrl, '通用订阅')">
                    <template #icon><n-icon :component="QrCodeOutline" /></template>
                    二维码
                  </n-button>
                </div>
              </div>
            </div>

            <div class="url-item">
              <div class="url-header">
                <span class="url-type">Clash 订阅</span>
                <n-tag size="small" type="success">Clash / Meta / ClashX</n-tag>
              </div>
              <div class="url-actions">
                <n-input class="url-input" :value="showSubUrls ? clashUrl : maskUrl(clashUrl)" readonly size="small" />
                <div class="url-buttons">
                  <n-button type="primary" size="small" @click="copyToClipboard(clashUrl, 'Clash 订阅地址')">
                    <template #icon><n-icon :component="CopyOutline" /></template>
                    复制
                  </n-button>
                  <n-button size="small" @click="showQrCode(clashUrl, 'Clash 订阅')">
                    <template #icon><n-icon :component="QrCodeOutline" /></template>
                    二维码
                  </n-button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Format Selector Card -->
        <div class="format-card-container">
          <div class="card-header">
            <h3 class="card-title">快速导入</h3>
            <span class="card-subtitle">选择客户端格式一键导入</span>
          </div>

          <div class="format-grid">
            <div
              v-for="fmt in formats"
              :key="fmt.type"
              class="format-item"
              :class="{ active: selectedFormat === fmt.type }"
              @click="selectedFormat = fmt.type"
            >
              <div class="format-item-icon">
                <img v-if="fmt.iconUrl" :src="fmt.iconUrl" :alt="fmt.name" loading="lazy" />
                <span v-else style="font-size: 32px;">{{ fmt.icon }}</span>
              </div>
              <span class="format-item-name">{{ fmt.name }}</span>
              <div class="format-item-actions">
                <n-button size="small" type="primary" @click.stop="copyToClipboard(getFormatUrl(fmt), fmt.name)">
                  复制
                </n-button>
                <n-button size="small" @click.stop="importFormat(fmt)">
                  导入
                </n-button>
              </div>
            </div>
          </div>
        </div>

        <!-- 设备列表卡片 -->
        <div class="device-card">
          <div class="card-header">
            <h3 class="card-title">我的设备</h3>
            <span class="card-subtitle">{{ devices.length }}/{{ subscription.device_limit }} 台设备</span>
          </div>
          <!-- Desktop Table -->
          <n-data-table
            v-if="!appStore.isMobile"
            :columns="deviceColumns"
            :data="devices"
            :pagination="false"
            :bordered="false"
            size="small"
          />
          <!-- Mobile Card List -->
          <div v-else class="mobile-device-list">
            <div v-if="devices.length === 0" class="empty-device-list">暂无设备记录</div>
            <div v-for="dev in devices" :key="dev.id" class="mobile-device-item">
              <div class="device-name">{{ dev.name || dev.ip || '未知设备' }}</div>
              <div class="device-info-row">
                <span class="device-label">IP</span>
                <span class="device-value">{{ dev.ip || '-' }}</span>
              </div>
              <div class="device-info-row">
                <span class="device-label">位置</span>
                <span class="device-value">{{ formatLocation(dev.location || dev.region || '') || '-' }}</span>
              </div>
              <div class="device-info-row">
                <span class="device-label">最后在线</span>
                <span class="device-value">{{ dev.last_active ? formatDate(dev.last_active) : '-' }}</span>
              </div>
            </div>
          </div>
        </div>
      </template>
    </n-spin>

    <!-- QR Code Drawer -->
    <common-drawer v-model:show="showQrModal" :title="qrTitle" :width="340">
      <div style="text-align: center;">
        <canvas ref="qrCanvas" style="margin: 0 auto;"></canvas>
        <p style="margin-top: 12px; color: #999; font-size: 13px;">使用客户端扫描二维码导入订阅</p>
      </div>
    </common-drawer>

    <!-- Reset Drawer -->
    <common-drawer v-model:show="showResetModal" title="重置订阅地址" :width="400" show-footer @confirm="handleResetSubscription" @cancel="showResetModal = false">
      <p>重置后原订阅地址将失效，所有设备需要重新配置。确定要继续吗？</p>
    </common-drawer>

    <!-- Convert Drawer -->
    <common-drawer v-model:show="showConvertModal" title="转换剩余天数" :width="400" show-footer @confirm="handleConvertToBalance" @cancel="showConvertModal = false">
      <p>将剩余 {{ remainingDays }} 天转换为余额，转换后订阅将立即失效。确定要继续吗？</p>
    </common-drawer>

    <!-- Upgrade Pay Drawer -->
    <common-drawer
      v-model:show="showUpgradePayModal"
      title="确认支付 - 升级订阅"
      :width="520"
    >
      <n-space vertical :size="16" class="drawer-section upgrade-pay-section">
        <n-descriptions :column="1" bordered>
          <n-descriptions-item label="升级内容">
            增加 {{ upgradeAddDevices }} 台设备<span v-if="upgradeExtendMonths > 0">，续期 {{ upgradeExtendMonths }} 月</span>
          </n-descriptions-item>
          <n-descriptions-item label="应付金额">
            <span style="color: #18a058; font-size: 20px; font-weight: bold;">¥{{ (upgradeOrderInfo?.final_amount ?? upgradeOrderInfo?.amount ?? 0).toFixed(2) }}</span>
          </n-descriptions-item>
          <n-descriptions-item label="账户余额">
            <span :style="{ color: (userBalance ?? 0) >= (upgradeOrderInfo?.final_amount ?? upgradeOrderInfo?.amount ?? 0) ? '#18a058' : '#e03050' }">
              ¥{{ (userBalance ?? 0).toFixed(2) }}
            </span>
          </n-descriptions-item>
          <n-descriptions-item v-if="useBalanceDeduct && paymentMethod !== 'balance'" label="余额抵扣">
            <span style="color: #18a058;">-¥{{ balanceDeductAmount.toFixed(2) }}</span>
          </n-descriptions-item>
          <n-descriptions-item v-if="useBalanceDeduct && paymentMethod !== 'balance'" label="还需支付">
            <span style="color: #e03050; font-size: 18px; font-weight: bold;">¥{{ remainingAmount.toFixed(2) }}</span>
          </n-descriptions-item>
        </n-descriptions>
        <div class="payment-method">
          <div class="pm-label">支付方式</div>
          <n-radio-group v-model:value="paymentMethod">
            <n-space vertical :size="8">
              <n-radio v-if="balanceEnabled" value="balance">余额支付</n-radio>
              <n-radio v-for="pm in paymentMethods" :key="pm.id" :value="'pm_' + pm.id">
                {{ getPaymentLabel(pm.pay_type) }}
              </n-radio>
            </n-space>
          </n-radio-group>
          <div v-if="paymentMethod !== 'balance' && (userBalance ?? 0) > 0 && balanceEnabled" style="margin-top: 8px;">
            <n-checkbox v-model:checked="useBalanceDeduct">
              使用余额抵扣 ¥{{ Math.min(userBalance ?? 0, finalPayAmount).toFixed(2) }}
            </n-checkbox>
          </div>
        </div>
      </n-space>
      <template #footer>
        <n-space justify="end" class="drawer-footer-actions">
          <n-button @click="showUpgradePayModal = false">取消</n-button>
          <n-button type="primary" :loading="paying" @click="handleUpgradePay">确认支付</n-button>
        </n-space>
      </template>
    </common-drawer>

    <!-- Pay QR Drawer -->
    <common-drawer v-model:show="showPayQrModal" title="扫码支付" :width="400" :mask-closable="false" @after-leave="stopPayPolling">
      <div v-if="isMobile" class="mobile-pay-panel">
        <p style="margin-bottom: 16px; color: #666;">请点击下方按钮完成支付</p>
        <n-button type="primary" size="large" block tag="a" :href="mobilePayUrl" target="_blank">打开支付App付款</n-button>
        <n-spin v-if="payPollingStatus" size="small" style="margin-top: 8px;" />
      </div>
      <div v-else class="desktop-pay-panel">
        <p style="margin-bottom: 16px; color: #666;">请使用支付宝扫描下方二维码完成支付</p>
        <canvas ref="payQrCanvas" style="margin: 0 auto;"></canvas>
        <n-spin v-if="payPollingStatus" size="small" style="margin-top: 8px;" />
      </div>
      <template #footer>
        <n-space justify="center"><n-button @click="showPayQrModal = false">取消支付</n-button></n-space>
      </template>
    </common-drawer>

    <!-- Crypto Pay Drawer -->
    <common-drawer v-model:show="showCryptoModal" title="加密货币支付" :width="480" :mask-closable="false" @after-leave="stopPayPolling">
      <div v-if="cryptoInfo" class="crypto-panel">
        <n-descriptions :column="1" bordered size="small" style="text-align: left;">
          <n-descriptions-item label="网络">{{ cryptoInfo.network }}</n-descriptions-item>
          <n-descriptions-item label="币种">{{ cryptoInfo.currency }}</n-descriptions-item>
          <n-descriptions-item label="转账金额">
            <span style="color: #e03050; font-size: 18px; font-weight: bold;">{{ cryptoInfo.amount_usdt }} {{ cryptoInfo.currency }}</span>
          </n-descriptions-item>
          <n-descriptions-item label="收款地址">
            <div style="word-break: break-all; font-family: monospace; font-size: 13px;">{{ cryptoInfo.wallet_address }}</div>
          </n-descriptions-item>
        </n-descriptions>
        <div style="margin-top: 16px;"><canvas ref="cryptoQrCanvas" style="margin: 0 auto;"></canvas></div>
        <n-alert type="warning" :bordered="false" style="margin-top: 12px; text-align: left;" size="small">
          请务必确认网络和币种正确，转账错误无法找回。
        </n-alert>
        <n-spin v-if="payPollingStatus" size="small" style="margin-top: 8px;" />
      </div>
      <template #footer>
        <n-space justify="center">
          <n-button @click="showCryptoModal = false">取消</n-button>
          <n-button type="primary" @click="handleCryptoTransferred">我已转账</n-button>
        </n-space>
      </template>
    </common-drawer>

    <!-- Upgrade Drawer -->
    <common-drawer v-model:show="showUpgradeModal" title="升级订阅" :width="480">
      <n-form class="drawer-section upgrade-form" label-placement="left" label-width="auto" :disabled="upgradeSubmitting" size="medium">
        <n-form-item label="增加设备数">
          <n-input-number v-model:value="upgradeAddDevices" :min="1" :max="50" :step="1" style="width: 100%;" />
        </n-form-item>
        <n-form-item label="续期月数">
          <n-input-number v-model:value="upgradeExtendMonths" :min="0" :max="120" style="width: 100%;" />
        </n-form-item>
        <div v-if="upgradeResult" class="upgrade-result">
          <n-descriptions :column="1" bordered size="small">
            <n-descriptions-item label="新增设备费用">¥{{ upgradeResult.fee_new_devices.toFixed(2) }}</n-descriptions-item>
            <n-descriptions-item label="续期费用">¥{{ upgradeResult.fee_extend.toFixed(2) }}</n-descriptions-item>
            <n-descriptions-item label="合计">
              <span style="color: #e03050; font-size: 18px; font-weight: bold;">¥{{ upgradeResult.total.toFixed(2) }}</span>
            </n-descriptions-item>
          </n-descriptions>
        </div>
      </n-form>
      <template #footer>
        <n-space justify="end" class="drawer-footer-actions">
          <n-button @click="showUpgradeModal = false">取消</n-button>
          <n-button type="primary" :loading="upgradeCalcLoading" @click="handleCalcUpgrade">计算金额</n-button>
          <n-button v-if="upgradeResult && upgradeResult.total > 0" type="success" :loading="upgradeSubmitting" @click="handleOpenUpgradePay">去支付</n-button>
        </n-space>
      </template>
    </common-drawer>
    <!-- Upgrade Success Drawer -->
    <common-drawer v-model:show="showUpgradeSuccess" title="升级成功" :width="440" show-footer :show-confirm="false" cancel-text="我知道了" @cancel="showUpgradeSuccess = false">
      <n-space vertical :size="14">
        <n-alert type="success" :bordered="false">
          您的订阅升级已生效。
        </n-alert>
        <n-descriptions v-if="upgradeSuccessInfo" :column="1" bordered>
          <n-descriptions-item label="增加设备数">+{{ upgradeSuccessInfo.addDevices }} 台</n-descriptions-item>
          <n-descriptions-item label="续期时长">{{ upgradeSuccessInfo.extendMonths > 0 ? `${upgradeSuccessInfo.extendMonths} 个月` : '未续期' }}</n-descriptions-item>
          <n-descriptions-item label="升级后设备上限">{{ upgradeSuccessInfo.deviceLimit }} 台</n-descriptions-item>
          <n-descriptions-item label="升级后到期时间">{{ formatDate(upgradeSuccessInfo.expireTime) }}</n-descriptions-item>
          <n-descriptions-item label="本次支付金额">¥{{ upgradeSuccessInfo.amount.toFixed(2) }}</n-descriptions-item>
        </n-descriptions>
      </n-space>
    </common-drawer>
  </div>
</template>
<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useMessage } from 'naive-ui'
import QRCode from 'qrcode'
import {
  CopyOutline, TimeOutline, PhonePortraitOutline, TrashOutline,
  RefreshOutline, SwapHorizontalOutline, MailOutline,
  CheckmarkCircle, CloseCircle, AlertCircle, QrCodeOutline,
  ArrowUpCircleOutline, WalletOutline, CalendarOutline, EyeOutline, EyeOffOutline
} from '@vicons/ionicons5'
import {
  getSubscription, getSubscriptionDevices, deleteDevice,
  resetSubscription, convertToBalance, sendSubscriptionEmail
} from '@/api/subscription'
import { calcUpgradePrice, createUpgradeOrder, payOrder, createPayment, getOrderStatus } from '@/api/order'
import { getPaymentMethods } from '@/api/common'
import { getDashboardInfo } from '@/api/user'
import { copyToClipboard as clipboardCopy } from '@/utils/clipboard'
import { safeRedirect } from '@/utils/security'
import { getErrorMessage } from '@/utils/error'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { formatLocation } from '@/utils/i18n'
import CommonDrawer from '@/components/CommonDrawer.vue'

const appStore = useAppStore()

const message = useMessage()
const router = useRouter()

const subscription = ref<any>(null)
const devices = ref<any[]>([])
const deviceColumns = [
  { title: '设备名称', key: 'device_name', width: 150 },
  { title: '客户端', key: 'software_name', width: 120, render: (row: any) => row.software_name || '未知' },
  { title: '版本', key: 'software_version', width: 80 },
  { title: '系统', key: 'os_name', width: 80, render: (row: any) => row.os_name || '未知' },
  { title: '设备型号', key: 'device_model', width: 150 },
  { title: '订阅类型', key: 'subscription_type', width: 100 },
  { title: '地区', key: 'region', width: 120, render: (row: any) => formatLocation(row.region || '') || '-' },
  { title: '最后访问', key: 'last_access', width: 150, render: (row: any) => formatDateTime(row.last_access) }
]
const loading = ref(false)
const showSubUrls = ref(false)
const showResetModal = ref(false)

function maskUrl(url: string) {
  if (!url || url.length < 20) return '••••••••'
  return url.substring(0, 20) + '••••••••' + url.substring(url.length - 6)
}
const showConvertModal = ref(false)
const sendingEmail = ref(false)
const selectedFormat = ref('clash')
const showUpgradeModal = ref(false)

const showUpgradeSuccess = ref(false)
const upgradeSuccessInfo = ref<{ addDevices: number; extendMonths: number; deviceLimit: number; expireTime: string; amount: number } | null>(null)
const showQrModal = ref(false)
const qrCanvas = ref<HTMLCanvasElement | null>(null)
const qrTitle = ref('')

// Upgrade
const upgradeAddDevices = ref(1)
const upgradeExtendMonths = ref(0)
const upgradeResult = ref<{ fee_extend: number; fee_new_devices: number; total: number } | null>(null)
const upgradeCalcLoading = ref(false)
const upgradeSubmitting = ref(false)
const upgradeOrderInfo = ref<any>(null)

// Payment
const showUpgradePayModal = ref(false)
const paymentMethod = ref('balance')
const paymentMethods = ref<any[]>([])
const balanceEnabled = ref(true)
const userBalance = ref<number>(0)
const useBalanceDeduct = ref(false)
const paying = ref(false)
const showPayQrModal = ref(false)
const payQrCanvas = ref<HTMLCanvasElement | null>(null)
const payPollingStatus = ref(false)
const mobilePayUrl = ref('')
const isMobile = ref(typeof window !== 'undefined' && window.innerWidth <= 767)
const showCryptoModal = ref(false)
const cryptoInfo = ref<any>(null)
const cryptoOrderNo = ref('')
const cryptoQrCanvas = ref<HTMLCanvasElement | null>(null)
let payPollTimer: ReturnType<typeof setInterval> | null = null
const finalPayAmount = computed(() => upgradeOrderInfo.value?.final_amount ?? upgradeOrderInfo.value?.amount ?? 0)
const balanceDeductAmount = computed(() => {
  if (paymentMethod.value === 'balance') return finalPayAmount.value
  if (useBalanceDeduct.value) return Math.min(userBalance.value ?? 0, finalPayAmount.value)
  return 0
})
const remainingAmount = computed(() => Math.max(0, finalPayAmount.value - balanceDeductAmount.value))

// 每个客户端的配置：type 对应后端 ?type=xxx 参数，urlKey 对应 GetUserSubscription 返回的字段名
const formats = [
  {
    type: 'clash',
    name: 'Clash / Meta',
    icon: '⚔️',
    iconUrl: 'https://fastly.jsdelivr.net/gh/walkxcode/dashboard-icons@main/png/clash.png',
    desc: 'Clash / Mihomo / ClashX',
    urlKey: 'token_clash_url',
    scheme: 'clash://install-config?url=',
  },
  {
    type: 'stash',
    name: 'Stash',
    icon: '📦',
    iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/stash.png',
    desc: 'iOS Clash',
    urlKey: 'token_stash_url',
    scheme: 'stash://install-config?url=',
  },
  {
    type: 'surge',
    name: 'Surge',
    icon: '🌊',
    iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/surge.png',
    desc: 'macOS / iOS',
    urlKey: 'token_surge_url',
    scheme: 'surge:///install-config?url=',
  },
  {
    type: 'loon',
    name: 'Loon',
    icon: '🎈',
    iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/loon.png',
    desc: 'iOS 客户端',
    urlKey: 'token_loon_url',
    scheme: 'loon://import/proxy?url=',
  },
  {
    type: 'quantumultx',
    name: 'Quantumult X',
    icon: '💠',
    iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/quantumultx.png',
    desc: 'iOS 客户端',
    urlKey: 'token_quantumultx_url',
    scheme: 'quantumult-x:///add-resource?remote-resource=',
  },
  {
    type: 'shadowrocket',
    name: 'Shadowrocket',
    icon: '🔴',
    iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/shadowrocket.png',
    desc: 'iOS 小火箭',
    urlKey: 'token_url',
    scheme: 'shadowrocket://add/',
  },
  {
    type: 'singbox',
    name: 'SingBox',
    icon: '📱',
    iconUrl: 'https://raw.githubusercontent.com/SagerNet/sing-box/testing/docs/assets/icon.svg',
    desc: '全平台通用',
    urlKey: 'token_singbox_url',
    scheme: '',
  },
  {
    type: 'v2ray',
    name: 'V2Ray / Hiddify',
    icon: '🚀',
    iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/V2ray.png',
    desc: '通用 Base64',
    urlKey: 'token_url',
    scheme: '',
  },
]

// 通用订阅 URL（Shadowrocket / V2Ray / Hiddify 使用）
const subscriptionUrl = computed(() => subscription.value?.token_url || '')
// Clash 订阅 URL
const clashUrl = computed(() => subscription.value?.token_clash_url || '')

// 根据格式定义获取对应 URL
const getFormatUrl = (fmt: typeof formats[0]) => {
  if (!subscription.value) return ''
  return subscription.value[fmt.urlKey] || subscription.value.token_url || ''
}

const statusClass = computed(() => {
  if (!subscription.value) return 'inactive'
  if (remainingDays.value <= 0) return 'expired'
  if (remainingDays.value <= 7) return 'warning'
  return 'active'
})
const statusText = computed(() => {
  if (!subscription.value) return '未激活'
  if (remainingDays.value <= 0) return '已过期'
  if (remainingDays.value <= 7) return '即将到期'
  return '使用中'
})
const statusIcon = computed(() => {
  if (!subscription.value || remainingDays.value <= 0) return CloseCircle
  if (remainingDays.value <= 7) return AlertCircle
  return CheckmarkCircle
})

const remainingDays = computed(() => {
  if (!subscription.value) return 0
  const diff = new Date(subscription.value.expire_time).getTime() - Date.now()
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)))
})

const canConvert = computed(() => remainingDays.value > 0)

const formatDate = (dateStr: string) => {
  if (!dateStr) return 'N/A'
  return new Date(dateStr).toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'
  })
}
const formatDateTime = (dateStr: string) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}天前`
  return date.toLocaleDateString('zh-CN')
}
const copyToClipboard = async (text: string, label: string) => {
  if (!text) { message.warning('暂无可用订阅'); return }
  const ok = await clipboardCopy(text)
  ok ? message.success(`${label}已复制到剪贴板`) : message.error('复制失败，请手动复制')
}

const importFormat = (fmt: typeof formats[0]) => {
  const url = getFormatUrl(fmt)
  if (!url) { message.warning('暂无可用订阅'); return }

  // V2RayN / Hiddify / SingBox 等桌面客户端没有 URL Scheme，直接复制
  if (!fmt.scheme) {
    copyToClipboard(url, fmt.name)
    message.info(`${fmt.name} 请在客户端中手动添加订阅地址`)
    return
  }

  try {
    // 检测浏览器系统并添加到订阅 URL
    const os = detectBrowserOS()
    const urlWithOS = url.includes('?') ? `${url}&browser_os=${os}` : `${url}?browser_os=${os}`
    const fullUrl = fmt.scheme + encodeURIComponent(urlWithOS)
    window.location.href = fullUrl
    message.success('正在打开客户端...')
  } catch (e) {
    message.error('打开失败，请确保已安装客户端')
    copyToClipboard(url, fmt.name)
  }
}

const detectBrowserOS = (): string => {
  const ua = navigator.userAgent.toLowerCase()
  if (ua.includes('mac os x') || ua.includes('macintosh')) return 'macos'
  if (ua.includes('windows')) return 'windows'
  if (ua.includes('linux')) return 'linux'
  return 'unknown'
}

const showQrCode = async (url: string, label: string) => {
  if (!url) { message.warning('暂无可用订阅'); return }
  const expiry = subscription.value?.expire_time ? formatDate(subscription.value.expire_time) : ''
  qrTitle.value = expiry ? `${label} (到期: ${expiry})` : label
  showQrModal.value = true
  await nextTick()
  if (qrCanvas.value) {
    QRCode.toCanvas(qrCanvas.value, url, { width: 240, margin: 2 })
  }
}

const getPaymentLabel = (payType: string) => {
  const labels: Record<string, string> = {
    epay: '在线支付', alipay: '支付宝', wxpay: '微信支付',
    qqpay: 'QQ支付', stripe: 'Stripe (国际卡)', crypto: '加密货币 (USDT)',
  }
  return labels[payType] || payType
}

const isQrCodeUrl = (url: string) => url.includes('qr.alipay.com') || (url.startsWith('https://qr.') && url.length < 200)

const fetchUserBalance = async () => {
  try { const res = await getDashboardInfo(); userBalance.value = res.data?.balance ?? 0 } catch {}
}
const loadPaymentMethods = async () => {
  try {
    const pmRes = await getPaymentMethods()
    const pmData = pmRes.data || {}
    paymentMethods.value = pmData.methods || []
    balanceEnabled.value = pmData.balance_enabled !== false
    if (!balanceEnabled.value && paymentMethods.value.length > 0) paymentMethod.value = 'pm_' + paymentMethods.value[0].id
  } catch {}
}
const buildUpgradeSuccessInfo = () => {
  upgradeSuccessInfo.value = {
    addDevices: upgradeAddDevices.value,
    extendMonths: upgradeExtendMonths.value,
    deviceLimit: subscription.value?.device_limit || 0,
    expireTime: subscription.value?.expire_time || '',
    amount: upgradeResult.value?.total || upgradeOrderInfo.value?.final_amount || upgradeOrderInfo.value?.amount || 0,
  }
  showUpgradeSuccess.value = true
}

const startPayPolling = (orderNo: string) => {
  stopPayPolling()
  let pollAttempts = 0
  const maxPollAttempts = 20
  payPollingStatus.value = true
  payPollTimer = setInterval(async () => {
    try {
      pollAttempts += 1
      const res = await getOrderStatus(orderNo)
      if (res.data?.status === 'paid') {
        stopPayPolling(); showPayQrModal.value = false; showCryptoModal.value = false
        await loadData()
        buildUpgradeSuccessInfo()
        message.success('支付成功，订阅已更新（设备数和到期时间已刷新）')
        return
      }
      if (pollAttempts >= maxPollAttempts) {
        stopPayPolling()
        message.warning('升级结果确认超时，请手动刷新当前订阅页面查看')
      }
    } catch {
      if (pollAttempts >= maxPollAttempts) {
        stopPayPolling()
        message.warning('升级结果确认超时，请手动刷新当前订阅页面查看')
      }
    }
  }, 3000)
}
const stopPayPolling = () => {
  payPollingStatus.value = false
  if (payPollTimer) { clearInterval(payPollTimer); payPollTimer = null }
}

const loadData = async () => {
  loading.value = true
  try {
    // Balance is shown at the top; fetch it regardless of subscription state
    const balanceP = fetchUserBalance()
    const pmP = loadPaymentMethods()

    const [subRes, devRes] = await Promise.all([
      getSubscription().catch((e: any) => {
        if (e?.response?.status === 404) return { data: null }
        throw e
      }),
      getSubscriptionDevices().catch((e: any) => {
        if (e?.response?.status === 404) return { data: [] }
        throw e
      }),
    ])
    subscription.value = subRes.data
    devices.value = devRes.data || []
    await Promise.all([balanceP, pmP])
  } catch (e: any) {
    if (e?.response?.status !== 404) message.error(getErrorMessage(e, '加载数据失败'))
  } finally { loading.value = false }
}

// Auto-show upgrade modal when devices near limit
watch(() => devices.value.length, (count) => {
  if (!subscription.value) return
  const limit = subscription.value.device_limit || 0
  if (limit > 0 && count >= limit - 1) {
    showUpgradeModal.value = true
    if (count >= limit) {
      message.warning('您的设备数量已达上限，建议升级套餐增加设备数')
    } else {
      message.info('您的设备数量即将达到上限，建议升级套餐')
    }
  }
})

const handleResetSubscription = async () => {
  try { await resetSubscription(); showResetModal.value = false; message.success('订阅地址已重置'); await loadData() }
  catch (e: any) { message.error(getErrorMessage(e, '重置订阅失败')) }
}
const handleConvertToBalance = async () => {
  try { await convertToBalance(); showConvertModal.value = false; message.success('转换成功'); await loadData() }
  catch (e: any) { message.error(getErrorMessage(e, '转换失败')) }
}
const handleSendEmail = async () => {
  sendingEmail.value = true
  try { await sendSubscriptionEmail(); message.success('订阅信息已发送到您的邮箱') }
  catch (e: any) { message.error(getErrorMessage(e, '发送失败')) }
  finally { sendingEmail.value = false }
}

const handleCalcUpgrade = async () => {
  upgradeCalcLoading.value = true; upgradeResult.value = null
  try {
    const res: any = await calcUpgradePrice({ add_devices: upgradeAddDevices.value, extend_months: upgradeExtendMonths.value || 0 })
    const d = res?.data ?? res
    if (d && typeof d.total === 'number') {
      upgradeResult.value = { fee_extend: d.fee_extend ?? 0, fee_new_devices: d.fee_new_devices ?? 0, total: d.total ?? 0 }
    }
  } catch (e: any) { message.error(getErrorMessage(e, '计算失败')) }
  finally { upgradeCalcLoading.value = false }
}
const handleOpenUpgradePay = async () => {
  if (!upgradeResult.value || upgradeResult.value.total <= 0) { message.warning('请先计算金额'); return }
  upgradeSubmitting.value = true
  try {
    const res: any = await createUpgradeOrder({ add_devices: upgradeAddDevices.value, extend_months: upgradeExtendMonths.value || 0 })
    upgradeOrderInfo.value = res.data
    useBalanceDeduct.value = false
    if (balanceEnabled.value) paymentMethod.value = 'balance'
    else if (paymentMethods.value.length > 0) paymentMethod.value = 'pm_' + paymentMethods.value[0].id
    showUpgradePayModal.value = true
  } catch (e: any) { message.error(getErrorMessage(e, '创建订单失败')) }
  finally { upgradeSubmitting.value = false }
}

const handleUpgradePay = async () => {
  if (!upgradeOrderInfo.value) return
  paying.value = true
  try {
    if (paymentMethod.value === 'balance') {
      await payOrder(upgradeOrderInfo.value.order_no, { payment_method: 'balance' })
      showUpgradePayModal.value = false
      await loadData()
      buildUpgradeSuccessInfo()
      message.success('支付成功，订阅已更新')
    } else if (paymentMethod.value.startsWith('pm_')) {
      const pmId = parseInt(paymentMethod.value.replace('pm_', ''))
      const paymentData: any = { order_id: upgradeOrderInfo.value.id, payment_method_id: pmId, is_mobile: isMobile.value }
      if (useBalanceDeduct.value && balanceDeductAmount.value > 0) {
        paymentData.use_balance = true; paymentData.balance_amount = balanceDeductAmount.value
      }
      const res = await createPayment(paymentData)
      const data = res.data
      if (data?.pay_type === 'crypto' && data?.crypto_info) {
        showUpgradePayModal.value = false; cryptoInfo.value = data.crypto_info
        cryptoOrderNo.value = data.order_no; showCryptoModal.value = true; startPayPolling(data.order_no); return
      }
      if (data?.payment_url) {
        showUpgradePayModal.value = false
        if (isMobile.value) {
          mobilePayUrl.value = data.payment_url; showPayQrModal.value = true; startPayPolling(upgradeOrderInfo.value.order_no)
        } else if (isQrCodeUrl(data.payment_url)) {
          showPayQrModal.value = true; await nextTick()
          if (payQrCanvas.value) QRCode.toCanvas(payQrCanvas.value, data.payment_url, { width: 240, margin: 2 })
          startPayPolling(upgradeOrderInfo.value.order_no)
        } else { safeRedirect(data.payment_url) }
      } else { message.info('支付已创建，请等待处理'); showUpgradePayModal.value = false; await loadData() }
    }
  } catch (e: any) { message.error(getErrorMessage(e, '支付失败')) }
  finally { paying.value = false }
}

const handleCryptoTransferred = () => {
  message.success('已记录，管理员确认到账后将为您开通服务')
  showCryptoModal.value = false; stopPayPolling(); loadData()
}

watch(showCryptoModal, async (val) => {
  if (val && cryptoInfo.value?.wallet_address) {
    await nextTick()
    if (cryptoQrCanvas.value) QRCode.toCanvas(cryptoQrCanvas.value, cryptoInfo.value.wallet_address, { width: 200, margin: 2 })
  }
})

onMounted(() => { loadData() })
onUnmounted(() => { stopPayPolling() })
</script>
<style scoped>
.subscription-page { padding: 16px; }
.empty-state { padding: 80px 0; text-align: center; }

/* Hero Card */
.hero-card { background: linear-gradient(135deg, #4a5fd7 0%, #7c3aed 100%); border-radius: 14px; overflow: hidden; }
.hero-card :deep(.n-card__content) { padding: 20px 24px; }
.hero-content { color: white; }
.hero-row-top { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.hero-left { display: flex; align-items: center; gap: 12px; }
.hero-right { flex-shrink: 0; }
.status-badge { display: inline-flex; align-items: center; gap: 5px; padding: 3px 10px; border-radius: 20px; font-size: 12px; font-weight: 600; }
.status-badge.active { background: rgba(24,160,88,0.4); }
.status-badge.warning { background: rgba(240,160,32,0.4); }
.status-badge.expired, .status-badge.inactive { background: rgba(224,48,80,0.4); }
.package-name { font-size: 18px; font-weight: 700; margin: 0; color: white; text-shadow: 0 1px 2px rgba(0,0,0,0.15); }
.hero-stats { display: flex; align-items: center; justify-content: center; gap: 24px; background: rgba(255,255,255,0.1); border-radius: 10px; padding: 14px 20px; }
.hero-stat { display: flex; flex-direction: column; align-items: center; min-width: 60px; }
.hero-stat-val { font-size: 22px; font-weight: 700; line-height: 1.2; }
.hero-stat-val.hero-stat-date { font-size: 14px; font-weight: 600; display: flex; align-items: center; gap: 4px; }
.hero-stat-label { font-size: 12px; opacity: 0.8; margin-top: 4px; }
.hero-stat-divider { width: 1px; height: 28px; background: rgba(255,255,255,0.25); }
.hero-actions { margin-top: 12px; display: flex; justify-content: flex-end; }
.hero-sub-action { background: rgba(255,255,255,0.15) !important; color: white !important; border: none !important; font-weight: 500 !important; font-size: 12px !important; }
.hero-sub-action:hover { background: rgba(255,255,255,0.3) !important; }
.hero-sub-action:disabled { background: rgba(255,255,255,0.08) !important; color: rgba(255,255,255,0.4) !important; }
/* Hero upgrade button */
.hero-upgrade-btn { background: rgba(255,255,255,0.95) !important; color: #4a5fd7 !important; border: none !important; font-weight: 600 !important; }
.hero-upgrade-btn:hover { background: #fff !important; }

/* Modern Hero */
.modern-hero { background: linear-gradient(135deg, #a8b5ff 0%, #c4a3e8 100%); border-radius: 16px; padding: 24px; color: #333; margin-bottom: 16px; }
.hero-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px; }
.hero-title-section { flex: 1; }
.package-title { font-size: 24px; font-weight: 700; margin: 8px 0 0 0; color: #333; text-shadow: 0 1px 2px rgba(255,255,255,0.5); }
.stats-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; }
.stat-item { background: rgba(255,255,255,0.6); border-radius: 12px; padding: 16px; display: flex; align-items: center; gap: 12px; backdrop-filter: blur(10px); border: 1px solid rgba(255,255,255,0.8); }
.stat-icon { width: 40px; height: 40px; border-radius: 10px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; color: white; }
.stat-icon.balance-icon { background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); }
.stat-icon.days-icon { background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%); }
.stat-icon.device-icon { background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%); }
.stat-icon.expire-icon { background: linear-gradient(135deg, #fa709a 0%, #fee140 100%); }
.stat-content { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.stat-label { font-size: 12px; opacity: 0.8; color: #555; }
.stat-value { font-size: 18px; font-weight: 700; line-height: 1.2; color: #333; }

/* Modern Hero Status Badge */
.modern-hero .status-badge { display: inline-flex; align-items: center; gap: 5px; padding: 4px 12px; border-radius: 20px; font-size: 12px; font-weight: 600; }
.modern-hero .status-badge.active { background: rgba(24,160,88,0.85); color: white; }
.modern-hero .status-badge.warning { background: rgba(240,160,32,0.85); color: white; }
.modern-hero .status-badge.expired, .modern-hero .status-badge.inactive { background: rgba(224,48,80,0.85); color: white; }

/* Upgrade Result */
.upgrade-result { margin-top: 12px; }

/* Two Column Layout */
.two-col { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-top: 16px; }
.compact-card :deep(.n-card__content) { padding: 12px 16px; }
.compact-card :deep(.n-card-header) { padding: 12px 16px 8px; }
.section-title { font-weight: 600; font-size: 14px; }

/* URL Card */
.url-card { background: white; border-radius: 12px; padding: 20px; margin-bottom: 16px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
.url-card-title { font-size: 16px; font-weight: 600; margin-bottom: 16px; color: #333; }
.url-item { margin-bottom: 16px; }
.url-item:last-child { margin-bottom: 0; }
.url-item-label { font-size: 13px; font-weight: 500; color: #666; margin-bottom: 8px; display: block; }
.url-item-actions { display: flex; gap: 8px; }
.url-item-actions .n-input { flex: 1; }

/* Format Card Container */
.format-card-container { background: white; border-radius: 12px; padding: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
.format-card-container .card-header { display: flex; flex-direction: column; gap: 4px; margin-bottom: 16px; }
.format-card-container .card-title { font-size: 16px; font-weight: 600; color: #333; margin: 0; }
.format-card-container .card-subtitle { font-size: 13px; color: #999; }

/* URL Section (legacy) */
.url-list { display: flex; flex-direction: column; gap: 10px; }
.url-header { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-bottom: 8px; }
.url-type { font-size: 14px; font-weight: 600; color: #333; }
.url-actions { display: flex; align-items: center; gap: 8px; }
.url-input { flex: 1; min-width: 0; }
.url-buttons { display: flex; gap: 8px; flex-wrap: wrap; }
.url-row { display: flex; align-items: center; gap: 8px; }
.url-label { font-size: 13px; font-weight: 500; color: #666; white-space: nowrap; min-width: 70px; }
.url-input-wrapper { display: flex; align-items: center; gap: 6px; flex: 1; }
.url-input { flex: 1; }

/* Format Grid */
.format-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; }
.format-card { padding: 10px 8px; border-radius: 8px; border: 1.5px solid #e8e8e8; cursor: pointer; text-align: center; transition: all 0.2s; }
.format-card:hover { border-color: #667eea; }
.format-card.active { border-color: #667eea; background: #667eea08; }
.format-icon { font-size: 20px; display: flex; justify-content: center; align-items: center; height: 24px; }
.format-icon-img { width: 22px; height: 22px; object-fit: contain; border-radius: 5px; }
.format-name { font-size: 12px; font-weight: 600; margin-top: 2px; }

/* Format Item */
.format-item { padding: 14px 12px; border-radius: 10px; border: 2px solid #e8e8e8; cursor: pointer; transition: all 0.2s; background: white; display: flex; flex-direction: column; align-items: center; gap: 8px; }
.format-item:hover { border-color: #667eea; transform: translateY(-2px); box-shadow: 0 4px 12px rgba(102,126,234,0.15); }
.format-item.active { border-color: #667eea; background: linear-gradient(135deg, #667eea08 0%, #764ba208 100%); }
.format-item-icon { display: flex; justify-content: center; align-items: center; width: 48px; height: 48px; }
.format-item-icon img { width: 48px; height: 48px; object-fit: contain; border-radius: 8px; }
.format-item-name { font-size: 14px; font-weight: 600; color: #333; text-align: center; }
.format-item-actions { display: flex; gap: 6px; width: 100%; margin-top: 4px; }
.format-item-actions .n-button { flex: 1; }

/* Payment */
.payment-method { padding: 4px 0; }
.pm-label { font-size: 14px; font-weight: 500; margin-bottom: 8px; }

/* Device Card */
.device-card { background: white; border-radius: 12px; padding: 20px; margin-bottom: 16px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }

/* Mobile Device List */
.mobile-device-list { display: flex; flex-direction: column; gap: 10px; }
.mobile-device-item { background: #f8f9fa; border-radius: 10px; padding: 14px; }
.device-name { font-weight: 600; font-size: 14px; margin-bottom: 8px; color: #333; }
.device-info-row { display: flex; justify-content: space-between; align-items: center; font-size: 13px; padding: 4px 0; }
.device-label { color: #888; font-size: 12px; }
.device-value { color: #555; }
.empty-device-list { text-align: center; padding: 20px 0; color: #999; font-size: 14px; }

/* Mobile */
@media (max-width: 767px) {
  .subscription-page { padding: 0 12px; }
  .hero-row-top { flex-direction: column; gap: 10px; align-items: flex-start; }
  .hero-right { align-self: flex-end; }
  .hero-stats { flex-wrap: wrap; gap: 16px; padding: 12px 16px; }
  .hero-stat-val { font-size: 18px; }
  .package-name { font-size: 16px; }
  .two-col { grid-template-columns: 1fr; }
  .url-row { flex-direction: column; align-items: flex-start; }
  .url-input-wrapper { width: 100%; }
  .format-grid { grid-template-columns: repeat(2, 1fr); }

  /* Modern Hero Mobile */
  .modern-hero { padding: 16px; }
  .hero-header { flex-direction: column; gap: 12px; }
  .package-title { font-size: 20px; }
  .stats-grid { grid-template-columns: repeat(2, 1fr); gap: 10px; }
  .stat-item { padding: 12px; }
  .stat-icon { width: 36px; height: 36px; }
  .stat-value { font-size: 16px; }
  .url-card, .format-card-container { padding: 16px; }
  .format-grid { grid-template-columns: repeat(2, 1fr); }
  .url-header { align-items: flex-start; }
  .url-actions { flex-direction: column; align-items: stretch; }
  .url-buttons { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; width: 100%; }
  .url-buttons .n-button { width: 100%; }
  .drawer-section { gap: 12px !important; }
  .drawer-footer-actions { width: 100%; }
  .drawer-footer-actions :deep(.n-button) { flex: 1 1 calc(50% - 6px); min-width: 0; }
  .mobile-pay-panel,
  .desktop-pay-panel,
  .crypto-panel { text-align: center; }
  .upgrade-pay-section :deep(.n-descriptions),
  .upgrade-form :deep(.n-descriptions),
  .crypto-panel :deep(.n-descriptions) { font-size: 13px; }
}
</style>
