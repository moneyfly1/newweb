<template>
  <div class="subscriptions-container">
    <n-card title="订阅管理">
      <template #header-extra>
        <n-space>
          <n-input v-model:value="searchQuery" placeholder="搜索用户/邮箱/订阅URL" clearable style="width: 240px" @keyup.enter="handleSearch">
            <template #prefix><n-icon><SearchOutline /></n-icon></template>
          </n-input>
          <n-select v-model:value="statusFilter" :options="statusOptions" style="width: 130px" @update:value="handleSearch" />
          <n-button @click="handleRefresh"><template #icon><n-icon><RefreshOutline /></n-icon></template>刷新</n-button>
        </n-space>
      </template>
      <n-data-table remote :columns="columns" :data="tableData" :loading="loading" :pagination="pagination" :bordered="false" :scroll-x="1400"
        @update:page="(p) => { pagination.page = p; fetchData() }"
        @update:page-size="(ps) => { pagination.pageSize = ps; pagination.page = 1; fetchData() }" />
    </n-card>

    <!-- Detail Drawer -->
    <n-drawer v-model:show="showDetailDrawer" :width="720" placement="right">
      <n-drawer-content :title="'订阅详情 - ' + (detailData.username || detailData.user_email || '')">
        <n-descriptions bordered :column="2" label-placement="left" size="small">
          <n-descriptions-item label="ID">{{ detailData.id }}</n-descriptions-item>
          <n-descriptions-item label="用户">{{ detailData.username || detailData.user_email || '-' }}</n-descriptions-item>
          <n-descriptions-item label="套餐">{{ detailData.package_name || '-' }}</n-descriptions-item>
          <n-descriptions-item label="状态"><n-tag :type="getStatusType(detailData.status)" size="small">{{ getStatusText(detailData.status) }}</n-tag></n-descriptions-item>
          <n-descriptions-item label="设备">{{ detailData.current_devices || 0 }} / {{ detailData.device_limit || 0 }}</n-descriptions-item>
          <n-descriptions-item label="剩余">
            <span :style="{ color: getRemainingDaysColor(detailData.expire_time), fontWeight: 'bold' }">{{ getRemainingDays(detailData.expire_time) }}</span>
          </n-descriptions-item>
          <n-descriptions-item label="到期时间" :span="2">{{ formatDate(detailData.expire_time) }}</n-descriptions-item>
          <n-descriptions-item label="通用次数">{{ detailData.universal_count || 0 }}</n-descriptions-item>
          <n-descriptions-item label="Clash次数">{{ detailData.clash_count || 0 }}</n-descriptions-item>
        </n-descriptions>
        <div v-if="detailData.universal_url" style="margin-top: 16px">
          <n-space align="center" style="margin-bottom: 12px">
            <n-button type="primary" size="small" @click="showQRModal = true">显示二维码</n-button>
            <n-button size="small" @click="copyText(detailData.universal_url)">复制通用链接</n-button>
            <n-button size="small" @click="copyText(detailData.clash_url)">复制Clash链接</n-button>
          </n-space>
          <div class="sub-url-section">
            <div class="sub-url-row"><span class="sub-url-label">通用订阅</span><code class="sub-url-text">{{ detailData.universal_url }}</code></div>
            <div class="sub-url-row"><span class="sub-url-label">Clash 订阅</span><code class="sub-url-text">{{ detailData.clash_url }}</code></div>
          </div>
        </div>
        <n-divider>在线设备 ({{ (detailData.devices || []).length }})</n-divider>
        <n-data-table v-if="detailData.devices && detailData.devices.length" :columns="deviceColumns" :data="detailData.devices" :bordered="false" size="small" :max-height="200" />
        <n-empty v-else description="暂无设备记录" size="small" />
        <template #footer>
          <n-space>
            <n-button type="success" @click="handleLoginAs(detailData)">后台</n-button>
            <n-button type="warning" @click="handleReset(detailData)">重置</n-button>
            <n-button type="info" @click="handleSendEmail(detailData)">发送</n-button>
            <n-button :type="detailData.is_active ? 'error' : 'success'" @click="handleToggleActive(detailData)">{{ detailData.is_active ? '禁用' : '启用' }}</n-button>
            <n-button type="error" @click="handleDeleteUser(detailData)">删除</n-button>
            <n-button @click="handleClearDevices(detailData)">清理设备</n-button>
          </n-space>
        </template>
      </n-drawer-content>
    </n-drawer>

    <!-- QR Code Modal -->
    <n-modal v-model:show="showQRModal" title="订阅二维码" preset="card" style="width: 640px">
      <div class="qr-grid" v-if="detailData.universal_url">
        <div class="qr-item">
          <div class="qr-title">Shadowrocket</div>
          <canvas :ref="(el) => renderQR(el, getShadowrocketUrl(detailData.universal_url))"></canvas>
          <n-button size="tiny" style="margin-top:8px" @click="copyText(getShadowrocketUrl(detailData.universal_url))">复制</n-button>
        </div>
        <div class="qr-item">
          <div class="qr-title">通用订阅</div>
          <canvas :ref="(el) => renderQR(el, detailData.universal_url)"></canvas>
          <n-button size="tiny" style="margin-top:8px" @click="copyText(detailData.universal_url)">复制</n-button>
        </div>
        <div class="qr-item">
          <div class="qr-title">Clash</div>
          <canvas :ref="(el) => renderQR(el, detailData.clash_url)"></canvas>
          <n-button size="tiny" style="margin-top:8px" @click="copyText(detailData.clash_url)">复制</n-button>
        </div>
      </div>
    </n-modal>

    <!-- Extend / Set Expire Modal -->
    <n-modal v-model:show="showExtendModal" title="延长/设置到期时间" preset="card" style="width: 480px">
      <n-space vertical :size="12">
        <div class="quick-time-label">快捷延长</div>
        <n-space>
          <n-button size="small" @click="quickAddDays(30)">+1月</n-button>
          <n-button size="small" @click="quickAddDays(90)">+3月</n-button>
          <n-button size="small" @click="quickAddDays(180)">+半年</n-button>
          <n-button size="small" @click="quickAddDays(365)">+一年</n-button>
          <n-button size="small" @click="quickAddDays(730)">+两年</n-button>
        </n-space>
        <n-input-number v-model:value="extendDays" :min="1" style="width:100%"><template #prefix>延长</template><template #suffix>天</template></n-input-number>
        <n-button type="primary" block @click="handleExtendSubmit" :loading="submitting">延长天数</n-button>
        <n-divider style="margin:8px 0">或手动选择到期时间</n-divider>
        <n-date-picker v-model:value="manualExpireTime" type="datetime" style="width:100%" clearable />
        <n-button type="warning" block @click="handleSetExpireTime" :loading="submitting" :disabled="!manualExpireTime">设置到期时间</n-button>
      </n-space>
    </n-modal>

    <!-- Device Limit Modal -->
    <n-modal v-model:show="showDeviceLimitModal" title="修改设备限制" preset="card" style="width: 400px">
      <n-space vertical :size="12">
        <n-space>
          <n-button size="small" @click="newDeviceLimit += 5">+5</n-button>
          <n-button size="small" @click="newDeviceLimit += 10">+10</n-button>
          <n-button size="small" @click="newDeviceLimit += 15">+15</n-button>
        </n-space>
        <n-input-number v-model:value="newDeviceLimit" :min="1" :max="999" style="width:100%"><template #prefix>上限</template><template #suffix>台</template></n-input-number>
      </n-space>
      <template #footer>
        <div style="display:flex;justify-content:flex-end;gap:12px">
          <n-button @click="showDeviceLimitModal = false">取消</n-button>
          <n-button type="primary" @click="handleDeviceLimitSubmit" :loading="submitting">确定</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
