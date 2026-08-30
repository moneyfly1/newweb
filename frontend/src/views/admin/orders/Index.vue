<template>
  <div class="admin-orders-page admin-page-shell">
    <div class="page-header" v-if="!appStore.isMobile">
      <div class="header-left">
        <h2 class="page-title">订单管理</h2>
        <p class="page-subtitle">查看和处理全站订单，支持退款、状态调整及财务对账</p>
      </div>
      <div class="header-right">
        <n-space>
          <n-button secondary @click="fetchOrders">
            <template #icon><n-icon><refresh-outline /></n-icon></template>
            刷新
          </n-button>
        </n-space>
      </div>
    </div>

    <div v-else class="mobile-toolbar">
      <div class="mobile-toolbar-title">订单管理</div>
      <div class="mobile-toolbar-controls">
        <div class="mobile-toolbar-row">
          <n-button size="small" @click="fetchOrders">
            <template #icon><n-icon><refresh-outline /></n-icon></template>
            刷新
          </n-button>
        </div>
      </div>
    </div>

    <!-- 统一搜索筛选工具栏（SearchFilterBar 组件，桌面单行不换行） -->
    <search-filter-bar
      v-model:values="filterValues"
      :filters="filterConfig"
      search-placeholder="订单号 / 用户ID / 邮箱"
      @search="handleSearch"
    />

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
        <n-space v-if="checkedRowKeys.length > 0 && !appStore.isMobile" align="center" class="batch-operations">
          <span class="batch-selected-text">已选择 {{ checkedRowKeys.length }} 项</span>
          <n-button size="small" type="success" @click="handleBatchMarkPaid">批量标记付款并开通</n-button>
          <n-button size="small" type="warning" @click="handleBatchCancel">批量取消</n-button>
          <n-button size="small" type="info" @click="handleBatchComplete">批量完成</n-button>
          <n-button size="small" type="error" @click="handleBatchRefund">批量退款</n-button>
          <n-button size="small" tertiary type="error" @click="handleBatchDelete">批量删除</n-button>
        </n-space>

        <n-data-table
          v-if="!appStore.isMobile"
          remote
          :columns="columns"
          :data="orders"
          :loading="loading"
          :pagination="pagination"
          :bordered="false"
          :single-line="false"
          :row-key="getRowKey"
          :checked-row-keys="checkedRowKeys"
          :scroll-x="1450"
          class="unified-admin-table"
          @update:checked-row-keys="(keys: Array<string | number>) => { checkedRowKeys = keys as string[] }"
          @update:page="(p: number) => { pagination.page = p; fetchOrders() }"
          @update:page-size="(ps: number) => { pagination.pageSize = ps; pagination.page = 1; fetchOrders() }"
        />

        <template v-else>
          <n-spin :show="loading">
            <div v-if="orders.length === 0" class="mobile-empty">暂无订单</div>
            <div v-else class="mobile-card-list">
              <div
                v-for="order in orders"
                :key="order.id"
                class="mobile-card order-mobile-card"
                @click="handleViewDetail(order)"
              >
                <div class="card-header">
                  <div class="card-title-block">
                    <div class="card-title mono">{{ order.order_no }}</div>
                    <div class="card-sub">UID: {{ order.user_id }} · {{ order.user_email || '-' }}</div>
                  </div>
                  <n-tag :type="getStatusType(order.status)" size="small">{{ getStatusText(order.status) }}</n-tag>
                </div>
                <div class="card-body">
                  <div class="card-row">
                    <span class="card-label">订单类型</span>
                    <span>{{ getOrderTypeText(order) }}</span>
                  </div>
                  <div class="card-row">
                    <span class="card-label">订单内容</span>
                    <span class="card-value-strong">{{ getOrderSummary(order) }}</span>
                  </div>
                  <div class="card-row">
                    <span class="card-label">实付金额</span>
                    <span class="amount-text">{{ formatCurrency(order.final_amount || order.amount || 0) }}</span>
                  </div>
                  <div class="card-row">
                    <span class="card-label">支付方式</span>
                    <span>{{ getPaymentMethodText(order) }}</span>
                  </div>
                  <div class="card-row">
                    <span class="card-label">创建时间</span>
                    <span>{{ formatDateTime(order.created_at) }}</span>
                  </div>
                </div>
                <div class="card-actions" @click.stop>
                  <n-button size="small" quaternary type="info" @click="handleViewDetail(order)">详情</n-button>
                  <n-button v-if="canMarkPaid(order)" size="small" type="success" @click="handleMarkPaid(order)">标记付款</n-button>
                  <n-button v-if="canCancel(order)" size="small" type="error" @click="handleCancel(order)">取消</n-button>
                  <n-button v-if="canComplete(order)" size="small" type="info" @click="handleComplete(order)">完成</n-button>
                  <n-button v-if="canRefund(order)" size="small" type="warning" @click="handleRefund(order)">退款</n-button>
                </div>
              </div>
            </div>
          </n-spin>
          <n-pagination
            v-if="orders.length > 0"
            v-model:page="pagination.page"
            v-model:page-size="pagination.pageSize"
            :item-count="pagination.itemCount"
            :page-sizes="pagination.pageSizes"
            :show-size-picker="pagination.showSizePicker"
            style="margin-top: 16px; justify-content: flex-end"
            @update:page="() => fetchOrders()"
            @update:page-size="() => { pagination.page = 1; fetchOrders() }"
          />
        </template>
      </n-space>
    </n-card>

    <common-drawer v-model:show="showDetailDrawer" title="订单流水详情" :width="appStore.isMobile ? '100%' : 540">
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
          <n-descriptions-item v-if="currentOrder.order_type === 'subscription_upgrade' && currentOrder.current_device_limit != null" label="设备上限变更">
            <span style="color: #999; text-decoration: line-through;">{{ currentOrder.current_device_limit }} 台</span>
            <span style="margin: 0 6px; color: #667eea;">→</span>
            <span style="color: #18a058; font-weight: 600;">{{ currentOrder.new_device_limit }} 台</span>
          </n-descriptions-item>
          <n-descriptions-item v-if="currentOrder.order_type === 'subscription_upgrade' && currentOrder.current_expire_time" label="到期时间变更">
            <div>
              <span style="color: #999; text-decoration: line-through; font-size: 13px;">{{ currentOrder.current_expire_time }}</span>
            </div>
            <div>
              <span style="margin: 0 6px; color: #667eea;">→</span>
              <span style="color: #18a058; font-weight: 600; font-size: 13px;">{{ currentOrder.new_expire_time }}</span>
            </div>
          </n-descriptions-item>
          <n-descriptions-item label="原始金额">{{ formatCurrency(currentOrder.amount) }}</n-descriptions-item>
          <n-descriptions-item label="优惠抵扣" v-if="currentOrder.discount_amount > 0">
            - {{ formatCurrency(currentOrder.discount_amount) }}
          </n-descriptions-item>
          <n-descriptions-item label="支付网关">{{ getPaymentMethodText(currentOrder) }}</n-descriptions-item>
          <n-descriptions-item label="商户订单号" v-if="currentOrder.payment_transaction_id">
            <div class="copyable-row wrap-copyable-row">
              <code class="gateway-no">{{ currentOrder.payment_transaction_id }}</code>
              <n-button size="tiny" quaternary @click="copyToClipboard(currentOrder.payment_transaction_id)">复制</n-button>
            </div>
          </n-descriptions-item>
          <n-descriptions-item label="平台流水号" v-if="currentOrder.gateway_trade_no">
            <div class="copyable-row wrap-copyable-row">
              <code class="gateway-no">{{ currentOrder.gateway_trade_no }}</code>
              <n-button size="tiny" quaternary @click="copyToClipboard(currentOrder.gateway_trade_no)">复制</n-button>
            </div>
          </n-descriptions-item>
          <n-descriptions-item label="创建时间">{{ formatFullDateTime(currentOrder.created_at) }}</n-descriptions-item>
          <n-descriptions-item label="支付时间" v-if="currentOrder.payment_time">
            {{ formatFullDateTime(currentOrder.payment_time) }}
          </n-descriptions-item>
        </n-descriptions>

        <div class="detail-actions">
          <n-divider />
          <n-space :justify="appStore.isMobile ? 'start' : 'end'" :wrap="true">
            <n-button v-if="canMarkPaid(currentOrder)" type="success" @click="handleMarkPaid(currentOrder)">标记已付款并开通</n-button>
            <n-button v-if="canCancel(currentOrder)" type="error" ghost @click="handleCancel(currentOrder)">取消订单</n-button>
            <n-button v-if="canComplete(currentOrder)" type="info" @click="handleComplete(currentOrder)">标记完成</n-button>
            <n-button v-if="canRefund(currentOrder)" type="warning" @click="handleRefund(currentOrder)">全额退款</n-button>
            <n-button v-if="canDelete(currentOrder)" type="error" tertiary @click="handleDelete(currentOrder)">删除记录</n-button>
          </n-space>
        </div>
      </div>
    </common-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, h, onMounted, watch } from 'vue'
