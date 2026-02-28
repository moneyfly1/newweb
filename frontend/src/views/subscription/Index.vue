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
        <!-- Hero Card: compact status + stats + upgrade inline -->
        <n-card class="hero-card" :bordered="false">
          <div class="hero-content">
            <div class="hero-top">
              <div class="status-section">
                <div class="status-badge" :class="statusClass">
                  <n-icon :size="18" :component="statusIcon" />
                  <span>{{ statusText }}</span>
                </div>
                <h2 class="package-name">{{ subscription.package_name || '当前套餐' }}</h2>
              </div>
              <div class="hero-stats">
                <div class="hero-stat">
                  <span class="hero-stat-val">¥{{ (userBalance ?? 0).toFixed(2) }}</span>
                  <span class="hero-stat-label">余额</span>
                </div>
                <div class="hero-stat-divider"></div>
                <div class="hero-stat">
                  <span class="hero-stat-val">{{ remainingDays }}</span>
                  <span class="hero-stat-label">剩余天数</span>
                  <n-button
                    class="hero-sub-action"
                    size="tiny"
                    :disabled="!canConvert"
                    @click="showConvertModal = true"
                  >
                    转换为余额
                  </n-button>
                </div>
                <div class="hero-stat-divider"></div>
                <div class="hero-stat">
                  <span class="hero-stat-val">{{ devices.length }}/{{ subscription.device_limit || 0 }}</span>
                  <span class="hero-stat-label">设备使用</span>
                </div>
                <div class="hero-stat-divider"></div>
                <div class="hero-stat">
                  <n-button size="small" class="hero-upgrade-btn" strong @click="showUpgradeModal = true">
                    <template #icon><n-icon :component="ArrowUpCircleOutline" /></template>
                    升级套餐
                  </n-button>
                </div>
              </div>
            </div>
            <div class="hero-meta">
              <span><n-icon :component="TimeOutline" :size="14" /> 到期：{{ formatDate(subscription.expire_time) }}</span>
            </div>
          </div>
        </n-card>

        <!-- Two Column Layout: URLs left, Formats right -->
        <div class="two-col">
          <!-- Left: Subscription URLs -->
          <n-card :bordered="false" class="section-card compact-card">
            <template #header><span class="section-title">订阅地址</span></template>
            <template #header-extra>
              <n-space :size="8" align="center">
                <n-button size="tiny" @click="handleSendEmail" :loading="sendingEmail">
                  <template #icon><n-icon :component="MailOutline" /></template>
                  发送到邮箱
                </n-button>
                <n-button size="tiny" type="warning" ghost @click="showResetModal = true" :disabled="!subscription">
                  <template #icon><n-icon :component="RefreshOutline" /></template>
                  重置订阅
                </n-button>
              </n-space>
            </template>
            <div class="url-list">
              <div class="url-row">
                <span class="url-label">通用订阅</span>
                <div class="url-input-wrapper">
                  <n-input :value="subscriptionUrl" readonly size="small" class="url-input" />
                  <n-button size="small" type="primary" @click="copyToClipboard(subscriptionUrl, '通用订阅地址')">
                    <template #icon><n-icon :component="CopyOutline" /></template>
                  </n-button>
                  <n-button size="small" @click="showQrCode(subscriptionUrl, '通用订阅')">
                    <template #icon><n-icon :component="QrCodeOutline" /></template>
                  </n-button>
                </div>
              </div>
              <div class="url-row">
                <span class="url-label">Clash 订阅</span>
                <div class="url-input-wrapper">
                  <n-input :value="clashUrl" readonly size="small" class="url-input" />
                  <n-button size="small" type="primary" @click="copyToClipboard(clashUrl, 'Clash 订阅地址')">
                    <template #icon><n-icon :component="CopyOutline" /></template>
                  </n-button>
                  <n-button size="small" @click="showQrCode(clashUrl, 'Clash 订阅')">
                    <template #icon><n-icon :component="QrCodeOutline" /></template>
                  </n-button>
                </div>
              </div>
            </div>
          </n-card>

          <!-- Right: Format Selector -->
          <n-card :bordered="false" class="section-card compact-card">
            <template #header><span class="section-title">选择订阅格式</span></template>
            <div class="format-grid">
              <div
                v-for="fmt in formats"
                :key="fmt.type"
                class="format-card"
                :class="{ active: selectedFormat === fmt.type }"
                @click="selectedFormat = fmt.type"
              >
                <div class="format-icon">
                  <img v-if="fmt.iconUrl" class="format-icon-img" :src="fmt.iconUrl" :alt="fmt.name" loading="lazy" />
                  <span v-else>{{ fmt.icon }}</span>
                </div>
                <div class="format-name">{{ fmt.name }}</div>
                <n-space :size="6" style="margin-top: 6px;">
                  <n-button size="tiny" type="primary" @click.stop="copyToClipboard(getFormatUrl(fmt.type), fmt.name)">复制</n-button>
                  <n-button size="tiny" @click.stop="importFormat(fmt)">导入</n-button>
                </n-space>
              </div>
            </div>
          </n-card>
        </div>
      </template>
    </n-spin>

    <!-- QR Code Modal -->
    <n-modal v-model:show="showQrModal" preset="card" :title="qrTitle" style="width: 340px; max-width: 92vw;" :bordered="false">
      <div style="text-align: center;">
        <canvas ref="qrCanvas" style="margin: 0 auto;"></canvas>
        <p style="margin-top: 12px; color: #999; font-size: 13px;">使用客户端扫描二维码导入订阅</p>
      </div>
    </n-modal>

    <!-- Reset Modal -->
    <n-modal v-model:show="showResetModal" preset="dialog" title="重置订阅地址"
      content="重置后原订阅地址将失效，所有设备需要重新配置。确定要继续吗？"
      positive-text="确定" negative-text="取消" @positive-click="handleResetSubscription" />

    <!-- Convert Modal -->
    <n-modal v-model:show="showConvertModal" preset="dialog" title="转换剩余天数"
      :content="`将剩余 ${remainingDays} 天转换为余额，转换后订阅将立即失效。确定要继续吗？`"
      positive-text="确定" negative-text="取消" @positive-click="handleConvertToBalance" />

    <!-- Upgrade Pay Modal -->
    <n-modal
      v-model:show="showUpgradePayModal"
      preset="card"
      title="确认支付 - 升级订阅"
      style="width: 520px; max-width: 92vw;"
      :bordered="false"
      :segmented="{ content: true }"
    >
      <n-space vertical :size="16">
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
        <n-space justify="end">
          <n-button @click="showUpgradePayModal = false">取消</n-button>
          <n-button type="primary" :loading="paying" @click="handleUpgradePay">确认支付</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Pay QR Modal -->
    <n-modal v-model:show="showPayQrModal" preset="card" title="扫码支付" style="width: 400px; max-width: 92vw;" :bordered="false" :mask-closable="false" @after-leave="stopPayPolling">
      <div v-if="isMobile" style="text-align: center;">
        <p style="margin-bottom: 16px; color: #666;">请点击下方按钮完成支付</p>
        <n-button type="primary" size="large" block tag="a" :href="mobilePayUrl" target="_blank">打开支付App付款</n-button>
        <n-spin v-if="payPollingStatus" size="small" style="margin-top: 8px;" />
      </div>
      <div v-else style="text-align: center;">
        <p style="margin-bottom: 16px; color: #666;">请使用支付宝扫描下方二维码完成支付</p>
        <canvas ref="payQrCanvas" style="margin: 0 auto;"></canvas>
        <n-spin v-if="payPollingStatus" size="small" style="margin-top: 8px;" />
      </div>
      <template #footer>
        <n-space justify="center"><n-button @click="showPayQrModal = false">取消支付</n-button></n-space>
      </template>
    </n-modal>

    <!-- Crypto Pay Modal -->
    <n-modal v-model:show="showCryptoModal" preset="card" title="加密货币支付" style="width: 480px; max-width: 92vw;" :bordered="false" :mask-closable="false" @after-leave="stopPayPolling">
      <div v-if="cryptoInfo" style="text-align: center;">
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
    </n-modal>

    <!-- Upgrade Modal -->
    <n-modal v-model:show="showUpgradeModal" preset="card" title="升级订阅" style="width: 480px; max-width: 92vw;" :bordered="false">
      <n-form label-placement="left" label-width="auto" :disabled="upgradeSubmitting" size="medium">
        <n-form-item label="增加设备数">
          <n-input-number v-model:value="upgradeAddDevices" :min="5" :max="50" :step="5" style="width: 100%;" />
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
        <n-space justify="end">
          <n-button @click="showUpgradeModal = false">取消</n-button>
          <n-button type="primary" :loading="upgradeCalcLoading" @click="handleCalcUpgrade">计算金额</n-button>
          <n-button v-if="upgradeResult && upgradeResult.total > 0" type="success" :loading="upgradeSubmitting" @click="handleOpenUpgradePay">去支付</n-button>
        </n-space>
      </template>
    </n-modal>
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
  ArrowUpCircleOutline
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
import { useRouter } from 'vue-router'