<script setup>
import { ref, h, onMounted, nextTick } from 'vue'
import { NButton, NTag, NSpace, useMessage, useDialog } from 'naive-ui'
import { SearchOutline, RefreshOutline } from '@vicons/ionicons5'
import QRCode from 'qrcode'
import { useRouter } from 'vue-router'
import {
  listAdminSubscriptions,
  getAdminSubscription,
  resetAdminSubscription,
  extendSubscription,
  updateSubscriptionDeviceLimit,
  sendSubscriptionEmail,
  setSubscriptionExpireTime,
  deleteUserFull,
  toggleUserActive,
  loginAsUser
} from '@/api/admin'

const message = useMessage()
const dialog = useDialog()
const router = useRouter()

const loading = ref(false)
const submitting = ref(false)
const searchQuery = ref('')
const statusFilter = ref(null)
const tableData = ref([])
const pagination = ref({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: true, pageSizes: [10, 20, 50, 100] })

const showDetailDrawer = ref(false)
const showQRModal = ref(false)
const showExtendModal = ref(false)
const showDeviceLimitModal = ref(false)
const detailData = ref({})
const extendDays = ref(30)
const manualExpireTime = ref(null)
const newDeviceLimit = ref(3)
const currentEditId = ref(null)

const statusOptions = [
  { label: '全部', value: null },
  { label: '活跃', value: 'active' },
  { label: '即将到期', value: 'expiring' },
  { label: '已过期', value: 'expired' },
  { label: '已禁用', value: 'disabled' }
]