import { usePageLoading } from '@/composables/usePageLoading'
import { NButton, NTag, NSpace, NIcon, NSelect, useMessage, useDialog, type DataTableColumns, type TagProps } from 'naive-ui'
import { RefreshOutline, ReceiptOutline, TimeOutline, MailOutline, LayersOutline } from '@vicons/ionicons5'
import { listAdminOrders, refundOrder, cancelOrder, completeOrder, deleteOrder, markOrderPaid, batchOrderAction, getAdminDashboard } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import CommonDrawer from '@/components/CommonDrawer.vue'
import SearchFilterBar from '@/components/SearchFilterBar.vue'
import { useRoute } from 'vue-router'
import { formatCurrency } from '@/utils/amount'
import { copyToClipboard as clipboardCopy } from '@/utils/clipboard'
import { formatDateTime, formatFullDateTime } from '@/utils/date'
import '@/styles/admin-common.css'

const message = useMessage()
const dialog = useDialog()
const appStore = useAppStore()
const route = useRoute()

const { loading, beginLoad, endLoad } = usePageLoading()
const orders = ref<any[]>([])
const orderStats = ref<any>({})
const searchQuery = ref((route.query.search as string) || '')
const statusFilter = ref(null)
const pagination = reactive({ page: 1, pageSize: 10, itemCount: 0, showSizePicker: true, pageSizes: [10, 20, 50, 100] })
const checkedRowKeys = ref<string[]>([])