const message = useMessage()
const router = useRouter()

const subscription = ref<any>(null)
const devices = ref<any[]>([])
const loading = ref(false)
const showResetModal = ref(false)
const showConvertModal = ref(false)
const sendingEmail = ref(false)
const selectedFormat = ref('clash')
const showUpgradeModal = ref(false)

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

const formats = [
  {
    type: 'clash',
    name: 'Clash',
    icon: '\u2694\uFE0F',
    iconUrl: 'https://fastly.jsdelivr.net/gh/walkxcode/dashboard-icons@main/png/clash.png',
    desc: 'Clash 系列',
  },
  {
    type: 'v2ray',
    name: 'V2Ray',
    icon: '\uD83D\uDE80',
    iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/V2ray.png',
    desc: 'V2Ray 格式',
  },
  {
    type: 'surge',
    name: 'Surge',
    icon: '\uD83C\uDF0A',
    iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/surge.png',
    desc: 'Surge 客户端',
  },
  {
    type: 'shadowrocket',
    name: 'Shadowrocket',
    icon: '\uD83D\uDD35',
    iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/shadowrocket.png',
    desc: 'iOS 客户端',
  },
  {
    type: 'quantumult',
    name: 'Quantumult X',
    icon: '\uD83D\uDCA0',
    iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/quantumultx.png',
    desc: 'iOS 客户端',
  },
  {
    type: 'stash',
    name: 'Stash',
    icon: '\uD83D\uDCE6',
    iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/stash.png',
    desc: 'iOS Clash',
  },
]

