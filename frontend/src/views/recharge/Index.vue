<template>
  <div class="recharge-container">
    <n-space vertical :size="24">
      <div class="header">
        <h1 class="title">余额充值</h1>
        <p class="subtitle">当前余额：¥{{ balance }}</p>
      </div>

      <n-card :bordered="false" class="main-card">
        <n-space vertical :size="20">
          <!-- Preset amounts -->
          <div>
            <div class="section-label">选择充值金额</div>
            <n-space :size="12" :wrap="true">
              <div v-for="a in presetAmounts" :key="a"
                class="amount-chip" :class="{ active: amount === a }"
                @click="amount = a">
                ¥{{ a }}
              </div>
            </n-space>
          </div>

          <!-- Custom amount -->
          <div>
            <div class="section-label">自定义金额</div>
            <n-input-number v-model:value="amount" :min="1" :max="10000" placeholder="输入充值金额" style="width: 100%;" />
          </div>

          <!-- Payment method -->
          <div>
            <div class="section-label">支付方式</div>
            <n-radio-group v-model:value="paymentMethodId">
              <n-space>
                <n-radio v-for="pm in paymentMethods" :key="pm.id" :value="pm.id">
                  {{ getPaymentLabel(pm.pay_type) }}
                </n-radio>
              </n-space>
            </n-radio-group>
            <div v-if="paymentMethods.length === 0 && !loading" style="color: #999; font-size: 14px; margin-top: 8px;">
              暂无可用支付方式，请联系管理员
            </div>
          </div>

          <!-- Submit -->
          <n-button type="primary" size="large" block :loading="submitting" :disabled="!amount || !paymentMethodId" @click="handleRecharge">
            立即充值 ¥{{ amount || 0 }}
          </n-button>
        </n-space>
      </n-card>
    </n-space>

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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, onUnmounted } from 'vue'
import { useMessage } from 'naive-ui'
import QRCode from 'qrcode'
import { getPaymentMethods, createRecharge } from '@/api/common'
import { getDashboardInfo } from '@/api/user'

const message = useMessage()

const loading = ref(false)
const submitting = ref(false)
const balance = ref('0.00')
const amount = ref<number | null>(null)
const paymentMethodId = ref<number | null>(null)
const paymentMethods = ref<any[]>([])
const presetAmounts = [10, 50, 100, 200, 500]
const showQrModal = ref(false)
const qrCanvas = ref<HTMLCanvasElement | null>(null)
const pollingStatus = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null

const getPaymentLabel = (payType: string) => {
  const labels: Record<string, string> = {
    epay: '在线支付',
    alipay: '支付宝',
    wxpay: '微信支付',
    qqpay: 'QQ支付',
  }
  return labels[payType] || payType
}

const loadData = async () => {
  loading.value = true
  try {
    const [dashRes, pmRes] = await Promise.all([getDashboardInfo(), getPaymentMethods()])
    balance.value = (dashRes.data?.balance ?? 0).toFixed(2)
    const pmData = pmRes.data || {}
    paymentMethods.value = pmData.methods || []
    if (paymentMethods.value.length > 0) {
      paymentMethodId.value = paymentMethods.value[0].id
    }
  } catch (e: any) {
    message.error(e.message || '加载数据失败')
  } finally {
    loading.value = false
  }
}
const isQrCodeUrl = (url: string) => {
  return url.includes('qr.alipay.com') || (url.startsWith('https://qr.') && url.length < 200)
}

const startPolling = (recordId: number) => {
  pollingStatus.value = true
  pollTimer = setInterval(async () => {
    try {
      const res = await getDashboardInfo()
      // Reload balance; also check recharge records
      const newBalance = (res.data?.balance ?? 0).toFixed(2)
      if (newBalance !== balance.value) {
        stopPolling()
        showQrModal.value = false
        balance.value = newBalance
        message.success('充值成功')
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

const handleRecharge = async () => {
  if (!amount.value || !paymentMethodId.value) {
    message.warning('请选择充值金额和支付方式')
    return
  }
  submitting.value = true
  try {
    const res = await createRecharge({ amount: amount.value, payment_method_id: paymentMethodId.value })
    const data = res.data
    const payUrl = data?.payment_url || data?.record?.payment_url
    if (payUrl) {
      if (isQrCodeUrl(payUrl)) {
        showQrModal.value = true
        await nextTick()
        if (qrCanvas.value) {
          QRCode.toCanvas(qrCanvas.value, payUrl, { width: 240, margin: 2 })
        }
        startPolling(data?.record?.id || 0)
      } else {
        window.location.href = payUrl
      }
    } else {
      message.success('充值订单已创建')
    }
  } catch (e: any) {
    message.error(e.message || '充值失败')
  } finally {
    submitting.value = false
  }
}

onUnmounted(() => { stopPolling() })

onMounted(() => { loadData() })
</script>
<style scoped>
.recharge-container { padding: 24px; max-width: 600px; margin: 0 auto; }
.header { text-align: center; margin-bottom: 16px; }
.title {
  font-size: 32px; font-weight: 600; margin: 0 0 8px 0;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text;
}
.subtitle { font-size: 16px; color: #666; margin: 0; }

.main-card { border-radius: 12px; }

.section-label {
  font-size: 14px; font-weight: 500; color: #333; margin-bottom: 12px;
}

.amount-chip {
  display: inline-flex; align-items: center; justify-content: center;
  min-width: 80px; padding: 10px 20px;
  border: 2px solid #e8e8e8; border-radius: 10px;
  font-size: 16px; font-weight: 600; color: #333;
  cursor: pointer; transition: all 0.2s ease;
  background: #fff; user-select: none;
}
.amount-chip:hover {
  border-color: #667eea; color: #667eea;
}
.amount-chip.active {
  border-color: #667eea; background: linear-gradient(135deg, #667eea08 0%, #764ba208 100%);
  color: #667eea;
}

@media (max-width: 767px) {
  .recharge-container { padding: 0; }
  .title { font-size: 24px; }
  .subtitle { font-size: 14px; }
  .amount-chip { min-width: 60px; padding: 8px 14px; font-size: 14px; }
}
</style>