const showDetailDrawer = ref(false)
const currentOrder = ref<any>(null)

const statusOptions = [
  { label: '待支付', value: 'pending' },
  { label: '已支付', value: 'paid' },
  { label: '已完成', value: 'completed' },
  { label: '已过期', value: 'expired' },
  { label: '已取消', value: 'cancelled' },
  { label: '已退款', value: 'refunded' }
]

// 统一筛选工具栏状态（值与原 refs 同步，保持业务逻辑不变）
// searchQuery 可能来自 URL 参数（route.query.search），filterValues 初始值需保持一致
const filterValues = reactive({
  search: searchQuery.value,
  status: null,
})
const filterConfig = [
  { key: 'status', placeholder: '所有状态', options: statusOptions },
]

const getStatusType = (s: string): TagProps['type'] => {
  const typeMap: Record<string, NonNullable<TagProps['type']>> = {
    pending: 'warning',
    paid: 'success',
    completed: 'info',
    expired: 'warning',
    cancelled: 'default',
    refunded: 'error'
  }
  return typeMap[s] || 'default'
}

const getStatusText = (s: string) => ({ pending: '待支付', paid: '已支付', completed: '已完成', expired: '已过期', cancelled: '已取消', refunded: '已退款' }[s] || s)
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
const getRowKey = (row: any) => `${row.order_type || 'package'}:${row.id}`
const isPackageOrder = (row: any) => row && row.order_type !== 'recharge'
const canMarkPaid = (row: any) => isPackageOrder(row) && ['pending', 'expired', 'cancelled'].includes(row.status)
const canCancel = (row: any) => isPackageOrder(row) && ['pending', 'expired'].includes(row.status)
const canComplete = (row: any) => isPackageOrder(row) && row.status === 'paid'
const canRefund = (row: any) => isPackageOrder(row) && ['paid', 'completed'].includes(row.status)
const canDelete = (row: any) => isPackageOrder(row) && ['cancelled', 'refunded'].includes(row.status)

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
      h('span', formatDateTime(row.created_at))
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
  beginLoad(orders.value.length > 0)
  try {
    const res = await listAdminOrders({ page: pagination.page, page_size: pagination.pageSize, search: searchQuery.value || undefined, status: statusFilter.value })
    orders.value = res.data.items || []
    pagination.itemCount = res.data.total || 0
    if (pagination.page === 1) {
      const dashRes = await getAdminDashboard()
      orderStats.value = dashRes.data
    }
  } finally {
    endLoad()
  }
}

