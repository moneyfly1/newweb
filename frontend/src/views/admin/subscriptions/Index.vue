<template>
  <div class="subscriptions-container">
    <n-card title="订阅管理">
      <template #header-extra>
        <n-space>
          <n-input
            v-model:value="searchQuery"
            placeholder="搜索用户/邮箱/订阅URL"
            clearable
            style="width: 240px"
            @keyup.enter="handleSearch"
          >
            <template #prefix><n-icon><SearchOutline /></n-icon></template>
          </n-input>
          <n-select
            v-model:value="statusFilter"
            :options="statusOptions"
            style="width: 130px"
            @update:value="handleSearch"
          />
          <n-button @click="handleRefresh">
            <template #icon><n-icon><RefreshOutline /></n-icon></template>
            刷新
          </n-button>
        </n-space>
      </template>

      <n-data-table
        remote
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :pagination="pagination"
        :bordered="false"
        :scroll-x="1400"
        @update:page="(p) => { pagination.page = p; fetchData() }"
        @update:page-size="(ps) => { pagination.pageSize = ps; pagination.page = 1; fetchData() }"
      />
    </n-card>

    <!-- Detail Modal -->
    <n-modal v-model:show="showDetailModal" title="订阅详情" preset="card" style="width: 720px">
      <n-descriptions bordered :column="2" label-placement="left">
        <n-descriptions-item label="ID">{{ detailData.id }}</n-descriptions-item>
        <n-descriptions-item label="用户">{{ detailData.user_email || '-' }}</n-descriptions-item>
        <n-descriptions-item label="套餐">{{ detailData.package_name || '-' }}</n-descriptions-item>
        <n-descriptions-item label="状态">
          <n-tag :type="getStatusType(detailData.status)" size="small">{{ getStatusText(detailData.status) }}</n-tag>
        </n-descriptions-item>
        <n-descriptions-item label="设备">{{ detailData.current_devices || 0 }} / {{ detailData.device_limit || 0 }}</n-descriptions-item>
        <n-descriptions-item label="剩余天数">
          <span :style="{ color: getRemainingDaysColor(detailData.expire_time), fontWeight: 'bold' }">{{ getRemainingDays(detailData.expire_time) }}</span>
        </n-descriptions-item>
        <n-descriptions-item label="到期时间" :span="2">{{ formatDate(detailData.expire_time) }}</n-descriptions-item>
        <n-descriptions-item label="通用次数">{{ detailData.universal_count || 0 }}</n-descriptions-item>
        <n-descriptions-item label="Clash次数">{{ detailData.clash_count || 0 }}</n-descriptions-item>
      </n-descriptions>

      <n-divider />

      <!-- Subscription URLs -->
      <div class="sub-url-section" v-if="detailData.universal_url">
        <div class="sub-url-row">
          <span class="sub-url-label">通用订阅</span>
          <code class="sub-url-text">{{ detailData.universal_url }}</code>
          <n-button size="tiny" @click="copyText(detailData.universal_url)">复制</n-button>
        </div>
        <div class="sub-url-row">
          <span class="sub-url-label">Clash 订阅</span>
          <code class="sub-url-text">{{ detailData.clash_url }}</code>
          <n-button size="tiny" @click="copyText(detailData.clash_url)">复制</n-button>
        </div>
      </div>

      <!-- QR Codes -->
      <div class="qr-section" v-if="detailData.universal_url">
        <n-divider />
        <div class="qr-grid">
          <div class="qr-item">
            <div class="qr-title">Shadowrocket 导入</div>
            <canvas :ref="(el) => renderQR(el, getShadowrocketUrl(detailData.universal_url))"></canvas>
            <n-button size="tiny" style="margin-top: 8px" @click="copyText(getShadowrocketUrl(detailData.universal_url))">复制链接</n-button>
          </div>
          <div class="qr-item">
            <div class="qr-title">通用订阅</div>
            <canvas :ref="(el) => renderQR(el, detailData.universal_url)"></canvas>
            <n-button size="tiny" style="margin-top: 8px" @click="copyText(detailData.universal_url)">复制链接</n-button>
          </div>
          <div class="qr-item">
            <div class="qr-title">Clash 订阅</div>
            <canvas :ref="(el) => renderQR(el, detailData.clash_url)"></canvas>
            <n-button size="tiny" style="margin-top: 8px" @click="copyText(detailData.clash_url)">复制链接</n-button>
          </div>
        </div>
      </div>
    </n-modal>

    <!-- Extend Modal -->
    <n-modal v-model:show="showExtendModal" title="延长订阅" preset="card" style="width: 400px">
      <n-space vertical>
        <n-space>
          <n-button @click="extendDays = 180">+半年</n-button>
          <n-button @click="extendDays = 365">+一年</n-button>
          <n-button @click="extendDays = 730">+两年</n-button>
        </n-space>
        <n-input-number v-model:value="extendDays" :min="1" placeholder="天数" style="width: 100%">
          <template #prefix>延长</template>
          <template #suffix>天</template>
        </n-input-number>
      </n-space>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 12px">
          <n-button @click="showExtendModal = false">取消</n-button>
          <n-button type="primary" @click="handleExtendSubmit" :loading="submitting">确定</n-button>
        </div>
      </template>
    </n-modal>

    <!-- Device Limit Modal -->
    <n-modal v-model:show="showDeviceLimitModal" title="修改设备限制" preset="card" style="width: 400px">
      <n-input-number v-model:value="newDeviceLimit" :min="1" :max="100" style="width: 100%">
        <template #prefix>设备上限</template>
        <template #suffix>台</template>
      </n-input-number>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 12px">
          <n-button @click="showDeviceLimitModal = false">取消</n-button>
          <n-button type="primary" @click="handleDeviceLimitSubmit" :loading="submitting">确定</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>


