<template>
  <div class="admin-orders-page admin-page-shell">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">订单管理</h2>
        <p class="page-subtitle">查看和处理全站订单，支持退款、状态调整及财务对账</p>
      </div>
      <div class="header-right">
        <n-space>
          <n-input
            v-model:value="searchQuery"
            placeholder="订单号 / 用户ID / 邮箱"
            clearable
            class="search-input"
            @keyup.enter="handleSearch"
          >
            <template #prefix><n-icon :component="SearchOutline" /></template>
          </n-input>
          <n-select
            v-model:value="statusFilter"
            placeholder="所有状态"
            clearable
            class="status-select"
            :options="statusOptions"
            @update:value="handleSearch"
          />
          <n-button secondary @click="fetchOrders">
            <template #icon><n-icon><refresh-outline /></n-icon></template>
            刷新
          </n-button>
        </n-space>
      </div>
    </div>

    <!-- Stats Summary - Desktop & Mobile -->
    <div class="stats-summary">
      <!-- Desktop: 4 columns -->
      <n-grid v-if="!appStore.isMobile" :cols="4" :x-gap="16">
        <n-grid-item>
          <div class="mini-stat-card">
            <div class="stat-label">今日营收</div>
            <div class="stat-value">¥{{ orderStats.today_revenue || '0.00' }}</div>
          </div>
        </n-grid-item>
        <n-grid-item>
          <div class="mini-stat-card">
            <div class="stat-label">本月营收</div>
            <div class="stat-value">¥{{ orderStats.month_revenue || '0.00' }}</div>
          </div>
        </n-grid-item>
        <n-grid-item>
          <div class="mini-stat-card">
            <div class="stat-label">待支付订单</div>
            <div class="stat-value text-warning">{{ orderStats.pending_count || 0 }}</div>
          </div>
        </n-grid-item>
        <n-grid-item>
          <div class="mini-stat-card">
            <div class="stat-label">退款订单</div>
            <div class="stat-value text-error">{{ orderStats.refunded_count || 0 }}</div>
          </div>
        </n-grid-item>
      </n-grid>
      <!-- Mobile: 2x2 grid -->
      <n-grid v-else :cols="2" :x-gap="8" :y-gap="8">
        <n-grid-item>
          <div class="mini-stat-card mobile-stat">
            <div class="stat-label">今日营收</div>
            <div class="stat-value">¥{{ orderStats.today_revenue || '0.00' }}</div>
          </div>
        </n-grid-item>
        <n-grid-item>
          <div class="mini-stat-card mobile-stat">
            <div class="stat-label">本月营收</div>
            <div class="stat-value">¥{{ orderStats.month_revenue || '0.00' }}</div>
          </div>
        </n-grid-item>
        <n-grid-item>
          <div class="mini-stat-card mobile-stat">
            <div class="stat-label">待支付</div>
            <div class="stat-value text-warning">{{ orderStats.pending_count || 0 }}</div>
          </div>
        </n-grid-item>
        <n-grid-item>
          <div class="mini-stat-card mobile-stat">
            <div class="stat-label">退款</div>
            <div class="stat-value text-error">{{ orderStats.refunded_count || 0 }}</div>
          </div>
        </n-grid-item>
      </n-grid>
    </div>

    <n-card :bordered="false" class="main-card">
      <n-space vertical :size="16">

        <!-- Batch operations -->
        <n-space v-if="checkedRowKeys.length > 0" align="center" class="batch-operations">
          <span class="batch-selected-text">已选择 {{ checkedRowKeys.length }} 项</span>
          <n-button size="small" type="warning" @click="handleBatchCancel">批量取消</n-button>
          <n-button size="small" type="error" @click="handleBatchRefund">批量退款</n-button>
        </n-space>

        <n-data-table
          remote
          :columns="columns"
          :data="orders"
          :loading="loading"
          :pagination="pagination"
          :bordered="false"
          :single-line="false"
          :row-key="(row: any) => row.id"
          :checked-row-keys="checkedRowKeys"
          :scroll-x="appStore.isMobile ? 1100 : 1450"
          class="unified-admin-table"
          @update:checked-row-keys="(keys: number[]) => { checkedRowKeys = keys }"
          @update:page="(p: number) => { pagination.page = p; fetchOrders() }"
          @update:page-size="(ps: number) => { pagination.pageSize = ps; pagination.page = 1; fetchOrders() }"
        />
      </n-space>
    </n-card>

    <common-drawer v-model:show="showDetailDrawer" title="订单流水详情" :width="540">
      <div v-if="currentOrder" class="detail-container">
        <div class="detail-header">
          <div class="amount-display">
            <div class="label">实付金额</div>
            <div class="value">{{ formatCurrency(currentOrder.final_amount || currentOrder.amount || 0) }}</div>
          </div>
          <n-tag :type="getStatusType(currentOrder.status)" round>{{ getStatusText(currentOrder.status) }}</n-tag>
        </div>

        <n-descriptions label-placement="left" :column="1" bordered size="small" class="detail-desc">
          <n-descriptions-item label="订单号">
            <div class="copyable-row">
              <code class="order-no-code">{{ currentOrder.order_no }}</code>
              <n-button size="tiny" quaternary @click="copyToClipboard(currentOrder.order_no)">复制</n-button>
            </div>
          </n-descriptions-item>
          <n-descriptions-item label="用户邮箱">{{ currentOrder.user_email || '-' }}</n-descriptions-item>
          <n-descriptions-item label="关联用户">ID: {{ currentOrder.user_id }}</n-descriptions-item>
          <n-descriptions-item label="订单类型">{{ currentOrder.order_type_text || '套餐订单' }}</n-descriptions-item>
          <n-descriptions-item label="订单内容">{{ currentOrder.order_summary || currentOrder.package_name || '-' }}</n-descriptions-item>
          <n-descriptions-item label="原始金额">{{ formatCurrency(currentOrder.amount) }}</n-descriptions-item>
          <n-descriptions-item label="优惠抵扣" v-if="currentOrder.discount_amount > 0">
            - {{ formatCurrency(currentOrder.discount_amount) }}
          </n-descriptions-item>
          <n-descriptions-item label="支付网关">{{ getPaymentMethodText(currentOrder.payment_method_name) }}</n-descriptions-item>
          <n-descriptions-item label="创建时间">{{ formatFullDate(currentOrder.created_at) }}</n-descriptions-item>
          <n-descriptions-item label="支付时间" v-if="currentOrder.payment_time">
            {{ formatFullDate(currentOrder.payment_time) }}
          </n-descriptions-item>
          <n-descriptions-item label="外部流水号" v-if="currentOrder.gateway_order_id">
            <code class="gateway-no">{{ currentOrder.gateway_order_id }}</code>
          </n-descriptions-item>
        </n-descriptions>

        <div class="detail-actions" v-if="['paid', 'completed', 'pending'].includes(currentOrder.status)">
          <n-divider />
          <n-space justify="end">
            <n-button v-if="currentOrder.status === 'pending'" type="error" ghost @click="handleCancel(currentOrder)">取消订单</n-button>
            <n-button v-if="currentOrder.status === 'paid'" type="success" @click="handleComplete(currentOrder)">标记完成</n-button>
            <n-button v-if="['paid', 'completed'].includes(currentOrder.status)" type="warning" @click="handleRefund(currentOrder)">全额退款</n-button>
          </n-space>
        </div>
      </div>
    </common-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, h, onMounted, watch } from 'vue'