const subscriptionUrl = computed(() => subscription.value?.universal_url || subscription.value?.subscription_url || '')
const clashUrl = computed(() => subscription.value?.clash_url || subscription.value?.subscription_url || '')

const getFormatUrl = (type: string) => {
  if (!subscription.value) return ''
  const base = subscription.value.subscription_url || subscription.value.universal_url || ''
  if (!base) return ''
  const sep = base.includes('?') ? '&' : '?'
  return `${base}${sep}format=${type}`
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
const copyToClipboard = async (text: string, label: string) => {
  if (!text) { message.warning('暂无可用订阅'); return }
  const ok = await clipboardCopy(text)
  ok ? message.success(`${label}已复制到剪贴板`) : message.error('复制失败，请手动复制')
}

const importFormat = (fmt: any) => {
  const url = getFormatUrl(fmt.type)
  if (!url) { message.warning('暂无可用订阅'); return }
  const schemeMap: Record<string, string> = {
    clash: 'clash://install-config?url=',
    stash: 'stash://install-config?url=',
    shadowrocket: 'sub://',
    surge: 'surge:///install-config?url=',
    quantumult: 'quantumult-x:///update-configuration?remote-resource=',
  }
  const scheme = schemeMap[fmt.type]
  if (scheme) {
    if (fmt.type === 'shadowrocket') {
      window.location.href = scheme + btoa(url)
    } else {
      window.location.href = scheme + encodeURIComponent(url)
    }
  } else {
    copyToClipboard(url, fmt.name)
  }
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
const startPayPolling = (orderNo: string) => {
  payPollingStatus.value = true
  payPollTimer = setInterval(async () => {
    try {
      const res = await getOrderStatus(orderNo)
      if (res.data?.status === 'paid') {
        stopPayPolling(); showPayQrModal.value = false; showCryptoModal.value = false
        message.success('支付成功，订阅已更新'); await loadData()
      }
    } catch {}
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
    if (e?.response?.status !== 404) message.error(e.message || '加载数据失败')
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
  try { await resetSubscription(); message.success('订阅地址已重置'); await loadData() }
  catch (e: any) { message.error(e.message || '重置订阅失败') }
}
const handleConvertToBalance = async () => {
  try { await convertToBalance(); message.success('转换成功'); await loadData() }
  catch (e: any) { message.error(e.message || '转换失败') }
}
const handleSendEmail = async () => {
  sendingEmail.value = true
  try { await sendSubscriptionEmail(); message.success('订阅信息已发送到您的邮箱') }
  catch (e: any) { message.error(e.message || '发送失败') }
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
  } catch (e: any) { message.error(e.message || '计算失败') }
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
  } catch (e: any) { message.error(e.message || '创建订单失败') }
  finally { upgradeSubmitting.value = false }
}

const handleUpgradePay = async () => {
  if (!upgradeOrderInfo.value) return
  paying.value = true
  try {
    if (paymentMethod.value === 'balance') {
      await payOrder(upgradeOrderInfo.value.order_no, { payment_method: 'balance' })
      message.success('支付成功，订阅已更新'); showUpgradePayModal.value = false; await loadData()
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
  } catch (e: any) { message.error(e.message || '支付失败') }
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
.hero-top { display: flex; justify-content: space-between; align-items: center; }
.status-section { display: flex; align-items: center; gap: 12px; }
.status-badge { display: inline-flex; align-items: center; gap: 6px; padding: 4px 12px; border-radius: 20px; font-size: 13px; font-weight: 600; }
.status-badge.active { background: rgba(24,160,88,0.4); }
.status-badge.warning { background: rgba(240,160,32,0.4); }
.status-badge.expired, .status-badge.inactive { background: rgba(224,48,80,0.4); }
.package-name { font-size: 18px; font-weight: 700; margin: 0; color: white; text-shadow: 0 1px 2px rgba(0,0,0,0.15); }
.hero-stats { display: flex; align-items: center; gap: 20px; }
.hero-stat { display: flex; flex-direction: column; align-items: center; }
.hero-stat-val { font-size: 24px; font-weight: 700; }
.hero-stat-label { font-size: 12px; opacity: 0.9; margin-top: 2px; }
.hero-sub-action { margin-top: 6px; background: rgba(255,255,255,0.2) !important; color: white !important; border: none !important; font-weight: 500 !important; }
.hero-sub-action:hover { background: rgba(255,255,255,0.35) !important; }
.hero-sub-action:disabled { background: rgba(255,255,255,0.1) !important; color: rgba(255,255,255,0.5) !important; }
.hero-sub-action:deep(.n-button__content) { font-size: 12px; }
.hero-stat-divider { width: 1px; height: 32px; background: rgba(255,255,255,0.3); }
.hero-meta { margin-top: 10px; font-size: 13px; opacity: 0.9; display: flex; align-items: center; gap: 4px; }
/* Hero upgrade button */
.hero-upgrade-btn { background: rgba(255,255,255,0.95) !important; color: #4a5fd7 !important; border: none !important; font-weight: 600 !important; }
.hero-upgrade-btn:hover { background: #fff !important; }

/* Upgrade Result */
.upgrade-result { margin-top: 12px; }

/* Two Column Layout */
.two-col { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-top: 16px; }
.compact-card :deep(.n-card__content) { padding: 12px 16px; }
.compact-card :deep(.n-card-header) { padding: 12px 16px 8px; }
.section-title { font-weight: 600; font-size: 14px; }
/* URL Section */
.url-list { display: flex; flex-direction: column; gap: 10px; }
.url-row { display: flex; align-items: center; gap: 8px; }
.url-label { font-size: 13px; font-weight: 500; color: #666; white-space: nowrap; min-width: 70px; }
.url-input-wrapper { display: flex; align-items: center; gap: 6px; flex: 1; }
.url-input { flex: 1; }

/* Format Grid */
.format-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; }
.format-card { padding: 10px 8px; border-radius: 8px; border: 1.5px solid #e8e8e8; cursor: pointer; text-align: center; transition: all 0.2s; }
.format-card:hover { border-color: #667eea; }
.format-card.active { border-color: #667eea; background: #667eea08; }
.format-icon { font-size: 20px; display: flex; justify-content: center; align-items: center; height: 24px; }
.format-icon-img { width: 22px; height: 22px; object-fit: contain; border-radius: 5px; }
.format-name { font-size: 12px; font-weight: 600; margin-top: 2px; }

/* Payment */
.payment-method { padding: 4px 0; }
.pm-label { font-size: 14px; font-weight: 500; margin-bottom: 8px; }

/* Mobile */
@media (max-width: 767px) {
  .subscription-page { padding: 0 12px; }
  .hero-top { flex-direction: column; gap: 12px; }
  .hero-stats { gap: 12px; }
  .hero-stat-val { font-size: 20px; }
  .package-name { font-size: 16px; }
  .two-col { grid-template-columns: 1fr; }
  .url-row { flex-direction: column; align-items: flex-start; }
  .url-input-wrapper { width: 100%; }
  .format-grid { grid-template-columns: repeat(2, 1fr); }
  .action-row { flex-direction: column; }
  .action-row .n-button { width: 100%; }
}
</style>
