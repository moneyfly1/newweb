<template>
  <div class="dashboard">
    <!-- Modern Welcome Card -->
    <div class="welcome-card">
      <div class="welcome-content">
        <div class="welcome-left">
          <h1 class="welcome-title">{{ greetingText }}，{{ info.username || '用户' }} 👋</h1>
          <div class="user-meta">
            <div class="level-badge" :style="{ background: levelColor }">
              <n-icon size="16" :component="RibbonOutline" />
              <span>{{ info.level_name || 'Lv.0' }}</span>
            </div>
            <n-tag v-if="info.discount_rate" type="success" size="small">
              {{ (info.discount_rate * 100).toFixed(0) }}% 折扣
            </n-tag>
          </div>
        </div>
        <div class="welcome-stats">
          <div class="stat-card balance-card">
            <div class="stat-icon">
              <n-icon size="24" :component="WalletOutline" />
            </div>
            <div class="stat-info">
              <span class="stat-label">账户余额</span>
              <span class="stat-value">¥{{ info.balance?.toFixed(2) || '0.00' }}</span>
            </div>
            <n-button type="primary" size="small" @click="$router.push('/recharge')">
              充值
            </n-button>
          </div>
          <div class="stat-card checkin-card">
            <div class="stat-icon">
              <n-icon size="24" :component="CalendarOutline" />
            </div>
            <div class="stat-info">
              <span class="stat-label">连续签到</span>
              <span class="stat-value">{{ checkinStatus.consecutive_days || 0 }} 天</span>
            </div>
            <n-button
              type="success"
              size="small"
              :disabled="checkinStatus.checked_in_today || checkinLoading"
              :loading="checkinLoading"
              @click="handleCheckIn"
            >
              {{ checkinStatus.checked_in_today ? '已签到' : '签到' }}
            </n-button>
          </div>
        </div>
      </div>
    </div>

    <!-- Main Two-Column -->
    <div class="main-grid">
      <!-- Left Column -->
      <div class="left-col">
        <!-- Subscription Info -->
        <div class="card">
          <div class="card-header">
            <span class="card-title">订阅信息</span>
            <n-button text type="primary" size="small" @click="$router.push('/subscription')">管理</n-button>
          </div>
          <n-spin :show="subscriptionLoading">
            <div v-if="subscription.token_url || subscription.token_clash_url" class="sub-info">
              <div class="sub-stats-row">
                <div class="sub-stat"><span class="sub-stat-label">剩余</span><n-tag :type="remainingDaysType" size="small" :bordered="false">{{ remainingDays }}天</n-tag></div>
                <div class="sub-stat"><span class="sub-stat-label">设备</span><span class="sub-stat-val">{{ subscription.current_devices || 0 }}/{{ subscription.device_limit || 0 }}</span></div>
                <div class="sub-stat"><span class="sub-stat-label">状态</span><n-tag :type="subscription.is_active ? 'success' : 'error'" size="small" :bordered="false">{{ subscription.is_active ? '使用中' : '未激活' }}</n-tag></div>
              </div>
              <div class="sub-urls">
                <div class="sub-url-row" v-if="subscription.token_clash_url">
                  <span class="sub-url-label">Clash</span>
                  <n-input :value="showSubUrls ? subscription.token_clash_url : maskUrl(subscription.token_clash_url)" readonly size="tiny" style="flex:1" />
                  <n-button size="tiny" @click="showSubUrls = !showSubUrls"><template #icon><n-icon :component="showSubUrls ? EyeOffOutline : EyeOutline" /></template></n-button>
                  <n-button size="tiny" @click="copyText(subscription.token_clash_url, 'Clash')"><template #icon><n-icon :component="CopyOutline" /></template></n-button>
                </div>
                <div class="sub-url-row" v-if="subscription.token_url">
                  <span class="sub-url-label">通用</span>
                  <n-input :value="showSubUrls ? subscription.token_url : maskUrl(subscription.token_url)" readonly size="tiny" style="flex:1" />
                  <n-button size="tiny" @click="showSubUrls = !showSubUrls"><template #icon><n-icon :component="showSubUrls ? EyeOffOutline : EyeOutline" /></template></n-button>
                  <n-button size="tiny" @click="copyText(subscription.token_url, '通用')"><template #icon><n-icon :component="CopyOutline" /></template></n-button>
                </div>
              </div>
            </div>
            <n-empty v-else description="暂无订阅" size="small">
              <template #extra><n-button size="small" type="primary" @click="$router.push('/shop')">购买套餐</n-button></template>
            </n-empty>
          </n-spin>
        </div>

        <!-- Quick Subscription -->
        <div class="card">
          <div class="card-header"><span class="card-title">快速订阅</span></div>
          <div v-if="quickSubItems.length" class="quick-sub-grid">
            <div v-for="item in quickSubItems" :key="item.name" class="quick-sub-item">
              <span class="qs-icon">
                <img
                  v-if="canShowIcon(`qs:${item.name}`, item.iconUrl)"
                  class="app-icon"
                  :src="item.iconUrl"
                  :alt="item.name"
                  loading="lazy"
                  @error="markIconFailed(`qs:${item.name}`)"
                />
                <span v-else>{{ item.icon }}</span>
              </span>
              <span class="qs-name">{{ item.name }}</span>
              <div class="qs-actions">
                <n-button size="tiny" @click="copyText(item.url, item.name)"><template #icon><n-icon :component="CopyOutline" /></template></n-button>
                <n-button v-if="item.importable" size="tiny" type="primary" @click="oneClickImport(item.client)"><template #icon><n-icon :component="CloudDownloadOutline" /></template></n-button>
              </div>
            </div>
          </div>
          <n-empty v-else description="暂无订阅" size="small" />
        </div>

        <!-- Announcements -->
        <div class="card">
          <div class="card-header"><span class="card-title">最近公告</span></div>
          <n-spin :show="announcementsLoading">
            <div v-if="announcements.length" class="announcement-list">
              <div v-for="a in announcements" :key="a.id" class="announcement-item">
                <n-tag :type="a.type === 'warning' ? 'warning' : 'info'" size="small" :bordered="false">{{ a.type === 'warning' ? '重要' : '通知' }}</n-tag>
                <span class="announcement-title">{{ a.title }}</span>
              </div>
            </div>
            <n-empty v-else description="暂无公告" size="small" />
          </n-spin>
        </div>
      </div>
      <!-- Right Column -->
      <div class="right-col">
        <!-- Client Downloads -->
        <div class="card" v-if="hasAnyClientUrl">
          <div class="card-header"><span class="card-title">客户端下载</span></div>
          <n-tabs type="segment" size="small" animated>
            <n-tab-pane v-for="tab in clientTabs" :key="tab.name" :name="tab.name" :tab="tab.label">
              <div class="client-grid">
                <div v-for="c in tab.clients" :key="c.key" class="client-card" @click="openUrl(c.url)">
                  <span class="client-icon">
                    <img
                      v-if="canShowIcon(`client:${c.key}`, c.iconUrl)"
                      class="app-icon"
                      :src="c.iconUrl"
                      :alt="c.name"
                      loading="lazy"
                      @error="markIconFailed(`client:${c.key}`)"
                    />
                    <span v-else>{{ c.icon }}</span>
                  </span>
                  <span class="client-name">{{ c.name }}</span>
                  <n-icon :component="DownloadOutline" size="14" color="#999" />
                </div>
              </div>
            </n-tab-pane>
          </n-tabs>
        </div>

        <!-- Quick Actions -->
        <div class="card">
          <div class="card-header"><span class="card-title">快捷操作</span></div>
          <div class="quick-actions-grid">
            <div class="quick-action" @click="$router.push('/shop')"><n-icon size="18" :component="CartOutline" color="#667eea" /><span>购买套餐</span></div>
            <div class="quick-action" @click="$router.push('/subscription')"><n-icon size="18" :component="LinkOutline" color="#764ba2" /><span>获取订阅</span></div>
            <div class="quick-action" @click="$router.push('/tickets')"><n-icon size="18" :component="ChatbubblesOutline" color="#f093fb" /><span>提交工单</span></div>
            <div class="quick-action" @click="$router.push('/invite')"><n-icon size="18" :component="PeopleOutline" color="#4facfe" /><span>邀请好友</span></div>
          </div>
        </div>

        <!-- Recent Orders -->
        <div class="card">
          <div class="card-header">
            <span class="card-title">最近订单</span>
            <n-button text type="primary" size="small" @click="$router.push('/orders')">查看全部</n-button>
          </div>
          <n-spin :show="ordersLoading">
            <div v-if="recentOrders.length" class="order-list">
              <div v-for="o in recentOrders" :key="o.id" class="order-item">
                <div class="order-left">
                  <span class="order-name">{{ o.package_name || '订单' }}</span>
                  <span class="order-time">{{ formatDate(o.created_at) }}</span>
                </div>
                <div class="order-right">
                  <span class="order-amount">¥{{ o.final_amount }}</span>
                  <n-tag :type="orderStatusType(o.status)" size="small" :bordered="false">{{ orderStatusText(o.status) }}</n-tag>
                </div>
              </div>
            </div>
            <n-empty v-else description="暂无订单" size="small" />
          </n-spin>
        </div>
      </div>
    </div>

    <!-- QR Code Modal -->
    <n-modal v-model:show="showQrCode" preset="card" title="Shadowrocket 二维码" style="max-width: 360px;">
      <div style="text-align: center;">
        <canvas ref="dashQrCanvas" style="max-width: 200px; border-radius: 8px;" />
        <p style="margin-top: 10px; font-size: 13px; color: #999;">使用 Shadowrocket 扫描二维码即可添加订阅</p>
      </div>
    </n-modal>
  </div>