import { NButton, NTag, NSpace, NIcon, NSelect, useMessage, useDialog, type DataTableColumns, type TagProps } from 'naive-ui'
import { SearchOutline, RefreshOutline, ReceiptOutline, TimeOutline, MailOutline, LayersOutline } from '@vicons/ionicons5'
import { listAdminOrders, refundOrder, cancelOrder, completeOrder, getAdminDashboard } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import CommonDrawer from '@/components/CommonDrawer.vue'
import { useRoute } from 'vue-router'
import { formatCurrency } from '@/utils/amount'
import '@/styles/admin-common.css'

const message = useMessage()
const dialog = useDialog()
const appStore = useAppStore()
const route = useRoute()

const loading = ref(false)
const orders = ref<any[]>([])
const orderStats = ref<any>({})
const searchQuery = ref((route.query.order_no as string) || '')
const statusFilter = ref(null)
const pagination = reactive({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: true, pageSizes: [20, 50, 100] })
const checkedRowKeys = ref<number[]>([])

const showDetailDrawer = ref(false)
const currentOrder = ref<any>(null)

const statusOptions = [
  { label: '待支付', value: 'pending' },
  { label: '已支付', value: 'paid' },
  { label: '已完成', value: 'completed' },
  { label: '已取消', value: 'cancelled' },
  { label: '已退款', value: 'refunded' }
]

const getStatusType = (s: string): TagProps['type'] => {
  const typeMap: Record<string, NonNullable<TagProps['type']>> = {
    pending: 'warning',
    paid: 'success',
    completed: 'info',
    cancelled: 'default',
    refunded: 'error'
  }
  return typeMap[s] || 'default'
}

