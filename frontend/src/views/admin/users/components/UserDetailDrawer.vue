<template>
  <n-drawer v-model:show="showDetailDrawer" :width="appStore.isMobile ? '100%' : 780" placement="right" closable>
    <n-drawer-content :title="'用户详情 - ' + (userDetail.username || userDetail.email || '')" closable>
      <n-descriptions bordered :column="appStore.isMobile ? 1 : 2" label-placement="left" size="small">
        <n-descriptions-item label="ID">{{ userDetail.id }}</n-descriptions-item>
        <n-descriptions-item label="用户名">{{ userDetail.username || '-' }}</n-descriptions-item>
        <n-descriptions-item label="邮箱">{{ userDetail.email || '-' }}</n-descriptions-item>
        <n-descriptions-item label="余额">{{ formatCurrency(userDetail.balance) }}</n-descriptions-item>
        <n-descriptions-item label="状态">
          <n-tag :type="userDetail.is_active ? 'success' : 'error'" size="small">{{ userDetail.is_active ? '激活' : '禁用' }}</n-tag>
          <n-tag v-if="userDetail.is_admin" type="warning" size="small" class="tag-spacing">管理员</n-tag>
        </n-descriptions-item>
        <n-descriptions-item label="等级">{{ userDetail.level_name || '无' }}</n-descriptions-item>
        <n-descriptions-item label="注册时间">{{ fmtDate(userDetail.created_at) }}</n-descriptions-item>
        <n-descriptions-item label="最后登录">{{ fmtDate(userDetail.last_login) }}</n-descriptions-item>
      </n-descriptions>

      <n-divider>订阅信息</n-divider>
      <template v-if="userDetail.subscription">
        <n-descriptions bordered :column="appStore.isMobile ? 1 : 2" label-placement="left" size="small">
          <n-descriptions-item label="套餐">{{ userDetail.package_name || '-' }}</n-descriptions-item>
          <n-descriptions-item label="状态">
            <n-tag :type="subStatusType(userDetail.subscription.status)" size="small">{{ subStatusText(userDetail.subscription.status) }}</n-tag>
            <n-tag v-if="!userDetail.subscription.is_active" type="error" size="small" class="tag-spacing">已停用</n-tag>
          </n-descriptions-item>
          <n-descriptions-item label="设备">{{ userDetail.subscription.current_devices || 0 }} / {{ userDetail.subscription.device_limit || 0 }}</n-descriptions-item>
          <n-descriptions-item label="到期时间">{{ fmtDate(userDetail.subscription.expire_time) }}</n-descriptions-item>
        </n-descriptions>
        <div v-if="userDetail.subscription_urls" class="url-section">
          <div v-for="item in getPrimarySubscriptionUrlRows()" :key="item.key" class="url-row">
            <span class="url-label">{{ item.label }}</span>
            <button type="button" class="url-text url-copy" :title="`点击复制：${item.url}`" @click="copySubscriptionUrl(item.url, item.label)">
              {{ item.url || '-' }}
            </button>
          </div>
          <n-collapse v-if="getMoreSubscriptionUrlRows().length" class="more-url-collapse">
            <n-collapse-item title="更多订阅地址" name="more-subscription-urls">
              <div v-for="item in getMoreSubscriptionUrlRows()" :key="item.key" class="url-row">
                <span class="url-label">{{ item.label }}</span>
                <button type="button" class="url-text url-copy" :title="`点击复制：${item.url}`" @click="copySubscriptionUrl(item.url, item.label)">
                  {{ item.url }}
                </button>
              </div>
            </n-collapse-item>
          </n-collapse>
        </div>
      </template>
      <n-empty v-else description="暂无订阅" size="small" />

      <n-tabs type="line" class="tabs-spacing" animated>
        <n-tab-pane name="custom-nodes" tab="专线分配">
          <div class="assign-panel">
            <n-form label-placement="top" size="small">
              <n-grid :cols="appStore.isMobile ? 1 : 2" :x-gap="12" :y-gap="8">
                <n-form-item-gi label="选择专线节点">
                  <n-select
                    v-model:value="assignCustomNodeIds"
                    multiple
                    filterable
                    remote
                    clearable
                    placeholder="搜索并选择专线节点"
                    :options="customNodeOptions"
                    :loading="loadingCustomNodeOptions"
                    @search="fetchCustomNodeOptions"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="专线独立到期时间">
                  <n-date-picker
                    v-model:value="assignExpiresAt"
                    type="datetime"
                    clearable
                    style="width: 100%"
                    placeholder="不设置则跟随订阅到期"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="显示模式">
                  <n-switch v-model:value="assignDedicatedOnly">
                    <template #checked>只显示专线节点</template>
                    <template #unchecked>显示全部节点</template>
                  </n-switch>
                </n-form-item-gi>
                <n-form-item-gi label="限制设备数量">
                  <n-switch v-model:value="assignLimitDevices">
                    <template #checked>跟随系统限制</template>
                    <template #unchecked>不限制设备数量</template>
                  </n-switch>
                </n-form-item-gi>
              </n-grid>
              <div class="assign-actions">
                <n-text depth="3" class="assign-hint">{{ customNodeAssignHint }}</n-text>
                <n-button type="primary" size="small" :loading="assigningCustomNode" @click="handleAssignCustomNodes">
                  分配专线
                </n-button>
              </div>
            </n-form>
          </div>
          <n-data-table
            v-if="userCustomNodes.length"
            :columns="customNodeCols"
            :data="userCustomNodes"
            :bordered="false"
            size="small"
            :loading="loadingUserCustomNodes"
            :max-height="280"
            :scroll-x="880"
            class="custom-node-table"
          />
          <n-empty v-else description="暂无专线分配" size="small" />
        </n-tab-pane>
        <n-tab-pane name="orders" tab="订单记录">
          <n-data-table v-if="(userDetail.recent_orders || []).length" :columns="orderCols" :data="userDetail.recent_orders" :bordered="false" size="small" :max-height="240" />
          <n-empty v-else description="暂无订单" size="small" />
        </n-tab-pane>
        <n-tab-pane name="devices" tab="设备记录">
          <n-data-table v-if="(userDetail.devices || []).length" :columns="deviceCols" :data="userDetail.devices" :bordered="false" size="small" :max-height="240" />
          <n-empty v-else description="暂无设备" size="small" />
        </n-tab-pane>
        <n-tab-pane name="logins" tab="登录历史">
          <n-data-table v-if="(userDetail.login_history || []).length" :columns="loginCols" :data="userDetail.login_history" :bordered="false" size="small" :max-height="240" />
          <n-empty v-else description="暂无记录" size="small" />
        </n-tab-pane>
        <n-tab-pane name="resets" tab="重置记录">
          <n-data-table v-if="(userDetail.resets || []).length" :columns="resetCols" :data="userDetail.resets" :bordered="false" size="small" :max-height="240" :scroll-x="900" />
          <n-empty v-else description="暂无记录" size="small" />
        </n-tab-pane>
        <n-tab-pane name="balance" tab="余额变动">
          <n-data-table v-if="(userDetail.balance_logs || []).length" :columns="balanceCols" :data="userDetail.balance_logs" :bordered="false" size="small" :max-height="240" />
          <n-empty v-else description="暂无记录" size="small" />
        </n-tab-pane>
        <n-tab-pane name="recharge" tab="充值记录">
          <n-data-table v-if="(userDetail.recharge_records || []).length" :columns="rechargeCols" :data="userDetail.recharge_records" :bordered="false" size="small" :max-height="240" />
          <n-empty v-else description="暂无记录" size="small" />
        </n-tab-pane>
      </n-tabs>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup>