<script setup>
import { ref, reactive, h, onMounted, nextTick } from 'vue'
import { NButton, NTag, NSpace, NIcon, NTooltip, useMessage, useDialog } from 'naive-ui'
import { SearchOutline, RefreshOutline } from '@vicons/ionicons5'
import QRCode from 'qrcode'
import {
  listAdminSubscriptions, getAdminSubscription,
  resetAdminSubscription, extendSubscription, updateSubscriptionDeviceLimit
} from '@/api/admin'

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const submitting = ref(false)
const showDetailModal = ref(false)
const showExtendModal = ref(false)
const showDeviceLimitModal = ref(false)
const tableData = ref([])
const searchQuery = ref('')
const statusFilter = ref('all')
const detailData = ref({})
const extendDays = ref(365)
const newDeviceLimit = ref(3)
const currentEditId = ref(null)

const statusOptions = [
  { label: '全部状态', value: 'all' },
  { label: '正常', value: 'active' },
  { label: '已过期', value: 'expired' },
  { label: '未激活', value: 'inactive' }
]

const pagination = reactive({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
})

const getStatusType = (status) => {
  const map = { active: 'success', expired: 'error', inactive: 'warning' }
  return map[status] || 'default'
}
const getStatusText = (status) => {
  const map = { active: '正常', expired: '已过期', inactive: '未激活' }
  return map[status] || status || '未知'
}
const getRemainingDays = (expireTime) => {
  if (!expireTime) return '-'
  const days = Math.ceil((new Date(expireTime) - Date.now()) / 86400000)
  return days > 0 ? `${days}天` : '已过期'
}
const getRemainingDaysColor = (expireTime) => {
  if (!expireTime) return '#999'
  const days = Math.ceil((new Date(expireTime) - Date.now()) / 86400000)
  if (days <= 0) return '#e03050'
  if (days <= 7) return '#f0a020'
  return '#18a058'
}
const formatDate = (d) => d ? new Date(d).toLocaleString('zh-CN') : '-'
const copyText = (text) => {
  if (!text) return
  navigator.clipboard.writeText(text).then(() => message.success('已复制'))
}

const getShadowrocketUrl = (universalUrl) => {
  if (!universalUrl) return ''
  return 'sub://' + btoa(universalUrl)
}

const renderQR = (canvas, text) => {
  if (!canvas || !text) return
  QRCode.toCanvas(canvas, text, { width: 180, margin: 2 }, (err) => {
    if (err) console.error('QR render error:', err)
  })
}

const quickExtend = async (row, days) => {
  try {
    await extendSubscription(row.id, { days })
    message.success(`已延长 ${days} 天`)
    fetchData()
  } catch (error) {
    message.error(error.message || '延期失败')
  }
}

const quickAddDevices = async (row, count) => {
  try {
    await updateSubscriptionDeviceLimit(row.id, { device_limit: (row.device_limit || 0) + count })
    message.success(`设备限制已增加 ${count}`)
    fetchData()
  } catch (error) {
    message.error(error.message || '更新失败')
  }
}

