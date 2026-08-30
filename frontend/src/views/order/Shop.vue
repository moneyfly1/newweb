<template>
  <div class="shop-container">
    <n-space vertical :size="24">
      <div class="header">
        <h1 class="title">套餐商城</h1>
        <p class="subtitle">选择适合您的订阅套餐</p>
        <div v-if="userBalance !== null" class="balance-info">
          <span>账户余额：</span>
          <span class="balance-amount">{{ formatCurrency(userBalance) }}</span>
        </div>
      </div>

      <n-spin :show="loading">
        <div class="packages-grid">
          <div v-for="pkg in packages" :key="pkg.id">
            <div
              class="package-card"
              :class="{ featured: pkg.is_featured }"
              @click="handleBuy(pkg)"
            >
              <div v-if="pkg.is_featured" class="badge">推荐</div>

              <div class="card-header">
                <h3 class="package-name">{{ pkg.name }}</h3>
                <div class="price-section">
                  <span class="currency">¥</span>
                  <span class="price">{{ pkg.price }}</span>
                </div>
              </div>

              <div class="card-body">
                <n-space vertical :size="12">
                  <div class="feature-item">
                    <n-icon :component="TimeOutline" :size="18" />
                    <span>有效期：{{ pkg.duration_days }} 天</span>
                  </div>
                  <div class="feature-item">
                    <n-icon :component="PhonePortraitOutline" :size="18" />
                    <span>设备数：{{ pkg.device_limit }} 台</span>
                  </div>
                  <div v-if="parseFeatures(pkg.features).length" class="features-list">
                    <div v-for="(f, i) in parseFeatures(pkg.features)" :key="i" class="feature-item feature-extra">
                      <n-icon :component="CheckmarkCircleOutline" :size="16" />
                      <span>{{ f }}</span>
                    </div>
                  </div>
                  <div v-if="pkg.description" class="description">{{ pkg.description }}</div>
                </n-space>
              </div>

              <div class="card-footer">
                <n-button type="primary" size="large" block strong>立即购买</n-button>
              </div>
            </div>
          </div>

          <!-- Custom Package as a grid item -->
          <div v-if="customEnabled" class="custom-package-card-wrap">
            <div class="package-card custom-card">
              <div class="card-header">
                <h3 class="package-name custom-name">自定义套餐</h3>
                <p class="custom-card-desc">自由选择设备数量和时长</p>
              </div>
              <div class="card-body">
                <div class="custom-inline-form">
                  <div class="custom-inline-row">
                    <span class="custom-inline-label">设备</span>
                    <n-input-number v-model:value="customDevices" :min="customMinDevices" :max="customMaxDevices" size="small" style="width: 100%" />
                  </div>
                  <div class="custom-inline-row">
                    <span class="custom-inline-label">时长</span>
                    <n-select v-model:value="customMonths" :options="customMonthOptions" size="small" style="width: 100%" />
                  </div>
                  <div v-if="customDiscountPercent > 0" class="custom-inline-discount">
                    省 {{ customDiscountPercent }}%
                  </div>
                </div>
                <div class="custom-inline-price">
                  <span class="currency">¥</span>
                  <span class="price">{{ customFinalPrice.toFixed(0) }}</span>
                </div>
              </div>
              <div class="card-footer">
                <n-button type="primary" size="large" block strong :loading="customOrdering" @click.stop="handleCustomBuy">
                  立即购买
                </n-button>
              </div>
            </div>
          </div>
        </div>
      </n-spin>

      <!-- 购买须知：信任合规说明（一次性展示） -->
      <n-alert type="info" :bordered="false" class="trust-block">
        <div class="trust-title">购买须知</div>
        <div class="trust-text">支付即代表同意《服务条款》与《隐私政策》。</div>
        <div class="trust-text">退款政策：购买后 7 天内未激活可联系客服申请退款，已激活订单不支持退款。</div>
        <div class="trust-text" v-if="supportContactText">{{ supportContactText }}</div>
        <n-space :size="12" wrap class="trust-links">
          <n-button text type="primary" size="small" @click="router.push('/terms')">《服务条款》</n-button>
          <n-button text type="primary" size="small" @click="router.push('/privacy')">《隐私政策》</n-button>
          <n-button text type="primary" size="small" @click="router.push('/help')">联系客服</n-button>
        </n-space>
      </n-alert>
    </n-space>

    <!-- Purchase Drawer -->
    <common-drawer
      v-model:show="showPaymentModal"
      title="确认购买"
      :width="520"
      show-footer
      :loading="paying"
      @confirm="handlePay"
      @cancel="showPaymentModal = false"
    >
      <n-space vertical :size="16" class="purchase-drawer-content">
        <n-descriptions :column="1" bordered>
          <n-descriptions-item label="套餐名称">{{ selectedPackage?.name }}</n-descriptions-item>
          <n-descriptions-item label="有效期">{{ selectedPackage?.duration_days }} 天</n-descriptions-item>
          <n-descriptions-item label="原价">¥{{ orderInfo?.amount }}</n-descriptions-item>
          <n-descriptions-item v-if="couponInfo" label="优惠">
            <span style="color: var(--danger-color);">-{{ formatCurrency(orderInfo?.amount - orderInfo?.final_amount) }}</span>
          </n-descriptions-item>
          <n-descriptions-item label="账户余额">
            <span :style="{ color: userBalance >= (orderInfo?.final_amount || 0) ? 'var(--success-color)' : 'var(--danger-color)' }">
              {{ formatCurrency(userBalance) }}
            </span>
          </n-descriptions-item>
          <n-descriptions-item label="实付金额">
            <span style="color: var(--success-color); font-size: 20px; font-weight: bold;">¥{{ orderInfo?.final_amount }}</span>
          </n-descriptions-item>
          <n-descriptions-item v-if="useBalanceDeduct && paymentMethod !== 'balance'" label="余额抵扣">
            <span style="color: var(--success-color);">-{{ formatCurrency(balanceDeductAmount) }}</span>
          </n-descriptions-item>
          <n-descriptions-item v-if="useBalanceDeduct && paymentMethod !== 'balance'" label="还需支付">
            <span style="color: var(--danger-color); font-size: 18px; font-weight: bold;">{{ formatCurrency(remainingAmount) }}</span>
          </n-descriptions-item>
        </n-descriptions>

        <!-- Coupon Input -->
        <div class="modal-coupon">
          <div class="coupon-group">
            <n-input v-model:value="couponCode" placeholder="输入优惠码（可选）" :disabled="verifying" size="small" />
            <n-button class="coupon-verify-btn" type="primary" size="small" :loading="verifying" @click="handleVerifyCoupon" ghost>验证</n-button>
          </div>
          <n-alert v-if="couponInfo" type="success" :bordered="false" style="margin-top: 8px;" size="small">
            优惠码有效：{{ couponInfo.description }}
          </n-alert>
        </div>

        <!-- Payment Method -->
        <div class="payment-method">
          <div class="pm-label">支付方式</div>
          <div class="pm-card-grid">
            <div
              v-if="balanceEnabled"
              class="pm-card"
              :class="{ selected: paymentMethod === 'balance', disabled: userBalance <= 0 }"
              :style="{ '--pm-brand': pmMeta('balance').brand }"
              role="radio"
              :aria-checked="paymentMethod === 'balance'"
              tabindex="0"
              @click="handleSelectPayment('balance')"
              @keydown.enter="handleSelectPayment('balance')"
            >
              <div class="pm-card-icon">{{ pmMeta('balance').icon }}</div>
              <div class="pm-card-body">
                <span class="pm-card-name">{{ pmMeta('balance').label }}</span>
                <span class="pm-card-desc">
                  余额 {{ formatCurrency(userBalance) }}
                  <span v-if="!canFullBalance && userBalance > 0" class="pm-insufficient">· 余额不足</span>
                </span>
              </div>
              <span class="pm-check"></span>
            </div>
            <div
              v-for="pm in paymentMethods"
              :key="pm.id"
              class="pm-card"
              :class="{ selected: paymentMethod === 'pm_' + pm.id }"
              :style="{ '--pm-brand': pmMeta(pm.pay_type).brand }"
              role="radio"
              :aria-checked="paymentMethod === 'pm_' + pm.id"
              tabindex="0"
              @click="handleSelectPayment('pm_' + pm.id)"
              @keydown.enter="handleSelectPayment('pm_' + pm.id)"
            >
              <div class="pm-card-icon">{{ pmMeta(pm.pay_type).icon }}</div>
              <div class="pm-card-body">
                <span class="pm-card-name">{{ pmMeta(pm.pay_type).label }}</span>
                <span class="pm-card-desc">{{ pmMeta(pm.pay_type).desc }}</span>
              </div>
              <span class="pm-check"></span>
            </div>
          </div>
          <div v-if="paymentMethod !== 'balance' && userBalance > 0 && balanceEnabled" style="margin-top: 8px;">
            <n-checkbox v-model:checked="useBalanceDeduct">
              使用余额抵扣 {{ formatCurrency(Math.min(userBalance, finalPayAmount)) }}
            </n-checkbox>
            <div v-if="useBalanceDeduct" style="margin-top: 4px; font-size: 13px; color: var(--text-color-secondary);">
              余额抵扣：{{ formatCurrency(balanceDeductAmount) }}，还需支付：<span style="color: var(--danger-color); font-weight: 600;">{{ formatCurrency(remainingAmount) }}</span>
            </div>
          </div>
        </div>
      </n-space>
    </common-drawer>

    <!-- QR Code Payment Drawer -->
    <common-drawer
      v-model:show="showQrModal"
      title="扫码支付"
      :width="400"
      :mask-closable="false"
      show-footer
      :show-confirm="false"
      cancel-text="取消支付"
      @cancel="showQrModal = false"
      @after-leave="stopPolling"
    >
      <div v-if="isMobile" class="mobile-pay-panel">
        <p style="margin-bottom: 16px; color: var(--text-color-secondary);">请点击下方按钮完成支付</p>
        <n-button type="primary" size="large" block tag="a" :href="mobilePayUrl" target="_blank">
          打开支付App付款
        </n-button>
        <p style="margin-top: 16px; color: var(--text-color-secondary); font-size: 13px;">支付完成后请返回此页面</p>
        <n-spin v-if="pollingStatus" size="small" style="margin-top: 8px;" />
      </div>
      <div v-else class="desktop-pay-panel">
        <p style="margin-bottom: 16px; color: var(--text-color-secondary);">请使用支付宝扫描下方二维码完成支付</p>
        <canvas ref="qrCanvas" style="margin: 0 auto;"></canvas>
        <p style="margin-top: 16px; color: var(--text-color-secondary); font-size: 13px;">支付成功后系统会自动确认并跳转，若未更新请前往订单页手动刷新</p>
        <n-spin v-if="pollingStatus" size="small" style="margin-top: 8px;" />
      </div>
    </common-drawer>

    <!-- Crypto Payment Drawer -->
    <common-drawer
      v-model:show="showCryptoModal"
      title="加密货币支付"
      :width="480"
      :mask-closable="false"
      show-footer
      confirm-text="我已转账"
      @confirm="handleCryptoTransferred"
      @cancel="showCryptoModal = false"
      @after-leave="stopPolling"
    >
      <div v-if="cryptoInfo" class="crypto-panel">
        <p style="margin-bottom: 16px; color: var(--text-color-secondary);">请转账以下金额到指定钱包地址</p>
        <n-descriptions :column="1" bordered size="small" style="text-align: left;">
          <n-descriptions-item label="网络">{{ cryptoInfo.network }}</n-descriptions-item>
          <n-descriptions-item label="币种">{{ cryptoInfo.currency }}</n-descriptions-item>
          <n-descriptions-item label="转账金额">
            <span style="color: var(--danger-color); font-size: 18px; font-weight: bold;">{{ cryptoInfo.amount_usdt }} {{ cryptoInfo.currency }}</span>
          </n-descriptions-item>
          <n-descriptions-item label="收款地址">
            <div style="word-break: break-all; font-family: monospace; font-size: 13px;">{{ cryptoInfo.wallet_address }}</div>
          </n-descriptions-item>
        </n-descriptions>
        <div style="margin-top: 16px;">
          <canvas ref="cryptoQrCanvas" style="margin: 0 auto;"></canvas>
        </div>
        <n-alert type="warning" :bordered="false" style="margin-top: 12px; text-align: left;" size="small">
          请务必确认网络和币种正确，转账错误无法找回。转账完成后请点击下方按钮，管理员将在确认到账后为您开通服务。
        </n-alert>
        <n-spin v-if="pollingStatus" size="small" style="margin-top: 8px;" />
      </div>
    </common-drawer>

    <!-- CodePay Page Payment Drawer -->
    <common-drawer
      v-model:show="showCodepayModal"
      title="码支付"
      :width="500"
      :mask-closable="false"
      show-footer
      :show-confirm="false"
      cancel-text="取消支付"
      @cancel="showCodepayModal = false"
      @after-leave="stopPolling"
    >
      <div class="codepay-window-container" style="text-align: center; padding: 24px 0;">
        <p style="margin-bottom: 20px; color: var(--text-color-secondary); font-size: 15px;">请在新打开的页面中完成支付</p>
        <n-button type="primary" size="large" @click="openCodepayWindow">
          打开支付页面
        </n-button>
        <p style="margin-top: 20px; color: var(--text-color-secondary); font-size: 13px;">
          如果页面被浏览器拦截，请允许弹出窗口
        </p>
        <p style="margin-top: 16px; color: var(--text-color-secondary); font-size: 13px;">
          支付完成后系统会自动确认并跳转，若未更新请前往订单页手动刷新
        </p>
        <n-spin v-if="pollingStatus" size="small" style="margin-top: 12px;" />
      </div>
    </common-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, onUnmounted, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import QRCode from 'qrcode'