import { computed, ref, h } from 'vue'
import { NButton, NTag, useDialog, useMessage } from 'naive-ui'
import {
  getUser,
  deleteUserDevice,
  listCustomNodes,
  getUserCustomNodes,
  assignCustomNodeToUser,
  unassignCustomNodeFromUser
} from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { formatCurrency } from '@/utils/amount'
import { parseDeviceInfo, translateBalanceChangeType, translateLoginStatus } from '@/utils/i18n'

const appStore = useAppStore()
const message = useMessage()
const dialog = useDialog()

const showDetailDrawer = ref(false)
const userDetail = ref({})
const userCustomNodes = ref([])
const customNodeOptions = ref([])
const assignCustomNodeIds = ref([])
const assignExpiresAt = ref(null)
const assignDedicatedOnly = ref(false)
const assignLimitDevices = ref(false)
const loadingUserCustomNodes = ref(false)
const loadingCustomNodeOptions = ref(false)
const assigningCustomNode = ref(false)

const open = async (userOrId) => {
  const userId = typeof userOrId === 'object' ? userOrId?.id || userOrId?.user_id : userOrId
  if (!userId) return
  try {
    const res = await getUser(userId)
    const d = res.data
    userDetail.value = {
      ...d.user,
      subscription: d.subscription || null,
      subscription_urls: d.subscription_urls || {},
      package_name: d.package_name || '',
      recent_orders: d.recent_orders || [],
      devices: d.devices || [],
      resets: d.resets || [],
      balance_logs: d.balance_logs || [],
      login_history: d.login_history || [],
      recharge_records: d.recharge_records || [],
    }
    await Promise.all([
      fetchUserCustomNodes(userId),
      fetchCustomNodeOptions('')
    ])
    showDetailDrawer.value = true
  } catch (error) {
    message.error('获取用户详情失败')
  }
}

