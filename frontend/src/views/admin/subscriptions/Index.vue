<template>
  <div class="subscriptions-container">
    <n-card title="订阅管理">
      <template #header-extra>
        <n-space>
          <n-input v-model:value="searchQuery" placeholder="搜索用户/邮箱/备注" clearable style="width: 200px" @keyup.enter="handleSearch">
            <template #prefix><n-icon><SearchOutline /></n-icon></template>
          </n-input>
          <n-select v-model:value="statusFilter" :options="statusOptions" style="width: 120px" @update:value="handleSearch" />
          <n-button @click="handleRefresh"><template #icon><n-icon><RefreshOutline /></n-icon></template></n-button>
        </n-space>
      </template>

      <!-- Desktop Table -->
      <template v-if="!appStore.isMobile">
        <n-space v-if="checkedRowKeys.length > 0" align="center" style="margin-bottom:12px">
          <span style="color:#666">已选择 {{ checkedRowKeys.length }} 项</span>
          <n-button size="small" type="success" @click="handleBatchEnable">批量启用</n-button>
          <n-button size="small" type="warning" @click="handleBatchDisable">批量禁用</n-button>
          <n-button size="small" type="info" @click="handleBatchEmail">批量发送</n-button>
          <n-button size="small" type="error" @click="handleBatchDelete">批量删除</n-button>
        </n-space>
        <n-data-table remote :columns="columns" :data="tableData" :loading="loading" :pagination="pagination" :bordered="false" :scroll-x="1500"
          :row-key="(row) => row.id"
          :checked-row-keys="checkedRowKeys"
          @update:checked-row-keys="(keys) => { checkedRowKeys = keys }"
          @update:page="(p) => { pagination.page = p; fetchData() }"
          @update:page-size="(ps) => { pagination.pageSize = ps; pagination.page = 1; fetchData() }" />
      </template>

      <!-- Mobile Card List -->
      <template v-else>
        <n-spin :show="loading">
          <div v-if="tableData.length === 0 && !loading" style="text-align:center;padding:40px 0;color:#999">暂无数据</div>
          <div class="mobile-card-list">
            <div v-for="row in tableData" :key="row.id" class="sub-card">
              <div class="sub-card-header">
                <div class="sub-user-info">
                  <div class="sub-avatar">{{ (row.username || row.user_email || 'U').charAt(0).toUpperCase() }}</div>
                  <div class="sub-user-meta">
                    <div class="sub-user-name">{{ row.username || row.user_email || '未知' }}</div>
                    <div class="sub-user-id">ID: {{ row.id }} · {{ row.package_name || '无套餐' }}</div>
                  </div>
                </div>
                <n-tag :type="getStatusType(row.status)" size="small">{{ getStatusText(row.status) }}</n-tag>
              </div>
              <div class="sub-section" :class="{ 'section-expired': isExpired(row) }">
                <div class="sub-section-row">
                  <span class="sub-section-label">到期时间</span>
                  <span class="sub-section-value" :style="{ color: getRemainingDaysColor(row.expire_time) }">{{ getRemainingDays(row.expire_time) }}</span>
                </div>
                <div class="sub-btn-row sub-btn-row-5">
                  <n-button size="tiny" @click="inlineAddTime(row, 30)">+1月</n-button>
                  <n-button size="tiny" @click="inlineAddTime(row, 90)">+3月</n-button>
                  <n-button size="tiny" @click="inlineAddTime(row, 180)">+半年</n-button>
                  <n-button size="tiny" @click="inlineAddTime(row, 365)">+1年</n-button>
                  <n-button size="tiny" @click="inlineAddTime(row, 730)">+2年</n-button>
                </div>
                <n-date-picker v-model:value="row._expireTs" type="datetime" size="small" style="width:100%;margin-top:6px" @update:value="(v) => inlineSetExpire(row, v)" clearable />
              </div>
              <div class="sub-section" :class="{ 'section-overlimit': isOverlimit(row) }">
                <div class="sub-section-row">
                  <span class="sub-section-label">设备限制</span>
                  <span class="sub-section-value">{{ row.current_devices || 0 }} / {{ row.device_limit || 0 }}</span>
                </div>
                <div class="sub-btn-row sub-btn-row-6">
                  <n-button size="tiny" type="error" @click="handleClearDevices(row)">清理</n-button>
                  <n-button size="tiny" @click="inlineSetDevice(row, 2)">2</n-button>
                  <n-button size="tiny" @click="inlineSetDevice(row, 5)">5</n-button>
                  <n-button size="tiny" @click="inlineSetDevice(row, 10)">10</n-button>
                  <n-button size="tiny" @click="inlineSetDevice(row, 20)">20</n-button>
                  <n-button size="tiny" @click="inlineSetDevice(row, 30)">30</n-button>
                </div>
              </div>
              <div class="sub-section">
                <div class="sub-section-row">
                  <span class="sub-section-label">备注</span>
                </div>
                <n-input v-model:value="row.user_notes" type="text" size="small" placeholder="输入备注..." @blur="saveNotes(row)" @keyup.enter="saveNotes(row)" />
              </div>
              <div class="sub-action-grid">
                <div class="sub-action-item" @click="handleViewDetail(row)">
                  <div class="sub-action-icon" style="background:#f0f0ff;color:#667eea"><n-icon :size="20"><SearchOutline /></n-icon></div>
                  <span>详情</span>
                </div>
                <div class="sub-action-item" @click="handleLoginAs(row)">
                  <div class="sub-action-icon" style="background:#f0f9eb;color:#18a058"><n-icon :size="20"><PersonOutline /></n-icon></div>
                  <span>后台</span>
                </div>
                <div class="sub-action-item" @click="handleReset(row)">
                  <div class="sub-action-icon" style="background:#ecf5ff;color:#2080f0"><n-icon :size="20"><RefreshOutline /></n-icon></div>
                  <span>重置</span>
                </div>
                <div class="sub-action-item" @click="handleSendEmail(row)">
                  <div class="sub-action-icon" style="background:#fdf6ec;color:#f0a020"><n-icon :size="20"><MailOutline /></n-icon></div>
                  <span>发邮件</span>
                </div>
                <div class="sub-action-item" :class="{ 'is-disabled': row.is_active && row.user_id === userStore.userInfo?.id }" @click="handleToggleActive(row)">
                  <div class="sub-action-icon" :style="row.is_active ? 'background:#fef0f0;color:#e03050' : 'background:#f0f9eb;color:#18a058'"><n-icon :size="20"><PowerOutline /></n-icon></div>
                  <span>{{ row.is_active ? '禁用' : '启用' }}</span>
                </div>
                <div class="sub-action-item" @click="copyText(getRowUniversalUrl(row))" v-if="getRowUniversalUrl(row)">
                  <div class="sub-action-icon" style="background:#ecf5ff;color:#2080f0"><n-icon :size="20"><CopyOutline /></n-icon></div>
                  <span>复制</span>
                </div>
                <div class="sub-action-item" @click="showSingleQR(row)" v-if="getRowUniversalUrl(row)">
                  <div class="sub-action-icon" style="background:#f3e8ff;color:#8b5cf6"><n-icon :size="20"><QrCodeOutline /></n-icon></div>
                  <span>二维码</span>
                </div>
                <div class="sub-action-item" @click="handleDeleteUser(row)">
                  <div class="sub-action-icon" style="background:#fef0f0;color:#e03050"><n-icon :size="20"><TrashOutline /></n-icon></div>
                  <span>删除</span>
                </div>
              </div>
            </div>
          </div>
          <div style="display:flex;justify-content:center;margin-top:16px">
            <n-pagination v-model:page="pagination.page" :page-count="Math.ceil((pagination.itemCount||0)/(pagination.pageSize||20))" @update:page="(p) => { pagination.page = p; fetchData() }" />
          </div>
        </n-spin>
      </template>
    </n-card>

    <!-- Detail Drawer -->
    <n-drawer v-model:show="showDetailDrawer" :width="appStore.isMobile ? '100%' : 780" placement="right" closable>
      <n-drawer-content :title="'用户详情 - ' + (detailData.username || detailData.user_email || '')" closable>
        <n-descriptions bordered :column="appStore.isMobile ? 1 : 2" label-placement="left" size="small">
          <n-descriptions-item label="ID">{{ detailData.user_id || detailData.id }}</n-descriptions-item>
          <n-descriptions-item label="用户名">{{ detailData.username || '-' }}</n-descriptions-item>
          <n-descriptions-item label="邮箱">{{ detailData.user_email || '-' }}</n-descriptions-item>
          <n-descriptions-item label="余额">¥{{ (detailData.user_balance ?? 0).toFixed(2) }}</n-descriptions-item>
          <n-descriptions-item label="状态">
            <n-tag :type="detailData.user_is_active ? 'success' : 'error'" size="small">{{ detailData.user_is_active ? '激活' : '禁用' }}</n-tag>
            <n-tag v-if="detailData.user_is_admin" type="warning" size="small" style="margin-left:4px">管理员</n-tag>
          </n-descriptions-item>
          <n-descriptions-item label="等级">{{ detailData.user_level_name || '无' }}</n-descriptions-item>
          <n-descriptions-item label="注册时间">{{ fmtDate(detailData.user_created_at) }}</n-descriptions-item>
          <n-descriptions-item label="最后登录">{{ fmtDate(detailData.user_last_login) }}</n-descriptions-item>
        </n-descriptions>

        <n-divider>订阅信息</n-divider>
        <n-descriptions bordered :column="appStore.isMobile ? 1 : 2" label-placement="left" size="small">
          <n-descriptions-item label="套餐">{{ detailData.package_name || '-' }}</n-descriptions-item>
          <n-descriptions-item label="状态">
            <n-tag :type="getStatusType(detailData.status)" size="small">{{ getStatusText(detailData.status) }}</n-tag>
            <n-tag v-if="!detailData.is_active" type="error" size="small" style="margin-left:4px">已停用</n-tag>
          </n-descriptions-item>
          <n-descriptions-item label="设备">{{ detailData.current_devices || 0 }} / {{ detailData.device_limit || 0 }}</n-descriptions-item>
          <n-descriptions-item label="到期时间">{{ fmtDate(detailData.expire_time) }}</n-descriptions-item>
        </n-descriptions>
        <div v-if="detailData.universal_url" style="margin-top:8px">
          <div class="url-row"><span class="url-label">通用</span><code class="url-text">{{ detailData.universal_url }}</code></div>
          <div class="url-row"><span class="url-label">Clash</span><code class="url-text">{{ detailData.clash_url }}</code></div>
        </div>

        <n-tabs type="line" style="margin-top:16px" animated>
          <n-tab-pane name="orders" tab="订单记录">
            <n-data-table v-if="(detailData.recent_orders||[]).length" :columns="orderCols" :data="detailData.recent_orders" :bordered="false" size="small" :max-height="240" />
            <n-empty v-else description="暂无订单" size="small" />
          </n-tab-pane>
          <n-tab-pane name="devices" tab="设备记录">
            <n-data-table v-if="(detailData.devices||[]).length" :columns="deviceCols" :data="detailData.devices" :bordered="false" size="small" :max-height="240" />
            <n-empty v-else description="暂无设备" size="small" />
          </n-tab-pane>
          <n-tab-pane name="logins" tab="登录历史">
            <n-data-table v-if="(detailData.login_history||[]).length" :columns="loginCols" :data="detailData.login_history" :bordered="false" size="small" :max-height="240" />
            <n-empty v-else description="暂无记录" size="small" />
          </n-tab-pane>
          <n-tab-pane name="resets" tab="重置记录">
            <n-data-table v-if="(detailData.resets||[]).length" :columns="resetCols" :data="detailData.resets" :bordered="false" size="small" :max-height="240" />
            <n-empty v-else description="暂无记录" size="small" />
          </n-tab-pane>
          <n-tab-pane name="balance" tab="余额变动">
            <n-data-table v-if="(detailData.balance_logs||[]).length" :columns="balanceCols" :data="detailData.balance_logs" :bordered="false" size="small" :max-height="240" />
            <n-empty v-else description="暂无记录" size="small" />
          </n-tab-pane>
          <n-tab-pane name="recharge" tab="充值记录">
            <n-data-table v-if="(detailData.recharge_records||[]).length" :columns="rechargeCols" :data="detailData.recharge_records" :bordered="false" size="small" :max-height="240" />
            <n-empty v-else description="暂无记录" size="small" />
          </n-tab-pane>
        </n-tabs>
      </n-drawer-content>
    </n-drawer>

    <!-- QR Code Modal (3 codes) -->
    <n-modal v-model:show="showQRModal" title="订阅二维码" preset="card" :style="{ width: appStore.isMobile ? '95%' : '640px' }">
      <div class="qr-grid" v-if="detailData.universal_url">
        <div class="qr-item">
          <div class="qr-title">Shadowrocket</div>
          <canvas :ref="(el) => renderQR(el, getShadowrocketUrl(detailData.universal_url))"></canvas>
        </div>
        <div class="qr-item">
          <div class="qr-title">通用订阅</div>
          <canvas :ref="(el) => renderQR(el, detailData.universal_url)"></canvas>
        </div>
        <div class="qr-item">
          <div class="qr-title">Clash</div>
          <canvas :ref="(el) => renderQR(el, detailData.clash_url)"></canvas>
        </div>
      </div>
    </n-modal>

    <!-- Single Shadowrocket QR Modal -->
    <n-modal v-model:show="showSingleQRModal" title="Shadowrocket 二维码" preset="card" :style="{ width: appStore.isMobile ? '85%' : '320px' }" @after-enter="renderSingleQRCode">
      <div style="text-align:center">
        <canvas ref="singleQRCanvas"></canvas>
      </div>
    </n-modal>
  </div>
