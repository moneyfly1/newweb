<template>
  <div class="admin-orders-page">
    <n-card title="订单管理" :bordered="false" class="page-card">
      <n-space vertical :size="16">
        <n-space>
          <n-input
            v-model:value="searchQuery"
            placeholder="搜索订单号"
            clearable
            style="width: 300px"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <n-icon :component="SearchOutline" />
            </template>
          </n-input>
          <n-select
            v-model:value="statusFilter"
            placeholder="状态筛选"
            clearable
            style="width: 150px"
            :options="statusOptions"
            @update:value="handleSearch"
          />
          <n-button type="primary" @click="handleSearch">
            <template #icon>
              <n-icon :component="SearchOutline" />
            </template>
            搜索
          </n-button>
        </n-space>

        <n-data-table
          :columns="columns"
          :data="orders"
          :loading="loading"
          :pagination="false"
          :bordered="false"
          :single-line="false"
          :scroll-x="1400"
        />

        <n-pagination
          v-model:page="currentPage"
          v-model:page-size="pageSize"
          :page-count="totalPages"
          :page-sizes="[10, 20, 50, 100]"
          show-size-picker
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </n-space>
    </n-card>

    <n-modal
      v-model:show="showDetailModal"
      preset="card"
      title="订单详情"
      style="width: 600px"
      :bordered="false"
    >
      <n-descriptions
        v-if="currentOrder"
        label-placement="left"
        :column="1"
        bordered
      >
        <n-descriptions-item label="订单号">
          {{ currentOrder.order_no }}
        </n-descriptions-item>
        <n-descriptions-item label="用户ID">
          {{ currentOrder.user_id }}
        </n-descriptions-item>
        <n-descriptions-item label="订单金额">
          ¥{{ currentOrder.amount.toFixed(2) }}
        </n-descriptions-item>
        <n-descriptions-item label="优惠金额">
          ¥{{ currentOrder.discount_amount != null ? Number(currentOrder.discount_amount).toFixed(2) : '0.00' }}
        </n-descriptions-item>
        <n-descriptions-item label="实付金额">
          <span style="color: #18a058; font-weight: 600; font-size: 16px">
            ¥{{ currentOrder.final_amount != null ? Number(currentOrder.final_amount).toFixed(2) : '0.00' }}
          </span>
        </n-descriptions-item>
        <n-descriptions-item label="支付方式">
          {{ getPaymentMethodText(currentOrder.payment_method_name) }}
        </n-descriptions-item>
        <n-descriptions-item label="订单状态">
          <n-tag :type="getStatusType(currentOrder.status)" size="small">
            {{ getStatusText(currentOrder.status) }}
          </n-tag>
        </n-descriptions-item>
        <n-descriptions-item label="创建时间">
          {{ new Date(currentOrder.created_at).toLocaleString('zh-CN') }}
        </n-descriptions-item>
      </n-descriptions>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { NButton, NTag, NSpace, NIcon, useMessage, useDialog } from 'naive-ui'
import { SearchOutline } from '@vicons/ionicons5'
import { listAdminOrders, refundOrder } from '@/api/admin'

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const orders = ref([])
const searchQuery = ref('')
const statusFilter = ref(null)
const currentPage = ref(1)
const pageSize = ref(20)
const totalPages = ref(0)

const showDetailModal = ref(false)
const currentOrder = ref(null)

const statusOptions = [
  { label: '全部', value: null },
  { label: '待支付', value: 'pending' },
  { label: '已支付', value: 'paid' },
  { label: '已取消', value: 'cancelled' },
  { label: '已退款', value: 'refunded' }
]

const getStatusType = (status) => {
  const typeMap = {
    pending: 'warning',
    paid: 'success',
    cancelled: 'default',
    refunded: 'error'
  }
  return typeMap[status] || 'default'
}

const getStatusText = (status) => {
  const textMap = {
    pending: '待支付',
    paid: '已支付',
    cancelled: '已取消',
    refunded: '已退款'
  }
  return textMap[status] || status
}