defineExpose({ open })

const fmtDate = (d) => d ? new Date(d).toLocaleString('zh-CN') : '-'
const subStatusType = (s) => ({ active: 'success', expiring: 'warning', expired: 'error' }[s] || 'default')
const subStatusText = (s) => ({ active: '活跃', expiring: '即将到期', expired: '已过期', disabled: '已禁用' }[s] || s || '-')

const subscriptionUrlOptions = [
  { key: 'universal_url', label: '通用', primary: true },
  { key: 'clash_url', label: 'Clash', primary: true },
  { key: 'stash_url', label: 'Stash', type: 'stash' },
  { key: 'surge_url', label: 'Surge', type: 'surge' },
  { key: 'quantumultx_url', label: 'Quantumult X', type: 'quantumultx' },
  { key: 'loon_url', label: 'Loon', type: 'loon' },
  { key: 'singbox_url', label: 'Sing-Box', type: 'singbox' },
  { key: 'shadowrocket_url', label: 'Shadowrocket', type: 'shadowrocket' },
  { key: 'v2ray_url', label: 'V2Ray', type: 'v2ray' },
  { key: 'hiddify_url', label: 'Hiddify', type: 'hiddify' }
]

const buildTypedSubscriptionUrl = (base, type) => {
  if (!base || !type) return base || ''
  if (['shadowrocket', 'v2ray', 'hiddify'].includes(type)) return base
  try {
    const url = new URL(base, window.location.origin)
    url.searchParams.set('type', type)
    return url.toString()
  } catch {
    const separator = base.includes('?') ? '&' : '?'
    return `${base}${separator}type=${encodeURIComponent(type)}`
  }
}

const getSubscriptionUrl = (item) => {
  const urls = userDetail.value.subscription_urls || {}
  const directUrl = urls[item.key]
  if (directUrl) return directUrl
  const baseUrl = urls.universal_url || urls.clash_url || userDetail.value.subscription?.subscription_url || ''
  return buildTypedSubscriptionUrl(baseUrl, item.type)
}

const getSubscriptionUrlRows = (primary) => subscriptionUrlOptions
  .filter(item => Boolean(item.primary) === primary)
  .map(item => ({ ...item, url: getSubscriptionUrl(item) }))
  .filter(item => item.url)

const getPrimarySubscriptionUrlRows = () => getSubscriptionUrlRows(true)
const getMoreSubscriptionUrlRows = () => getSubscriptionUrlRows(false)

const copySubscriptionUrl = async (url, label) => {
  if (!url) return
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(url)
    } else {
      const textarea = document.createElement('textarea')
      textarea.value = url
      textarea.setAttribute('readonly', '')
      textarea.style.position = 'fixed'
      textarea.style.left = '-9999px'
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
    }
    message.success(`${label}订阅地址已复制`)
  } catch {
    message.error('复制失败，请手动复制')
  }
}