</template>
<script setup>
import { ref, h, onMounted, nextTick } from 'vue'
import { NButton, NTag, NSpace, NDatePicker, NInputNumber, NInput, NDataTable, NEmpty, useMessage, useDialog } from 'naive-ui'
import { SearchOutline, RefreshOutline, PersonOutline, MailOutline, PowerOutline, TrashOutline, CopyOutline, QrCodeOutline } from '@vicons/ionicons5'
import QRCode from 'qrcode'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'
import { copyToClipboard as clipboardCopy } from '@/utils/clipboard'
import {
  listAdminSubscriptions, getAdminSubscription, resetAdminSubscription,
  extendSubscription, updateSubscriptionDeviceLimit, sendSubscriptionEmail,
  setSubscriptionExpireTime, deleteUserFull, toggleUserActive, loginAsUser,
  deleteUserDevice, updateUserNotes
} from '@/api/admin'

const message = useMessage()
const dialog = useDialog()
const router = useRouter()
const appStore = useAppStore()
const userStore = useUserStore()

const loading = ref(false)
const searchQuery = ref('')
const statusFilter = ref(null)
const tableData = ref([])
const pagination = ref({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: true, pageSizes: [10, 20, 50, 100] })
const showDetailDrawer = ref(false)
const showQRModal = ref(false)
const showSingleQRModal = ref(false)
const singleQRUrl = ref('')
const detailData = ref({})
const checkedRowKeys = ref([])

