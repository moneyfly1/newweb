<template>
  <div class="shop-container">
    <n-space vertical :size="24">
      <div class="header">
        <h1 class="title">套餐商城</h1>
        <p class="subtitle">选择适合您的订阅套餐</p>
        <div v-if="userBalance !== null" class="balance-info">
          <span>账户余额：</span>
          <span class="balance-amount">¥{{ userBalance.toFixed(2) }}</span>
        </div>
      </div>

      <n-spin :show="loading">
        <n-grid :cols="1" :x-gap="20" :y-gap="20" responsive="screen" :s="2" :l="3">
          <n-grid-item v-for="pkg in packages" :key="pkg.id">
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
          </n-grid-item>
        </n-grid>
      </n-spin>
    </n-space>

    <!-- Purchase Modal -->
    <n-modal
      v-model:show="showPaymentModal"
      preset="card"
      title="确认购买"
      style="width: 520px; max-width: 92vw;"
      :bordered="false"
      :segmented="{ content: true }"
    >
      <n-space vertical :size="16">
        <n-descriptions :column="1" bordered>
          <n-descriptions-item label="套餐名称">{{ selectedPackage?.name }}</n-descriptions-item>
          <n-descriptions-item label="有效期">{{ selectedPackage?.duration_days }} 天</n-descriptions-item>
          <n-descriptions-item label="原价">¥{{ orderInfo?.amount }}</n-descriptions-item>
          <n-descriptions-item v-if="couponInfo" label="优惠">
            <span style="color: #e03050;">-¥{{ (orderInfo?.amount - orderInfo?.final_amount).toFixed(2) }}</span>
          </n-descriptions-item>
          <n-descriptions-item label="账户余额">
            <span :style="{ color: userBalance >= (orderInfo?.final_amount || 0) ? '#18a058' : '#e03050' }">
              ¥{{ userBalance?.toFixed(2) }}
            </span>
          </n-descriptions-item>
          <n-descriptions-item label="实付金额">
            <span style="color: #18a058; font-size: 20px; font-weight: bold;">¥{{ orderInfo?.final_amount }}</span>
          </n-descriptions-item>
          <n-descriptions-item v-if="useBalanceDeduct && paymentMethod !== 'balance'" label="余额抵扣">
            <span style="color: #18a058;">-¥{{ balanceDeductAmount.toFixed(2) }}</span>
          </n-descriptions-item>
          <n-descriptions-item v-if="useBalanceDeduct && paymentMethod !== 'balance'" label="还需支付">
            <span style="color: #e03050; font-size: 18px; font-weight: bold;">¥{{ remainingAmount.toFixed(2) }}</span>
          </n-descriptions-item>
        </n-descriptions>

        <!-- Coupon Input in Modal -->
        <div class="modal-coupon">
          <n-input-group>
            <n-input v-model:value="couponCode" placeholder="输入优惠码（可选）" :disabled="verifying" size="small" />
            <n-button type="primary" size="small" :loading="verifying" @click="handleVerifyCoupon" ghost>验证</n-button>
          </n-input-group>
          <n-alert v-if="couponInfo" type="success" :bordered="false" style="margin-top: 8px;" size="small">
            优惠码有效：{{ couponInfo.description }}
          </n-alert>
        </div>

        <!-- Payment Method -->
        <div class="payment-method">
          <div class="pm-label">支付方式</div>
          <n-radio-group v-model:value="paymentMethod">
            <n-space vertical :size="8">
              <n-radio v-if="balanceEnabled" value="balance" :disabled="userBalance <= 0">
                余额支付 (¥{{ userBalance.toFixed(2) }})
                <span v-if="!canFullBalance && userBalance > 0" style="color: #e03050; font-size: 12px; margin-left: 4px;">余额不足</span>
              </n-radio>
              <n-radio v-for="pm in paymentMethods" :key="pm.id" :value="'pm_' + pm.id">
                {{ getPaymentLabel(pm.pay_type) }}
              </n-radio>
            </n-space>
          </n-radio-group>
          <div v-if="paymentMethod !== 'balance' && userBalance > 0 && balanceEnabled" style="margin-top: 8px;">
            <n-checkbox v-model:checked="useBalanceDeduct">
              使用余额抵扣 ¥{{ Math.min(userBalance, finalPayAmount).toFixed(2) }}
            </n-checkbox>
            <div v-if="useBalanceDeduct" style="margin-top: 4px; font-size: 13px; color: #666;">
              余额抵扣：¥{{ balanceDeductAmount.toFixed(2) }}，还需支付：<span style="color: #e03050; font-weight: 600;">¥{{ remainingAmount.toFixed(2) }}</span>
            </div>
          </div>
        </div>
      </n-space>

      <template #footer>
        <n-space justify="end">
          <n-button @click="showPaymentModal = false">取消</n-button>
          <n-button type="primary" :loading="paying" @click="handlePay">确认支付</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- QR Code Payment Modal -->
    <n-modal
      v-model:show="showQrModal"
      preset="card"
      title="扫码支付"
      style="width: 400px; max-width: 92vw;"
      :bordered="false"
      :mask-closable="false"
      @after-leave="stopPolling"
    >
      <div style="text-align: center;">
        <p style="margin-bottom: 16px; color: #666;">请使用支付宝扫描下方二维码完成支付</p>
        <canvas ref="qrCanvas" style="margin: 0 auto;"></canvas>
        <p style="margin-top: 16px; color: #999; font-size: 13px;">支付完成后将自动跳转...</p>
        <n-spin v-if="pollingStatus" size="small" style="margin-top: 8px;" />
      </div>
      <template #footer>
        <n-space justify="center">
          <n-button @click="showQrModal = false">取消支付</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Crypto Payment Modal -->
    <n-modal
      v-model:show="showCryptoModal"
      preset="card"
      title="加密货币支付"
      style="width: 480px; max-width: 92vw;"
      :bordered="false"
      :mask-closable="false"
      @after-leave="stopPolling"
    >
      <div v-if="cryptoInfo" style="text-align: center;">
        <p style="margin-bottom: 16px; color: #666;">请转账以下金额到指定钱包地址</p>
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
        <div style="margin-top: 16px;">
          <canvas ref="cryptoQrCanvas" style="margin: 0 auto;"></canvas>
        </div>
        <n-alert type="warning" :bordered="false" style="margin-top: 12px; text-align: left;" size="small">
          请务必确认网络和币种正确，转账错误无法找回。转账完成后请点击下方按钮，管理员将在确认到账后为您开通服务。
        </n-alert>
        <n-spin v-if="pollingStatus" size="small" style="margin-top: 8px;" />
      </div>
      <template #footer>
        <n-space justify="center">
          <n-button @click="showCryptoModal = false">取消</n-button>
          <n-button type="primary" @click="handleCryptoTransferred">我已转账</n-button>
        </n-space>
      </template>
    </n-modal>
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
import { listPackages, verifyCoupon, getPaymentMethods } from '@/api/common'
import { createOrder, payOrder, createPayment, getOrderStatus } from '@/api/order'
import { getDashboardInfo } from '@/api/user'