import {
  TimeOutline, PhonePortraitOutline, CheckmarkCircleOutline
} from '@vicons/ionicons5'
import { listPackages, verifyCoupon, getPaymentMethods, getPublicConfig } from '@/api/common'
import { createOrder, payOrder, createPayment, getOrderStatus, createCustomOrder } from '@/api/order'
import { getDashboardInfo } from '@/api/user'
import { safeRedirect } from '@/utils/security'
import { formatCurrency } from '@/utils/amount'
import { getErrorMessage, silentCatch } from '@/utils/error'
import CommonDrawer from '@/components/CommonDrawer.vue'

const router = useRouter()
const message = useMessage()

const loading = ref(false)
const packages = ref<any[]>([])
const publicConfig = ref<Record<string, string>>({})
const couponCode = ref('')
const verifying = ref(false)
const couponInfo = ref<any>(null)
const showPaymentModal = ref(false)
const selectedPackage = ref<any>(null)
const orderInfo = ref<any>(null)
const paying = ref(false)
const paymentMethod = ref('balance')
const paymentMethods = ref<any[]>([])
const balanceEnabled = ref(true)
const showQrModal = ref(false)
const qrCanvas = ref<HTMLCanvasElement | null>(null)
const cryptoQrCanvas = ref<HTMLCanvasElement | null>(null)
const pollingStatus = ref(false)
const buyingId = ref<number | null>(null)
const userBalance = ref<number>(0)
const useBalanceDeduct = ref(false)
const isMobile = ref(window.innerWidth <= 767)
const mobilePayUrl = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null
let pollAttempts = 0
const maxPollAttempts = 20

