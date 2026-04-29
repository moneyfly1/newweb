<template>
  <div class="recharge-container">
    <n-space vertical :size="20">
      <div class="header">
        <h1 class="title">余额充值</h1>
        <p class="subtitle">当前余额：<span class="balance-val">¥{{ balance }}</span></p>
      </div>

      <!-- 待支付充值提示 -->
      <n-card v-if="pendingRecords.length > 0" :bordered="false" class="pending-card">
        <div class="pending-header">
          <n-icon :component="TimeOutline" size="18" color="#f0a020" />
          <span class="pending-title">有 {{ pendingRecords.length }} 条充值待支付</span>
        </div>
        <div class="pending-list">
          <div v-for="r in pendingRecords" :key="r.id" class="pending-item">
            <div class="pending-info">
              <span class="pending-amount">¥{{ r.amount }}</span>
              <span class="pending-time">{{ formatDateTime(r.created_at) }}</span>
            </div>
            <n-space :size="8">
              <n-button size="small" type="primary" @click="openPay(r)">继续支付</n-button>
              <n-button size="small" @click="handleCancel(r)">取消</n-button>
            </n-space>
          </div>
        </div>
      </n-card>

      <!-- 新建充值 -->
      <n-card :bordered="false" class="main-card">
        <n-space vertical :size="20">
          <div>
            <div class="section-label">选择充值金额</div>
            <n-space :size="12" :wrap="true">
              <div
                v-for="a in presetAmounts" :key="a"
                class="amount-chip" :class="{ active: amount === a }"
                @click="amount = a"
              >¥{{ a }}</div>
            </n-space>
          </div>

          <div>
            <div class="section-label">自定义金额</div>
            <n-input-number
              v-model:value="amount"
              :min="1" :max="10000"
              placeholder="输入充值金额"
              style="width: 100%;"
            />
          </div>

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

          <n-button
            type="primary" size="large" block
            :loading="submitting"
            :disabled="!amount || !paymentMethodId"
            @click="handleRecharge"
          >
            立即充值 ¥{{ amount || 0 }}
          </n-button>
        </n-space>
      </n-card>
    </n-space>

    <!-- 支付方式选择 Drawer（继续支付待支付记录）-->
    <common-drawer
      v-model:show="showPayDrawer"
      title="继续支付充值"
      :width="440"
      show-footer
      :loading="payingPending"
      @confirm="handlePendingPay"
      @cancel="showPayDrawer = false"
    >
      <n-space vertical :size="16">
        <n-descriptions :column="1" bordered>
          <n-descriptions-item label="充值金额">
            <span style="color: #18a058; font-size: 18px; font-weight: bold;">¥{{ pendingTarget?.amount }}</span>
          </n-descriptions-item>
          <n-descriptions-item label="订单号">{{ pendingTarget?.order_no }}</n-descriptions-item>
        </n-descriptions>
        <div>
          <div class="section-label">支付方式</div>
          <n-radio-group v-model:value="pendingPayMethodId">
            <n-space>
              <n-radio v-for="pm in paymentMethods" :key="pm.id" :value="pm.id">
                {{ getPaymentLabel(pm.pay_type) }}
              </n-radio>
            </n-space>
          </n-radio-group>
        </div>
      </n-space>
    </common-drawer>

    <!-- 扫码支付 Drawer -->
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
      <div style="text-align: center;">
        <p style="margin-bottom: 16px; color: #666;">请使用支付宝扫描下方二维码完成支付</p>
        <canvas ref="qrCanvas" style="margin: 0 auto; display: block;"></canvas>
        <p style="margin-top: 16px; color: #999; font-size: 13px;">支付后通常 1–5 秒到账，如未更新可继续刷新状态</p>
        <n-spin v-if="pollingStatus" size="small" style="margin-top: 8px;" />
      </div>
    </common-drawer>

    <!-- 手机支付 Drawer -->
    <common-drawer
      v-model:show="showMobilePayModal"
      title="手机支付"
      :width="400"
      :mask-closable="false"
      show-footer
      :show-confirm="false"
      cancel-text="取消支付"
      @cancel="showMobilePayModal = false"
      @after-leave="stopPolling"
    >
      <div style="text-align: center; padding: 24px 0;">
        <p style="margin-bottom: 20px; color: #555; font-size: 15px;">请点击下方按钮完成支付</p>
        <n-button type="primary" size="large" block tag="a" :href="mobilePayUrl" target="_blank">
          打开支付 App 付款
        </n-button>
        <p style="margin-top: 16px; color: #999; font-size: 13px;">支付完成后将自动更新余额...</p>
        <n-spin v-if="pollingStatus" size="small" style="margin-top: 8px;" />
      </div>
    </common-drawer>
    <!-- 充值成功提示 -->
    <common-drawer
      v-model:show="showRechargeSuccess"
      title="充值成功"
      :width="420"
      show-footer
      :show-confirm="false"
      cancel-text="我知道了"
      @cancel="showRechargeSuccess = false"
    >
      <n-space vertical :size="14">
        <n-alert type="success" :bordered="false">
          您已成功充值 <strong>¥{{ rechargeSuccessInfo?.amount.toFixed(2) || '0.00' }}</strong>。
        </n-alert>
        <n-descriptions :column="1" bordered>
          <n-descriptions-item label="充值金额">¥{{ rechargeSuccessInfo?.amount.toFixed(2) || '0.00' }}</n-descriptions-item>
          <n-descriptions-item label="当前余额">¥{{ rechargeSuccessInfo?.balance || '0.00' }}</n-descriptions-item>
          <n-descriptions-item label="余额用途">可用于购买套餐和升级设备数量</n-descriptions-item>
        </n-descriptions>
      </n-space>
    </common-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, onUnmounted } from 'vue'