const handleSearch = () => {
  searchQuery.value = filterValues.search || ''
  statusFilter.value = filterValues.status
  pagination.page = 1
  fetchOrders()
}
const handleViewDetail = (row: any) => { currentOrder.value = row; showDetailDrawer.value = true }

const handleRefund = (row: any) => {
  dialog.warning({
    title: '确认全额退款',
    content: `订单 ${row.order_no} 将退款 ${formatCurrency(row.final_amount || row.amount)}。线上支付会原路退回支付渠道，余额支付才退回用户余额。`,
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
    content: '确定要取消此订单吗？取消后用户不能继续支付此订单。',
    positiveText: '确定',
    onPositiveClick: async () => {
      await cancelOrder(row.id)
      message.success('已取消')
      showDetailDrawer.value = false
      fetchOrders()
    }
  })
}

const handleMarkPaid = (row: any) => {
  dialog.warning({
    title: '标记已付款并开通权限',
    content: `确认订单 ${row.order_no} 已线下收款或需要人工放行？系统会立即开通/续期/升级对应订阅权限。`,
    positiveText: '确认开通',
    onPositiveClick: async () => {
      await markOrderPaid(row.id)
      message.success('已标记付款并开通权限')
      showDetailDrawer.value = false
      fetchOrders()
    }
  })
}

const handleComplete = (row: any) => {
  dialog.info({
    title: '手动标记完成',
    content: '此操作只将已支付订单标记为完成，不会重复开通订阅。',
    positiveText: '确定完成',
    onPositiveClick: async () => {
      await completeOrder(row.id)
      message.success('已标记为完成')
      showDetailDrawer.value = false
      fetchOrders()
    }
  })
}

const handleDelete = (row: any) => {
  dialog.warning({
    title: '删除订单记录',
    content: `确定删除订单 ${row.order_no} 吗？只能删除已取消或已退款的订单记录。`,
    positiveText: '删除',
    onPositiveClick: async () => {
      await deleteOrder(row.id)
      message.success('订单记录已删除')
      showDetailDrawer.value = false
      fetchOrders()
    }
  })
}

const getSelectedOrders = () => orders.value.filter(o => checkedRowKeys.value.includes(getRowKey(o)))

const handleBatchMarkPaid = () => {
  const markable = getSelectedOrders().filter(canMarkPaid)
  if (markable.length === 0) {
    message.warning('没有可标记付款的套餐订单')
    return
  }
  dialog.warning({
    title: '批量标记付款并开通',
    content: `确定要将 ${markable.length} 个订单标记为已付款并开通对应权限吗？`,
    positiveText: '确认开通',
    onPositiveClick: async () => {
      const res = await batchOrderAction({ ids: markable.map(o => o.id), action: 'mark_paid' })
      message.success(`批量开通完成：成功 ${res.data.success} 个，失败 ${res.data.failed} 个`)
      checkedRowKeys.value = []
      fetchOrders()
    }
  })
}

const handleBatchCancel = () => {
  const cancellable = getSelectedOrders().filter(canCancel)
  if (cancellable.length === 0) {
    message.warning('没有可取消的订单')
    return
  }
  dialog.warning({
    title: '批量取消订单',
    content: `确定要取消选中的 ${cancellable.length} 个订单吗？`,
    positiveText: '确定',
    onPositiveClick: async () => {
      const res = await batchOrderAction({ ids: cancellable.map(o => o.id), action: 'cancel' })
      message.success(`批量取消完成：成功 ${res.data.success} 个，失败 ${res.data.failed} 个`)
      checkedRowKeys.value = []
      fetchOrders()
    }
  })
}