const getStatusType = (status) => {
  const m = { active: 'success', expiring: 'warning', expired: 'error', disabled: 'default' }
  return m[status] || 'default'
}
const getStatusText = (status) => {
  const m = { active: '活跃', expiring: '即将到期', expired: '已过期', disabled: '已禁用' }
  return m[status] || status || '-'
}
const formatDate = (d) => d ? new Date(d).toLocaleString('zh-CN') : '-'
const getRemainingDays = (expireTime) => {
  if (!expireTime) return '-'
  const diff = new Date(expireTime) - Date.now()
  if (diff <= 0) return '已过期'
  const days = Math.ceil(diff / 86400000)
  return days > 365 ? `${Math.floor(days / 365)}年${days % 365}天` : `${days}天`
}
const getRemainingDaysColor = (expireTime) => {
  if (!expireTime) return '#999'
  const diff = new Date(expireTime) - Date.now()
  if (diff <= 0) return '#e03050'
  const days = diff / 86400000
  if (days <= 3) return '#e03050'
  if (days <= 7) return '#f0a020'
  return '#18a058'
}
const getShadowrocketUrl = (url) => 'sub://' + btoa(url)
const copyText = (text) => {
  navigator.clipboard.writeText(text).then(() => message.success('已复制')).catch(() => message.error('复制失败'))
}
const renderQR = (el, text) => {
  if (!el || !text) return
  nextTick(() => { QRCode.toCanvas(el, text, { width: 180, margin: 1 }) })
}
// Table columns — matching old project: user, expire time, devices, remaining, actions (6 buttons in 2 rows)
const columns = [
  { title: 'ID', key: 'id', width: 60, fixed: 'left' },
  {
    title: '用户', key: 'user_email', width: 180, fixed: 'left',
    render: (row) => h('div', {}, [
      h('div', { style: 'font-weight:500' }, row.username || row.user_email || '未知'),
      h(NButton, { size: 'tiny', type: 'success', style: 'margin-top:4px', onClick: () => handleViewDetail(row) }, { default: () => '详情' })
    ])
  },
  {
    title: '到期时间', key: 'expire_time', width: 140,
    render: (row) => h('div', { style: { color: getRemainingDaysColor(row.expire_time), fontWeight: 'bold', fontSize: '13px' } }, [
      h('div', {}, getRemainingDays(row.expire_time)),
      h('div', { style: 'font-weight:normal;font-size:12px;color:#999;margin-top:2px' }, formatDate(row.expire_time))
    ])
  },
  {
    title: '设备', key: 'devices', width: 80, align: 'center',
    render: (row) => h('span', {}, `${row.current_devices || 0}/${row.device_limit || 0}`)
  },
  {
    title: '状态', key: 'status', width: 80,
    render: (row) => h(NTag, { type: getStatusType(row.status), size: 'small' }, { default: () => getStatusText(row.status) })
  },
  { title: '套餐', key: 'package_name', width: 120, ellipsis: { tooltip: true } },
  {
    title: '操作', key: 'actions', width: 340, fixed: 'right',
    render: (row) => h('div', { class: 'action-btn-grid' }, [
      h(NButton, { size: 'small', type: 'success', onClick: () => handleLoginAs(row) }, { default: () => '后台' }),
      h(NButton, { size: 'small', type: 'primary', onClick: () => handleExtend(row) }, { default: () => '时间' }),
      h(NButton, { size: 'small', type: 'info', onClick: () => handleSendEmail(row) }, { default: () => '发送' }),
      h(NButton, { size: 'small', type: row.is_active ? 'warning' : 'success', onClick: () => handleToggleActive(row) }, { default: () => row.is_active ? '禁用' : '启用' }),
      h(NButton, { size: 'small', type: 'error', onClick: () => handleDeleteUser(row) }, { default: () => '删除' }),
      h(NButton, { size: 'small', onClick: () => handleDeviceLimit(row) }, { default: () => '设备' }),
    ])
  }
]

const deviceColumns = [
  { title: '设备名', key: 'device_name', ellipsis: { tooltip: true } },
  { title: 'IP', key: 'ip_address', width: 140 },
  { title: '最后活跃', key: 'last_active', width: 170, render: (row) => formatDate(row.last_access || row.updated_at) }
]

const fetchData = async () => {
  loading.value = true
  try {
    const params = { page: pagination.value.page, page_size: pagination.value.pageSize, search: searchQuery.value || undefined, status: statusFilter.value || undefined }
    const res = await listAdminSubscriptions(params)
    tableData.value = res.data.items || []
    pagination.value.itemCount = res.data.total || 0
  } catch (e) { message.error('获取订阅列表失败') }
  finally { loading.value = false }
}
const handleSearch = () => { pagination.value.page = 1; fetchData() }
const handleRefresh = () => fetchData()