const statusOptions = [
  { label: '全部', value: null }, { label: '活跃', value: 'active' },
  { label: '即将到期', value: 'expiring' }, { label: '已过期', value: 'expired' }, { label: '已禁用', value: 'disabled' }
]
const getStatusType = (s) => ({ active: 'success', expiring: 'warning', expired: 'error', disabled: 'default' }[s] || 'default')
const getStatusText = (s) => ({ active: '活跃', expiring: '即将到期', expired: '已过期', disabled: '已禁用' }[s] || s || '-')
const formatDate = (d) => d ? new Date(d).toLocaleString('zh-CN') : '-'
const isExpired = (row) => row.expire_time && new Date(row.expire_time) < Date.now()
const isOverlimit = (row) => (row.current_devices || 0) > (row.device_limit || 0)
const getRemainingDays = (t) => {
  if (!t) return '-'; const diff = new Date(t) - Date.now()
  if (diff <= 0) return '已过期'; const d = Math.ceil(diff / 86400000)
  return d > 365 ? `${Math.floor(d / 365)}年${d % 365}天` : `${d}天`
}
const getRemainingDaysColor = (t) => {
  if (!t) return '#999'; const diff = (new Date(t) - Date.now()) / 86400000
  if (diff <= 0) return '#e03050'; if (diff <= 3) return '#e03050'; if (diff <= 7) return '#f0a020'; return '#18a058'
}
const getShadowrocketUrl = (url) => 'sub://' + btoa(url)
/** Effective subscription URL: from API or fallback from current origin + subscription_url token */
const getRowUniversalUrl = (row) => {
  if (row?.universal_url) return row.universal_url
  if (row?.subscription_url && typeof window !== 'undefined') {
    const base = window.location.origin.replace(/\/$/, '')
    return `${base}/api/v1/sub/${row.subscription_url}`
  }
  return ''
}
const copyText = async (text) => { const ok = await clipboardCopy(text); ok ? message.success('已复制') : message.error('复制失败') }
const saveNotes = async (row) => {
  try { await updateUserNotes(row.user_id, row.user_notes || ''); message.success('备注已保存') }
  catch { message.error('保存备注失败') }
}
const renderQR = (el, text) => { if (el && text) nextTick(() => QRCode.toCanvas(el, text, { width: 180, margin: 1 })) }
const singleQRCanvas = ref(null)
const showSingleQR = (row) => {
  const url = getRowUniversalUrl(row)
  if (!url) return
  singleQRUrl.value = getShadowrocketUrl(url)
  showSingleQRModal.value = true
}
const renderSingleQRCode = () => {
  nextTick(() => {
    if (singleQRCanvas.value && singleQRUrl.value) {
      QRCode.toCanvas(singleQRCanvas.value, singleQRUrl.value, { width: 220, margin: 1 })
    }
  })
}