// Custom package
const customEnabled = ref(false)
const customPricePerDeviceYear = ref(40)
const customMinDevices = ref(1)
const customMaxDevices = ref(20)
const customMinMonths = ref(6)
const customDiscountTiers = ref<{ months: number; discount: number }[]>([])
const customDevices = ref(5)
const customMonths = ref(12)
const customCouponCode = ref('')
const customOrdering = ref(false)

const customBasePrice = computed(() => {
  return Math.round(customPricePerDeviceYear.value * customDevices.value * (customMonths.value / 12) * 100) / 100
})
const customDiscountPercent = computed(() => {
  let best = 0
  for (const tier of customDiscountTiers.value) {
    if (customMonths.value >= tier.months && tier.discount > best) best = tier.discount
  }
  return best
})
const customFinalPrice = computed(() => {
  return Math.round(customBasePrice.value * (1 - customDiscountPercent.value / 100) * 100) / 100
})
const customMonthOptions = computed(() => {
  return customDiscountTiers.value.map(tier => ({
    label: tier.discount > 0 ? `${tier.months}个月 (省${tier.discount}%)` : `${tier.months}个月`,
    value: tier.months
  }))
})

const finalPayAmount = computed(() => orderInfo.value?.final_amount || 0)
const canFullBalance = computed(() => userBalance.value >= finalPayAmount.value)
const balanceDeductAmount = computed(() => {
  if (paymentMethod.value === 'balance') return finalPayAmount.value
  if (useBalanceDeduct.value) return Math.min(userBalance.value, finalPayAmount.value)
  return 0
})
const remainingAmount = computed(() => {
  return Math.max(0, finalPayAmount.value - balanceDeductAmount.value)
})