import { useMessage, useDialog } from 'naive-ui'
import { TimeOutline } from '@vicons/ionicons5'
import QRCode from 'qrcode'
import { getPaymentMethods, createRecharge, listRechargeRecords, getRechargeStatus, cancelRecharge, createRechargePayment } from '@/api/common'
import { getDashboardInfo } from '@/api/user'
import { useAppStore } from '@/stores/app'
import { safeRedirect } from '@/utils/security'
import { getErrorMessage } from '@/utils/error'
import CommonDrawer from '@/components/CommonDrawer.vue'

const message = useMessage()
const dialog = useDialog()
const appStore = useAppStore()

const loading = ref(false)
const submitting = ref(false)
const showRechargeSuccess = ref(false)
const rechargeSuccessInfo = ref<{ amount: number; balance: string } | null>(null)
const balance = ref('0.00')
const amount = ref<number | null>(null)
const paymentMethodId = ref<number | null>(null)
const paymentMethods = ref<any[]>([])
const presetAmounts = [10, 50, 100, 200, 500]

// 待支付记录
const pendingRecords = ref<any[]>([])
const showPayDrawer = ref(false)
const pendingTarget = ref<any>(null)
const pendingPayMethodId = ref<number | null>(null)
const payingPending = ref(false)

// QR / 手机
const showQrModal = ref(false)
const showMobilePayModal = ref(false)
const qrCanvas = ref<HTMLCanvasElement | null>(null)
const mobilePayUrl = ref('')
const pollingStatus = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null
let pollingRecordId = 0
let pollAttempts = 0
const maxPollAttempts = 20

const getPaymentLabel = (payType: string) => {
  const labels: Record<string, string> = { epay: '在线支付', alipay: '支付宝', wxpay: '微信支付', qqpay: 'QQ支付', stripe: 'Stripe', codepay: '码支付', codepay_alipay: '码支付-支付宝', codepay_wxpay: '码支付-微信' }
  return labels[payType] || payType
}

