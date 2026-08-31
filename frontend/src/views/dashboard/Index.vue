<template>
  <div
    class="dashboard"
    :class="{ 'dash-ready': loaded }"
    @touchstart.passive="pullTouchStart"
    @touchmove.passive="pullTouchMove"
    @touchend.passive="pullTouchEnd"
  >
    <!-- 下拉刷新指示器（App 原生感） -->
    <transition name="fade">
      <div v-if="pullDistance > 0 || pullRefreshing" class="pull-indicator" :style="{ transform: `translate(-50%, ${Math.min(pullDistance, 70) - 40}px)` }">
        <n-spin v-if="pullRefreshing" size="small" />
        <span v-else>{{ pullDistance >= 55 ? '释放刷新' : '下拉刷新' }}</span>
      </div>
    </transition>
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
          <div class="welcome-stat balance-card dash-fade-up" style="animation-delay: 0ms">
            <div class="stat-icon">
              <n-icon size="24" :component="WalletOutline" />
            </div>
            <div class="stat-info">
              <span class="stat-label">账户余额</span>
              <span class="stat-value">{{ formatCurrency(balanceCount) }}</span>
            </div>
            <n-button type="primary" size="small" @click="$router.push('/recharge')">
              充值
            </n-button>
          </div>
          <div class="welcome-stat checkin-card dash-fade-up" style="animation-delay: 80ms">
            <div class="stat-icon">
              <n-icon size="24" :component="CalendarOutline" />
            </div>
            <div class="stat-info">
              <span class="stat-label">连续签到</span>
              <span class="stat-value">{{ checkinDaysCount }} 天</span>
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
        <div class="card dash-fade-up" style="animation-delay: 160ms">
          <div class="card-header">
            <span class="card-title">订阅信息</span>
            <n-button text type="primary" size="small" @click="$router.push('/subscription')">管理</n-button>
          </div>
          <n-spin :show="subscriptionLoading">
            <div v-if="subscription.token_url || subscription.token_clash_url" class="sub-info">
              <!-- 剩余天数大字 + 到期日 -->
              <div class="sub-days-block">
                <div class="sub-days-text">
                  <span class="sub-days-number">剩余 <span class="sub-days-big">{{ remainingDaysCount }}</span> 天</span>
                  <span class="sub-days-expire">到期：{{ formatDate(subscription.expire_time) }}</span>
                </div>
                <n-tag :type="remainingDaysType" size="small" :bordered="false" class="sub-days-tag">
                  {{ remainingDays > 30 ? '充足' : remainingDays >= 7 ? '即将到期' : '已到期' }}
                </n-tag>
              </div>
              <div class="sub-stats-row">
                <div class="sub-stat"><span class="sub-stat-label">设备</span><span class="sub-stat-val">{{ subscription.current_devices || 0 }}/{{ subscription.device_limit || 0 }}</span></div>
                <div class="sub-stat"><span class="sub-stat-label">状态</span><n-tag :type="subscription.is_active ? 'success' : 'error'" size="small" :bordered="false">{{ subscription.is_active ? '使用中' : '未激活' }}</n-tag></div>
              </div>
              <div class="sub-urls">
                <div class="dash-protocol-exclude">
                  <div class="dash-protocol-exclude-head">
                    <span>协议排除</span>
                    <n-button v-if="excludedProtocols.length" size="tiny" quaternary @click="excludedProtocols = []">清空</n-button>
                  </div>
                  <n-checkbox-group v-model:value="excludedProtocols">
                    <n-space :size="[8, 6]" wrap>
                      <n-checkbox v-for="protocol in protocolExcludeOptions" :key="protocol.value" :value="protocol.value">
                        {{ protocol.label }}
                      </n-checkbox>
                    </n-space>
                  </n-checkbox-group>
                </div>
                <div class="sub-url-row shadowrocket-qr-row" v-if="shadowrocketQrData">
                  <div class="shadowrocket-qr-card">
                    <div class="shadowrocket-qr-header">
                      <div>
                        <div class="shadowrocket-qr-title">Shadowrocket 扫码订阅</div>
                        <div class="shadowrocket-qr-desc">打开 Shadowrocket 扫描即可直接添加订阅</div>
                      </div>
                      <n-button size="tiny" type="primary" @click="oneClickImport('shadowrocket')">一键导入</n-button>
                    </div>
                    <canvas ref="dashQrCanvas" class="shadowrocket-qr-canvas" />
                  </div>
                </div>
                <div class="sub-url-row" v-if="subscription.token_clash_url">
                  <span class="sub-url-label">Clash</span>
                  <n-input :value="showSubUrls ? clashSubscriptionUrl : maskUrl(clashSubscriptionUrl)" readonly size="tiny" style="flex:1" />
                  <n-button size="tiny" @click="showSubUrls = !showSubUrls"><template #icon><n-icon :component="showSubUrls ? EyeOffOutline : EyeOutline" /></template></n-button>
                  <n-button size="tiny" @click="copyText(clashSubscriptionUrl, 'Clash')"><template #icon><n-icon :component="CopyOutline" /></template></n-button>
                </div>
                <div class="sub-url-row" v-if="subscription.token_url">
                  <span class="sub-url-label">通用</span>
                  <n-input :value="showSubUrls ? universalSubscriptionUrl : maskUrl(universalSubscriptionUrl)" readonly size="tiny" style="flex:1" />
                  <n-button size="tiny" @click="showSubUrls = !showSubUrls"><template #icon><n-icon :component="showSubUrls ? EyeOffOutline : EyeOutline" /></template></n-button>
                  <n-button size="tiny" @click="copyText(universalSubscriptionUrl, '通用')"><template #icon><n-icon :component="CopyOutline" /></template></n-button>
                </div>
                <n-collapse v-if="moreSubscriptionUrls.length" class="dash-more-subscriptions">
                  <n-collapse-item title="更多订阅地址" name="more-subscriptions">
                    <div v-for="item in moreSubscriptionUrls" :key="item.client" class="sub-url-row">
                      <span class="sub-url-label">{{ item.name }}</span>
                      <n-input :value="showSubUrls ? item.url : maskUrl(item.url)" readonly size="tiny" style="flex:1" />
                      <n-button size="tiny" @click="copyText(item.url, item.name)"><template #icon><n-icon :component="CopyOutline" /></template></n-button>
                    </div>
                  </n-collapse-item>
                </n-collapse>
              </div>
            </div>
            <n-empty v-else description="暂无订阅" size="small">
              <template #extra><n-button size="small" type="primary" @click="$router.push('/shop')">去购买</n-button></template>
            </n-empty>
          </n-spin>
        </div>

        <!-- Quick Subscription -->
        <div class="card dash-fade-up" style="animation-delay: 240ms">
          <div class="card-header"><span class="card-title">快速订阅</span></div>
          <div v-if="quickSubItems.length" class="quick-sub-grid">
            <div
              v-for="item in quickSubItems"
              :key="item.name"
              class="quick-sub-item"
              :class="{ 'is-open': expandedQuickSub === item.name }"
            >
              <button class="qs-main" type="button" @click="toggleQuickSub(item.name)">
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
                <span class="qs-hint">点击选择操作</span>
              </button>
              <div v-if="expandedQuickSub === item.name" class="qs-actions">
                <n-button size="small" @click="copyText(item.url, item.name)">
                  <template #icon><n-icon :component="CopyOutline" /></template>
                  复制订阅
                </n-button>
                <n-button v-if="item.importable" size="small" type="primary" @click="oneClickImport(item.client)">
                  <template #icon><n-icon :component="CloudDownloadOutline" /></template>
                  一键导入
                </n-button>
              </div>
            </div>
          </div>
          <n-empty v-else description="暂无订阅" size="small" />
        </div>

        <!-- Announcements -->
        <div class="card dash-fade-up" style="animation-delay: 320ms">
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

        <!-- 签到记录 -->
        <div class="card dash-fade-up" style="animation-delay: 400ms">
          <div class="card-header">
            <span class="card-title">签到记录</span>
            <n-button v-if="checkinHistory.length" text type="primary" size="small" @click="loadCheckinHistory">刷新</n-button>
          </div>
          <div v-if="checkinHistory.length" class="checkin-history-list">
            <div v-for="h in checkinHistory" :key="h.id" class="checkin-history-item">
              <div class="checkin-history-left">
                <n-icon :size="14" :component="CalendarOutline" />
                <span>{{ formatDate(h.created_at) }}</span>
              </div>
              <span class="checkin-history-amount">+{{ formatAmount(h.amount) }} 元</span>
            </div>
          </div>
          <n-empty v-else description="暂无签到记录" size="small" />
        </div>
      </div>
      <!-- Right Column -->
      <div class="right-col">
        <!-- Quick Actions -->
        <div class="card dash-fade-up" style="animation-delay: 480ms">
          <div class="card-header"><span class="card-title">快捷操作</span></div>
          <div class="quick-actions-grid">
            <div class="quick-action" @click="$router.push('/shop')"><n-icon size="18" :component="CartOutline" color="var(--primary-color)" /><span>购买套餐</span></div>
            <div class="quick-action" @click="$router.push('/subscription')"><n-icon size="18" :component="LinkOutline" color="var(--primary-color)" /><span>获取订阅</span></div>
            <div class="quick-action" @click="$router.push('/tickets')"><n-icon size="18" :component="ChatbubblesOutline" color="var(--primary-color)" /><span>提交工单</span></div>
            <div class="quick-action" @click="$router.push('/invite')"><n-icon size="18" :component="PeopleOutline" color="var(--primary-color)" /><span>邀请好友</span></div>
          </div>
        </div>

        <!-- Recent Orders -->
        <div class="card dash-fade-up" style="animation-delay: 560ms">
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
                  <span class="order-amount">{{ formatCurrency(o.final_amount ?? o.amount) }}</span>
                  <n-tag :type="orderStatusType(o.status)" size="small" :bordered="false">{{ orderStatusText(o.status) }}</n-tag>
                </div>
              </div>
            </div>
            <n-empty v-else description="暂无订单" size="small" />
          </n-spin>
        </div>

        <!-- Client Downloads -->
        <div class="card dash-fade-up" v-if="hasAnyClientUrl" style="animation-delay: 640ms">
          <div class="card-header">
            <span class="card-title">软件下载</span>
            <n-tag v-if="currentClientTabLabel" size="small" type="info" :bordered="false">
              {{ currentClientTabLabel }}
            </n-tag>
          </div>
          <n-tabs v-model:value="activeClientTab" type="segment" size="small" animated>
            <n-tab-pane v-for="tab in clientTabs" :key="tab.name" :name="tab.name" :tab="tab.label">
              <div class="client-grid">
                <button v-for="c in tab.clients" :key="c.key" class="client-card" type="button" @click="handleClientClick(c)">
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
                  <n-spin v-if="downloadingKey === c.key" size="small" />
                  <n-icon v-else :component="DownloadOutline" size="14" color="#999" />
                </button>
              </div>
            </n-tab-pane>
          </n-tabs>
        </div>
      </div>
    </div>

    <!-- QR Code Modal -->
  </div>