const fmtDate = (d) => d ? new Date(d).toLocaleString('zh-CN') : '-'
const subStatusType = (s) => ({ active: 'success', expiring: 'warning', expired: 'error' }[s] || 'default')
const subStatusText = (s) => ({ active: '活跃', expiring: '即将到期', expired: '已过期', disabled: '已禁用' }[s] || s || '-')

const handleDeleteDevice = (device) => {
  dialog.warning({
    title: '确认删除设备',
    content: `确定要删除设备 ${device.device_name || device.software_name || '未知设备'} 吗？`,
    positiveText: '删除', negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteUserDevice(detailData.value.user_id, device.id)
        message.success('设备已删除')
        handleViewDetail({ id: detailData.value.id })
      } catch (error) { message.error('删除设备失败') }
    }
  })
}

const orderCols = [
  { title: '订单号', key: 'order_no', width: 180, ellipsis: { tooltip: true } },
  { title: '金额', key: 'final_amount', width: 90, render: (r) => `¥${(r.final_amount ?? r.amount ?? 0).toFixed(2)}` },
  { title: '状态', key: 'status', width: 80 },
  { title: '时间', key: 'created_at', width: 160, render: (r) => fmtDate(r.created_at) }
]
const deviceCols = [
  { title: '设备名', key: 'device_name', ellipsis: { tooltip: true }, render: (r) => r.device_name || r.software_name || '未知设备' },
  { title: 'IP', key: 'ip_address', width: 130, render: (r) => r.ip_address || '-' },
  { title: '最后活跃', key: 'last_access', width: 160, render: (r) => fmtDate(r.last_access || r.updated_at) },
  {
    title: '操作', key: 'actions', width: 80,
    render: (r) => h(NButton, { size: 'small', type: 'error', secondary: true, onClick: () => handleDeleteDevice(r) }, { default: () => '删除' })
  }
]
const loginCols = [
  { title: 'IP', key: 'ip_address', width: 130, render: (r) => r.ip_address || '-' },
  { title: '位置', key: 'location', width: 100, render: (r) => r.location || '-' },
  { title: 'UA', key: 'user_agent', ellipsis: { tooltip: true }, render: (r) => r.user_agent || '-' },
  { title: '状态', key: 'login_status', width: 70, render: (r) => h(NTag, { type: r.login_status === 'success' ? 'success' : 'error', size: 'small' }, { default: () => r.login_status === 'success' ? '成功' : '失败' }) },
  { title: '时间', key: 'login_time', width: 160, render: (r) => fmtDate(r.login_time) }
]
const resetCols = [
  { title: '操作者', key: 'reset_by', width: 100, render: (r) => r.reset_by || '-' },
  { title: '类型', key: 'reset_type', width: 80 },
  { title: '原因', key: 'reason', ellipsis: { tooltip: true } },
  { title: '时间', key: 'created_at', width: 160, render: (r) => fmtDate(r.created_at) }
]
const balanceCols = [
  { title: '类型', key: 'change_type', width: 90 },
  { title: '金额', key: 'amount', width: 90, render: (r) => `¥${(r.amount ?? 0).toFixed(2)}` },
  { title: '变动后', key: 'balance_after', width: 90, render: (r) => `¥${(r.balance_after ?? 0).toFixed(2)}` },
  { title: '说明', key: 'description', ellipsis: { tooltip: true }, render: (r) => r.description || '-' },
  { title: '时间', key: 'created_at', width: 160, render: (r) => fmtDate(r.created_at) }
]
const rechargeCols = [
  { title: '金额', key: 'amount', width: 90, render: (r) => `¥${(r.amount ?? 0).toFixed(2)}` },
  { title: '方式', key: 'payment_method', width: 100 },
  { title: '状态', key: 'status', width: 80 },
  { title: '时间', key: 'created_at', width: 160, render: (r) => fmtDate(r.created_at) }
]
// Desktop table columns with INLINE controls
const columns = [
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 60, sorter: (a, b) => a.id - b.id, resizable: true },
  {
    title: '用户', key: 'user_email', width: 160, resizable: true,
    render: (row) => h('div', {}, [
      h('div', { style: 'font-weight:500;font-size:13px' }, row.username || row.user_email || '未知'),
      h(NButton, { size: 'tiny', type: 'success', style: 'margin-top:4px', onClick: () => handleViewDetail(row) }, { default: () => '详情' })
    ])
  },
  {
    title: 'SR', key: 'qr', width: 50,
    render: (row) => getRowUniversalUrl(row) ? h(NButton, { size: 'tiny', quaternary: true, onClick: () => showSingleQR(row) }, { icon: () => h('span', { style: 'font-size:16px' }, '📱') }) : '-'
  },
  {
    title: '到期时间', key: 'expire_time', width: 260, resizable: true,
    sorter: (a, b) => new Date(a.expire_time || 0) - new Date(b.expire_time || 0),
    render: (row) => h('div', { class: isExpired(row) ? 'inline-cell cell-expired' : 'inline-cell' }, [
      h('div', { style: { color: getRemainingDaysColor(row.expire_time), fontWeight: 'bold', fontSize: '13px', marginBottom: '4px' } }, getRemainingDays(row.expire_time)),
      h(NDatePicker, { value: row._expireTs, type: 'datetime', size: 'small', style: 'width:100%', clearable: true, onUpdateValue: (v) => inlineSetExpire(row, v) }),
      h('div', { class: 'inline-quick-btns' }, [
        h(NButton, { size: 'tiny', onClick: () => inlineAddTime(row, 30) }, { default: () => '+1月' }),
        h(NButton, { size: 'tiny', onClick: () => inlineAddTime(row, 90) }, { default: () => '+3月' }),
        h(NButton, { size: 'tiny', onClick: () => inlineAddTime(row, 180) }, { default: () => '+半年' }),
        h(NButton, { size: 'tiny', onClick: () => inlineAddTime(row, 365) }, { default: () => '+1年' }),
        h(NButton, { size: 'tiny', onClick: () => inlineAddTime(row, 730) }, { default: () => '+2年' }),
      ])
    ])
  },
  {
    title: '设备限制', key: 'device_limit', width: 260, resizable: true,
    sorter: (a, b) => (a.device_limit || 0) - (b.device_limit || 0),
    render: (row) => h('div', { class: isOverlimit(row) ? 'inline-cell cell-overlimit' : 'inline-cell' }, [
      h('div', { style: 'font-size:13px;margin-bottom:4px' }, `在线 ${row.current_devices || 0} / 上限 ${row.device_limit || 0}`),
      h(NInputNumber, { value: row.device_limit, min: 0, max: 999, size: 'small', style: 'width:100%', onUpdateValue: (v) => inlineSetDevice(row, v) }),
      h('div', { class: 'inline-quick-btns' }, [
        h(NButton, { size: 'tiny', onClick: () => inlineSetDevice(row, 2) }, { default: () => '2' }),
        h(NButton, { size: 'tiny', onClick: () => inlineSetDevice(row, 5) }, { default: () => '5' }),
        h(NButton, { size: 'tiny', onClick: () => inlineSetDevice(row, 10) }, { default: () => '10' }),
        h(NButton, { size: 'tiny', onClick: () => inlineSetDevice(row, 20) }, { default: () => '20' }),
        h(NButton, { size: 'tiny', onClick: () => inlineSetDevice(row, 30) }, { default: () => '30' }),
      ])
    ])
  },
  {
    title: '备注', key: 'user_notes', width: 160, resizable: true,
    render: (row) => h(NInput, {
      value: row.user_notes || '', size: 'small', placeholder: '备注...',
      onUpdateValue: (v) => { row.user_notes = v },
      onBlur: () => saveNotes(row)
    })
  },
  {
    title: '状态', key: 'status', width: 80, resizable: true,
    render: (row) => h(NTag, { type: getStatusType(row.status), size: 'small' }, { default: () => getStatusText(row.status) })
  },
  {
    title: '操作', key: 'actions', width: 200, fixed: 'right',
    render: (row) => h('div', { class: 'action-btn-grid' }, [
      h(NButton, { size: 'small', type: 'success', onClick: () => handleLoginAs(row) }, { default: () => '后台' }),
      h(NButton, { size: 'small', type: 'warning', onClick: () => handleReset(row) }, { default: () => '重置' }),
      h(NButton, { size: 'small', type: 'info', onClick: () => handleSendEmail(row) }, { default: () => '发送' }),
      h(NButton, { size: 'small', type: row.is_active ? 'error' : 'success', disabled: row.is_active && row.user_id === userStore.userInfo?.id, onClick: () => handleToggleActive(row) }, { default: () => row.is_active ? '禁用' : '启用' }),
      h(NButton, { size: 'small', type: 'error', onClick: () => handleDeleteUser(row) }, { default: () => '删除' }),
      h(NButton, { size: 'small', onClick: () => handleClearDevices(row) }, { default: () => '清理' }),
    ])
  }
]