const formatDateTime = (d: string) => {
  if (!d) return '-'
  return new Date(d).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

const loadData = async () => {
  loading.value = true
  try {
    const [dashRes, pmRes, rcRes] = await Promise.all([
      getDashboardInfo(),
      getPaymentMethods(),
      listRechargeRecords({ page: 1, page_size: 20, status: 'pending' }),
    ])
    balance.value = (dashRes.data?.balance ?? 0).toFixed(2)
    const pmData = pmRes.data || {}
    paymentMethods.value = pmData.methods || []
    if (paymentMethods.value.length > 0) {
      paymentMethodId.value = paymentMethods.value[0].id
      pendingPayMethodId.value = paymentMethods.value[0].id
    }
    pendingRecords.value = rcRes.data?.items || []
  } catch (e: any) {
    message.error(getErrorMessage(e, '加载数据失败'))
  } finally {
    loading.value = false
  }
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

const checkRechargeStatus = async (recordId: number) => {
  const res = await getRechargeStatus(recordId)
  if (res.data?.status === 'paid') {
    stopPolling()
    showQrModal.value = false
    showMobilePayModal.value = false
    await loadData()
    rechargeSuccessInfo.value = {
      amount: Number(res.data?.amount || amount.value || pendingTarget.value?.amount || 0),
      balance: balance.value,
    }
    showRechargeSuccess.value = true
    message.success('充值成功，余额已到账')
    return true
  }
  return false
}

const startPolling = (recordId: number) => {
  stopPolling()
  pollingRecordId = recordId
  pollAttempts = 0
  pollingStatus.value = true
  checkRechargeStatus(recordId).catch(() => {})
  pollTimer = setInterval(async () => {
    try {
      pollAttempts += 1
      const handled = await checkRechargeStatus(recordId)
      if (handled) {
        return
      }
      if (pollAttempts >= maxPollAttempts) {
        stopPolling()
        message.warning('充值结果确认较慢，支付已提交，请稍后刷新充值页面查看状态')
      }
    } catch {
      if (pollAttempts >= maxPollAttempts) {
        stopPolling()
        message.warning('充值结果确认较慢，支付已提交，请稍后刷新充值页面查看状态')
      }
    }
  }, 3000)
}

const stopPolling = () => {
  pollingStatus.value = false
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
}

const handlePayUrl = async (payUrl: string, recordId: number, paymentMode?: 'qrcode' | 'page' | 'redirect') => {
  if (paymentMode === 'page') {
    safeRedirect(payUrl)
    startPolling(recordId)
    return
  }
  if (paymentMode === 'redirect') {
    safeRedirect(payUrl)
    return
  }
  if (paymentMode === 'qrcode' || isQrCodeUrl(payUrl)) {
    if (appStore.isMobile) {
      mobilePayUrl.value = payUrl
      showMobilePayModal.value = true
    } else {
      showQrModal.value = true
      await nextTick()
      if (qrCanvas.value) QRCode.toCanvas(qrCanvas.value, payUrl, { width: 240, margin: 2 })
    }
    startPolling(recordId)
    return
  }
  safeRedirect(payUrl)
}

const handleRecharge = async () => {
  if (!amount.value || !paymentMethodId.value) {
    message.warning('请选择充值金额和支付方式')
    return
  }
  submitting.value = true
  try {
    // createRecharge 带 payment_method_id 时后端会直接生成 payment_url
    const res = await createRecharge({ amount: amount.value, payment_method_id: paymentMethodId.value })
    const data = res.data
    const payUrl = data?.payment_url || data?.record?.payment_url
    const paymentMode = data?.payment_mode
    const recordId = data?.record?.id || data?.id || 0
    if (payUrl) {
      await handlePayUrl(payUrl, recordId, paymentMode)
    } else {
      message.success('充值订单已创建，请等待处理')
      loadData()
    }
  } catch (e: any) {
    message.error(getErrorMessage(e, '充值失败'))
  } finally {
    submitting.value = false
  }
}

// 继续支付待支付记录
const openPay = (record: any) => {
  pendingTarget.value = record
  if (paymentMethods.value.length > 0) pendingPayMethodId.value = paymentMethods.value[0].id
  showPayDrawer.value = true
}

const handlePendingPay = async () => {
  if (!pendingTarget.value || !pendingPayMethodId.value) return
  payingPending.value = true
  try {
    const res = await createRechargePayment(pendingTarget.value.id, {
      recharge_id: pendingTarget.value.id,
      payment_method_id: pendingPayMethodId.value,
      is_mobile: appStore.isMobile,
    })
    showPayDrawer.value = false
    const payUrl = res.data?.payment_url
    const paymentMode = res.data?.payment_mode
    if (payUrl) {
      await handlePayUrl(payUrl, pendingTarget.value.id, paymentMode)
    } else {
      message.info('支付订单已创建，请等待处理')
    }
  } catch (e: any) {
    message.error(getErrorMessage(e, '支付失败'))
  } finally {
    payingPending.value = false
  }
}

const handleCancel = (record: any) => {
  dialog.warning({
    title: '取消充值',
    content: `确定要取消此充值记录（¥${record.amount}）吗？`,
    positiveText: '确定', negativeText: '取消',
    onPositiveClick: async () => {
      try { await cancelRecharge(record.id); message.success('充值已取消'); loadData() }
      catch (e: any) { message.error(getErrorMessage(e, '取消失败')) }
    },
  })
}

onUnmounted(() => { stopPolling() })
onMounted(() => { loadData() })
</script>

<style scoped>
.recharge-container { padding: 24px; }
.header { text-align: center; margin-bottom: 4px; }
.title {
  font-size: 32px; font-weight: 600; margin: 0 0 8px 0;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text;
}
.subtitle { font-size: 16px; color: #666; margin: 0; }
.balance-val { color: #18a058; font-weight: 700; }

/* 待支付提示卡 */
.pending-card { border-radius: 12px; border: 1.5px solid #f0a020; background: #fffbf0; }
.pending-header { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; }
.pending-title { font-size: 14px; font-weight: 600; color: #b76e00; }
.pending-list { display: flex; flex-direction: column; gap: 10px; }
.pending-item { display: flex; align-items: center; justify-content: space-between; padding: 10px 14px; background: white; border-radius: 8px; border: 1px solid #f0e0b0; }
.pending-info { display: flex; flex-direction: column; gap: 2px; }
.pending-amount { font-size: 16px; font-weight: 700; color: #18a058; }
.pending-time { font-size: 12px; color: #999; }

.main-card { border-radius: 12px; }
.section-label { font-size: 14px; font-weight: 500; color: #333; margin-bottom: 12px; }

.amount-chip {
  display: inline-flex; align-items: center; justify-content: center;
  min-width: 80px; padding: 10px 20px;
  border: 2px solid #e8e8e8; border-radius: 10px;
  font-size: 16px; font-weight: 600; color: #333;
  cursor: pointer; transition: all 0.2s;
  background: #fff; user-select: none;
}
.amount-chip:hover { border-color: #667eea; color: #667eea; }
.amount-chip.active { border-color: #667eea; background: #667eea12; color: #667eea; }

@media (max-width: 767px) {
  .recharge-container { padding: 0 12px; }
  .title { font-size: 24px; }
  .subtitle { font-size: 14px; }
  .amount-chip { min-width: 60px; padding: 8px 14px; font-size: 14px; }
  .pending-item { flex-direction: column; align-items: flex-start; gap: 10px; }
}
</style>