const router = useRouter()
const message = useMessage()

const loading = ref(false)
const packages = ref<any[]>([])
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
const userBalance = ref<number>(0)
const useBalanceDeduct = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null

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


const loadPackages = async () => {
  loading.value = true
  try {
    const [pkgRes, pmRes] = await Promise.all([listPackages(), getPaymentMethods()])
    packages.value = pkgRes.data || []
    const pmData = pmRes.data || {}
    paymentMethods.value = pmData.methods || []
    balanceEnabled.value = pmData.balance_enabled !== false
    // Auto-select first available method
    if (!balanceEnabled.value && paymentMethods.value.length > 0) {
      paymentMethod.value = 'pm_' + paymentMethods.value[0].id
    }
  } catch (e: any) {
    message.error(e.message || '加载套餐失败')
  } finally { loading.value = false }
}

const fetchUserBalance = async () => {
  try {
    const res = await getDashboardInfo()
    userBalance.value = res.data?.balance || 0
  } catch {}
}

const getPaymentLabel = (payType: string) => {
  const labels: Record<string, string> = {
    epay: '在线支付',
    alipay: '支付宝',
    wxpay: '微信支付',
    qqpay: 'QQ支付',
    stripe: 'Stripe (国际卡)',
    crypto: '加密货币 (USDT)',
  }
  return labels[payType] || payType
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
    message.error(e.message || '优惠码无效')
    couponInfo.value = null
  } finally { verifying.value = false }
}

const handleBuy = async (pkg: any) => {
  selectedPackage.value = pkg
  try {
    const payload: any = { package_id: pkg.id }
    if (couponCode.value.trim()) payload.coupon_code = couponCode.value
    const res = await createOrder(payload)
    orderInfo.value = res.data
    showPaymentModal.value = true
  } catch (e: any) {
    message.error(e.message || '创建订单失败')
  }
}

const isQrCodeUrl = (url: string) => {
  return url.includes('qr.alipay.com') || (url.startsWith('https://qr.') && url.length < 200)
}