// Fetch data
const fetchData = async () => {
  loading.value = true
  try {
    const params = { page: pagination.value.page, page_size: pagination.value.pageSize, search: searchQuery.value || undefined, status: statusFilter.value || undefined }
    const res = await listAdminSubscriptions(params)
    const items = res.data.items || []
    items.forEach(r => { r._expireTs = r.expire_time ? new Date(r.expire_time).getTime() : null })
    tableData.value = items
    pagination.value.itemCount = res.data.total || 0
  } catch { message.error('获取订阅列表失败') }
  finally { loading.value = false }
}
const handleSearch = () => { pagination.value.page = 1; fetchData() }
const handleRefresh = () => fetchData()

// Inline time operations
const inlineAddTime = async (row, days) => {
  try { await extendSubscription(row.id, { days }); message.success(`已延长 ${days} 天`); fetchData() }
  catch { message.error('延长失败') }
}
const inlineSetExpire = async (row, ts) => {
  if (!ts) return
  try { await setSubscriptionExpireTime(row.id, { expire_time: new Date(ts).toISOString() }); message.success('到期时间已设置'); fetchData() }
  catch { message.error('设置失败') }
}

// Inline device operations
const inlineAddDevice = async (row, n) => {
  const newLimit = (row.device_limit || 0) + n
  try { await updateSubscriptionDeviceLimit(row.id, { device_limit: newLimit }); message.success(`设备上限已设为 ${newLimit}`); fetchData() }
  catch { message.error('更新失败') }
}
const inlineSetDevice = async (row, v) => {
  if (v == null || v === row.device_limit) return
  try { await updateSubscriptionDeviceLimit(row.id, { device_limit: v }); message.success(`设备上限已设为 ${v}`); fetchData() }
  catch { message.error('更新失败') }
}