</template>
<script setup lang="ts">
import { ref, computed, onMounted, onActivated, onUnmounted, nextTick, watch } from 'vue'
import { useMessage } from 'naive-ui'
import {
  WalletOutline, RibbonOutline, CartOutline, LinkOutline,
  ChatbubblesOutline, PeopleOutline, CopyOutline, CloudDownloadOutline,
  DownloadOutline, CalendarOutline, EyeOutline, EyeOffOutline,
} from '@vicons/ionicons5'
import { getDashboardInfo, checkIn, getCheckInStatus, getCheckInHistory } from '@/api/user'
import { listPublicAnnouncements, getPublicConfig } from '@/api/common'
import { getClientDownloadUrl, resolvePanDownloadUrl } from '@/utils/githubDownload'
import { listOrders } from '@/api/order'
import { getSubscription } from '@/api/subscription'
import { copyToClipboard as clipboardCopy } from '@/utils/clipboard'
import { formatCurrency, formatAmount } from '@/utils/amount'
import { formatDate } from '@/utils/date'
import { usePullRefresh } from '@/composables/usePullRefresh'
import { useCountUp } from '@/composables/useCountUp'

const message = useMessage()

const info = ref<any>({})
const subscription = ref<any>({})
const announcements = ref<any[]>([])
const recentOrders = ref<any[]>([])
const announcementsLoading = ref(false)
const ordersLoading = ref(false)
const subscriptionLoading = ref(false)
const loaded = ref(false)
const dashQrCanvas = ref<HTMLCanvasElement | null>(null)
const showSubUrls = ref(false)
const expandedQuickSub = ref('')
const excludedProtocols = ref<string[]>([])
const protocolExcludeOptions = [
  { label: 'VMess', value: 'vmess' },
  { label: 'VLESS', value: 'vless' },
  { label: 'Trojan', value: 'trojan' },
  { label: 'SS', value: 'ss' },
  { label: 'SSR', value: 'ssr' },
  { label: 'Hysteria', value: 'hysteria' },
  { label: 'Hysteria2', value: 'hysteria2' },
  { label: 'TUIC', value: 'tuic' },
  { label: 'AnyTLS', value: 'anytls' },
  { label: 'SOCKS', value: 'socks' },
  { label: 'SOCKS5', value: 'socks5' },
  { label: 'HTTP', value: 'http' },
  { label: 'WireGuard', value: 'wireguard' },
]