const columns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '用户', key: 'user_email', ellipsis: { tooltip: true }, width: 180 },
  {
    title: '套餐', key: 'package_name', width: 120,
    render: (row) => row.package_name || '-'
  },
  {
    title: '状态', key: 'status', width: 80,
    render: (row) => h(NTag, { type: getStatusType(row.status), size: 'small' }, { default: () => getStatusText(row.status) })
  },
  {
    title: '设备', key: 'current_devices', width: 200,
    render: (row) => h(NSpace, { size: 4, align: 'center' }, {
      default: () => [
        h('span', {}, `${row.current_devices || 0}/${row.device_limit || 0}`),
        h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => quickAddDevices(row, 5) }, { default: () => '+5' }),
        h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => quickAddDevices(row, 10) }, { default: () => '+10' }),
        h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => quickAddDevices(row, 15) }, { default: () => '+15' }),
      ]
    })
  },
  {
    title: '到期/剩余', key: 'expire_time', width: 280,
    render: (row) => h(NSpace, { size: 4, align: 'center' }, {
      default: () => [
        h('span', {
          style: { color: getRemainingDaysColor(row.expire_time), fontWeight: 'bold', minWidth: '50px', display: 'inline-block' }
        }, getRemainingDays(row.expire_time)),
        h(NButton, { size: 'tiny', quaternary: true, type: 'info', onClick: () => quickExtend(row, 180) }, { default: () => '+半年' }),
        h(NButton, { size: 'tiny', quaternary: true, type: 'info', onClick: () => quickExtend(row, 365) }, { default: () => '+一年' }),
        h(NButton, { size: 'tiny', quaternary: true, type: 'info', onClick: () => quickExtend(row, 730) }, { default: () => '+两年' }),
      ]
    })
  },
  {
    title: '操作', key: 'actions', width: 280, fixed: 'right',
    render: (row) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, { size: 'small', type: 'info', text: true, onClick: () => handleViewDetail(row) }, { default: () => '详情' }),
        h(NButton, { size: 'small', type: 'primary', text: true, onClick: () => handleExtend(row) }, { default: () => '延期' }),
        h(NButton, { size: 'small', text: true, onClick: () => handleDeviceLimit(row) }, { default: () => '设备' }),
        h(NButton, { size: 'small', type: 'warning', text: true, onClick: () => handleReset(row) }, { default: () => '重置' }),
        h(NButton, { size: 'small', type: 'error', text: true, onClick: () => handleToggleActive(row) },
          { default: () => row.is_active ? '禁用' : '启用' }),
      ]
    })
  }
]

const fetchData = async () => {
  loading.value = true
  try {
    const params = { page: pagination.page, page_size: pagination.pageSize }
    if (searchQuery.value) params.search = searchQuery.value
    if (statusFilter.value && statusFilter.value !== 'all') params.status = statusFilter.value
    const res = await listAdminSubscriptions(params)
    tableData.value = res.data.items || []
    pagination.itemCount = res.data.total || 0
  } catch (error) {
    message.error(error.message || '获取订阅列表失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => { pagination.page = 1; fetchData() }
const handleRefresh = () => fetchData()

const handleViewDetail = async (row) => {
  try {
    const res = await getAdminSubscription(row.id)
    detailData.value = res.data
    showDetailModal.value = true
  } catch (error) {
    message.error(error.message || '获取详情失败')
  }
}

const handleExtend = (row) => {
  currentEditId.value = row.id
  extendDays.value = 365
  showExtendModal.value = true
}

const handleExtendSubmit = async () => {
  submitting.value = true
  try {
    await extendSubscription(currentEditId.value, { days: extendDays.value })
    message.success(`已延长 ${extendDays.value} 天`)
    showExtendModal.value = false
    fetchData()
  } catch (error) {
    message.error(error.message || '延期失败')
  } finally {
    submitting.value = false
  }
}

const handleDeviceLimit = (row) => {
  currentEditId.value = row.id
  newDeviceLimit.value = row.device_limit || 3
  showDeviceLimitModal.value = true
}

const handleDeviceLimitSubmit = async () => {
  submitting.value = true
  try {
    await updateSubscriptionDeviceLimit(currentEditId.value, { device_limit: newDeviceLimit.value })
    message.success('设备限制已更新')
    showDeviceLimitModal.value = false
    fetchData()
  } catch (error) {
    message.error(error.message || '更新失败')
  } finally {
    submitting.value = false
  }
}

const handleReset = (row) => {
  dialog.warning({
    title: '确认重置',
    content: `确定要重置用户 "${row.user_email}" 的订阅吗？将生成新的订阅地址并清除所有设备。`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await resetAdminSubscription(row.id)
        message.success('订阅已重置')
        fetchData()
      } catch (error) {
        message.error(error.message || '重置失败')
      }
    }
  })
}

const handleToggleActive = (row) => {
  const action = row.is_active ? '禁用' : '启用'
  dialog.warning({
    title: `确认${action}`,
    content: `确定要${action}用户 "${row.user_email}" 的订阅吗？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await updateSubscriptionDeviceLimit(row.id, { is_active: !row.is_active })
        message.success(`已${action}`)
        fetchData()
      } catch (error) {
        message.error(error.message || `${action}失败`)
      }
    }
  })
}

onMounted(() => fetchData())
</script>

<style scoped>
.subscriptions-container {
  padding: 20px;
}
.sub-url-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.sub-url-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.sub-url-label {
  flex-shrink: 0;
  font-weight: 500;
  width: 80px;
}
.sub-url-text {
  flex: 1;
  font-size: 12px;
  background: var(--n-color-embedded, #f5f5f5);
  padding: 4px 8px;
  border-radius: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.qr-grid {
  display: flex;
  justify-content: space-around;
  gap: 16px;
}
.qr-item {
  display: flex;
  flex-direction: column;
  align-items: center;
}
.qr-title {
  font-weight: 500;
  margin-bottom: 8px;
  font-size: 13px;
}

@media (max-width: 767px) {
  .subscriptions-container { padding: 8px; }
}
</style>