const startPolling = (orderNo: string) => {
  pollingStatus.value = true
  pollTimer = setInterval(async () => {
    try {
      const res = await getOrderStatus(orderNo)
      if (res.data?.status === 'paid') {
        stopPolling()
        showQrModal.value = false
        message.success('支付成功')
        router.push('/orders')
      }
    } catch {}
  }, 3000)
}

const stopPolling = () => {
  pollingStatus.value = false
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

const showCryptoModal = ref(false)
const cryptoInfo = ref<any>(null)
const cryptoOrderNo = ref('')

const handlePay = async () => {
  if (!orderInfo.value) return
  paying.value = true
  try {
    if (paymentMethod.value === 'balance') {
      await payOrder(orderInfo.value.order_no, { payment_method: 'balance' })
      message.success('支付成功')
      showPaymentModal.value = false
      router.push('/orders')
    } else if (paymentMethod.value.startsWith('pm_')) {
      const pmId = parseInt(paymentMethod.value.replace('pm_', ''))
      const paymentData: any = { order_id: orderInfo.value.id, payment_method_id: pmId }
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
        if (isQrCodeUrl(data.payment_url)) {
          showQrModal.value = true
          await nextTick()
          if (qrCanvas.value) {
            QRCode.toCanvas(qrCanvas.value, data.payment_url, { width: 240, margin: 2 })
          }
          startPolling(orderInfo.value.order_no)
        } else {
          window.location.href = data.payment_url
        }
      } else {
        message.info('支付已创建，请等待处理')
        showPaymentModal.value = false
        router.push('/orders')
      }
    }
  } catch (e: any) {
    message.error(e.message || '支付失败')
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
.shop-container { padding: 24px; max-width: 1400px; margin: 0 auto; }
.header { text-align: center; margin-bottom: 16px; }
.title {
  font-size: 32px; font-weight: 600; margin: 0 0 8px 0;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text;
}
.subtitle { font-size: 16px; color: #666; margin: 0; }

.balance-info {
  text-align: center; margin-top: 8px; font-size: 15px; color: #666;
}
.balance-amount {
  color: #18a058; font-weight: 700; font-size: 18px;
}

.package-card {
  background: #fff; border-radius: 12px; padding: 24px;
  border: 2px solid #e8e8e8; transition: all 0.3s ease;
  cursor: pointer; position: relative; height: 100%;
  display: flex; flex-direction: column;
}
.package-card:hover { transform: translateY(-8px); box-shadow: 0 12px 24px rgba(0,0,0,0.1); border-color: #667eea; }
.package-card.featured { border-color: #667eea; border-width: 3px; background: linear-gradient(135deg, #667eea08 0%, #764ba208 100%); }
.badge {
  position: absolute; top: -12px; right: 24px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #fff; padding: 4px 16px; border-radius: 12px; font-size: 14px; font-weight: 600;
}
.card-header { text-align: center; margin-bottom: 24px; }
.package-name { font-size: 24px; font-weight: 600; margin: 0 0 16px 0; color: #333; }
.price-section { display: flex; align-items: baseline; justify-content: center; }
.currency { font-size: 24px; color: #667eea; font-weight: 600; }
.price { font-size: 48px; font-weight: 700; color: #667eea; margin-left: 4px; }
.card-body { flex: 1; margin-bottom: 24px; }
.feature-item { display: flex; align-items: center; gap: 8px; color: #666; font-size: 15px; }
.feature-item .n-icon { color: #667eea; }
.feature-extra .n-icon { color: #18a058; }
.features-list { margin-top: 8px; padding-top: 8px; border-top: 1px dashed #e8e8e8; }
.description {
  margin-top: 8px; padding: 12px; background: #f5f5f5;
  border-radius: 8px; color: #666; font-size: 14px; line-height: 1.6;
}
.card-footer { margin-top: auto; }

.modal-coupon { padding: 8px 0; }
.payment-method { padding: 4px 0; }
.pm-label { font-size: 14px; font-weight: 500; margin-bottom: 8px; color: #333; }

/* Mobile Responsive */
@media (max-width: 767px) {
  .shop-container { padding: 0; }
  .title { font-size: 24px; }
  .subtitle { font-size: 14px; }
  .package-card { padding: 18px 14px; }
  .package-card:hover { transform: none; }
  .package-name { font-size: 20px; margin-bottom: 10px; }
  .price { font-size: 36px; }
  .currency { font-size: 20px; }
  .feature-item { font-size: 14px; }
  .badge { top: -10px; right: 16px; font-size: 12px; padding: 3px 12px; }
}
</style>
