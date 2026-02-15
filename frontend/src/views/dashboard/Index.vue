<template>
  <div class="dashboard">
    <!-- Welcome Banner -->
    <div class="welcome-banner">
      <div class="welcome-text">
        <h1 class="welcome-title">{{ greetingText }}，{{ info.username || '用户' }}</h1>
        <p class="welcome-sub">欢迎回来，祝您使用愉快</p>
      </div>
      <div class="welcome-decoration"></div>
    </div>

    <!-- User Level & Balance Cards -->
    <n-grid :x-gap="20" :y-gap="20" cols="1 l:3" responsive="screen" style="margin-top: 20px;">
      <n-gi>
        <n-card :bordered="false" class="section-card level-card">
          <div class="level-header">
            <div class="level-badge" :style="{ background: levelColor }">
              <n-icon size="20" :component="RibbonOutline" />
              <span>{{ info.level_name || 'Lv.0' }}</span>
            </div>
            <n-tag v-if="info.discount_rate" type="success" size="small" :bordered="false">
              {{ (info.discount_rate * 100).toFixed(0) }}% 折扣
            </n-tag>
          </div>
          <div class="level-progress">
            <div class="progress-info">
              <span class="progress-label">升级进度</span>
              <span class="progress-text">{{ info.current_exp || 0 }} / {{ info.next_level_exp || 100 }}</span>
            </div>
            <n-progress
              type="line"
              :percentage="levelProgress"
              :show-indicator="false"
              :color="levelColor"
              :rail-color="'rgba(0,0,0,0.06)'"
              :height="8"
              border-radius="4px"
            />
          </div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card :bordered="false" class="section-card balance-card">
          <div class="balance-content">
            <div class="balance-info">
              <div class="balance-label">账户余额</div>
              <div class="balance-value">¥{{ info.balance?.toFixed(2) || '0.00' }}</div>
            </div>
            <n-button type="primary" size="large" @click="$router.push('/recharge')">
              <template #icon>
                <n-icon :component="WalletOutline" />
              </template>
              充值
            </n-button>
          </div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card :bordered="false" class="section-card checkin-card">
          <div class="checkin-content">
            <div class="checkin-info">
              <div class="checkin-label">每日签到</div>
              <div class="checkin-days">
                <span class="checkin-days-num">{{ checkinStatus.consecutive_days || 0 }}</span>
                <span class="checkin-days-text">天连续签到</span>
              </div>
            </div>
            <n-button
              :type="checkinStatus.checked_in_today ? 'default' : 'success'"
              size="large"
              :disabled="checkinStatus.checked_in_today || checkinLoading"
              :loading="checkinLoading"
              @click="handleCheckIn"
            >
              <template #icon>
                <n-icon :component="CalendarOutline" />
              </template>
              {{ checkinStatus.checked_in_today ? '已签到' : '签到' }}
            </n-button>
          </div>
        </n-card>
      </n-gi>
    </n-grid>

    <!-- Subscription Info Card -->
    <n-card :bordered="false" class="section-card" style="margin-top: 20px;">
      <template #header>
        <span class="section-title-text">订阅信息</span>
      </template>
      <n-spin :show="subscriptionLoading">
        <div v-if="subscription.clash_url || subscription.universal_url || subscription.subscription_url" class="subscription-content">
          <div class="subscription-row">
            <div class="subscription-item" v-if="subscription.clash_url">
              <div class="sub-label">Clash 订阅地址</div>
              <div class="sub-value-row">
                <n-input
                  :value="subscription.clash_url"
                  readonly
                  size="small"
                  placeholder="暂无订阅"
                  style="flex: 1; max-width: 400px;"
                />
                <n-button size="small" @click="copyText(subscription.clash_url, 'Clash 订阅地址')">
                  <template #icon>
                    <n-icon :component="CopyOutline" />
                  </template>
                  复制
                </n-button>
              </div>
            </div>
            <div class="subscription-item" v-if="subscription.universal_url" style="margin-top: 12px;">
              <div class="sub-label">通用订阅地址 (V2Ray / Shadowrocket / Hiddify)</div>
              <div class="sub-value-row">
                <n-input
                  :value="subscription.universal_url"
                  readonly
                  size="small"
                  placeholder="暂无订阅"
                  style="flex: 1; max-width: 400px;"
                />
                <n-button size="small" @click="copyText(subscription.universal_url, '通用订阅地址')">
                  <template #icon>
                    <n-icon :component="CopyOutline" />
                  </template>
                  复制
                </n-button>
              </div>
            </div>
          </div>
          <n-divider style="margin: 16px 0;" />
          <div class="subscription-stats">
            <div class="sub-stat-item">
              <div class="sub-stat-label">剩余天数</div>
              <div class="sub-stat-value">
                <n-tag :type="remainingDaysType" size="medium" :bordered="false">
                  {{ remainingDays }} 天
                </n-tag>
              </div>
            </div>
            <div class="sub-stat-item">
              <div class="sub-stat-label">设备使用</div>
              <div class="sub-stat-value">
                <span class="device-usage">{{ subscription.current_devices || 0 }} / {{ subscription.device_limit || 0 }}</span>
              </div>
            </div>
            <div class="sub-stat-item">
              <div class="sub-stat-label">订阅状态</div>
              <div class="sub-stat-value">
                <n-tag :type="subscription.is_active ? 'success' : 'error'" size="medium" :bordered="false">
                  {{ subscription.is_active ? '使用中' : '未激活' }}
                </n-tag>
              </div>
            </div>
          </div>
        </div>
        <n-empty v-else description="暂无订阅信息" />
      </n-spin>
    </n-card>

    <!-- Quick Subscription Links -->
    <n-card :bordered="false" class="section-card" style="margin-top: 20px;">
      <template #header>
        <span class="section-title-text">快速订阅</span>
      </template>
      <div v-if="subscription.clash_url || subscription.universal_url || subscription.subscription_url" class="quick-sub-links">
        <!-- Clash -->
        <div class="quick-sub-item">
          <div class="quick-sub-info">
            <span class="client-icon">🔵</span>
            <span class="quick-sub-name">Clash 订阅</span>
          </div>
          <div class="quick-sub-actions">
            <n-button size="small" @click="copyText(subscription.clash_url || subscription.subscription_url, 'Clash 订阅地址')">
              <template #icon><n-icon :component="CopyOutline" /></template>
              复制
            </n-button>
            <n-button size="small" type="primary" @click="oneClickImport('clash')">
              <template #icon><n-icon :component="CloudDownloadOutline" /></template>
              导入
            </n-button>
          </div>
        </div>
        <!-- Shadowrocket -->
        <div class="quick-sub-item">
          <div class="quick-sub-info">
            <span class="client-icon">🚀</span>
            <span class="quick-sub-name">Shadowrocket</span>
          </div>
          <div class="quick-sub-actions">
            <n-button size="small" @click="copyText(subscription.universal_url || subscription.subscription_url, 'Shadowrocket 订阅地址')">
              <template #icon><n-icon :component="CopyOutline" /></template>
              复制
            </n-button>
            <n-button size="small" type="primary" @click="oneClickImport('shadowrocket')">
              <template #icon><n-icon :component="CloudDownloadOutline" /></template>
              导入
            </n-button>
            <n-button size="small" @click="showQrCode = true">
              <template #icon><n-icon :component="QrCodeOutline" /></template>
              二维码
            </n-button>
          </div>
        </div>
        <!-- V2Ray / Hiddify -->
        <div class="quick-sub-item">
          <div class="quick-sub-info">
            <span class="client-icon">🟢</span>
            <span class="quick-sub-name">V2Ray / Hiddify 通用</span>
          </div>
          <div class="quick-sub-actions">
            <n-button size="small" @click="copyText(subscription.universal_url || subscription.subscription_url, '通用订阅地址')">
              <template #icon><n-icon :component="CopyOutline" /></template>
              复制
            </n-button>
          </div>
        </div>
        <!-- Stash -->
        <div class="quick-sub-item">
          <div class="quick-sub-info">
            <span class="client-icon">🟡</span>
            <span class="quick-sub-name">Stash</span>
          </div>
          <div class="quick-sub-actions">
            <n-button size="small" @click="copyText(subscription.clash_url || subscription.subscription_url, 'Stash 订阅地址')">
              <template #icon><n-icon :component="CopyOutline" /></template>
              复制
            </n-button>
            <n-button size="small" type="primary" @click="oneClickImport('stash')">
              <template #icon><n-icon :component="CloudDownloadOutline" /></template>
              导入
            </n-button>
          </div>
        </div>
      </div>
      <n-empty v-else description="暂无订阅，请先购买套餐" />
    </n-card>

    <!-- Client Downloads -->
    <n-card :bordered="false" class="section-card" style="margin-top: 20px;" v-if="hasAnyClientUrl">
      <template #header>
        <span class="section-title-text">客户端下载</span>
      </template>
      <n-tabs type="segment" size="small" animated>
        <n-tab-pane name="windows" tab="Windows" v-if="windowsClients.length">
          <div class="client-grid">
            <div v-for="c in windowsClients" :key="c.key" class="client-card" @click="openUrl(c.url)">
              <span class="client-card-icon">{{ c.icon }}</span>
              <span class="client-card-name">{{ c.name }}</span>
              <n-icon :component="DownloadOutline" size="16" color="#999" />
            </div>
          </div>
        </n-tab-pane>
        <n-tab-pane name="android" tab="Android" v-if="androidClients.length">
          <div class="client-grid">
            <div v-for="c in androidClients" :key="c.key" class="client-card" @click="openUrl(c.url)">
              <span class="client-card-icon">{{ c.icon }}</span>
              <span class="client-card-name">{{ c.name }}</span>
              <n-icon :component="DownloadOutline" size="16" color="#999" />
            </div>
          </div>
        </n-tab-pane>
        <n-tab-pane name="macos" tab="macOS" v-if="macClients.length">
          <div class="client-grid">
            <div v-for="c in macClients" :key="c.key" class="client-card" @click="openUrl(c.url)">
              <span class="client-card-icon">{{ c.icon }}</span>
              <span class="client-card-name">{{ c.name }}</span>
              <n-icon :component="DownloadOutline" size="16" color="#999" />
            </div>
          </div>
        </n-tab-pane>
        <n-tab-pane name="ios" tab="iOS" v-if="iosClients.length">
          <div class="client-grid">
            <div v-for="c in iosClients" :key="c.key" class="client-card" @click="openUrl(c.url)">
              <span class="client-card-icon">{{ c.icon }}</span>
              <span class="client-card-name">{{ c.name }}</span>
              <n-icon :component="DownloadOutline" size="16" color="#999" />
            </div>
          </div>
        </n-tab-pane>
        <n-tab-pane name="linux" tab="Linux" v-if="linuxClients.length">
          <div class="client-grid">
            <div v-for="c in linuxClients" :key="c.key" class="client-card" @click="openUrl(c.url)">
              <span class="client-card-icon">{{ c.icon }}</span>
              <span class="client-card-name">{{ c.name }}</span>
              <n-icon :component="DownloadOutline" size="16" color="#999" />
            </div>
          </div>
        </n-tab-pane>
      </n-tabs>
    </n-card>

    <!-- QR Code Modal -->
    <n-modal v-model:show="showQrCode" preset="card" title="Shadowrocket 二维码" style="max-width: 360px;">
      <div class="qr-modal-content">
        <img v-if="qrCodeUrl" :src="qrCodeUrl" alt="订阅二维码" class="qr-image" />
        <p class="qr-tip">使用 Shadowrocket 扫描二维码即可添加订阅</p>
      </div>
    </n-modal>

    <!-- Quick Actions -->
    <n-card :bordered="false" class="section-card" style="margin-top: 20px;">
      <template #header>
        <span class="section-title-text">快捷操作</span>
      </template>
      <n-grid :x-gap="12" :y-gap="12" cols="2 s:4" responsive="screen">
        <n-gi>
          <div class="quick-action" @click="$router.push('/shop')">
            <n-icon size="24" :component="CartOutline" color="#667eea" />
            <span>购买套餐</span>
          </div>
        </n-gi>
        <n-gi>
          <div class="quick-action" @click="$router.push('/subscription')">
            <n-icon size="24" :component="LinkOutline" color="#764ba2" />
            <span>获取订阅</span>
          </div>
        </n-gi>
        <n-gi>
          <div class="quick-action" @click="$router.push('/tickets')">
            <n-icon size="24" :component="ChatbubblesOutline" color="#f093fb" />
            <span>提交工单</span>
          </div>
        </n-gi>
        <n-gi>
          <div class="quick-action" @click="$router.push('/invite')">
            <n-icon size="24" :component="PeopleOutline" color="#4facfe" />
            <span>邀请好友</span>
          </div>
        </n-gi>
      </n-grid>
    </n-card>

    <!-- Announcements & Recent Orders -->
    <n-grid :x-gap="20" :y-gap="20" cols="1 l:2" responsive="screen" style="margin-top: 20px;">
      <n-gi>
        <n-card :bordered="false" class="section-card">
          <template #header>
            <span class="section-title-text">最近公告</span>
          </template>
          <n-spin :show="announcementsLoading">
            <n-empty v-if="!announcements.length" description="暂无公告" />
            <div v-else class="announcement-list">
              <div v-for="a in announcements" :key="a.id" class="announcement-item">
                <n-tag :type="a.type === 'warning' ? 'warning' : 'info'" size="small" :bordered="false">
                  {{ a.type === 'warning' ? '重要' : '通知' }}
                </n-tag>
                <span class="announcement-title">{{ a.title }}</span>
                <span class="announcement-time">{{ formatDate(a.created_at) }}</span>
              </div>
            </div>
          </n-spin>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card :bordered="false" class="section-card">
          <template #header>
            <div class="section-header-row">
              <span class="section-title-text">最近订单</span>
              <n-button text type="primary" @click="$router.push('/orders')">查看全部</n-button>
            </div>
          </template>
          <n-spin :show="ordersLoading">
            <n-empty v-if="!recentOrders.length" description="暂无订单" />
            <div v-else class="order-list">
              <div v-for="o in recentOrders" :key="o.id" class="order-item">
                <div class="order-item-left">
                  <span class="order-name">{{ o.package_name || '订单' }}</span>
                  <span class="order-time">{{ formatDate(o.created_at) }}</span>
                </div>
                <div class="order-item-right">
                  <span class="order-amount">¥{{ o.final_amount }}</span>
                  <n-tag :type="orderStatusType(o.status)" size="small" :bordered="false">
                    {{ orderStatusText(o.status) }}
                  </n-tag>
                </div>
              </div>
            </div>
          </n-spin>
        </n-card>
      </n-gi>
    </n-grid>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import {
  WalletOutline, CloudOutline, PhonePortraitOutline, RibbonOutline,
  CartOutline, LinkOutline, ChatbubblesOutline, PeopleOutline,
  CopyOutline, CloudDownloadOutline, DownloadOutline, QrCodeOutline,
  CalendarOutline,
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
const clientConfig = ref<Record<string, string>>({})

// Check-in
const checkinStatus = ref<any>({})
const checkinLoading = ref(false)

async function handleCheckIn() {
  checkinLoading.value = true
  try {
    const res: any = await checkIn()
    message.success(`签到成功！获得 ${res.data.amount} 元奖励，已连续签到 ${res.data.consecutive_days} 天`)
    checkinStatus.value.checked_in_today = true
    checkinStatus.value.consecutive_days = res.data.consecutive_days
    // Refresh balance
    try {
      const dashRes: any = await getDashboardInfo()
      if (dashRes.data) info.value.balance = dashRes.data.balance
    } catch {}
  } catch (error: any) {
    message.error(error.message || '签到失败')
  } finally {
    checkinLoading.value = false
  }
}

// Client definitions
const allClients = {
  windows: [
    { key: 'client_clash_windows_url', name: 'Clash for Windows', icon: '🔵' },
    { key: 'client_v2rayn_url', name: 'V2rayN', icon: '🟢' },
    { key: 'client_mihomo_windows_url', name: 'Mihomo Party', icon: '🟣' },
    { key: 'client_hiddify_windows_url', name: 'Hiddify', icon: '🟠' },
    { key: 'client_flclash_windows_url', name: 'FlClash', icon: '⚡' },
  ],
  android: [
    { key: 'client_clash_android_url', name: 'Clash Meta', icon: '🔵' },
    { key: 'client_v2rayng_url', name: 'V2rayNG', icon: '🟢' },
    { key: 'client_hiddify_android_url', name: 'Hiddify', icon: '🟠' },
  ],
  macos: [
    { key: 'client_flclash_macos_url', name: 'FlClash', icon: '⚡' },
    { key: 'client_mihomo_macos_url', name: 'Mihomo Party', icon: '🟣' },
  ],
  ios: [
    { key: 'client_shadowrocket_url', name: 'Shadowrocket', icon: '🚀' },
    { key: 'client_stash_url', name: 'Stash', icon: '🟡' },
  ],
  linux: [
    { key: 'client_clash_linux_url', name: 'Clash', icon: '🐧' },
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
  Object.values(allClients).flat().some(c => clientConfig.value[c.key])
)

const qrCodeUrl = computed(() => {
  const url = subscription.value.universal_url || subscription.value.subscription_url
  if (!url) return ''
  return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(url)}&ecc=M&margin=10`
})

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
    'Lv.0': '#999999',
    'Lv.1': '#52c41a',
    'Lv.2': '#1890ff',
    'Lv.3': '#722ed1',
    'Lv.4': '#eb2f96',
    'Lv.5': '#fa8c16',
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
  const now = new Date().getTime()
  const expire = new Date(subscription.value.expire_time).getTime()
  const days = Math.ceil((expire - now) / (1000 * 60 * 60 * 24))
  return Math.max(days, 0)
})

const remainingDaysType = computed(() => {
  const days = remainingDays.value
  if (days > 30) return 'success'
  if (days >= 7) return 'warning'
  return 'error'
})

const orderStatusType = (status: string) => {
  const map: Record<string, string> = { pending: 'warning', paid: 'success', cancelled: 'default', expired: 'error', refunded: 'info' }
  return (map[status] || 'default') as any
}

const orderStatusText = (status: string) => {
  const map: Record<string, string> = { pending: '待支付', paid: '已支付', cancelled: '已取消', expired: '已过期', refunded: '已退款' }
  return map[status] || status
}

function formatDate(d: string) {
  if (!d) return ''
  return new Date(d).toLocaleDateString('zh-CN')
}

async function copyText(text: string, label: string) {
  const ok = await clipboardCopy(text)
  ok ? message.success(`${label}已复制到剪贴板`) : message.error('复制失败，请手动复制')
}

function oneClickImport(client: string) {
  const clashUrl = subscription.value.clash_url || subscription.value.subscription_url
  const universalUrl = subscription.value.universal_url || subscription.value.subscription_url
  if (!clashUrl && !universalUrl) {
    message.warning('暂无订阅地址')
    return
  }
  const siteName = info.value.site_name || 'CBoard'
  switch (client) {
    case 'clash':
      window.location.href = `clash://install-config?url=${encodeURIComponent(clashUrl)}&name=${encodeURIComponent(siteName)}`
      break
    case 'shadowrocket':
      window.location.href = `shadowrocket://add/sub://${btoa(universalUrl)}#${encodeURIComponent(siteName)}`
      break
    case 'stash':
      window.location.href = `clash://install-config?url=${encodeURIComponent(clashUrl)}&name=${encodeURIComponent(siteName)}`
      break
    default:
      copyText(universalUrl, '订阅地址')
      return
  }
  message.info(`正在打开 ${client} 客户端...`)
}

function openUrl(url: string) {
  if (url) window.open(url, '_blank')
}

onMounted(async () => {
  try {
    const res: any = await getDashboardInfo()
    info.value = res.data || {}
  } catch {}

  try {
    const res: any = await getPublicConfig()
    if (res.data) {
      clientConfig.value = res.data
      if (res.data.site_name) info.value.site_name = res.data.site_name
    }
  } catch {}

  subscriptionLoading.value = true
  try {
    const res: any = await getSubscription()
    subscription.value = res.data || {}
  } catch {} finally {
    subscriptionLoading.value = false
  }

  announcementsLoading.value = true
  try {
    const res: any = await listPublicAnnouncements()
    announcements.value = (res.data?.items || res.data || []).slice(0, 5)
  } catch {} finally {
    announcementsLoading.value = false
  }

  ordersLoading.value = true
  try {
    const res: any = await listOrders({ page: 1, page_size: 5 })
    recentOrders.value = (res.data?.items || []).slice(0, 5)
  } catch {} finally {
    ordersLoading.value = false
  }

  // Load check-in status
  try {
    const res: any = await getCheckInStatus()
    if (res.data) {
      checkinStatus.value = res.data
    }
  } catch {}
})
</script>

<style scoped>
.dashboard { max-width: 1200px; margin: 0 auto; }

.welcome-banner {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 16px;
  padding: 32px;
  color: white;
  position: relative;
  overflow: hidden;
}
.welcome-decoration {
  position: absolute;
  right: -20px;
  top: -20px;
  width: 160px;
  height: 160px;
  border-radius: 50%;
  background: rgba(255,255,255,0.1);
}
.welcome-title { font-size: 26px; font-weight: 700; margin: 0 0 6px 0; }
.welcome-sub { font-size: 14px; opacity: 0.85; margin: 0; }

.section-card { border-radius: 12px; }
.section-title-text { font-weight: 600; }
.section-header-row { display: flex; justify-content: space-between; align-items: center; width: 100%; }

/* Level Card */
.level-card {
  background: linear-gradient(135deg, #f5af1908, #f5af1920);
  border: 1px solid #f5af1930;
}
.level-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}
.level-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: 20px;
  color: white;
  font-weight: 600;
  font-size: 16px;
}
.level-progress {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.progress-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
}
.progress-label { color: #999; }
.progress-text { font-weight: 600; color: #333; }

/* Balance Card */
.balance-card {
  background: linear-gradient(135deg, #667eea08, #667eea20);
  border: 1px solid #667eea30;
}
.balance-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.balance-label { font-size: 13px; color: #999; margin-bottom: 8px; }
.balance-value { font-size: 32px; font-weight: 700; color: #667eea; }

/* Check-in Card */
.checkin-card {
  background: linear-gradient(135deg, #18a05808, #18a05820);
  border: 1px solid #18a05830;
}
.checkin-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.checkin-label { font-size: 13px; color: #999; margin-bottom: 8px; }
.checkin-days { display: flex; align-items: baseline; gap: 4px; }
.checkin-days-num { font-size: 32px; font-weight: 700; color: #18a058; }
.checkin-days-text { font-size: 14px; color: #666; }

/* Subscription Info */
.subscription-content {
  display: flex;
  flex-direction: column;
  gap: 0;
}
.subscription-row {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.subscription-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.sub-label {
  font-size: 13px;
  color: #999;
  font-weight: 500;
}
.sub-value-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.subscription-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 20px;
}
.sub-stat-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.sub-stat-label {
  font-size: 13px;
  color: #999;
}
.sub-stat-value {
  font-size: 16px;
  font-weight: 600;
}
.device-usage {
  color: #333;
  font-size: 18px;
}

/* Quick Subscription Links */
.quick-sub-links {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.quick-sub-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-radius: 10px;
  background: var(--n-color-embedded);
  transition: background 0.2s;
}
.quick-sub-item:hover {
  background: var(--n-color-hover);
}
.quick-sub-info {
  display: flex;
  align-items: center;
  gap: 12px;
}
.quick-sub-name {
  font-size: 15px;
  font-weight: 500;
}
.quick-sub-actions {
  display: flex;
  gap: 8px;
}

.client-icon { font-size: 20px; }

/* Client Downloads */
.client-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
  padding: 12px 0;
}
.client-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 16px;
  border-radius: 10px;
  background: var(--n-color-embedded);
  cursor: pointer;
  transition: all 0.2s;
}
.client-card:hover {
  background: var(--n-color-hover);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
}
.client-card-icon { font-size: 20px; }
.client-card-name { flex: 1; font-size: 14px; font-weight: 500; }

/* QR Code Modal */
.qr-modal-content { display: flex; flex-direction: column; align-items: center; gap: 16px; padding: 12px 0; }
.qr-image { width: 200px; height: 200px; border-radius: 8px; }
.qr-tip { font-size: 13px; color: #999; text-align: center; margin: 0; }

.quick-action {
  display: flex; flex-direction: column; align-items: center; gap: 8px;
  padding: 20px 12px; border-radius: 10px; cursor: pointer;
  transition: background 0.2s; background: var(--n-color-embedded);
}
.quick-action:hover { background: var(--n-color-hover); }
.quick-action span { font-size: 13px; color: #666; }

.announcement-list { display: flex; flex-direction: column; gap: 12px; }
.announcement-item { display: flex; align-items: center; gap: 10px; font-size: 14px; }
.announcement-title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.announcement-time { font-size: 12px; color: #bbb; flex-shrink: 0; }

.order-list { display: flex; flex-direction: column; gap: 12px; }
.order-item {
  display: flex; justify-content: space-between; align-items: center;
  padding: 10px 12px; border-radius: 8px; background: var(--n-color-embedded);
}
.order-item-left { display: flex; flex-direction: column; gap: 4px; }
.order-name { font-size: 14px; font-weight: 500; }
.order-time { font-size: 12px; color: #999; }
.order-item-right { display: flex; align-items: center; gap: 10px; }
.order-amount { font-size: 15px; font-weight: 600; color: #18a058; }

/* Mobile Responsive */
@media (max-width: 767px) {
  .dashboard { padding: 0; }
  .welcome-banner { padding: 20px 16px; border-radius: 12px; }
  .welcome-title { font-size: 20px; }
  .welcome-sub { font-size: 13px; }
  .welcome-decoration { width: 100px; height: 100px; right: -10px; top: -10px; }

  .section-card { border-radius: 10px; }

  .balance-value { font-size: 24px; }
  .balance-content { flex-direction: column; align-items: flex-start; gap: 12px; }
  .balance-content .n-button { width: 100%; }

  .checkin-days-num { font-size: 24px; }
  .checkin-content { flex-direction: column; align-items: flex-start; gap: 12px; }
  .checkin-content .n-button { width: 100%; }

  .sub-value-row { flex-direction: column; align-items: stretch; }
  .sub-value-row .n-input { max-width: 100% !important; }

  .subscription-stats { grid-template-columns: repeat(3, 1fr); gap: 12px; }

  .quick-sub-item { flex-direction: column; align-items: flex-start; gap: 10px; padding: 12px; }
  .quick-sub-actions { width: 100%; flex-wrap: wrap; }
  .quick-sub-actions .n-button { flex: 1; min-width: 70px; }

  .client-grid { grid-template-columns: repeat(2, 1fr); gap: 8px; }
  .client-card { padding: 10px 12px; }
  .client-card-name { font-size: 13px; }

  .quick-action { padding: 14px 8px; }
  .quick-action span { font-size: 12px; }

  .announcement-item { font-size: 13px; }
  .announcement-time { display: none; }

  .order-item { padding: 8px 10px; }
  .order-name { font-size: 13px; }
  .order-amount { font-size: 14px; }
}
</style>