// 客服入口（来自公共配置，缺失时隐藏）
const supportContactText = computed(() => {
  const parts: string[] = []
  const email = (publicConfig.value.support_email || '').trim()
  const qq = (publicConfig.value.support_qq || '').trim()
  const telegram = (publicConfig.value.support_telegram || '').trim()
  if (email) parts.push(`邮箱 ${email}`)
  if (qq) parts.push(`QQ ${qq}`)
  if (telegram) parts.push(`Telegram @${telegram.replace(/^@/, '')}`)
  return parts.length ? `客服入口：${parts.join(' · ')}` : ''
})


const loadPackages = async () => {
  loading.value = true
  try {
    const [pkgRes, pmRes, cfgRes] = await Promise.all([listPackages(), getPaymentMethods(), getPublicConfig()])
    packages.value = pkgRes.data || []
    const pmData = pmRes.data || {}
    paymentMethods.value = pmData.methods || []
    balanceEnabled.value = pmData.balance_enabled !== false
    // Auto-select first available method
    if (!balanceEnabled.value && paymentMethods.value.length > 0) {
      paymentMethod.value = 'pm_' + paymentMethods.value[0].id
    }
    // Custom package config
    const cfg = cfgRes.data || {}
    publicConfig.value = cfg
    customEnabled.value = cfg.custom_package_enabled === 'true' || cfg.custom_package_enabled === '1'
    if (cfg.custom_package_price_per_device_year) customPricePerDeviceYear.value = parseFloat(cfg.custom_package_price_per_device_year) || 40
    if (cfg.custom_package_min_devices) customMinDevices.value = parseInt(cfg.custom_package_min_devices) || 1
    if (cfg.custom_package_max_devices) customMaxDevices.value = parseInt(cfg.custom_package_max_devices) || 20
    if (cfg.custom_package_min_months) customMinMonths.value = parseInt(cfg.custom_package_min_months) || 6
    if (cfg.custom_package_duration_discounts) {
      try {
        customDiscountTiers.value = JSON.parse(cfg.custom_package_duration_discounts)
      } catch (e) {
        silentCatch(e, 'parse custom_package_duration_discounts')
      }
    }
    customDevices.value = Math.max(customMinDevices.value, Math.min(customDevices.value, customMaxDevices.value))
    customMonths.value = Math.max(customMinMonths.value, customMonths.value)
  } catch (e: any) {
    message.error(getErrorMessage(e, '加载套餐失败'))
  } finally { loading.value = false }
}