const getNodeName = (row) => row.node?.display_name || row.node?.name || row.display_name || row.name || '-'
const getNodeProtocol = (row) => row.node?.protocol || row.protocol || '-'
const getNodeEndpoint = (row) => {
  const domain = row.node?.domain || row.domain || ''
  const port = row.node?.port || row.port || ''
  if (!domain) return '-'
  return port ? `${domain}:${port}` : domain
}
const customNodeAssignHint = computed(() => {
  if (assignDedicatedOnly.value && !assignLimitDevices.value) return '用户订阅只显示专线节点，且不限制设备数量。'
  if (assignDedicatedOnly.value && assignLimitDevices.value) return '用户订阅只显示专线节点，设备数量跟随系统限制。'
  if (!assignDedicatedOnly.value && !assignLimitDevices.value) return '专线节点附加到公共节点列表，且不限制设备数量。'
  return '专线节点附加到公共节点列表，设备数量跟随系统限制。'
})

const fetchUserCustomNodes = async (userId = userDetail.value.id) => {
  if (!userId) return
  loadingUserCustomNodes.value = true
  try {
    const res = await getUserCustomNodes(userId)
    userCustomNodes.value = res.data?.items || res.data || []
  } catch (error) {
    userCustomNodes.value = []
    message.error('获取专线分配失败')
  } finally {
    loadingUserCustomNodes.value = false
  }
}

const fetchCustomNodeOptions = async (query = '') => {
  loadingCustomNodeOptions.value = true
  try {
    const res = await listCustomNodes({
      page: 1,
      page_size: 50,
      search: String(query || '').trim()
    })
    customNodeOptions.value = (res.data?.items || []).map(node => ({
      label: `${node.display_name || node.name || `专线节点 ${node.id}`} · ${node.protocol || '-'} · ${node.domain || '-'}`,
      value: node.id
    }))
  } catch (error) {
    message.error('获取专线节点失败')
  } finally {
    loadingCustomNodeOptions.value = false
  }
}

const resetAssignForm = () => {
  assignCustomNodeIds.value = []
  assignExpiresAt.value = null
  assignDedicatedOnly.value = false
  assignLimitDevices.value = false
}

const handleAssignCustomNodes = async () => {
  if (!userDetail.value.id) return
  if (!assignCustomNodeIds.value.length) {
    message.warning('请选择要分配的专线节点')
    return
  }
  assigningCustomNode.value = true
  try {
    await assignCustomNodeToUser(userDetail.value.id, {
      custom_node_ids: assignCustomNodeIds.value,
      expires_at: assignExpiresAt.value ? new Date(assignExpiresAt.value).toISOString() : null,
      dedicated_only: assignDedicatedOnly.value,
      limit_devices: assignLimitDevices.value
    })
    message.success('专线分配成功')
    resetAssignForm()
    await fetchUserCustomNodes()
  } catch (error) {
    message.error(error.message || '专线分配失败')
  } finally {
    assigningCustomNode.value = false
  }
}

const handleUnassignCustomNode = (row) => {
  const nodeId = row.custom_node_id || row.node?.id
  if (!userDetail.value.id || !nodeId) return
  dialog.warning({
    title: '确认解除专线分配',
    content: `确定要解除 ${getNodeName(row)} 的专线分配吗？`,
    positiveText: '解除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await unassignCustomNodeFromUser(userDetail.value.id, nodeId)
        message.success('已解除专线分配')
        await fetchUserCustomNodes()
      } catch (error) {
        message.error(error.message || '解除专线分配失败')
      }
    }
  })
}

const countryNameMap = {
  CN: '中国', HK: '中国香港', MO: '中国澳门', TW: '中国台湾',
  US: '美国', JP: '日本', KR: '韩国', SG: '新加坡',
  GB: '英国', UK: '英国', DE: '德国', FR: '法国',
  CA: '加拿大', AU: '澳大利亚', RU: '俄罗斯', IN: '印度',
  TH: '泰国', VN: '越南', MY: '马来西亚', PH: '菲律宾',
  ID: '印度尼西亚'
}
const countryAliasMap = {
  china: '中国', hongkong: '中国香港', 'hong kong': '中国香港',
  macao: '中国澳门', macau: '中国澳门', taiwan: '中国台湾',
  'united states': '美国', usa: '美国', 'united kingdom': '英国',
  uk: '英国', japan: '日本', korea: '韩国', 'south korea': '韩国',
  singapore: '新加坡', germany: '德国', france: '法国',
  canada: '加拿大', australia: '澳大利亚', russia: '俄罗斯',
  india: '印度', thailand: '泰国', vietnam: '越南',
  malaysia: '马来西亚', philippines: '菲律宾', indonesia: '印度尼西亚'
}
const regionDisplayNames = typeof Intl !== 'undefined' && Intl.DisplayNames
  ? new Intl.DisplayNames(['zh-CN'], { type: 'region' })
  : null