const handleViewDetail = async (row) => {
  try { const res = await getAdminSubscription(row.id); detailData.value = res.data; showDetailDrawer.value = true }
  catch { message.error('获取详情失败') }
}
const handleLoginAs = async (row) => {
  try {
    const res = await loginAsUser(row.user_id || row.id)
    const { access_token, user } = res.data
    localStorage.setItem('admin_token', localStorage.getItem('token') || '')
    localStorage.setItem('admin_user', localStorage.getItem('user') || '')
    localStorage.setItem('token', access_token)
    localStorage.setItem('user', JSON.stringify(user))
    window.open('/', '_blank')
  } catch { message.error('登录失败') }
}
const handleReset = (row) => {
  dialog.warning({ title: '确认重置', content: '重置将生成新的订阅链接，旧链接失效。确定？', positiveText: '确定', negativeText: '取消',
    onPositiveClick: async () => { try { await resetAdminSubscription(row.id); message.success('已重置'); fetchData() } catch { message.error('重置失败') } }
  })
}
const handleToggleActive = (row) => {
  if (row.is_active && row.user_id === userStore.userInfo?.id) {
    message.error('不能禁用自己')
    return
  }
  const a = row.is_active ? '禁用' : '启用'
  dialog.warning({ title: `确认${a}`, content: `确定要${a}该用户吗？`, positiveText: '确定', negativeText: '取消',
    onPositiveClick: async () => { try { await toggleUserActive(row.user_id); message.success(`已${a}`); fetchData() } catch (e) { message.error(e?.response?.data?.message || `${a}失败`) } }
  })
}
const handleSendEmail = async (row) => {
  try { await sendSubscriptionEmail(row.id); message.success('已发送') } catch { message.error('发送失败') }
}
const handleDeleteUser = (row) => {
  dialog.error({ title: '确认删除', content: '永久删除该用户及所有数据，不可恢复！', positiveText: '删除', negativeText: '取消',
    onPositiveClick: async () => { try { await deleteUserFull(row.user_id); message.success('已删除'); showDetailDrawer.value = false; fetchData() } catch { message.error('删除失败') } }
  })
}
const handleClearDevices = (row) => {
  dialog.warning({ title: '确认清理', content: '清除该订阅下所有设备记录？', positiveText: '确定', negativeText: '取消',
    onPositiveClick: async () => { try { await resetAdminSubscription(row.id); message.success('已清理'); fetchData() } catch { message.error('清理失败') } }
  })
}

