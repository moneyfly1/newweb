<template>
  <div class="order-container">
    <n-space vertical :size="24">
      <div class="header">
        <h1 class="title">我的订单</h1>
        <n-button type="primary" @click="router.push('/shop')">购买套餐</n-button>
      </div>

      <n-card :bordered="false" class="main-card">
        <n-tabs v-model:value="activeTab" type="line" animated @update:value="handleTabChange">
          <n-tab-pane name="orders" tab="全部订单">
            <!-- Status Filter -->
            <n-space :size="8" style="margin-bottom: 16px;">
              <n-button
                v-for="sf in statusFilters" :key="sf.value"
                :type="orderStatusFilter === sf.value ? 'primary' : 'default'"
                size="small"
                :ghost="orderStatusFilter === sf.value"
                @click="orderStatusFilter = sf.value; loadOrders()"
              >{{ sf.label }}</n-button>
            </n-space>

            <n-data-table
              v-if="!appStore.isMobile"
              :columns="orderColumns"
              :data="orders"
              :loading="ordersLoading"
              :pagination="orderPagination"
              :bordered="false"
              :single-line="false"
              :scroll-x="900"
            />

            <!-- Mobile card list -->
            <div v-else class="mobile-card-list">
              <div v-if="orders.length === 0 && !ordersLoading" class="mobile-empty">暂无订单</div>
              <div v-for="order in orders" :key="order.id" class="mobile-card">
                <div class="card-row">
                  <span class="label">订单号</span>
                  <span class="value">{{ order.order_no }}</span>
                </div>
                <div class="card-row">
                  <span class="label">套餐</span>
                  <span class="value">{{ order.package_name || '-' }}</span>
                </div>
                <div class="card-row">
                  <span class="label">金额</span>
                  <span class="value" style="color: #18a058; font-weight: 600;">¥{{ order.final_amount }}</span>
                </div>
                <div class="card-row">
                  <span class="label">状态</span>
                  <span class="value"><n-tag :type="getStatusType(order.status)" size="small">{{ getStatusText(order.status) }}</n-tag></span>
                </div>
                <div class="card-row">
                  <span class="label">时间</span>
                  <span class="value">{{ formatDateTime(order.created_at) }}</span>
                </div>
                <div class="card-actions">
                  <n-button size="small" type="info" @click="detailOrder = order; showDetailModal = true">详情</n-button>
                  <n-button v-if="order.status === 'pending'" size="small" type="primary" @click="currentOrder = order; showPaymentModal = true">支付</n-button>
                  <n-button v-if="order.status === 'pending'" size="small" @click="handleCancelOrder(order)">取消</n-button>
                </div>
              </div>
            </div>
          </n-tab-pane>

          <n-tab-pane name="recharge" tab="充值记录">
            <n-data-table
              v-if="!appStore.isMobile"
              :columns="rechargeColumns"
              :data="rechargeRecords"
              :loading="rechargeLoading"
              :pagination="rechargePagination"
              :bordered="false"
              :single-line="false"
              :scroll-x="700"
            />

            <div v-else class="mobile-card-list">
              <div v-if="rechargeRecords.length === 0 && !rechargeLoading" class="mobile-empty">暂无充值记录</div>
              <div v-for="record in rechargeRecords" :key="record.id" class="mobile-card">
                <div class="card-row">
                  <span class="label">订单号</span>
                  <span class="value">{{ record.order_no }}</span>
                </div>
                <div class="card-row">
                  <span class="label">金额</span>
                  <span class="value" style="color: #18a058; font-weight: 600;">¥{{ record.amount }}</span>
                </div>
                <div class="card-row">
                  <span class="label">状态</span>
                  <span class="value"><n-tag :type="getStatusType(record.status)" size="small">{{ getStatusText(record.status) }}</n-tag></span>
                </div>
                <div class="card-row">
                  <span class="label">时间</span>
                  <span class="value">{{ formatDateTime(record.created_at) }}</span>
                </div>
                <div class="card-actions" v-if="record.status === 'pending'">
                  <n-button size="small" @click="handleCancelRecharge(record)">取消</n-button>
                </div>
              </div>
            </div>
          </n-tab-pane>
        </n-tabs>
      </n-card>
    </n-space>

    <!-- Payment Modal -->
    <n-modal
      v-model:show="showPaymentModal"
      preset="card"
      title="确认支付"
      style="width: 500px; max-width: 92vw;"
      :bordered="false"
      :segmented="{ content: true }"
    >
      <n-space vertical :size="16">
        <n-descriptions :column="1" bordered>
          <n-descriptions-item label="订单号">{{ currentOrder?.order_no }}</n-descriptions-item>
          <n-descriptions-item label="套餐名称">{{ currentOrder?.package_name }}</n-descriptions-item>
          <n-descriptions-item label="原价">¥{{ currentOrder?.amount }}</n-descriptions-item>
          <n-descriptions-item v-if="currentOrder?.discount_amount" label="优惠">
            <span style="color: #e03050;">-¥{{ currentOrder?.discount_amount }}</span>
          </n-descriptions-item>
          <n-descriptions-item label="实付金额">
            <span style="color: #18a058; font-size: 18px; font-weight: bold;">¥{{ currentOrder?.final_amount }}</span>
          </n-descriptions-item>
        </n-descriptions>
        <!-- Payment Method -->
        <div>
          <div style="font-size: 14px; font-weight: 500; margin-bottom: 8px; color: #333;">支付方式</div>
          <n-radio-group v-model:value="orderPayMethod">
            <n-space>
              <n-radio v-if="pmBalanceEnabled" value="balance">余额支付</n-radio>
              <n-radio v-for="pm in pmMethods" :key="pm.id" :value="'pm_' + pm.id">
                {{ pmLabel(pm.pay_type) }}
              </n-radio>
            </n-space>
          </n-radio-group>
        </div>
      </n-space>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showPaymentModal = false">取消</n-button>
          <n-button type="primary" :loading="paying" @click="handlePay">确认支付</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Order Detail Modal -->
    <n-modal
      v-model:show="showDetailModal"
      preset="card"
      title="订单详情"
      style="width: 560px; max-width: 92vw;"
      :bordered="false"
    >
      <n-descriptions :column="1" bordered v-if="detailOrder">
        <n-descriptions-item label="订单号">{{ detailOrder.order_no }}</n-descriptions-item>
        <n-descriptions-item label="套餐名称">{{ detailOrder.package_name }}</n-descriptions-item>
        <n-descriptions-item label="原价">¥{{ detailOrder.amount }}</n-descriptions-item>
        <n-descriptions-item label="优惠金额">¥{{ detailOrder.discount_amount || '0.00' }}</n-descriptions-item>
        <n-descriptions-item label="实付金额">
          <span style="color: #18a058; font-weight: 600;">¥{{ detailOrder.final_amount }}</span>
        </n-descriptions-item>
        <n-descriptions-item label="支付方式">{{ detailOrder.payment_method || '-' }}</n-descriptions-item>
        <n-descriptions-item label="状态">
          <n-tag :type="getStatusType(detailOrder.status)" size="small">{{ getStatusText(detailOrder.status) }}</n-tag>
        </n-descriptions-item>
        <n-descriptions-item label="创建时间">{{ formatDateTime(detailOrder.created_at) }}</n-descriptions-item>
        <n-descriptions-item v-if="detailOrder.paid_at" label="支付时间">{{ formatDateTime(detailOrder.paid_at) }}</n-descriptions-item>
      </n-descriptions>
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
  </div>