const getPaymentMethodText = (method) => {
  const methodMap = {
    alipay: '支付宝',
    wechat: '微信支付',
    balance: '余额支付',
    yipay: '易支付',
    stripe: 'Stripe',
    paypal: 'PayPal'
  }
  return methodMap[method] || method
}

const columns = [
  { title: 'ID', key: 'id', width: 80, fixed: 'left' },
  {
    title: '订单号',
    key: 'order_no',
    width: 200,
    ellipsis: { tooltip: true },
    fixed: 'left'
  },
  { title: '用户ID', key: 'user_id', width: 100 },
  {
    title: '订单金额',
    key: 'amount',
    width: 120,
    render: (row) => `¥${row.amount.toFixed(2)}`
  },
  {
    title: '优惠金额',
    key: 'discount_amount',
    width: 120,
    render: (row) => row.discount_amount != null ? `¥${Number(row.discount_amount).toFixed(2)}` : '¥0.00'
  },
  {
    title: '实付金额',
    key: 'final_amount',
    width: 120,
    render: (row) => h(
      'span',
      { style: 'color: #18a058; font-weight: 600' },
      row.final_amount != null ? `¥${Number(row.final_amount).toFixed(2)}` : '¥0.00'
    )
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row) => h(
      NTag,
      { type: getStatusType(row.status), size: 'small' },
      { default: () => getStatusText(row.status) }
    )
  },
  {
    title: '支付方式',
    key: 'payment_method_name',
    width: 120,
    render: (row) => getPaymentMethodText(row.payment_method_name)
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 180,
    render: (row) => new Date(row.created_at).toLocaleString('zh-CN')
  },
  {
    title: '操作',
    key: 'actions',
    width: 180,
    fixed: 'right',
    render: (row) => h(
      NSpace,
      {},
      {
        default: () => [
          h(
            NButton,
            {
              size: 'small',
              type: 'info',
              onClick: () => handleViewDetail(row)
            },
            { default: () => '查看详情' }
          ),
          row.status === 'paid' && h(
            NButton,
            {
              size: 'small',
              type: 'error',
              onClick: () => handleRefund(row)
            },
            { default: () => '退款' }
          )
        ].filter(Boolean)
      }
    )
  }
]

const fetchOrders = async () => {
  loading.value = true
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value,
      order_no: searchQuery.value || undefined,
      status: statusFilter.value
    }
    const response = await listAdminOrders(params)
    orders.value = response.data.items || []
    totalPages.value = Math.ceil((response.data.total || 0) / pageSize.value)
  } catch (error) {
    message.error('获取订单列表失败：' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  currentPage.value = 1
  fetchOrders()
}

const handlePageChange = (page) => {
  currentPage.value = page
  fetchOrders()
}

const handlePageSizeChange = (size) => {
  pageSize.value = size
  currentPage.value = 1
  fetchOrders()
}

const handleViewDetail = (row) => {
  currentOrder.value = row
  showDetailModal.value = true
}

const handleRefund = (row) => {
  dialog.warning({
    title: '确认退款',
    content: `确定要退款订单 ${row.order_no} 吗？退款金额为 ¥${row.final_amount != null ? Number(row.final_amount).toFixed(2) : '0.00'}`,
    positiveText: '确定退款',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await refundOrder(row.id)
        message.success('退款成功')
        fetchOrders()
      } catch (error) {
        message.error('退款失败：' + (error.message || '未知错误'))
      }
    }
  })
}

onMounted(() => {
  fetchOrders()
})
</script>

<style scoped>
.admin-orders-page {
  padding: 20px;
}

.page-card {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

:deep(.n-data-table) {
  font-size: 14px;
}

:deep(.n-data-table .n-data-table-th) {
  font-weight: 600;
}

@media (max-width: 767px) {
  .admin-orders-page { padding: 8px; }
}
</style>