const fetchUserBalance = async () => {
  try {
    const res = await getDashboardInfo()
    userBalance.value = res.data?.balance || 0
  } catch (e) {
    silentCatch(e, 'fetchUserBalance')
  }
}

interface PmMeta { payType: string; label: string; brand: string; icon: string; desc: string }
// 支付方式品牌元数据：品牌色 + CSS 绘制图标字母 + 主文案（纯展示层，不改变支付逻辑）
const pmMetaList: PmMeta[] = [
  { payType: 'balance', label: '余额支付', brand: 'var(--primary-color)', icon: '余', desc: '账户余额直接抵扣' },
  { payType: 'alipay', label: '支付宝', brand: '#1677ff', icon: '支', desc: '扫码或跳转支付' },
  { payType: 'wxpay', label: '微信支付', brand: '#07c160', icon: '微', desc: '扫码或跳转支付' },
  { payType: 'qqpay', label: 'QQ 支付', brand: '#12b7f5', icon: 'Q', desc: '扫码或跳转支付' },
  { payType: 'stripe', label: 'Stripe', brand: '#635bff', icon: 'S', desc: '国际信用卡支付' },
  { payType: 'crypto', label: '加密货币 (USDT)', brand: '#f7931a', icon: '₮', desc: '链上转账' },
  { payType: 'codepay', label: '码支付', brand: '#7c3aed', icon: '码', desc: '扫码或跳转支付' },
  { payType: 'codepay_alipay', label: '码支付-支付宝', brand: '#7c3aed', icon: '码', desc: '扫码或跳转支付' },
  { payType: 'codepay_wxpay', label: '码支付-微信', brand: '#7c3aed', icon: '码', desc: '扫码或跳转支付' },
]
const pmMeta = (payType: string): PmMeta =>
  pmMetaList.find(m => m.payType === payType) || {
    payType, label: payType, brand: 'var(--primary-color)',
    icon: (payType[0] || '?').toUpperCase(), desc: '在线支付',
  }

const handleSelectPayment = (value: string) => {
  if (value === 'balance' && userBalance.value <= 0) {
    message.warning('余额不足，请先充值')
    return
  }
  paymentMethod.value = value
}

const isCodepayPayType = (payType?: string) => {
  return !!payType && (payType === 'codepay' || payType.startsWith('codepay_'))
}

const isCodepayPaymentMethod = () => {
  if (!paymentMethod.value.startsWith('pm_')) return false
  const methodId = parseInt(paymentMethod.value.replace('pm_', ''))
  const method = paymentMethods.value.find(pm => pm.id === methodId)
  return isCodepayPayType(method?.pay_type)
}

const parseFeatures = (features: any): string[] => {
  if (!features) return []
  if (Array.isArray(features)) return features
  try { return JSON.parse(features) } catch { return [] }
}

const handleVerifyCoupon = async () => {
  if (!couponCode.value.trim()) { message.warning('请输入优惠码'); return }
  verifying.value = true
  try {
    const res = await verifyCoupon({ code: couponCode.value, package_id: selectedPackage.value?.id || 0 })
    couponInfo.value = res.data
    message.success('优惠码验证成功')
    // Re-create order with coupon
    if (selectedPackage.value) {
      const payload: any = { package_id: selectedPackage.value.id }
      if (couponCode.value.trim()) payload.coupon_code = couponCode.value
      const orderRes = await createOrder(payload)
      orderInfo.value = orderRes.data
    }
  } catch (e: any) {
    message.error(getErrorMessage(e, '优惠码无效'))
    couponInfo.value = null
  } finally { verifying.value = false }
}

const handleCustomBuy = async () => {
  if (customDevices.value <= 0 || customMonths.value <= 0) {
    message.warning('设备数和月数必须大于0')
    return
  }
  customOrdering.value = true
  try {
    const payload: any = { devices: customDevices.value, months: customMonths.value }
    if (customCouponCode.value.trim()) payload.coupon_code = customCouponCode.value
    const res = await createCustomOrder(payload)
    orderInfo.value = res.data
    // 计算准确的天数：使用月份转换为天数的更准确方式
    // 1个月约30.44天，12个月约365天
    const durationDays = Math.round(customMonths.value * 30.44)
    selectedPackage.value = { name: `自定义套餐 (${customDevices.value}设备/${customMonths.value}月)`, duration_days: durationDays }
    showPaymentModal.value = true
  } catch (e: any) {
    message.error(getErrorMessage(e, '创建订单失败'))
  } finally { customOrdering.value = false }
}