function maskUrl(url: string) {
  if (!url || url.length < 20) return '••••••••'
  return url.substring(0, 20) + '••••••••' + url.substring(url.length - 6)
}
const clientConfig = ref<Record<string, string>>({})
const activeClientTab = ref('windows')
const checkinStatus = ref<any>({})
const checkinHistory = ref<any[]>([])
const loadCheckinHistory = async () => {
  try {
    const res: any = await getCheckInHistory({ page: 1, page_size: 5 })
    checkinHistory.value = res.data?.items || []
  } catch {}
}
const checkinLoading = ref(false)

async function handleCheckIn() {
  checkinLoading.value = true
  try {
    const res: any = await checkIn()
    const amt = Number(res.data.amount)
    const amtStr = formatAmount(amt)
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
    { key: 'client_v2rayn_url', name: 'V2rayN', clientKey: 'v2rayN', icon: '🟢', iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/V2ray.png' },
    { key: 'client_clashparty_windows_url', name: 'Clash Party', clientKey: 'clash-party', icon: '🟣', iconUrl: 'https://fastly.jsdelivr.net/gh/mihomo-party-org/clash-party@smart_core/images/icon-black.png' },
    { key: 'client_hiddify_windows_url', name: 'Hiddify', clientKey: 'hiddify-app', icon: '🟠', iconUrl: 'https://raw.githubusercontent.com/hiddify/hiddify-app/main/assets/images/logo.svg' },
    { key: 'client_flclash_windows_url', name: 'FlClash', clientKey: 'FlClash', icon: '⚡', iconUrl: 'https://fastly.jsdelivr.net/gh/chen08209/FlClash@main/assets/images/icon.png' },
  ],
  android: [
    { key: 'client_clash_android_url', name: 'Clash Meta', clientKey: 'clash-meta', icon: '🔵', iconUrl: 'https://fastly.jsdelivr.net/gh/walkxcode/dashboard-icons@main/png/clash.png' },
    { key: 'client_v2rayng_url', name: 'V2rayNG', clientKey: 'v2rayNG', icon: '🟢', iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/V2ray.png' },
    { key: 'client_hiddify_android_url', name: 'Hiddify', clientKey: 'hiddify-app', icon: '🟠', iconUrl: 'https://raw.githubusercontent.com/hiddify/hiddify-app/main/assets/images/logo.svg' },
  ],
  macos: [
    { key: 'client_flclash_macos_url', name: 'FlClash', clientKey: 'FlClash', icon: '⚡', iconUrl: 'https://fastly.jsdelivr.net/gh/chen08209/FlClash@main/assets/images/icon.png' },
    { key: 'client_clashparty_macos_url', name: 'Clash Party', clientKey: 'clash-party', icon: '🟣', iconUrl: 'https://fastly.jsdelivr.net/gh/mihomo-party-org/clash-party@smart_core/images/icon-black.png' },
  ],
  ios: [
    { key: 'client_shadowrocket_url', name: 'Shadowrocket', icon: '🚀', iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/shadowrocket.png' },
    { key: 'client_stash_url', name: 'Stash', icon: '🟡', iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/stash.png' },
  ],
  linux: [
    { key: 'client_clash_linux_url', name: 'Clash', icon: '🐧', iconUrl: 'https://fastly.jsdelivr.net/gh/walkxcode/dashboard-icons@main/png/clash.png' },
    { key: 'client_singbox_url', name: 'Sing-box', icon: '📦', iconUrl: 'https://raw.githubusercontent.com/SagerNet/sing-box/testing/docs/assets/icon.svg' },
  ],
}

// 显示规则：配置了 URL 的客户端，或配置了 clientKey（GitHub 自动解析）的客户端都显示；
// URL 为 pan:// 标记或为空时点击自动获取 GitHub 最新版直链。
const filterClients = (list: typeof allClients.windows) =>
  list.filter(c => clientConfig.value[c.key] || c.clientKey).map(c => ({
    ...c,
    url: clientConfig.value[c.key] || '',
    auto: !clientConfig.value[c.key] || String(clientConfig.value[c.key]).startsWith('pan://'),
  }))

const windowsClients = computed(() => filterClients(allClients.windows))
const androidClients = computed(() => filterClients(allClients.android))
const macClients = computed(() => filterClients(allClients.macos))
const iosClients = computed(() => filterClients(allClients.ios))
const linuxClients = computed(() => filterClients(allClients.linux))
const hasAnyClientUrl = computed(() =>
  windowsClients.value.length || androidClients.value.length || macClients.value.length || iosClients.value.length || linuxClients.value.length
)

const detectedPlatform = computed(() => {
  if (typeof navigator === 'undefined') return 'windows'
  const ua = navigator.userAgent.toLowerCase()
  const platform = navigator.platform?.toLowerCase() || ''
  if (/iphone|ipad|ipod/.test(ua)) return 'ios'
  if (/android/.test(ua)) return 'android'
  if (platform.includes('mac')) return 'macos'
  if (platform.includes('linux')) return 'linux'
  return 'windows'
})

const clientTabSource = computed(() => [
  { name: 'windows', label: 'Windows', clients: windowsClients.value },
  { name: 'android', label: 'Android', clients: androidClients.value },
  { name: 'macos', label: 'macOS', clients: macClients.value },
  { name: 'ios', label: 'iOS', clients: iosClients.value },
  { name: 'linux', label: 'Linux', clients: linuxClients.value },
].filter(t => t.clients.length))

const preferredClientTab = computed(() => {
  if (clientTabSource.value.some(t => t.name === detectedPlatform.value)) return detectedPlatform.value
  return clientTabSource.value[0]?.name || 'windows'
})

const clientTabs = computed(() => {
  const preferred = preferredClientTab.value
  return [...clientTabSource.value].sort((a, b) => {
    if (a.name === preferred) return -1
    if (b.name === preferred) return 1
    return 0
  })
})

const currentClientTabLabel = computed(() => {
  const tab = clientTabSource.value.find(t => t.name === activeClientTab.value)
  return tab ? `当前系统：${tab.label}` : ''
})

watch(preferredClientTab, (tab) => {
  activeClientTab.value = tab
}, { immediate: true })

function buildTypedSubscriptionUrl(base: string, type: string) {
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

function withExcludedProtocols(rawUrl: string) {
  if (!rawUrl) return ''
  const exclude = excludedProtocols.value.join(',')
  try {
    const url = new URL(rawUrl, window.location.origin)
    if (exclude) url.searchParams.set('exclude', exclude)
    else url.searchParams.delete('exclude')
    return url.toString()
  } catch {
    if (!exclude) return rawUrl
    const separator = rawUrl.includes('?') ? '&' : '?'
    return `${rawUrl}${separator}exclude=${encodeURIComponent(exclude)}`
  }
}

function getSubscriptionUrl(key: string, type: string) {
  const s = subscription.value || {}
  const rawUrl = s[key] || buildTypedSubscriptionUrl(s.token_url || s.token_clash_url || '', type)
  return withExcludedProtocols(rawUrl)
}

const universalSubscriptionUrl = computed(() => getSubscriptionUrl('token_url', ''))
const clashSubscriptionUrl = computed(() => getSubscriptionUrl('token_clash_url', 'clash'))

const quickSubItems = computed(() => {
  return [
    {
      name: 'Clash / Meta', icon: '⚔️',
      iconUrl: 'https://fastly.jsdelivr.net/gh/walkxcode/dashboard-icons@main/png/clash.png',
      url: getSubscriptionUrl('token_clash_url', 'clash'), client: 'clash', importable: true,
    },
    {
      name: 'Stash', icon: '📦',
      iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/stash.png',
      url: getSubscriptionUrl('token_stash_url', 'stash'), client: 'stash', importable: true,
    },
    {
      name: 'Surge', icon: '🌊',
      iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/surge.png',
      url: getSubscriptionUrl('token_surge_url', 'surge'), client: 'surge', importable: true,
    },
    {
      name: 'Loon', icon: '🎈',
      iconUrl: 'https://fastly.jsdelivr.net/gh/Orz-3/mini@master/Color/loon.png',
      url: getSubscriptionUrl('token_loon_url', 'loon'), client: 'loon', importable: true,
    },
    {
      name: 'QuantumultX', icon: '💠',
      iconUrl: 'https://raw.githubusercontent.com/Koolson/Qure/master/IconSet/Quantumult_X.png',
      url: getSubscriptionUrl('token_quantumultx_url', 'quantumultx'), client: 'quantumultx', importable: true,
    },
    {
      name: 'Shadowrocket', icon: '🔴',
      iconUrl: 'https://raw.githubusercontent.com/Koolson/Qure/master/IconSet/Rocket.png',
      url: getSubscriptionUrl('token_shadowrocket_url', 'shadowrocket'), client: 'shadowrocket', importable: true,
    },
    {
      name: 'SingBox', icon: '📱',
      iconUrl: 'https://raw.githubusercontent.com/SagerNet/sing-box/testing/docs/assets/icon.svg',
      url: getSubscriptionUrl('token_singbox_url', 'singbox'), client: 'singbox', importable: false,
    },
    {
      name: 'V2Ray / Hiddify', icon: '🚀',
      iconUrl: 'https://raw.githubusercontent.com/hiddify/hiddify-app/main/assets/images/logo.svg',
      url: getSubscriptionUrl('token_v2ray_url', 'v2ray'), client: 'v2ray', importable: false,
    },
  ].filter(i => i.url)
})

const moreSubscriptionUrls = computed(() => quickSubItems.value.filter(item => !['clash', 'v2ray'].includes(item.client)))

const iconFailed = ref<Record<string, boolean>>({})
function markIconFailed(key: string) {
  iconFailed.value[key] = true
}
function canShowIcon(key: string, url?: string) {
  return !!url && !iconFailed.value[key]
}

function toggleQuickSub(name: string) {
  expandedQuickSub.value = expandedQuickSub.value === name ? '' : name
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
  return colors[info.value.level_name] || 'var(--primary-color)'
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

// 数字滚动（余额 / 签到天数 / 订阅剩余天数）：数据就绪后 600ms 内从 0 滚到目标值
const { value: balanceCount } = useCountUp(() => Number(info.value.balance) || 0)
const { value: checkinDaysCount } = useCountUp(() => checkinStatus.value.consecutive_days || 0)
const { value: remainingDaysCount } = useCountUp(() => remainingDays.value)

const shadowrocketQrData = computed(() => {
  const url = getSubscriptionUrl('token_shadowrocket_url', 'shadowrocket')
  if (!url) return ''
  return 'sub://' + btoa(url)
})

watch(quickSubItems, (items) => {
  if (!items.some(item => item.name === expandedQuickSub.value)) {
    expandedQuickSub.value = ''
  }
})

watch(shadowrocketQrData, async (value) => {
  await nextTick()
  if (!dashQrCanvas.value) return
  const canvas = dashQrCanvas.value
  const ctx = canvas.getContext('2d')
  ctx?.clearRect(0, 0, canvas.width, canvas.height)
  if (!value) return
  const QRCode = (await import('qrcode')).default
  await QRCode.toCanvas(canvas, value, { width: 200, margin: 2 })
})

const orderStatusType = (status: string) => {
  const map: Record<string, string> = { pending: 'warning', paid: 'success', cancelled: 'default', expired: 'error', refunded: 'info' }
  return (map[status] || 'default') as any
}
const orderStatusText = (status: string) => {
  const map: Record<string, string> = { pending: '待支付', paid: '已支付', cancelled: '已取消', expired: '已过期', refunded: '已退款' }
  return map[status] || status
}

async function copyText(text: string, label: string) {
  const ok = await clipboardCopy(text)
  ok ? message.success(`${label}已复制到剪贴板`) : message.error('复制失败，请手动复制')
}
function oneClickImport(client: string) {
  const subName = info.value.site_name || '订阅'
  switch (client) {
    case 'clash':
      window.location.href = `clash://install-config?url=${encodeURIComponent(getSubscriptionUrl('token_clash_url', 'clash'))}&name=${encodeURIComponent(subName)}`; break
    case 'stash':
      window.location.href = `stash://install-config?url=${encodeURIComponent(getSubscriptionUrl('token_stash_url', 'stash'))}&name=${encodeURIComponent(subName)}`; break
    case 'surge':
      window.location.href = `surge:///install-config?url=${encodeURIComponent(getSubscriptionUrl('token_surge_url', 'surge'))}`; break
    case 'loon':
      window.location.href = `loon://import/proxy?url=${encodeURIComponent(getSubscriptionUrl('token_loon_url', 'loon'))}`; break
    case 'quantumultx':
      window.location.href = `quantumult-x:///add-resource?remote-resource=${encodeURIComponent(JSON.stringify({ server_remote: [getSubscriptionUrl('token_quantumultx_url', 'quantumultx')] }))}`; break
    case 'shadowrocket':
      window.location.href = `shadowrocket://add/${encodeURIComponent(getSubscriptionUrl('token_shadowrocket_url', 'shadowrocket'))}`; break
    default:
      copyText(universalSubscriptionUrl.value, '订阅地址'); return
  }
  message.info(`正在打开 ${client} 客户端...`)
}

function openUrl(url: string) { if (url) window.open(url, '_blank') }

// 自动客户端点击：动态解析 GitHub 最新版直链（pan:// 或未配置 URL）
const downloadingKey = ref('')
async function handleClientClick(c: any) {
  if (c.auto && c.clientKey) {
    if (downloadingKey.value) return
    downloadingKey.value = c.key
    try {
      const resolved = await getClientDownloadUrl(c.clientKey, clientConfig.value)
      window.open(resolved, '_blank')
    } catch (e: any) {
      message.error(e?.message || '获取下载链接失败，请稍后重试')
    } finally {
      downloadingKey.value = ''
    }
    return
  }
  // 普通 URL：pan:// 标记转后端解析接口
  openUrl(resolvePanDownloadUrl(c.url))
}

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

// 下拉刷新：重新拉取首页数据（App 原生感）
const { distance: pullDistance, refreshing: pullRefreshing, onTouchStart: pullTouchStart, onTouchMove: pullTouchMove, onTouchEnd: pullTouchEnd } =
  usePullRefresh(loadDashboardData)

const handleVisibilityChange = () => {
  if (!document.hidden) {
    loadDashboardData().catch(() => {})
  }
}

async function loadFullDashboardData() {
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
  loadCheckinHistory()
  // 数据就绪：触发卡片依次入场（仅首次置 true，KeepAlive 激活刷新不重播动画）
  if (!loaded.value) loaded.value = true
}

onMounted(() => {
  loadFullDashboardData()
  document.addEventListener('visibilitychange', handleVisibilityChange)
})

// KeepAlive 缓存激活时刷新数据（余额/订阅/订单保持最新）
onActivated(() => {
  loadFullDashboardData()
})

onUnmounted(() => {
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
</script>
<style scoped>
.dashboard { padding: 0; }

/* 下拉刷新指示器（App 原生感） */
.pull-indicator {
  position: fixed;
  top: 0;
  left: 50%;
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 96px;
  height: 34px;
  padding: 0 14px;
  border-radius: 999px;
  background: var(--bg-color, #fff);
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.12);
  font-size: 12px;
  color: var(--text-color-secondary, #666);
  transition: transform 0.15s ease;
}
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

/* Welcome Card */
.welcome-card { background: var(--brand-gradient); border-radius: 12px; padding: 20px 24px; margin-bottom: 12px; color: #fff; }
.welcome-content { display: flex; justify-content: space-between; align-items: center; gap: 24px; }
.welcome-left { flex: 1; }
.welcome-title { font-size: 22px; font-weight: 700; margin: 0 0 8px 0; text-shadow: 0 1px 2px rgba(0,0,0,0.2); color: #fff; }
.user-meta { display: flex; align-items: center; gap: 8px; }
.level-badge { display: inline-flex; align-items: center; gap: 4px; padding: 4px 12px; border-radius: 16px; color: white; font-weight: 600; font-size: 13px; background: rgba(0,0,0,0.18); }
.welcome-stats { display: flex; gap: 12px; }
.welcome-stat { background: rgba(255,255,255,0.16); border-radius: 12px; padding: 14px 16px; display: flex; align-items: center; gap: 12px; backdrop-filter: blur(10px); min-width: 200px; border: 1px solid rgba(255,255,255,0.28); }
.stat-icon { width: 40px; height: 40px; border-radius: 10px; background: rgba(255,255,255,0.22); display: flex; align-items: center; justify-content: center; flex-shrink: 0; color: white; }
.stat-info { display: flex; flex-direction: column; gap: 2px; flex: 1; }
.stat-label { font-size: 12px; opacity: 0.9; color: rgba(255,255,255,0.85); }
.stat-value { font-size: 18px; font-weight: 700; line-height: 1.2; color: #fff; }
.welcome-stat .n-button { flex-shrink: 0; background: rgba(255,255,255,0.95) !important; color: var(--primary-color) !important; border: none !important; font-weight: 600 !important; }
.welcome-stat .n-button:hover { opacity: 0.9; }
.welcome-stat .n-button:disabled { background: rgba(255,255,255,0.5) !important; color: rgba(0,0,0,0.5) !important; }

/* Main Grid */
.main-grid { display: grid; grid-template-columns: 1.25fr 1fr; gap: 12px; margin-top: 12px; }
.left-col, .right-col { display: flex; flex-direction: column; gap: 12px; }

/* Card */
.card { background: var(--bg-color); border-radius: 10px; padding: 14px 16px; box-shadow: 0 1px 3px rgba(0,0,0,0.06); }
.card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.card-title { font-size: 14px; font-weight: 600; color: var(--text-color); }

/* Subscription Info */
.sub-days-block {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
  padding: 12px 14px;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--primary-color-soft) 0%, var(--primary-color-hover) 100%);
  border: 1px solid var(--primary-color-hover);
}
.sub-days-text { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.sub-days-number { font-size: 14px; font-weight: 600; color: var(--text-color); }
.sub-days-big { font-size: 30px; font-weight: 800; line-height: 1.1; color: var(--primary-color); margin: 0 2px; }
.sub-days-expire { font-size: 12px; color: var(--text-color-secondary); }
.sub-days-tag { flex-shrink: 0; }
.sub-stats-row { display: flex; gap: 16px; margin-bottom: 10px; }
.sub-stat { display: flex; align-items: center; gap: 6px; }
.sub-stat-label { font-size: 12px; color: var(--text-color-secondary); }
.sub-stat-val { font-size: 13px; font-weight: 600; }
.sub-urls { display: flex; flex-direction: column; gap: 6px; }
.sub-url-row { display: flex; align-items: center; gap: 6px; }
.dash-protocol-exclude {
  padding: 8px 10px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--primary-color-soft);
}
.dash-protocol-exclude-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-color);
}
.dash-protocol-exclude :deep(.n-checkbox) {
  font-size: 12px;
}
.dash-more-subscriptions {
  margin-top: 2px;
}
.dash-more-subscriptions .sub-url-row {
  margin-bottom: 6px;
}
.dash-more-subscriptions .sub-url-row:last-child {
  margin-bottom: 0;
}
.shadowrocket-qr-row { align-items: stretch; }
.shadowrocket-qr-card {
  width: 100%; padding: 12px; border-radius: 10px;
  background: linear-gradient(135deg, var(--primary-color-soft) 0%, var(--primary-color-hover) 100%);
  border: 1px solid var(--primary-color-hover);
}
.shadowrocket-qr-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
.shadowrocket-qr-title { font-size: 14px; font-weight: 600; color: var(--text-color); }
.shadowrocket-qr-desc { margin-top: 4px; font-size: 12px; color: var(--text-color-secondary); }
.shadowrocket-qr-canvas { display: block; margin: 0 auto; max-width: 200px; border-radius: 8px; background: var(--bg-color); }
.sub-url-label { font-size: 12px; color: var(--text-color-secondary); min-width: 36px; font-weight: 500; }

/* Quick Subscription */
.quick-sub-grid { display: flex; flex-direction: column; gap: 10px; }
.quick-sub-item { border-radius: 10px; background: var(--primary-color-soft); border: 1px solid transparent; transition: all 0.2s; }
.quick-sub-item.is-open { background: var(--primary-color-soft); border-color: #d8e0ff; }
.qs-main { width: 100%; display: flex; align-items: center; gap: 10px; padding: 10px 12px; border: 0; background: transparent; cursor: pointer; text-align: left; }
.qs-main:hover { background: rgba(102,126,234,0.04); }
.qs-icon { width: 28px; height: 28px; display: inline-flex; align-items: center; justify-content: center; font-size: 16px; flex-shrink: 0; }
.qs-name { flex: 1; font-size: 13px; font-weight: 600; color: var(--text-color); }
.qs-hint { font-size: 12px; color: var(--text-color-secondary); flex-shrink: 0; }
.qs-actions { display: flex; gap: 8px; padding: 0 12px 12px 50px; flex-wrap: wrap; }

/* Client Downloads */
.client-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; margin-top: 8px; }
.client-card { display: flex; align-items: center; gap: 8px; width: 100%; min-height: 38px; padding: 8px 10px; border: 1px solid transparent; border-radius: 8px; background: var(--primary-color-soft); color: inherit; cursor: pointer; transition: background 0.2s, border-color 0.2s; }
.client-card:hover { background: var(--primary-color-soft); border-color: #dfe4ee; }
.client-card:focus-visible { outline: 2px solid rgba(102,126,234,0.45); outline-offset: 2px; }
.client-icon { display: inline-flex; align-items: center; justify-content: center; width: 20px; height: 20px; font-size: 16px; flex-shrink: 0; }
.client-name { flex: 1; font-size: 12px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.app-icon { width: 18px; height: 18px; object-fit: contain; border-radius: 4px; }

/* Quick Actions */
.quick-actions-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; }
.quick-action { display: flex; flex-direction: column; align-items: center; gap: 4px; padding: 10px 6px; border-radius: 8px; background: var(--primary-color-soft); cursor: pointer; transition: background 0.2s; }
.quick-action:hover { background: var(--primary-color-soft); }
.quick-action span { font-size: 11px; color: var(--text-color); }

/* Announcements */
.announcement-list { display: flex; flex-direction: column; gap: 6px; }
.announcement-item { display: flex; align-items: center; gap: 8px; font-size: 13px; }
.announcement-title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

/* Orders */
.order-list { display: flex; flex-direction: column; gap: 6px; }
.order-item { display: flex; justify-content: space-between; align-items: center; padding: 6px 8px; border-radius: 6px; background: var(--primary-color-soft); }
.order-left { display: flex; flex-direction: column; gap: 2px; }
.order-name { font-size: 13px; font-weight: 500; }
.order-time { font-size: 11px; color: var(--text-color-secondary); }
.order-right { display: flex; align-items: center; gap: 8px; }
.order-amount { font-size: 14px; font-weight: 600; color: var(--success-color); }

/* Mobile */
@media (max-width: 767px) {
  .dashboard { padding: 12px; }
  .welcome-card { padding: 18px 16px; margin-bottom: 12px; border-radius: 16px; }
  .welcome-content { display: grid; gap: 14px; }
  .welcome-left { min-width: 0; }
  .welcome-title { font-size: 20px; line-height: 1.25; white-space: normal; word-break: break-word; }
  /* 统计卡横向滑动（App 仪表盘风格，scroll-snap 吸附） */
  .welcome-stats {
    display: flex;
    flex-direction: row;
    gap: 10px;
    width: 100%;
    overflow-x: auto;
    scroll-snap-type: x mandatory;
    -webkit-overflow-scrolling: touch;
    padding-bottom: 4px;
    margin: 0 -4px;
    padding-left: 4px;
    padding-right: 4px;
  }
  .welcome-stat {
    flex: 0 0 78%;
    min-width: 0;
    scroll-snap-align: start;
    border-radius: 14px;
  }
  .welcome-stat .n-button {
    white-space: nowrap;
  }
  .main-grid { grid-template-columns: 1fr; }
  .card { border-radius: 14px; }
  .card-header { align-items: center; gap: 8px; padding-bottom: 10px; }
  .client-grid { grid-template-columns: repeat(2, 1fr); }
  .quick-actions-grid { grid-template-columns: repeat(4, 1fr); gap: 8px; }
  .quick-action { padding: 12px 4px; border-radius: 12px; }
  .quick-action span { font-size: 11px; }
  .sub-days-block { align-items: flex-start; }
  .sub-stats-row { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; }
  .sub-stat { min-width: 0; }
  .sub-url-row { display: grid; grid-template-columns: 44px minmax(0, 1fr) 32px 32px; gap: 6px; align-items: center; }
  .sub-url-label { min-width: 0; }
  .shadowrocket-qr-row {
    display: block;
  }
  .dash-more-subscriptions .sub-url-row {
    grid-template-columns: 44px minmax(0, 1fr) 32px;
  }
  .shadowrocket-qr-header { flex-direction: column; align-items: stretch; }
  .shadowrocket-qr-canvas { max-width: 180px; }
  .qs-main { display: grid; grid-template-columns: 28px minmax(0, 1fr); align-items: center; }
  .qs-name { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .qs-hint { grid-column: 2; width: auto; padding-left: 0; }
  .qs-actions { display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); padding: 0 12px 12px 12px; }
  .order-item { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 8px; align-items: center; }
  .order-right { justify-content: flex-end; flex-wrap: wrap; }
}

.checkin-history-list { display: flex; flex-direction: column; gap: 6px; }
.checkin-history-item { display: flex; justify-content: space-between; align-items: center; padding: 6px 8px; border-radius: 8px; background: var(--primary-color-soft, rgba(79,70,229,0.04)); font-size: 13px; }
.checkin-history-left { display: flex; align-items: center; gap: 6px; color: var(--text-color-secondary, #666); }
.checkin-history-amount { color: var(--success-color, #059669); font-weight: 600; }

/* 卡片入场：数据加载完成后依次 fade-up（stagger 80ms，仅一次性） */
.dash-fade-up {
  opacity: 0;
  visibility: hidden;
}
.dash-ready .dash-fade-up {
  visibility: visible;
  animation: dash-fade-up 0.45s cubic-bezier(0.22, 1, 0.36, 1) both;
}
@keyframes dash-fade-up {
  from {
    opacity: 0;
    transform: translateY(14px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
@media (prefers-reduced-motion: reduce) {
  .dash-fade-up {
    animation: none !important;
    opacity: 1 !important;
    visibility: visible !important;
  }
}
</style>