const countryNameFromText = (value) => {
  if (!value || typeof value !== 'string') return ''
  const text = value.trim()
  if (!text) return ''
  const code = text.toUpperCase()
  if (/^[A-Z]{2}$/.test(code)) return countryNameMap[code] || regionDisplayNames?.of(code) || code
  return countryAliasMap[text.toLowerCase()] || ''
}

const parseMaybeJSON = (value) => {
  if (typeof value !== 'string') return value
  const text = value.trim()
  if (!text || !/^[{[]/.test(text)) return value
  try { return JSON.parse(text) } catch { return value }
}

const formatCountryOnly = (location) => {
  const parsed = parseMaybeJSON(location)
  if (!parsed) return '-'
  if (typeof parsed === 'object') {
    const code = parsed.country_code || parsed.countryCode || parsed.country_iso || parsed.countryISO || parsed.iso_code
    const name = parsed.country_name || parsed.countryName || parsed.country || parsed.region || parsed.location
    return countryNameFromText(code) || countryNameFromText(name) || name || '-'
  }
  if (typeof parsed !== 'string') return '-'
  const text = parsed.trim()
  const directName = countryNameFromText(text)
  if (directName) return directName
  const firstSegment = text.split(/[,·|/]+/).filter(Boolean)[0]?.trim() || text
  return countryNameFromText(firstSegment) || firstSegment || '-'
}

const handleDeleteDevice = (device) => {
  dialog.warning({
    title: '确认删除设备',
    content: `确定要删除设备 ${device.device_name || device.software_name || '未知设备'} 吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteUserDevice(userDetail.value.id, device.id)
        message.success('设备已删除')
        await open(userDetail.value.id)
      } catch (error) {
        message.error('删除设备失败：' + (error.message || '未知错误'))
      }
    }
  })
}

const customNodeCols = [
  { title: '节点', key: 'name', width: 170, ellipsis: { tooltip: true }, render: (r) => getNodeName(r) },
  { title: '协议', key: 'protocol', width: 90, render: (r) => h(NTag, { size: 'small', type: 'info' }, { default: () => getNodeProtocol(r) }) },
  { title: '地址', key: 'endpoint', width: 190, ellipsis: { tooltip: true }, render: (r) => getNodeEndpoint(r) },
  { title: '显示模式', key: 'dedicated_only', width: 120, render: (r) => r.dedicated_only ? '仅专线' : '全部节点' },
  { title: '设备限制', key: 'limit_devices', width: 120, render: (r) => r.limit_devices ? '跟随系统' : '不限制' },
  { title: '独立到期', key: 'expires_at', width: 160, render: (r) => fmtDate(r.expires_at) },
  { title: '分配时间', key: 'created_at', width: 160, render: (r) => fmtDate(r.created_at) },
  { title: '操作', key: 'actions', width: 90, fixed: 'right', render: (r) => h(NButton, { size: 'small', type: 'error', secondary: true, onClick: () => handleUnassignCustomNode(r) }, { default: () => '解除' }) }
]
const orderCols = [
  { title: '订单号', key: 'order_no', width: 180, ellipsis: { tooltip: true } },
  { title: '金额', key: 'final_amount', width: 90, render: (r) => formatCurrency(r.final_amount ?? r.amount ?? 0) },
  { title: '状态', key: 'status', width: 80 },
  { title: '时间', key: 'created_at', width: 160, render: (r) => fmtDate(r.created_at) }
]
const deviceCols = [
  { title: '设备名', key: 'device_name', width: 120, ellipsis: { tooltip: true }, render: (r) => r.device_name || '未知设备' },
  { title: '客户端', key: 'software_name', width: 120, render: (r) => r.software_name || '未知' },
  { title: '版本', key: 'software_version', width: 80, render: (r) => r.software_version || '-' },
  { title: '系统', key: 'os_name', width: 80, render: (r) => r.os_name || '-' },
  { title: '设备型号', key: 'device_model', width: 130, ellipsis: { tooltip: true }, render: (r) => r.device_model || '-' },
  { title: '订阅类型', key: 'subscription_type', width: 90, render: (r) => r.subscription_type || '-' },
  { title: 'IP', key: 'ip_address', width: 130, render: (r) => r.ip_address || '-' },
  { title: '地区', key: 'region', width: 100, render: (r) => formatCountryOnly(r.location || r.region) },
  { title: '最后活跃', key: 'last_access', width: 150, render: (r) => fmtDate(r.last_access || r.updated_at) },
  { title: '操作', key: 'actions', width: 80, render: (r) => h(NButton, { size: 'small', type: 'error', secondary: true, onClick: () => handleDeleteDevice(r) }, { default: () => '删除' }) }
]
const loginCols = [
  { title: 'IP', key: 'ip_address', width: 130, render: (r) => r.ip_address || '-' },
  { title: '位置', key: 'location', width: 150, render: (r) => formatCountryOnly(r.location) },
  { title: '设备', key: 'user_agent', width: 180, ellipsis: { tooltip: true }, render: (r) => parseDeviceInfo(r.user_agent) },
  { title: '状态', key: 'login_status', width: 70, render: (r) => h(NTag, { type: r.login_status === 'success' ? 'success' : 'error', size: 'small' }, { default: () => translateLoginStatus(r.login_status) }) },
  { title: '时间', key: 'login_time', width: 160, render: (r) => fmtDate(r.login_time) }
]
const resetCols = [
  { title: '操作者', key: 'reset_by', width: 80, render: (r) => r.reset_by || '-' },
  { title: '类型', key: 'reset_type', width: 80 },
  { title: '原订阅地址', key: 'old_subscription_url', width: 180, ellipsis: { tooltip: true }, render: (r) => r.old_subscription_url || '-' },
  { title: '新订阅地址', key: 'new_subscription_url', width: 180, ellipsis: { tooltip: true }, render: (r) => r.new_subscription_url || '-' },
  { title: '设备(前/后)', key: 'devices', width: 90, render: (r) => `${r.device_count_before ?? 0} → ${r.device_count_after ?? 0}` },
  { title: '原因', key: 'reason', ellipsis: { tooltip: true } },
  { title: '时间', key: 'created_at', width: 160, render: (r) => fmtDate(r.created_at) }
]
const balanceCols = [
  { title: '类型', key: 'change_type', width: 110, render: (r) => translateBalanceChangeType(r.change_type) },
  { title: '金额', key: 'amount', width: 90, render: (r) => formatCurrency(r.amount) },
  { title: '变动后', key: 'balance_after', width: 90, render: (r) => formatCurrency(r.balance_after) },
  { title: '说明', key: 'description', ellipsis: { tooltip: true }, render: (r) => r.description || '-' },
  { title: '时间', key: 'created_at', width: 160, render: (r) => fmtDate(r.created_at) }
]
const rechargeCols = [
  { title: '金额', key: 'amount', width: 90, render: (r) => formatCurrency(r.amount) },
  { title: '方式', key: 'payment_method', width: 100 },
  { title: '状态', key: 'status', width: 80 },
  { title: '时间', key: 'created_at', width: 160, render: (r) => fmtDate(r.created_at) }
]
</script>

<style scoped>
.url-section { margin-top: 10px; }
.url-row { display: flex; align-items: flex-start; gap: 8px; margin-bottom: 6px; }
.url-label { flex: 0 0 86px; font-size: 12px; line-height: 28px; color: var(--text-color-secondary, #666); }
.url-text {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  line-height: 20px;
  word-break: break-all;
  color: var(--text-color, #333);
  background: rgba(0,0,0,0.03);
  padding: 4px 8px;
  border-radius: 4px;
}
.url-copy {
  appearance: none;
  border: 1px solid transparent;
  text-align: left;
  cursor: pointer;
  transition: background-color 0.2s ease, border-color 0.2s ease;
}
.url-copy:hover { background: rgba(24, 160, 88, 0.08); border-color: rgba(24, 160, 88, 0.28); }
.more-url-collapse { margin-top: 6px; }
.assign-panel {
  margin-bottom: 12px;
  padding: 12px;
  border: 1px solid rgba(0,0,0,0.08);
  border-radius: 6px;
  background: rgba(0,0,0,0.02);
}
.assign-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.assign-hint {
  flex: 1;
  min-width: 0;
  font-size: 12px;
}
.custom-node-table { margin-top: 8px; }
</style>