const handleBuy = async (pkg: any) => {
  selectedPackage.value = pkg
  buyingId.value = pkg.id
  try {
    const payload: any = { package_id: pkg.id }
    if (couponCode.value.trim()) payload.coupon_code = couponCode.value
    const res = await createOrder(payload)
    orderInfo.value = res.data
    showPaymentModal.value = true
  } catch (e: any) {
    message.error(getErrorMessage(e, '创建订单失败'))
  } finally { buyingId.value = null }
}

const isQrCodeUrl = (url: string) => {
  // 支付宝二维码
  if (url.includes('qr.alipay.com')) return true
  // 通用二维码链接（短链接）
  if (url.startsWith('https://qr.') && url.length < 200) return true
  // 码支付二维码（通常是短链接或包含特定关键词）
  if (url.includes('qrcode') || url.includes('qr_code')) return true
  // 微信支付二维码
  if (url.includes('wxpay') && url.startsWith('weixin://')) return true
  // 其他常见二维码模式：短链接（长度小于100）且以 http 开头
  if ((url.startsWith('http://') || url.startsWith('https://')) && url.length < 100) return true
  return false
}

const isCodepayPageUrl = (url: string) => {
  // 码支付的submit.php页面（需要在iframe中显示）
  return url.includes('/xpay/epay/submit.php') || url.includes('/submit.php')
}

const goToPurchaseSuccess = (orderNo: string) => {
  router.push({ name: 'PaymentReturn', query: { order_no: orderNo, source: 'purchase', redirect: 'dashboard' } })
}

const startPolling = (orderNo: string) => {
  stopPolling()
  pollAttempts = 0
  pollingStatus.value = true
  // 递归 setTimeout 代替 setInterval：上一次请求完成后间隔 3s 再发起下一次，
  // 避免慢请求（最长 15s 超时）与 3s 定时器重叠导致并发轮询
  const pollOnce = async () => {
    if (!pollingStatus.value) return
    pollAttempts += 1
    try {
      const res = await getOrderStatus(orderNo)
      if (res.data?.status === 'paid') {
        stopPolling()
        showQrModal.value = false
        showCodepayModal.value = false
        await fetchUserBalance()
        goToPurchaseSuccess(orderNo)
        return
      }
      if (pollAttempts >= maxPollAttempts) {
        stopPolling()
        message.warning('支付结果确认超时，请前往订单页手动刷新查看')
        return
      }
    } catch {
      if (pollAttempts >= maxPollAttempts) {
        stopPolling()
        message.warning('支付结果确认超时，请前往订单页手动刷新查看')
        return
      }
    }
    if (pollingStatus.value) {
      pollTimer = setTimeout(pollOnce, 3000)
    }
  }
  pollTimer = setTimeout(pollOnce, 3000)
}