</template>

<script setup lang="tsx">
import { ref, onMounted, h, nextTick, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, useDialog, NButton, NSpace, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import QRCode from 'qrcode'
import { listOrders, payOrder, cancelOrder, createPayment, getOrderStatus } from '@/api/order'
import { listRechargeRecords, cancelRecharge, getPaymentMethods } from '@/api/common'
import { useAppStore } from '@/stores/app'
import { safeRedirect } from '@/utils/security'

const router = useRouter()
const appStore = useAppStore()
const message = useMessage()
const dialog = useDialog()

const activeTab = ref('orders')
const ordersLoading = ref(false)
const rechargeLoading = ref(false)
const orders = ref<any[]>([])
const rechargeRecords = ref<any[]>([])
const showPaymentModal = ref(false)
const showDetailModal = ref(false)
const currentOrder = ref<any>(null)
const detailOrder = ref<any>(null)
const paying = ref(false)
const orderStatusFilter = ref('')
const orderPayMethod = ref('balance')
const pmMethods = ref<any[]>([])
const pmBalanceEnabled = ref(true)
const showQrModal = ref(false)
const qrCanvas = ref<HTMLCanvasElement | null>(null)
const pollingStatus = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null

const pmLabel = (payType: string) => {
  const labels: Record<string, string> = { epay: '在线支付', alipay: '支付宝', wxpay: '微信支付', qqpay: 'QQ支付' }
  return labels[payType] || payType
}

const loadPaymentMethods = async () => {
  try {
    const res = await getPaymentMethods()
    const data = res.data || {}
    pmMethods.value = data.methods || []
    pmBalanceEnabled.value = data.balance_enabled !== false
    if (!pmBalanceEnabled.value && pmMethods.value.length > 0) {
      orderPayMethod.value = 'pm_' + pmMethods.value[0].id
    }
  } catch {}
}

const statusFilters = [
  { label: '全部', value: '' },
  { label: '待支付', value: 'pending' },
  { label: '已支付', value: 'paid' },
  { label: '已取消', value: 'cancelled' },
  { label: '已过期', value: 'expired' },
  { label: '已退款', value: 'refunded' },
]

const getStatusType = (s: string) => {
  const m: Record<string, any> = { pending: 'warning', paid: 'success', cancelled: 'default', expired: 'error', refunded: 'info' }
  return m[s] || 'default'
}
const getStatusText = (s: string) => {
  const m: Record<string, string> = { pending: '待支付', paid: '已支付', cancelled: '已取消', expired: '已过期', refunded: '已退款' }
  return m[s] || s
}


const orderPagination = ref({
  page: 1, pageSize: 10, itemCount: 0,
  showSizePicker: true, pageSizes: [10, 20, 50],
  onChange: (p: number) => { orderPagination.value.page = p; loadOrders() },
  onUpdatePageSize: (ps: number) => { orderPagination.value.pageSize = ps; orderPagination.value.page = 1; loadOrders() }
})

const rechargePagination = ref({
  page: 1, pageSize: 10, itemCount: 0,
  showSizePicker: true, pageSizes: [10, 20, 50],
  onChange: (p: number) => { rechargePagination.value.page = p; loadRechargeRecords() },
  onUpdatePageSize: (ps: number) => { rechargePagination.value.pageSize = ps; rechargePagination.value.page = 1; loadRechargeRecords() }
})

const formatDateTime = (dateStr: string) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'
  })
}