const handleViewDetail = async (row) => {
  try {
    const res = await getAdminSubscription(row.id)
    detailData.value = res.data
    showDetailDrawer.value = true
  } catch (e) { message.error('获取详情失败') }
}
// Login as user
const handleLoginAs = async (row) => {
  const userId = row.user_id || row.id
  try {
    const res = await loginAsUser(userId)
    const { access_token, refresh_token, user } = res.data
    localStorage.setItem('admin_token', localStorage.getItem('token') || '')
    localStorage.setItem('admin_user', localStorage.getItem('user') || '')
    localStorage.setItem('token', access_token)
    localStorage.setItem('user', JSON.stringify(user))
    window.open('/', '_blank')
  } catch (e) { message.error('登录失败') }
}

// Extend / Set expire time
const handleExtend = (row) => {
  currentEditId.value = row.id
  extendDays.value = 30
  manualExpireTime.value = null
  showExtendModal.value = true
}
const quickAddDays = (days) => { extendDays.value = days }
const handleExtendSubmit = async () => {
  submitting.value = true
  try {
    await extendSubscription(currentEditId.value, { days: extendDays.value })
    message.success(`已延长 ${extendDays.value} 天`)
    showExtendModal.value = false
    fetchData()
    if (showDetailDrawer.value) handleViewDetail({ id: currentEditId.value })
  } catch (e) { message.error('延长失败') }
  finally { submitting.value = false }
}
const handleSetExpireTime = async () => {
  if (!manualExpireTime.value) return
  submitting.value = true
  try {
    const t = new Date(manualExpireTime.value).toISOString()
    await setSubscriptionExpireTime(currentEditId.value, { expire_time: t })
    message.success('到期时间已设置')
    showExtendModal.value = false
    fetchData()
    if (showDetailDrawer.value) handleViewDetail({ id: currentEditId.value })
  } catch (e) { message.error('设置失败') }
  finally { submitting.value = false }
}

// Device limit
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
    if (showDetailDrawer.value) handleViewDetail({ id: currentEditId.value })
  } catch (e) { message.error('更新失败') }
  finally { submitting.value = false }
}

// Reset subscription
const handleReset = (row) => {
  dialog.warning({
    title: '确认重置', content: '重置将生成新的订阅链接，旧链接将失效，所有设备将被清除。确定继续？',
    positiveText: '确定重置', negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await resetAdminSubscription(row.id)
        message.success('订阅已重置')
        if (showDetailDrawer.value) handleViewDetail({ id: row.id })
        fetchData()
      } catch (e) { message.error('重置失败') }
    }
  })
}

// Toggle active
const handleToggleActive = (row) => {
  const action = row.is_active ? '禁用' : '启用'
  dialog.warning({
    title: `确认${action}`, content: `确定要${action}该用户吗？`,
    positiveText: `确定${action}`, negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await toggleUserActive(row.user_id)
        message.success(`已${action}`)
        if (showDetailDrawer.value) handleViewDetail({ id: row.id })
        fetchData()
      } catch (e) { message.error(`${action}失败`) }
    }
  })
}

// Send email
const handleSendEmail = async (row) => {
  try {
    await sendSubscriptionEmail(row.id)
    message.success('订阅地址已发送到用户邮箱')
  } catch (e) { message.error('发送失败') }
}

// Delete user
const handleDeleteUser = (row) => {
  dialog.error({
    title: '确认删除用户', content: '此操作将永久删除该用户及其所有数据（订阅、订单、设备等），不可恢复！',
    positiveText: '确定删除', negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteUserFull(row.user_id)
        message.success('用户已删除')
        showDetailDrawer.value = false
        fetchData()
      } catch (e) { message.error('删除失败') }
    }
  })
}

// Clear devices
const handleClearDevices = (row) => {
  dialog.warning({
    title: '确认清理设备', content: '将清除该订阅下所有设备记录并重置设备计数。确定继续？',
    positiveText: '确定清理', negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await resetAdminSubscription(row.id)
        message.success('设备已清理')
        if (showDetailDrawer.value) handleViewDetail({ id: row.id })
        fetchData()
      } catch (e) { message.error('清理失败') }
    }
  })
}

onMounted(() => fetchData())
</script>

<style scoped>
.subscriptions-container { padding: 20px; }
.sub-url-section { background: #f5f5f5; border-radius: 6px; padding: 12px; }
.sub-url-row { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.sub-url-row:last-child { margin-bottom: 0; }
.sub-url-label { font-size: 12px; color: #666; white-space: nowrap; min-width: 70px; }
.sub-url-text { font-size: 12px; word-break: break-all; color: #333; background: none; padding: 0; }
.qr-grid { display: flex; gap: 24px; justify-content: center; flex-wrap: wrap; }
.qr-item { text-align: center; }
.qr-title { font-weight: 600; margin-bottom: 8px; }
.quick-time-label { font-size: 13px; color: #666; }
:deep(.action-btn-grid) {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 4px;
}
@media (max-width: 767px) {
  .subscriptions-container { padding: 8px; }
  .qr-grid { flex-direction: column; align-items: center; }
}
</style>