// Batch operations
const handleBatchEnable = () => {
  dialog.warning({ title: '批量启用', content: `确定启用选中的 ${checkedRowKeys.value.length} 个用户？`, positiveText: '确定', negativeText: '取消',
    onPositiveClick: async () => {
      try {
        const selected = tableData.value.filter(r => checkedRowKeys.value.includes(r.id))
        await Promise.all(selected.filter(r => !r.is_active).map(r => toggleUserActive(r.user_id)))
        message.success('批量启用完成'); checkedRowKeys.value = []; fetchData()
      } catch { message.error('批量启用失败') }
    }
  })
}

const handleBatchDisable = () => {
  const selected = tableData.value.filter(r => checkedRowKeys.value.includes(r.id))
  const toDisable = selected.filter(r => r.is_active && r.user_id !== userStore.userInfo?.id)
  if (toDisable.length === 0) {
    message.warning(selected.some(r => r.user_id === userStore.userInfo?.id) ? '不能禁用自己，已跳过' : '没有可禁用的用户')
    return
  }
  if (selected.some(r => r.user_id === userStore.userInfo?.id)) message.info('已排除当前登录账号')
  dialog.warning({ title: '批量禁用', content: `确定禁用选中的 ${toDisable.length} 个用户？`, positiveText: '确定', negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await Promise.all(toDisable.map(r => toggleUserActive(r.user_id)))
        message.success('批量禁用完成'); checkedRowKeys.value = []; fetchData()
      } catch { message.error('批量禁用失败') }
    }
  })
}