const orderColumns: DataTableColumns<any> = [
  { title: '订单号', key: 'order_no', width: 180, resizable: true, ellipsis: { tooltip: true } },
  { title: '套餐名称', key: 'package_name', width: 140, resizable: true },
  { title: '原价', key: 'amount', width: 90, resizable: true, sorter: (a, b) => Number(a.amount) - Number(b.amount), render: (r) => `¥${r.amount}` },
  { title: '优惠', key: 'discount_amount', width: 90, resizable: true, render: (r) => r.discount_amount ? `-¥${r.discount_amount}` : '-' },
  {
    title: '实付', key: 'final_amount', width: 90, resizable: true,
    sorter: (a, b) => Number(a.final_amount) - Number(b.final_amount),
    render: (r) => h('span', { style: 'color:#18a058;font-weight:600' }, `¥${r.final_amount}`)
  },
  { title: '状态', key: 'status', width: 90, resizable: true, render: (r) => h(NTag, { type: getStatusType(r.status), size: 'small' }, { default: () => getStatusText(r.status) }) },
  { title: '支付方式', key: 'payment_method', width: 90, resizable: true, render: (r) => r.payment_method || '-' },
  { title: '创建时间', key: 'created_at', width: 170, resizable: true, sorter: (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(), render: (r) => formatDateTime(r.created_at) },
  {
    title: '操作', key: 'actions', width: 200, fixed: 'right',
    render: (row) => {
      const btns: any[] = [
        h(NButton, { size: 'small', quaternary: true, type: 'info', onClick: () => { detailOrder.value = row; showDetailModal.value = true } }, { default: () => '详情' })
      ]
      if (row.status === 'pending') {
        btns.push(h(NButton, { size: 'small', type: 'primary', onClick: () => { currentOrder.value = row; showPaymentModal.value = true } }, { default: () => '支付' }))
        btns.push(h(NButton, { size: 'small', onClick: () => handleCancelOrder(row) }, { default: () => '取消' }))
      }
      return h(NSpace, { size: 4 }, { default: () => btns })
    }
  }
]


const rechargeColumns: DataTableColumns<any> = [
  { title: '订单号', key: 'order_no', width: 180, resizable: true, ellipsis: { tooltip: true } },
  { title: '金额', key: 'amount', width: 100, resizable: true, sorter: (a, b) => Number(a.amount) - Number(b.amount), render: (r) => h('span', { style: 'color:#18a058;font-weight:600' }, `¥${r.amount}`) },
  { title: '状态', key: 'status', width: 100, resizable: true, render: (r) => h(NTag, { type: getStatusType(r.status), size: 'small' }, { default: () => getStatusText(r.status) }) },
  { title: '支付方式', key: 'payment_method', width: 100, resizable: true, render: (r) => r.payment_method || '-' },
  { title: '创建时间', key: 'created_at', width: 170, resizable: true, sorter: (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(), render: (r) => formatDateTime(r.created_at) },
  {
    title: '操作', key: 'actions', width: 100, fixed: 'right',
    render: (row) => {
      if (row.status === 'pending') {
        return h(NButton, { size: 'small', onClick: () => handleCancelRecharge(row) }, { default: () => '取消' })
      }
      return h('span', { style: 'color:#999' }, '-')
    }
  }
]

const loadOrders = async () => {
  ordersLoading.value = true
  try {
    const params: any = { page: orderPagination.value.page, page_size: orderPagination.value.pageSize }
    if (orderStatusFilter.value) params.status = orderStatusFilter.value
    const res = await listOrders(params)
    orders.value = res.data?.items || []
    orderPagination.value.itemCount = res.data?.total || 0
  } catch (e: any) { message.error(e.message || '加载订单失败') }
  finally { ordersLoading.value = false }
}

const loadRechargeRecords = async () => {
  rechargeLoading.value = true
  try {
    const res = await listRechargeRecords({ page: rechargePagination.value.page, page_size: rechargePagination.value.pageSize })
    rechargeRecords.value = res.data?.items || []
    rechargePagination.value.itemCount = res.data?.total || 0
  } catch (e: any) { message.error(e.message || '加载充值记录失败') }
  finally { rechargeLoading.value = false }
}

const handleTabChange = (tab: string) => {
  if (tab === 'orders') loadOrders()
  else loadRechargeRecords()
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
        loadOrders()
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

const handlePay = async () => {
  if (!currentOrder.value) return
  paying.value = true
  try {
    if (orderPayMethod.value === 'balance') {
      await payOrder(currentOrder.value.order_no, { payment_method: 'balance' })
      message.success('支付成功')
      showPaymentModal.value = false
      loadOrders()
    } else if (orderPayMethod.value.startsWith('pm_')) {
      const pmId = parseInt(orderPayMethod.value.replace('pm_', ''))
      const res = await createPayment({ order_id: currentOrder.value.id, payment_method_id: pmId })
      const data = res.data
      if (data?.payment_url) {
        showPaymentModal.value = false
        if (isQrCodeUrl(data.payment_url)) {
          showQrModal.value = true
          await nextTick()
          if (qrCanvas.value) {
            QRCode.toCanvas(qrCanvas.value, data.payment_url, { width: 240, margin: 2 })
          }
          startPolling(currentOrder.value.order_no)
        } else {
          safeRedirect(data.payment_url)
        }
      } else {
        message.info('支付已创建，请等待处理')
        showPaymentModal.value = false
        loadOrders()
      }
    }
  } catch (e: any) { message.error(e.message || '支付失败') }
  finally { paying.value = false }
}

onUnmounted(() => { stopPolling() })

const handleCancelOrder = (order: any) => {
  dialog.warning({
    title: '取消订单',
    content: `确定要取消订单 ${order.order_no} 吗？`,
    positiveText: '确定', negativeText: '取消',
    onPositiveClick: async () => {
      try { await cancelOrder(order.order_no); message.success('订单已取消'); loadOrders() }
      catch (e: any) { message.error(e.message || '取消订单失败') }
    }
  })
}

const handleCancelRecharge = (record: any) => {
  dialog.warning({
    title: '取消充值',
    content: `确定要取消此充值记录吗？`,
    positiveText: '确定', negativeText: '取消',
    onPositiveClick: async () => {
      try { await cancelRecharge(record.id); message.success('充值已取消'); loadRechargeRecords() }
      catch (e: any) { message.error(e.message || '取消充值失败') }
    }
  })
}

onMounted(() => { loadOrders(); loadPaymentMethods() })
</script>

<style scoped>
.order-container { padding: 24px; }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.title {
  font-size: 28px; font-weight: 600; margin: 0;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text;
}
.main-card { border-radius: 12px; }

/* Mobile Responsive */
@media (max-width: 767px) {
  .order-container { padding: 0 12px; }
  .header { margin-bottom: 4px; }
  .title { font-size: 22px; }
}

</style>