const stopPolling = () => {
  pollingStatus.value = false
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

const showCryptoModal = ref(false)
const cryptoInfo = ref<any>(null)
const cryptoOrderNo = ref('')
const showCodepayModal = ref(false)
const codepayUrl = ref('')

const openCodepayWindow = () => {
  if (codepayUrl.value) {
    window.open(codepayUrl.value, '_blank', 'width=800,height=700,scrollbars=yes,resizable=yes')
  }
}

const showCodepayPayment = async (payUrl: string, orderNo: string) => {
  codepayUrl.value = payUrl
  showCodepayModal.value = true
  await nextTick()
  openCodepayWindow()
  startPolling(orderNo)
}

const showQrPayment = async (payUrl: string, orderNo: string) => {
  if (isMobile.value) {
    mobilePayUrl.value = payUrl
    showQrModal.value = true
  } else {
    showQrModal.value = true
    await nextTick()
    if (qrCanvas.value) {
      QRCode.toCanvas(qrCanvas.value, payUrl, { width: 240, margin: 2 })
    }
  }
  startPolling(orderNo)
}

const handlePay = async () => {
  if (!orderInfo.value) return
  paying.value = true
  try {
    if (paymentMethod.value === 'balance') {
      await payOrder(orderInfo.value.order_no, { payment_method: 'balance' })
      showPaymentModal.value = false
      goToPurchaseSuccess(orderInfo.value.order_no)
    } else if (paymentMethod.value.startsWith('pm_')) {
      const pmId = parseInt(paymentMethod.value.replace('pm_', ''))
      const paymentData: any = { order_id: orderInfo.value.id, payment_method_id: pmId, is_mobile: isMobile.value }
      if (useBalanceDeduct.value && balanceDeductAmount.value > 0) {
        paymentData.use_balance = true
        paymentData.balance_amount = balanceDeductAmount.value
      }
      const res = await createPayment(paymentData)
      const data = res.data

      // Crypto payment: show wallet info modal
      if (data?.pay_type === 'crypto' && data?.crypto_info) {
        showPaymentModal.value = false
        cryptoInfo.value = data.crypto_info
        cryptoOrderNo.value = data.order_no
        showCryptoModal.value = true
        startPolling(data.order_no)
        return
      }

      if (data?.payment_url) {
        showPaymentModal.value = false
        const forceCodepayPopup = isCodepayPaymentMethod() || isCodepayPayType(data?.pay_type)

        if (forceCodepayPopup && (data?.payment_mode === 'qrcode' || isQrCodeUrl(data.payment_url))) {
          await showQrPayment(data.payment_url, orderInfo.value.order_no)
        } else if (forceCodepayPopup || data?.payment_mode === 'page') {
          await showCodepayPayment(data.payment_url, orderInfo.value.order_no)
        } else if (data?.payment_mode === 'qrcode') {
          await showQrPayment(data.payment_url, orderInfo.value.order_no)
        } else if (data?.payment_mode === 'redirect') {
          safeRedirect(data.payment_url)
        } else if (isCodepayPageUrl(data.payment_url)) {
          await showCodepayPayment(data.payment_url, orderInfo.value.order_no)
        } else if (isQrCodeUrl(data.payment_url)) {
          await showQrPayment(data.payment_url, orderInfo.value.order_no)
        } else {
          safeRedirect(data.payment_url)
        }
      } else {
        message.info('支付已创建，请等待处理')
        showPaymentModal.value = false
        router.push('/orders')
      }
    }
  } catch (e: any) {
    message.error(getErrorMessage(e, '支付失败'))
  } finally { paying.value = false }
}

// Render crypto wallet address as QR code when modal opens
watch(showCryptoModal, async (val) => {
  if (val && cryptoInfo.value?.wallet_address) {
    await nextTick()
    if (cryptoQrCanvas.value) {
      QRCode.toCanvas(cryptoQrCanvas.value, cryptoInfo.value.wallet_address, { width: 200, margin: 2 })
    }
  }
})

const handleCryptoTransferred = () => {
  message.success('已记录，管理员确认到账后将为您开通服务')
  showCryptoModal.value = false
  stopPolling()
  router.push('/orders')
}

onUnmounted(() => { stopPolling() })

onMounted(() => {
  loadPackages()
  fetchUserBalance()
})
</script>

<style scoped>
.shop-container { padding: 24px; }
.header { text-align: center; margin-bottom: 16px; }
.title {
  font-size: 32px; font-weight: 600; margin: 0 0 8px 0;
  background: var(--brand-gradient);
  -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text;
}
.subtitle { font-size: 16px; color: var(--text-color-secondary, #666); margin: 0; }

.balance-info {
  text-align: center; margin-top: 8px; font-size: 15px; color: var(--text-color-secondary, #666);
}
.balance-amount {
  color: var(--success-color); font-weight: 700; font-size: 18px;
}

.packages-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
}

.package-card {
  background: var(--bg-color, #fff); border-radius: 12px; padding: 24px;
  border: 2px solid var(--border-color, #e8e8e8); transition: border-color 0.3s ease, box-shadow 0.3s ease;
  cursor: pointer; position: relative; height: 100%;
  display: flex; flex-direction: column;
}
.package-card:hover { transform: translateY(-8px); box-shadow: 0 12px 24px rgba(0,0,0,0.1); border-color: var(--primary-color); }
.package-card.featured { border-color: var(--primary-color); border-width: 3px; background: linear-gradient(135deg, rgba(102,126,234,0.06) 0%, rgba(118,75,162,0.06) 100%); }
.badge {
  position: absolute; top: -12px; right: 24px;
  background: var(--brand-gradient);
  color: #fff; padding: 4px 16px; border-radius: 12px; font-size: 14px; font-weight: 600;
}
.card-header { text-align: center; margin-bottom: 24px; }
.package-name { font-size: 24px; font-weight: 600; margin: 0 0 16px 0; color: var(--text-color, #333); }
.price-section { display: flex; align-items: baseline; justify-content: center; }
.currency { font-size: 24px; color: var(--primary-color); font-weight: 600; }
.price { font-size: 48px; font-weight: 700; color: var(--primary-color); margin-left: 4px; }
.card-body { flex: 1; margin-bottom: 24px; }
.feature-item { display: flex; align-items: center; gap: 8px; color: var(--text-color-secondary, #666); font-size: 15px; }
.feature-item .n-icon { color: var(--primary-color); }
.feature-extra .n-icon { color: var(--success-color); }
.features-list { margin-top: 8px; padding-top: 8px; border-top: 1px dashed var(--border-color, #e8e8e8); }
.description {
  margin-top: 8px; padding: 12px; background: rgba(0,0,0,0.03);
  border-radius: 8px; color: var(--text-color-secondary, #666); font-size: 14px; line-height: 1.6;
}
.card-footer { margin-top: auto; }

/* Custom package card */
.custom-card { border-style: dashed; cursor: default; }
.custom-card:hover { transform: none; border-color: var(--primary-color); }
.custom-name { margin-bottom: 4px; }
.custom-card-desc { font-size: 13px; color: var(--text-color-secondary, #999); margin: 0; }
.custom-inline-form { display: flex; flex-direction: column; gap: 12px; }
.custom-inline-row { display: flex; align-items: center; gap: 8px; }
.custom-inline-label { font-size: 13px; color: var(--text-color-secondary, #666); min-width: 32px; flex-shrink: 0; }
.custom-inline-discount { text-align: center; font-size: 12px; color: var(--success-color); font-weight: 500; }
.custom-inline-price { display: flex; align-items: baseline; justify-content: center; margin-top: 12px; }

.modal-coupon { padding: 8px 0; }
.coupon-group { display: flex; gap: 8px; align-items: stretch; }
.coupon-group .n-input { flex: 1; min-width: 0; }
.coupon-verify-btn { flex-shrink: 0; }
.payment-method { padding: 4px 0; }
.pm-label { font-size: 14px; font-weight: 500; margin-bottom: 8px; color: var(--text-color, #333); }

/* 支付方式品牌卡片 */
.pm-card-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }
.pm-card {
  --pm-brand: var(--primary-color);
  position: relative;
  display: flex; align-items: center; gap: 10px;
  padding: 12px 14px;
  border: 2px solid var(--border-color, #e8e8e8);
  border-radius: 12px;
  background: var(--bg-color, #fff);
  cursor: pointer;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
  min-width: 0;
  outline: none;
}
.pm-card:hover { border-color: color-mix(in srgb, var(--pm-brand) 50%, var(--border-color)); box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06); }
.pm-card:focus-visible { box-shadow: 0 0 0 2px color-mix(in srgb, var(--pm-brand) 40%, transparent); }
.pm-card.selected {
  border-color: var(--pm-brand);
  background: color-mix(in srgb, var(--pm-brand) 8%, var(--bg-color, #fff));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--pm-brand) 25%, transparent);
}
.pm-card.disabled { opacity: 0.5; cursor: not-allowed; }
.pm-card.disabled:hover { border-color: var(--border-color); box-shadow: none; }
.pm-card-icon {
  width: 40px; height: 40px; border-radius: 10px; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
  background: var(--pm-brand);
  color: #fff; font-size: 18px; font-weight: 700;
}
.pm-card-body { display: flex; flex-direction: column; gap: 2px; min-width: 0; flex: 1; }
.pm-card-name { font-size: 14px; font-weight: 600; color: var(--text-color, #333); }
.pm-card-desc { font-size: 12px; color: var(--text-color-secondary, #666); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.pm-insufficient { color: var(--danger-color); font-weight: 600; }
.pm-check {
  flex-shrink: 0; width: 20px; height: 20px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-size: 12px; color: transparent;
  border: 2px solid var(--border-color, #ddd);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}
.pm-card.selected .pm-check {
  background: var(--pm-brand); border-color: var(--pm-brand); color: #fff;
}
.mobile-pay-panel,
.desktop-pay-panel,
.crypto-panel { text-align: center; }

/* 购买须知：信任合规说明 */
.trust-block { text-align: left; }
.trust-title { font-size: 14px; font-weight: 600; margin-bottom: 6px; color: var(--text-color, #333); }
.trust-text { font-size: 13px; line-height: 1.7; color: var(--text-color-secondary, #666); }
.trust-links { margin-top: 8px; }

/* Mobile Responsive */
@media (max-width: 767px) {
  .shop-container { padding: 12px; }
  .title { font-size: 24px; }
  .subtitle { font-size: 14px; }
  .packages-grid { grid-template-columns: repeat(2, 1fr); gap: 12px; }
  .package-card { min-height: 100%; padding: 16px 14px; border-radius: 16px; }
  
  .package-card:hover { transform: none; }
  .card-header { margin-bottom: 14px; }
  .package-name { font-size: 16px; line-height: 1.3; margin-bottom: 8px; word-break: break-word; }
  .price { font-size: 28px; }
  .currency { font-size: 16px; }
  .card-body { margin-bottom: 14px; }
  .feature-item { align-items: flex-start; font-size: 13px; line-height: 1.45; gap: 4px; }
  .feature-item .n-icon { margin-top: 1px; flex-shrink: 0; }
  .badge { top: -10px; right: 12px; font-size: 11px; padding: 2px 10px; }
  .description { padding: 8px; font-size: 12px; }
  .custom-inline-row { display: grid; grid-template-columns: 34px minmax(0, 1fr); }
  .purchase-drawer-content { gap: 12px !important; }
  .coupon-group { flex-direction: column; }
  .coupon-verify-btn { width: 100%; }
  .purchase-drawer-content :deep(.n-descriptions) { font-size: 13px; }
  .pm-card { padding: 10px 12px; border-radius: 10px; }
  .pm-card-icon { width: 36px; height: 36px; font-size: 16px; border-radius: 9px; }
  .pm-card-desc { white-space: normal; line-height: 1.3; }
}

@media (max-width: 400px) {
  .packages-grid { grid-template-columns: 1fr; }
  .pm-card-grid { grid-template-columns: 1fr; }
}
</style>