const getStatusText = (s: string) => ({ pending: '待支付', paid: '已支付', completed: '已完成', cancelled: '已取消', refunded: '已退款' }[s] || s)
const getPaymentMethodText = (row: any) => {
  const m = row.payment_method_name
  const nameMap: Record<string, string> = { alipay: '支付宝', wechat: '微信支付', balance: '余额支付', stripe: 'Stripe', epay: '易支付' }
  if (nameMap[m]) return nameMap[m]
  if (m) return m
  return row.status === 'pending' ? '待选择' : '未支付'
}
const getOrderTypeTag = (type: string): TagProps['type'] => ({ package: 'info', custom_package: 'warning', subscription_upgrade: 'success' }[type] as TagProps['type'] || 'default')
const getOrderTypeText = (row: any) => row.order_type_text || '套餐订单'
const getOrderSummary = (row: any) => row.order_summary || row.package_name || '-'

const columns: DataTableColumns<any> = [
  { type: 'selection' },
  {
    title: '订单信息',
    key: 'order_no',
    minWidth: 240,
    render: (row: any) => h('div', { class: 'cell-block order-block' }, [
      h('div', { class: 'cell-title order-no' }, row.order_no),
      h('div', { class: 'cell-sub' }, `UID: ${row.user_id}`)
    ])
  },
  {
    title: '用户邮箱',
    key: 'user_email',
    minWidth: 220,
    render: (row: any) => h('div', { class: 'cell-inline' }, [
      h(NIcon, { component: MailOutline, size: 14, class: 'inline-icon' }),
      h('span', { class: 'email-text' }, row.user_email || '-')
    ])
  },
  {
    title: '订单类型',
    key: 'order_type',
    width: 120,
    render: (row: any) => h(NTag, { type: getOrderTypeTag(row.order_type), size: 'small', round: true, bordered: false }, { default: () => getOrderTypeText(row) })
  },
  {
    title: '订单内容',
    key: 'order_summary',
    minWidth: 220,
    render: (row: any) => h('div', { class: 'cell-inline' }, [
      h(NIcon, { component: LayersOutline, size: 14, class: 'inline-icon' }),
      h('span', { class: 'summary-text' }, getOrderSummary(row))
    ])
  },
  {
    title: '实付金额',
    key: 'final_amount',
    width: 120,
    render: (row: any) => h('span', { class: 'amount-text' }, formatCurrency(row.final_amount || row.amount || 0))
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row: any) => h(NTag, { type: getStatusType(row.status), size: 'small', round: true, ghost: true }, { default: () => getStatusText(row.status) })
  },
  {
    title: '支付方式',
    key: 'payment_method_name',
    width: 110,
    render: (row: any) => h('span', { class: 'plain-text' }, getPaymentMethodText(row))
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 180,
    render: (row: any) => h('div', { class: 'cell-inline time-text left-text' }, [
      h(NIcon, { component: TimeOutline, size: 14, class: 'inline-icon' }),
      h('span', formatDate(row.created_at))
    ])
  },
  {
    title: '操作',
    key: 'actions',
    width: 120,
    fixed: 'right',
    render: (row: any) => h(NButton, { size: 'small', quaternary: true, type: 'primary', onClick: () => handleViewDetail(row) }, { default: () => '管理订单' })
  }
]

const fetchOrders = async () => {
  loading.value = true
  try {
    const res = await listAdminOrders({ page: pagination.page, page_size: pagination.pageSize, order_no: searchQuery.value || undefined, status: statusFilter.value })
    orders.value = res.data.items || []
    pagination.itemCount = res.data.total || 0
    if (pagination.page === 1) {
      const dashRes = await getAdminDashboard()
      orderStats.value = dashRes.data
    }
  } finally {
    loading.value = false
  }
}

const handleSearch = () => { pagination.page = 1; fetchOrders() }
const handleViewDetail = (row: any) => { currentOrder.value = row; showDetailDrawer.value = true }

const handleRefund = (row: any) => {
  dialog.warning({
    title: '确认全额退款',
    content: `订单 ${row.order_no} 将退款 ${formatCurrency(row.final_amount || row.amount)} 到用户余额。`,
    positiveText: '确认退款',
    onPositiveClick: async () => {
      await refundOrder(row.id)
      message.success('已退款')
      showDetailDrawer.value = false
      fetchOrders()
    }
  })
}

const handleCancel = (row: any) => {
  dialog.warning({
    title: '取消订单',
    content: '确定要取消此待支付订单吗？',
    positiveText: '确定',
    onPositiveClick: async () => {
      await cancelOrder(row.id)
      message.success('已取消')
      showDetailDrawer.value = false
      fetchOrders()
    }
  })
}

const handleComplete = (row: any) => {
  dialog.info({
    title: '手动标记完成',
    content: '此操作将直接激活用户的订阅套餐。',
    positiveText: '确定完成',
    onPositiveClick: async () => {
      await completeOrder(row.id)
      message.success('已标记为完成')
      showDetailDrawer.value = false
      fetchOrders()
    }
  })
}