</template>
<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useMessage } from 'naive-ui'
import {
  WalletOutline, RibbonOutline, CartOutline, LinkOutline,
  ChatbubblesOutline, PeopleOutline, CopyOutline, CloudDownloadOutline,
  DownloadOutline, QrCodeOutline, CalendarOutline, EyeOutline, EyeOffOutline,
} from '@vicons/ionicons5'
import { getDashboardInfo, checkIn, getCheckInStatus } from '@/api/user'
import { listPublicAnnouncements, getPublicConfig } from '@/api/common'
import { listOrders } from '@/api/order'
import { getSubscription } from '@/api/subscription'
import { copyToClipboard as clipboardCopy } from '@/utils/clipboard'

const message = useMessage()

const info = ref<any>({})
const subscription = ref<any>({})
const announcements = ref<any[]>([])
const recentOrders = ref<any[]>([])
const announcementsLoading = ref(false)
const ordersLoading = ref(false)
const subscriptionLoading = ref(false)
const showQrCode = ref(false)
const dashQrCanvas = ref<HTMLCanvasElement | null>(null)
const showSubUrls = ref(false)

function maskUrl(url: string) {
  if (!url || url.length < 20) return '••••••••'
  return url.substring(0, 20) + '••••••••' + url.substring(url.length - 6)
}
const clientConfig = ref<Record<string, string>>({})
const checkinStatus = ref<any>({})
const checkinLoading = ref(false)