const handleBatchComplete = () => {
  const completable = getSelectedOrders().filter(canComplete)
  if (completable.length === 0) {
    message.warning('没有可完成的已支付订单')
    return
  }
  dialog.info({
    title: '批量标记完成',
    content: `确定要将 ${completable.length} 个已支付订单标记为完成吗？`,
    positiveText: '确定完成',
    onPositiveClick: async () => {
      const res = await batchOrderAction({ ids: completable.map(o => o.id), action: 'complete' })
      message.success(`批量完成：成功 ${res.data.success} 个，失败 ${res.data.failed} 个`)
      checkedRowKeys.value = []
      fetchOrders()
    }
  })
}

const handleBatchRefund = () => {
  const refundable = getSelectedOrders().filter(canRefund)
  if (refundable.length === 0) {
    message.warning('没有可退款的订单')
    return
  }
  dialog.warning({
    title: '批量退款',
    content: `确定要退款选中的 ${refundable.length} 个订单吗？线上支付会逐笔原路退回，余额支付才退回余额。`,
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

const handleBatchDelete = () => {
  const deletable = getSelectedOrders().filter(canDelete)
  if (deletable.length === 0) {
    message.warning('没有可删除的已取消/已退款订单')
    return
  }
  dialog.warning({
    title: '批量删除订单',
    content: `确定要删除选中的 ${deletable.length} 个订单记录吗？`,
    positiveText: '确定删除',
    onPositiveClick: async () => {
      const res = await batchOrderAction({ ids: deletable.map(o => o.id), action: 'delete' })
      message.success(`批量删除完成：成功 ${res.data.success} 个，失败 ${res.data.failed} 个`)
      checkedRowKeys.value = []
      fetchOrders()
    }
  })
}

const copyToClipboard = async (text: string) => {
  const ok = await clipboardCopy(text)
  ok ? message.success('已复制到剪贴板') : message.error('复制失败')
}

watch(() => route.query.search, (searchVal) => {
  if (typeof searchVal === 'string' && searchVal !== searchQuery.value) {
    searchQuery.value = searchVal
    filterValues.search = searchVal
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

.mobile-empty { text-align: center; color: #999; padding: 40px 0; }
.order-mobile-card { cursor: pointer; }
.card-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; padding: 12px 14px; border-bottom: 1px solid var(--border-color, #f0f0f0); }
.card-title-block { min-width: 0; }
.card-title { font-weight: 600; color: var(--text-color, #333); }
.card-title.mono { font-family: monospace; font-size: 13px; word-break: break-all; }
.card-sub { margin-top: 4px; font-size: 12px; color: var(--text-color-secondary, #999); word-break: break-all; }
.card-body { padding: 10px 14px; }
.card-row { display: flex; justify-content: space-between; gap: 12px; padding: 4px 0; font-size: 13px; }
.card-row > span:last-child { text-align: right; color: var(--text-color, #333); word-break: break-word; }
.card-label { color: var(--text-color-secondary, #999); flex-shrink: 0; }
.card-value-strong { font-weight: 500; }

.detail-header { display: flex; justify-content: space-between; align-items: flex-end; margin-bottom: 20px; padding: 0 4px; gap: 12px; }
.amount-display .label { font-size: 12px; color: #888; margin-bottom: 4px; }
.amount-display .value { font-size: 32px; font-weight: 800; color: #10b981; line-height: 1; }
.copyable-row { display: flex; align-items: center; gap: 8px; }
.wrap-copyable-row { align-items: flex-start; flex-wrap: wrap; }
.order-no-code { background: #f5f5f5; padding: 2px 6px; border-radius: 4px; font-family: monospace; font-size: 12px; }
.gateway-no { font-size: 11px; color: #666; word-break: break-all; }

@media (max-width: 767px) {
  .admin-page-shell { padding: 12px; }
  .stats-summary { margin-bottom: 16px; }
  .mini-stat-card { padding: 14px 12px; }
  .mini-stat-card .stat-value { font-size: 18px; }
  .mobile-toolbar-row :deep(.n-select),
  .mobile-toolbar-row :deep(.n-base-selection) {
    min-width: 0;
  }
  .mobile-toolbar-row > *:first-child {
    flex: 1;
    min-width: 0;
  }
  .mobile-toolbar-row > .n-button {
    flex-shrink: 0;
  }
  .detail-header {
    flex-direction: column;
    align-items: flex-start;
  }
  .amount-display .value {
    font-size: 28px;
  }
}
</style>