const handleBatchCancel = () => {
  const selected = orders.value.filter(o => checkedRowKeys.value.includes(o.id))
  const pending = selected.filter(o => o.status === 'pending')
  if (pending.length === 0) {
    message.warning('没有可取消的待支付订单')
    return
  }
  dialog.warning({
    title: '批量取消订单',
    content: `确定要取消选中的 ${pending.length} 个待支付订单吗？`,
    positiveText: '确定',
    onPositiveClick: async () => {
      try {
        await Promise.all(pending.map(o => cancelOrder(o.id)))
        message.success('批量取消完成')
        checkedRowKeys.value = []
        fetchOrders()
      } catch { message.error('批量取消失败') }
    }
  })
}

const handleBatchRefund = () => {
  const selected = orders.value.filter(o => checkedRowKeys.value.includes(o.id))
  const refundable = selected.filter(o => ['paid', 'completed'].includes(o.status))
  if (refundable.length === 0) {
    message.warning('没有可退款的订单')
    return
  }
  dialog.warning({
    title: '批量退款',
    content: `确定要退款选中的 ${refundable.length} 个订单吗？`,
    positiveText: '确定退款',
    onPositiveClick: async () => {
      try {
        await Promise.all(refundable.map(o => refundOrder(o.id)))
        message.success('批量退款完成')
        checkedRowKeys.value = []
        fetchOrders()
      } catch { message.error('批量退款失败') }
    }
  })
}

const formatDate = (d: string) => d ? new Date(d).toLocaleDateString('zh-CN') + ' ' + new Date(d).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) : '-'
const formatFullDate = (d: string) => d ? new Date(d).toLocaleString('zh-CN') : '-'

const copyToClipboard = (text: string) => {
  navigator.clipboard.writeText(text)
  message.success('已复制到剪贴板')
}

watch(() => route.query.order_no, (orderNo) => {
  if (typeof orderNo === 'string' && orderNo !== searchQuery.value) {
    searchQuery.value = orderNo
    handleSearch()
  }
})

onMounted(() => fetchOrders())
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px; }
.page-title { margin: 0; font-size: 24px; font-weight: 700; color: var(--n-title-text-color); }
.page-subtitle { margin: 4px 0 0; color: #888; font-size: 14px; }

.stats-summary { margin-bottom: 24px; }
.mini-stat-card {
  padding: 16px;
  border-radius: 12px;
  background: white;
  box-shadow: 0 2px 8px rgba(0,0,0,0.04);
}
.mini-stat-card .stat-label { font-size: 12px; color: #888; margin-bottom: 4px; }
.mini-stat-card .stat-value { font-size: 20px; font-weight: 700; color: #333; }
.text-warning { color: #f59e0b !important; }
.text-error { color: #ef4444 !important; }

.main-card { border-radius: 12px; box-shadow: 0 4px 16px rgba(0,0,0,0.05); }
.unified-admin-table :deep(.n-data-table-th),
.unified-admin-table :deep(.n-data-table-td) {
  text-align: left;
}
.unified-admin-table :deep(.n-data-table-td__content) {
  justify-content: flex-start;
  text-align: left;
}

.cell-block { display: flex; flex-direction: column; align-items: flex-start; gap: 4px; text-align: left; }
.cell-title { font-weight: 600; color: #1f2937; }
.cell-sub { font-size: 12px; color: #6b7280; }
.cell-inline { display: flex; align-items: center; justify-content: flex-start; gap: 6px; text-align: left; }
.inline-icon { color: #94a3b8; }
.order-no { font-family: monospace; font-size: 13px; }
.email-text, .summary-text, .plain-text { color: #334155; }
.amount-text { font-weight: 700; color: #10b981; font-size: 14px; }
.time-text { color: #64748b; font-size: 13px; }
.left-text { justify-content: flex-start; }

.detail-header { display: flex; justify-content: space-between; align-items: flex-end; margin-bottom: 20px; padding: 0 4px; }
.amount-display .label { font-size: 12px; color: #888; margin-bottom: 4px; }
.amount-display .value { font-size: 32px; font-weight: 800; color: #10b981; line-height: 1; }
.copyable-row { display: flex; align-items: center; gap: 8px; }
.order-no-code { background: #f5f5f5; padding: 2px 6px; border-radius: 4px; font-family: monospace; font-size: 12px; }
.gateway-no { font-size: 11px; color: #666; word-break: break-all; }

@media (max-width: 767px) {
  .admin-page-shell { padding: 12px; }
  .page-header { flex-direction: column; align-items: flex-start; gap: 16px; }
}
</style>