async function handleCheckIn() {
  checkinLoading.value = true
  try {
    const res: any = await checkIn()
    const amt = Number(res.data.amount)
    const amtStr = amt < 1 ? amt.toFixed(2) : amt.toString()
    message.success(`签到成功！获得 ${amtStr} 元奖励，已连续签到 ${res.data.consecutive_days} 天`)
    checkinStatus.value.checked_in_today = true
    checkinStatus.value.consecutive_days = res.data.consecutive_days
    try { const dashRes: any = await getDashboardInfo(); if (dashRes.data) info.value.balance = dashRes.data.balance } catch {}
  } catch (error: any) { message.error(error.message || '签到失败') }
  finally { checkinLoading.value = false }
}

const allClients = {
  windows: [
    { key: 'client_clash_windows_url', name: 'Clash for Windows', icon: '🔵', iconUrl: 'https://fastly.jsdelivr.net/gh/walkxcode/dashboard-icons@main/png/clash.png' },
    { key: 'client_v2rayn_url', name: 'V2rayN', icon: '🟢', iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/V2ray.png' },
    { key: 'client_clashparty_windows_url', name: 'Clash Party', icon: '🟣', iconUrl: 'https://fastly.jsdelivr.net/gh/mihomo-party-org/clash-party@smart_core/images/icon-black.png' },
    { key: 'client_hiddify_windows_url', name: 'Hiddify', icon: '🟠' },
    { key: 'client_flclash_windows_url', name: 'FlClash', icon: '⚡', iconUrl: 'https://fastly.jsdelivr.net/gh/chen08209/FlClash@main/assets/images/icon.png' },
  ],
  android: [
    { key: 'client_clash_android_url', name: 'Clash Meta', icon: '🔵', iconUrl: 'https://fastly.jsdelivr.net/gh/walkxcode/dashboard-icons@main/png/clash.png' },
    { key: 'client_v2rayng_url', name: 'V2rayNG', icon: '🟢', iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/V2ray.png' },
    { key: 'client_hiddify_android_url', name: 'Hiddify', icon: '🟠' },
  ],
  macos: [
    { key: 'client_flclash_macos_url', name: 'FlClash', icon: '⚡', iconUrl: 'https://fastly.jsdelivr.net/gh/chen08209/FlClash@main/assets/images/icon.png' },
    { key: 'client_clashparty_macos_url', name: 'Clash Party', icon: '🟣', iconUrl: 'https://fastly.jsdelivr.net/gh/mihomo-party-org/clash-party@smart_core/images/icon-black.png' },
  ],
  ios: [
    { key: 'client_shadowrocket_url', name: 'Shadowrocket', icon: '🚀', iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/shadowrocket.png' },
    { key: 'client_stash_url', name: 'Stash', icon: '🟡', iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/stash.png' },
  ],
  linux: [
    { key: 'client_clash_linux_url', name: 'Clash', icon: '🐧', iconUrl: 'https://fastly.jsdelivr.net/gh/walkxcode/dashboard-icons@main/png/clash.png' },
    { key: 'client_singbox_url', name: 'Sing-box', icon: '📦' },
  ],
}

const filterClients = (list: typeof allClients.windows) =>
  list.filter(c => clientConfig.value[c.key]).map(c => ({ ...c, url: clientConfig.value[c.key] }))

const windowsClients = computed(() => filterClients(allClients.windows))
const androidClients = computed(() => filterClients(allClients.android))
const macClients = computed(() => filterClients(allClients.macos))
const iosClients = computed(() => filterClients(allClients.ios))
const linuxClients = computed(() => filterClients(allClients.linux))
const hasAnyClientUrl = computed(() =>
  windowsClients.value.length || androidClients.value.length || macClients.value.length || iosClients.value.length || linuxClients.value.length
)

const clientTabs = computed(() => [
  { name: 'windows', label: 'Windows', clients: windowsClients.value },
  { name: 'android', label: 'Android', clients: androidClients.value },
  { name: 'macos', label: 'macOS', clients: macClients.value },
  { name: 'ios', label: 'iOS', clients: iosClients.value },
  { name: 'linux', label: 'Linux', clients: linuxClients.value },
].filter(t => t.clients.length))

const quickSubItems = computed(() => {
  const s = subscription.value
  return [
    {
      name: 'Clash / Meta', icon: '⚔️',
      iconUrl: 'https://fastly.jsdelivr.net/gh/walkxcode/dashboard-icons@main/png/clash.png',
      url: s.token_clash_url, client: 'clash', importable: true,
    },
    {
      name: 'Stash', icon: '📦',
      iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/stash.png',
      url: s.token_stash_url || s.token_clash_url, client: 'stash', importable: true,
    },
    {
      name: 'Surge', icon: '🌊',
      iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/surge.png',
      url: s.token_surge_url, client: 'surge', importable: true,
    },
    {
      name: 'Loon', icon: '🎈',
      iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/loon.png',
      url: s.token_loon_url, client: 'loon', importable: true,
    },
    {
      name: 'QuantumultX', icon: '💠',
      iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/quantumultx.png',
      url: s.token_quantumultx_url, client: 'quantumultx', importable: true,
    },
    {
      name: 'Shadowrocket', icon: '🔴',
      iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/shadowrocket.png',
      url: s.token_url, client: 'shadowrocket', importable: true,
    },
    {
      name: 'SingBox', icon: '📱',
      iconUrl: 'https://raw.githubusercontent.com/SagerNet/sing-box/testing/docs/assets/icon.svg',
      url: s.token_singbox_url, client: 'singbox', importable: false,
    },
    {
      name: 'V2Ray / Hiddify', icon: '🚀',
      iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/V2ray.png',
      url: s.token_url, client: 'v2ray', importable: false,
    },
  ].filter(i => i.url)
})

const iconFailed = ref<Record<string, boolean>>({})
function markIconFailed(key: string) {
  iconFailed.value[key] = true
}
function canShowIcon(key: string, url?: string) {
  return !!url && !iconFailed.value[key]
}

const greetingText = computed(() => {
  const h = new Date().getHours()
  if (h < 6) return '夜深了'
  if (h < 12) return '早上好'
  if (h < 14) return '中午好'
  if (h < 18) return '下午好'
  return '晚上好'
})

const levelColor = computed(() => {
  const colors: Record<string, string> = {
    'Lv.0': '#999999', 'Lv.1': '#52c41a', 'Lv.2': '#1890ff',
    'Lv.3': '#722ed1', 'Lv.4': '#eb2f96', 'Lv.5': '#fa8c16',
  }
  return colors[info.value.level_name] || '#667eea'
})

const levelProgress = computed(() => {
  const current = info.value.current_exp || 0
  const next = info.value.next_level_exp || 100
  return Math.min((current / next) * 100, 100)
})

const remainingDays = computed(() => {
  if (!subscription.value.expire_time) return 0
  const days = Math.ceil((new Date(subscription.value.expire_time).getTime() - Date.now()) / (1000 * 60 * 60 * 24))
  return Math.max(days, 0)
})

const remainingDaysType = computed(() => {
  if (remainingDays.value > 30) return 'success'
  if (remainingDays.value >= 7) return 'warning'
  return 'error'
})

// 本地生成 QR 码，不发送订阅 URL 到第三方服务
const qrCodeData = computed(() => {
  const url = subscription.value.token_url
  if (!url) return ''
  const subName = info.value.site_name || '订阅'
  return url + (url.includes('#') ? '' : `#${encodeURIComponent(subName)}`)
})

watch(showQrCode, async (val) => {
  if (val && qrCodeData.value) {
    await nextTick()
    if (dashQrCanvas.value) {
      const QRCode = (await import('qrcode')).default
      QRCode.toCanvas(dashQrCanvas.value, qrCodeData.value, { width: 200, margin: 2 })
    }
  }
})

const orderStatusType = (status: string) => {
  const map: Record<string, string> = { pending: 'warning', paid: 'success', cancelled: 'default', expired: 'error', refunded: 'info' }
  return (map[status] || 'default') as any
}
const orderStatusText = (status: string) => {
  const map: Record<string, string> = { pending: '待支付', paid: '已支付', cancelled: '已取消', expired: '已过期', refunded: '已退款' }
  return map[status] || status
}

function formatDate(d: string) { if (!d) return ''; return new Date(d).toLocaleDateString('zh-CN') }

async function copyText(text: string, label: string) {
  const ok = await clipboardCopy(text)
  ok ? message.success(`${label}已复制到剪贴板`) : message.error('复制失败，请手动复制')
}
function oneClickImport(client: string) {
  const s = subscription.value
  const subName = info.value.site_name || '订阅'
  const getUrl = (key: string) => s[key] || s.token_url || ''
  switch (client) {
    case 'clash':
      window.location.href = `clash://install-config?url=${encodeURIComponent(getUrl('token_clash_url'))}&name=${encodeURIComponent(subName)}`; break
    case 'stash':
      window.location.href = `stash://install-config?url=${encodeURIComponent(getUrl('token_stash_url') || getUrl('token_clash_url'))}&name=${encodeURIComponent(subName)}`; break
    case 'surge':
      window.location.href = `surge:///install-config?url=${encodeURIComponent(getUrl('token_surge_url'))}`; break
    case 'loon':
      window.location.href = `loon://import/proxy?url=${encodeURIComponent(getUrl('token_loon_url'))}`; break
    case 'quantumultx':
      window.location.href = `quantumult-x:///add-resource?remote-resource=${encodeURIComponent(JSON.stringify({ server_remote: [getUrl('token_quantumultx_url')] }))}`; break
    case 'shadowrocket':
      window.location.href = `shadowrocket://add/${encodeURIComponent(getUrl('token_url'))}`; break
    default:
      copyText(getUrl('token_url'), '订阅地址'); return
  }
  message.info(`正在打开 ${client} 客户端...`)
}

function openUrl(url: string) { if (url) window.open(url, '_blank') }

const loadDashboardData = async () => {
  const [dashRes, subRes, ordersRes, checkinRes] = await Promise.allSettled([
    getDashboardInfo(),
    getSubscription(),
    listOrders({ page: 1, page_size: 5 }),
    getCheckInStatus(),
  ])

  if (dashRes.status === 'fulfilled') { const res: any = dashRes.value; info.value = res.data || {} }
  if (subRes.status === 'fulfilled') { const res: any = subRes.value; subscription.value = res.data || {} }
  if (ordersRes.status === 'fulfilled') { const res: any = ordersRes.value; recentOrders.value = (res.data?.items || []).slice(0, 5) }
  if (checkinRes.status === 'fulfilled') { const res: any = checkinRes.value; if (res.data) checkinStatus.value = res.data }
}

const handleVisibilityChange = () => {
  if (!document.hidden) {
    // Page became visible, refresh data
    loadDashboardData().catch(() => {})
  }
}

onMounted(async () => {
  subscriptionLoading.value = true
  announcementsLoading.value = true
  ordersLoading.value = true

  const [dashRes, configRes, subRes, annRes, ordersRes, checkinRes] = await Promise.allSettled([
    getDashboardInfo(),
    getPublicConfig(),
    getSubscription(),
    listPublicAnnouncements(),
    listOrders({ page: 1, page_size: 5 }),
    getCheckInStatus(),
  ])

  if (dashRes.status === 'fulfilled') { const res: any = dashRes.value; info.value = res.data || {} }
  if (configRes.status === 'fulfilled') {
    const res: any = configRes.value
    if (res.data) { clientConfig.value = res.data; if (res.data.site_name) info.value.site_name = res.data.site_name }
  }
  if (subRes.status === 'fulfilled') { const res: any = subRes.value; subscription.value = res.data || {} }
  subscriptionLoading.value = false
  if (annRes.status === 'fulfilled') { const res: any = annRes.value; announcements.value = (res.data?.items || res.data || []).slice(0, 5) }
  announcementsLoading.value = false
  if (ordersRes.status === 'fulfilled') { const res: any = ordersRes.value; recentOrders.value = (res.data?.items || []).slice(0, 5) }
  ordersLoading.value = false
  if (checkinRes.status === 'fulfilled') { const res: any = checkinRes.value; if (res.data) checkinStatus.value = res.data }

  // Listen for visibility changes to auto-refresh
  document.addEventListener('visibilitychange', handleVisibilityChange)
})

onUnmounted(() => {
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
</script>
<style scoped>
.dashboard { padding: 0; }

/* Welcome Card */
.welcome-card { background: linear-gradient(135deg, #e0c3fc 0%, #8ec5fc 100%); border-radius: 12px; padding: 20px 24px; margin-bottom: 12px; color: #333; }
.welcome-content { display: flex; justify-content: space-between; align-items: center; gap: 24px; }
.welcome-left { flex: 1; }
.welcome-title { font-size: 22px; font-weight: 700; margin: 0 0 8px 0; text-shadow: 0 1px 2px rgba(255,255,255,0.5); color: #333; }
.user-meta { display: flex; align-items: center; gap: 8px; }
.level-badge { display: inline-flex; align-items: center; gap: 4px; padding: 4px 12px; border-radius: 16px; color: white; font-weight: 600; font-size: 13px; background: rgba(102,126,234,0.85); }
.welcome-stats { display: flex; gap: 12px; }
.stat-card { background: rgba(255,255,255,0.7); border-radius: 12px; padding: 14px 16px; display: flex; align-items: center; gap: 12px; backdrop-filter: blur(10px); min-width: 200px; border: 1px solid rgba(255,255,255,0.9); }
.stat-icon { width: 40px; height: 40px; border-radius: 10px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); display: flex; align-items: center; justify-content: center; flex-shrink: 0; color: white; }
.stat-info { display: flex; flex-direction: column; gap: 2px; flex: 1; }
.stat-label { font-size: 12px; opacity: 0.8; color: #555; }
.stat-value { font-size: 18px; font-weight: 700; line-height: 1.2; color: #333; }
.stat-card .n-button { flex-shrink: 0; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%) !important; color: white !important; border: none !important; font-weight: 600 !important; }
.stat-card .n-button:hover { opacity: 0.9; }
.stat-card .n-button:disabled { background: rgba(102,126,234,0.5) !important; color: rgba(255,255,255,0.7) !important; }

/* Top Bar */
.top-bar {
  background: linear-gradient(135deg, #4a5fd7 0%, #7c3aed 100%);
  border-radius: 10px; padding: 14px 20px; color: white; position: relative; overflow: hidden;
  display: flex; justify-content: space-between; align-items: center;
}
.welcome-section { display: flex; align-items: center; gap: 24px; flex: 1; z-index: 1; }
.welcome-title { font-size: 18px; font-weight: 700; margin: 0; white-space: nowrap; text-shadow: 0 1px 2px rgba(0,0,0,0.15); }
.welcome-decoration { position: absolute; right: -20px; top: -20px; width: 100px; height: 100px; border-radius: 50%; background: rgba(255,255,255,0.08); }
.top-stats { display: flex; align-items: center; gap: 16px; }
.top-stat { display: flex; align-items: center; gap: 6px; }
.top-stat-divider { width: 1px; height: 20px; background: rgba(255,255,255,0.35); }
.top-stat-label { font-size: 12px; opacity: 0.9; }
.top-stat-val { font-size: 15px; font-weight: 700; }
/* Solid white buttons on gradient backgrounds */
.gradient-btn { background: rgba(255,255,255,0.95) !important; color: #4a5fd7 !important; border: none !important; font-weight: 600 !important; }
.gradient-btn:hover { background: #fff !important; }
.gradient-btn:disabled { background: rgba(255,255,255,0.5) !important; color: rgba(74,95,215,0.6) !important; }
.level-badge { display: inline-flex; align-items: center; gap: 4px; padding: 2px 10px; border-radius: 12px; color: white; font-weight: 600; font-size: 12px; }
/* Tags on gradient background need solid styling */
.top-bar :deep(.n-tag) { background: rgba(255,255,255,0.2) !important; color: white !important; font-weight: 600; }
.top-bar :deep(.n-tag .n-tag__content) { color: white !important; }

/* Main Grid */
.main-grid { display: grid; grid-template-columns: 1.25fr 1fr; gap: 12px; margin-top: 12px; }
.left-col, .right-col { display: flex; flex-direction: column; gap: 12px; }

/* Card */
.card { background: white; border-radius: 10px; padding: 14px 16px; box-shadow: 0 1px 3px rgba(0,0,0,0.06); }
.card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.card-title { font-size: 14px; font-weight: 600; color: #333; }

/* Subscription Info */
.sub-stats-row { display: flex; gap: 16px; margin-bottom: 10px; }
.sub-stat { display: flex; align-items: center; gap: 6px; }
.sub-stat-label { font-size: 12px; color: #999; }
.sub-stat-val { font-size: 13px; font-weight: 600; }
.sub-urls { display: flex; flex-direction: column; gap: 6px; }
.sub-url-row { display: flex; align-items: center; gap: 6px; }
.sub-url-label { font-size: 12px; color: #666; min-width: 36px; font-weight: 500; }

/* Quick Subscription */
.quick-sub-grid { display: flex; flex-direction: column; gap: 6px; }
.quick-sub-item { display: flex; align-items: center; gap: 8px; padding: 6px 8px; border-radius: 6px; background: #f8f8fa; }
.qs-icon { font-size: 16px; }
.qs-name { flex: 1; font-size: 13px; font-weight: 500; }
.qs-actions { display: flex; gap: 4px; }

/* Client Downloads */
.client-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 6px; margin-top: 6px; }
.client-card { display: flex; align-items: center; gap: 6px; padding: 8px; border-radius: 6px; background: #f8f8fa; cursor: pointer; transition: background 0.2s; }
.client-card:hover { background: #eef0f5; }
.client-icon { font-size: 16px; }
.client-name { flex: 1; font-size: 12px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.app-icon { width: 18px; height: 18px; object-fit: contain; border-radius: 4px; }

/* Quick Actions */
.quick-actions-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; }
.quick-action { display: flex; flex-direction: column; align-items: center; gap: 4px; padding: 10px 6px; border-radius: 8px; background: #f8f8fa; cursor: pointer; transition: background 0.2s; }
.quick-action:hover { background: #eef0f5; }
.quick-action span { font-size: 11px; color: #555; }

/* Announcements */
.announcement-list { display: flex; flex-direction: column; gap: 6px; }
.announcement-item { display: flex; align-items: center; gap: 8px; font-size: 13px; }
.announcement-title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

/* Orders */
.order-list { display: flex; flex-direction: column; gap: 6px; }
.order-item { display: flex; justify-content: space-between; align-items: center; padding: 6px 8px; border-radius: 6px; background: #f8f8fa; }
.order-left { display: flex; flex-direction: column; gap: 2px; }
.order-name { font-size: 13px; font-weight: 500; }
.order-time { font-size: 11px; color: #999; }
.order-right { display: flex; align-items: center; gap: 8px; }
.order-amount { font-size: 14px; font-weight: 600; color: #18a058; }

/* Mobile */
@media (max-width: 767px) {
  .dashboard { padding: 0 12px; }
  .welcome-card { padding: 16px; margin-bottom: 16px; }
  .welcome-content { flex-direction: column; align-items: flex-start; }
  .welcome-title { font-size: 20px; }
  .welcome-stats { flex-direction: column; width: 100%; }
  .stat-card { min-width: auto; width: 100%; }
  .main-grid { grid-template-columns: 1fr; }
  .client-grid { grid-template-columns: repeat(2, 1fr); }
  .quick-actions-grid { grid-template-columns: repeat(2, 1fr); }
  .sub-stats-row { flex-wrap: wrap; }
}
</style>