const handleBatchEmail = async () => {
  try {
    await Promise.all(checkedRowKeys.value.map(id => sendSubscriptionEmail(id)))
    message.success('批量发送完成'); checkedRowKeys.value = []
  } catch { message.error('批量发送失败') }
}

const handleBatchDelete = () => {
  dialog.error({ title: '批量删除', content: `确定删除选中的 ${checkedRowKeys.value.length} 个用户及其所有数据？不可恢复！`, positiveText: '删除', negativeText: '取消',
    onPositiveClick: async () => {
      try {
        const selected = tableData.value.filter(r => checkedRowKeys.value.includes(r.id))
        await Promise.all(selected.map(r => deleteUserFull(r.user_id)))
        message.success('批量删除完成'); checkedRowKeys.value = []; fetchData()
      } catch { message.error('批量删除失败') }
    }
  })
}

onMounted(() => fetchData())
</script>
<style scoped>
.subscriptions-container { padding: 20px; }
/* Desktop inline cells */
:deep(.inline-cell) { padding: 6px; border-radius: 6px; }
:deep(.cell-expired) { background: #fef0f0; border: 1px solid #f56c6c; }
:deep(.cell-overlimit) { background: #fef0f0; border: 1px solid #f56c6c; }
:deep(.inline-quick-btns) { display: flex; gap: 4px; margin-top: 4px; justify-content: center; }
:deep(.inline-quick-btns .n-button) { flex: 1; }
:deep(.action-btn-grid) { display: grid; grid-template-columns: repeat(3, 1fr); gap: 4px; }
/* Mobile cards */
.mobile-card-list { display: flex; flex-direction: column; gap: 12px; }
.sub-card { background: #fff; border-radius: 12px; overflow: hidden; box-shadow: 0 1px 4px rgba(0,0,0,0.08); }
.sub-card-header { display: flex; align-items: center; justify-content: space-between; padding: 12px 14px; border-bottom: 1px solid #f0f0f0; }
.sub-user-info { display: flex; align-items: center; gap: 10px; flex: 1; min-width: 0; }
.sub-avatar { width: 36px; height: 36px; border-radius: 50%; background: #667eea; color: #fff; display: flex; align-items: center; justify-content: center; font-weight: 600; font-size: 15px; flex-shrink: 0; }
.sub-user-meta { flex: 1; min-width: 0; }
.sub-user-name { font-weight: 600; font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sub-user-id { font-size: 12px; color: #999; margin-top: 2px; }
.sub-section { padding: 10px 14px; border-bottom: 1px solid #f5f5f5; }
.section-expired { background: #fef0f0; }
.section-overlimit { background: #fef0f0; }
.sub-section-row { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.sub-section-label { font-size: 13px; color: #909399; }
.sub-section-value { font-size: 14px; font-weight: 600; }
.sub-btn-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 6px; }
.sub-btn-row-5 { grid-template-columns: repeat(5, 1fr); }
.sub-btn-row-6 { grid-template-columns: repeat(6, 1fr); }
.sub-action-grid { display: grid; grid-template-columns: repeat(4, 1fr); padding: 10px 8px; }
.sub-action-item { display: flex; flex-direction: column; align-items: center; gap: 4px; padding: 8px 4px; cursor: pointer; border-radius: 8px; }
.sub-action-item:active { background: #f5f7fa; }
.sub-action-item.is-disabled { opacity: 0.5; cursor: not-allowed; pointer-events: none; }
.sub-action-icon { width: 40px; height: 40px; border-radius: 10px; display: flex; align-items: center; justify-content: center; }
.sub-action-item span { font-size: 11px; color: #606266; }
.qr-grid { display: flex; gap: 24px; justify-content: center; flex-wrap: wrap; }
.qr-item { text-align: center; }
.qr-title { font-weight: 600; margin-bottom: 8px; }
.url-row { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.url-label { font-size: 12px; color: #666; min-width: 40px; }
.url-text { font-size: 12px; word-break: break-all; color: #333; background: #f5f5f5; padding: 2px 6px; border-radius: 3px; }
@media (max-width: 767px) {
  .subscriptions-container { padding: 8px; }
  .qr-grid { flex-direction: column; align-items: center; }
}
</style>
