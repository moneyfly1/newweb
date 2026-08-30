<template>
  <div
    class="admin-dashboard"
    @touchstart.passive="pullTouchStart"
    @touchmove.passive="pullTouchMove"
    @touchend.passive="pullTouchEnd"
  >
    <!-- 下拉刷新指示器 -->
    <transition name="fade">
      <div v-if="pullDistance > 0 || pullRefreshing" class="pull-indicator" :style="{ transform: `translate(-50%, ${Math.min(pullDistance, 70) - 40}px)` }">
        <n-spin v-if="pullRefreshing" size="small" />
        <span v-else>{{ pullDistance >= 55 ? '释放刷新' : '下拉刷新' }}</span>
      </div>
    </transition>
    <div class="welcome-section">
      <div class="welcome-text">
        <h2>工作控制台</h2>
        <p>欢迎回来，管理员。以下是站点的最新运行状态和关键指标。</p>
      </div>
      <div class="welcome-action">
        <n-button secondary type="primary" @click="loadDashboard">
          <template #icon><n-icon><refresh-outline /></n-icon></template>
          刷新数据
        </n-button>
      </div>
    </div>

    <n-space vertical :size="24">
      <n-grid :cols="appStore.isMobile ? 2 : 4" :x-gap="16" :y-gap="16">
        <n-grid-item>
          <div class="metric-card metric-primary">
            <div class="metric-label">总用户</div>
            <div class="metric-value">{{ stats.total_users || 0 }}</div>
            <div class="metric-sub">今日新增: {{ stats.new_users_today || 0 }}</div>
            <div class="metric-icon"><n-icon :size="48"><people-outline /></n-icon></div>
          </div>
        </n-grid-item>
        <n-grid-item>
          <div class="metric-card metric-success">
            <div class="metric-label">活跃订阅</div>
            <div class="metric-value">{{ stats.active_subscriptions || 0 }}</div>
            <div class="metric-sub">付费率: {{ conversionRate }}%</div>
            <div class="metric-icon"><n-icon :size="48"><checkmark-circle-outline /></n-icon></div>
          </div>
        </n-grid-item>
        <n-grid-item>
          <div class="metric-card metric-warning">
            <div class="metric-label">今日营收</div>
            <div class="metric-value">¥{{ stats.today_revenue || 0 }}</div>
            <div class="metric-sub">待支付: {{ stats.pending_orders || 0 }}</div>
            <div class="metric-icon"><n-icon :size="48"><trending-up-outline /></n-icon></div>
          </div>
        </n-grid-item>
        <n-grid-item>
          <div class="metric-card metric-info">
            <div class="metric-label">月度营收</div>
            <div class="metric-value">¥{{ stats.month_revenue || 0 }}</div>
            <div class="metric-sub">待处理工单: {{ stats.pending_tickets || 0 }}</div>
            <div class="metric-icon"><n-icon :size="48"><wallet-outline /></n-icon></div>
          </div>
        </n-grid-item>
      </n-grid>

      <n-grid :cols="appStore.isMobile ? 1 : 3" :x-gap="16" :y-gap="16">
        <n-grid-item span="2">
          <n-card title="收入趋势（近30天）" :bordered="false" class="glass-card shadow-sm">
            <div class="chart-shell">
              <revenue-bar-chart :data="revenueTrend" />
            </div>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card title="待办任务" :bordered="false" class="glass-card shadow-sm">
            <n-list hoverable clickable>
              <n-list-item @click="$router.push('/admin/orders?status=pending')">
                <template #prefix><n-icon :size="20" color="var(--warning-color)"><cart-outline /></n-icon></template>
                <n-thing title="待支付订单" :description="`${stats.pending_orders || 0} 个订单正在等待用户支付`" />
              </n-list-item>
              <n-list-item @click="$router.push('/admin/tickets')">
                <template #prefix><n-icon :size="20" color="var(--success-color)"><chatbubble-ellipses-outline /></n-icon></template>
                <n-thing title="待处理工单" :description="`${stats.pending_tickets || 0} 个工单需要管理员回复`" />
              </n-list-item>
              <n-list-item @click="$router.push('/admin/abnormal-users')">
                <template #prefix><n-icon :size="20" color="var(--danger-color)"><alert-circle-outline /></n-icon></template>
                <n-thing title="异常用户提醒" description="有用户存在频繁重置订阅的行为" />
              </n-list-item>
            </n-list>
          </n-card>
        </n-grid-item>
      </n-grid>

      <n-grid :cols="appStore.isMobile ? 1 : 2" :x-gap="16" :y-gap="16">
        <n-grid-item>
          <n-card title="新注册用户" :bordered="false" class="glass-card shadow-sm">
            <div v-if="recentUsers.length" class="activity-list">
              <button
                v-for="user in recentUsers"
                :key="user.id"
                type="button"
                class="activity-item"
                @click="goToUserSubscription(user)"
              >
                <div class="activity-main">
                  <div class="activity-title">{{ user.email || user.username || `用户 #${user.id}` }}</div>
                  <div class="activity-meta">账号：{{ user.username || '-' }}</div>
                </div>
                <div class="activity-side">
                  <div class="activity-time">{{ formatFullTime(user.created_at) }}</div>
                  <div class="activity-relative">{{ formatRelativeTime(user.created_at) }}</div>
                </div>
              </button>
            </div>
            <n-empty v-else description="暂无新注册用户" size="small" />
            <template #footer>
              <n-button quaternary block @click="$router.push('/admin/subscriptions')">查看订阅管理</n-button>
            </template>
          </n-card>
        </n-grid-item>

        <n-grid-item>
          <n-card title="最近订单" :bordered="false" class="glass-card shadow-sm">
            <div v-if="recentOrders.length" class="activity-list">
              <button
                v-for="order in recentOrders"
                :key="order.id"
                type="button"
                class="activity-item"
                @click="goToOrder(order)"
              >
                <div class="activity-main">
                  <div class="activity-title">{{ order.user_email || `用户 #${order.user_id}` }}</div>
                  <div class="activity-meta">
                    <span class="amount">{{ formatCurrency(order.final_amount || order.amount || 0) }}</span>
                    <n-tag :type="getOrderStatusType(order.status)" size="small" round :bordered="false">
                      {{ getOrderStatusText(order.status) }}
                    </n-tag>
                  </div>
                </div>
                <div class="activity-side">
                  <div class="activity-time">{{ formatFullTime(order.created_at) }}</div>
                  <div class="activity-relative">{{ formatRelativeTime(order.created_at) }}</div>
                </div>
              </button>
            </div>
            <n-empty v-else description="暂无订单" size="small" />
            <template #footer>
              <n-button quaternary block @click="$router.push('/admin/orders')">查看全部订单</n-button>
            </template>
          </n-card>
        </n-grid-item>
      </n-grid>

      <!-- 系统监控 + 签到统计（后端已实现、前端接入） -->
      <n-grid :cols="appStore.isMobile ? 1 : 2" :x-gap="16" :y-gap="16">
        <n-grid-item>
          <n-card title="系统监控" :bordered="false" class="glass-card shadow-sm">
            <n-grid :cols="appStore.isMobile ? 2 : 3" :x-gap="12" :y-gap="12">
              <n-grid-item>
                <div class="monitor-item">
                  <div class="monitor-value">{{ monitoring.user_count || 0 }}</div>
                  <div class="monitor-label">用户总数</div>
                </div>
              </n-grid-item>
              <n-grid-item>
                <div class="monitor-item">
                  <div class="monitor-value">{{ monitoring.node_count || 0 }}</div>
                  <div class="monitor-label">节点总数</div>
                </div>
              </n-grid-item>
              <n-grid-item>
                <div class="monitor-item">
                  <div class="monitor-value">{{ monitoring.active_subscriptions || 0 }}</div>
                  <div class="monitor-label">活跃订阅</div>
                </div>
              </n-grid-item>
              <n-grid-item>
                <div class="monitor-item">
                  <div class="monitor-value" :style="{ color: 'var(--warning-color)' }">{{ monitoring.pending_orders || 0 }}</div>
                  <div class="monitor-label">待支付订单</div>
                </div>
              </n-grid-item>
              <n-grid-item>
                <div class="monitor-item">
                  <div class="monitor-value" :style="{ color: 'var(--warning-color)' }">{{ monitoring.pending_tickets || 0 }}</div>
                  <div class="monitor-label">待处理工单</div>
                </div>
              </n-grid-item>
              <n-grid-item>
                <div class="monitor-item">
                  <div class="monitor-value" :style="{ color: 'var(--success-color)' }">{{ checkinStats.today_count || 0 }}</div>
                  <div class="monitor-label">今日签到</div>
                </div>
              </n-grid-item>
            </n-grid>
            <template #footer>
              <n-space justify="end" :size="8">
                <n-button quaternary size="tiny" @click="loadMonitoring">刷新</n-button>
              </n-space>
            </template>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card title="签到统计" :bordered="false" class="glass-card shadow-sm">
            <div class="checkin-stats">
              <div class="checkin-stat-main">
                <div class="checkin-today">{{ checkinStats.today_count || 0 }} <span>人今日签到</span></div>
                <div class="checkin-reward">累计发放 {{ formatCurrency(checkinStats.today_total_reward || 0) }}</div>
              </div>
              <n-divider />
              <div class="checkin-sub">
                <span>累计签到 {{ checkinStats.total_count || 0 }} 次</span>
                <span v-if="checkinStats.settings">
                  奖励区间 ¥{{ (checkinStats.settings.min_reward || 0) / 100 }} ~ ¥{{ (checkinStats.settings.max_reward || 0) / 100 }}
                  <n-tag v-if="checkinStats.settings.enabled === false" size="tiny" type="error" :bordered="false" style="margin-left: 6px">已关闭</n-tag>
                </span>
              </div>
            </div>
            <template #footer>
              <n-space justify="end" :size="8">
                <n-button quaternary size="tiny" @click="loadCheckinStats">刷新</n-button>
              </n-space>
            </template>
          </n-card>
        </n-grid-item>
      </n-grid>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent, ref, computed, onMounted } from 'vue'
import { usePullRefresh } from '@/composables/usePullRefresh'
import { useMessage, type TagProps } from 'naive-ui'
import {
  PeopleOutline, CheckmarkCircleOutline, TrendingUpOutline, WalletOutline,
  CartOutline, ChatbubbleEllipsesOutline, RefreshOutline, AlertCircleOutline
} from '@vicons/ionicons5'
import { useRouter } from 'vue-router'
import { getAdminDashboard, getMonitoring, getCheckInStats } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { formatCurrency } from '@/utils/amount'

const router = useRouter()
const appStore = useAppStore()
const message = useMessage()
const RevenueBarChart = defineAsyncComponent(() => import('./components/RevenueBarChart.vue'))

const stats = ref<any>({
  total_users: 0,
  active_subscriptions: 0,
  today_revenue: 0,
  month_revenue: 0,
  pending_orders: 0,
  pending_tickets: 0,
  new_users_today: 0,
})

const recentUsers = ref<any[]>([])
const recentOrders = ref<any[]>([])
const revenueTrend = ref<{ date: string; value: number }[]>([])

const conversionRate = computed(() => {
  if (!stats.value.total_users) return 0
  return ((stats.value.active_subscriptions / stats.value.total_users) * 100).toFixed(1)
})

const loadDashboard = async () => {
  try {
    const res = await getAdminDashboard()
    const data = res.data
    stats.value = data
    recentUsers.value = data.recent_users || []
    recentOrders.value = data.recent_orders || []
    revenueTrend.value = data.revenue_trend || []
  } catch (error: any) {
    message.error('仪表盘加载失败')
  }
}

const getOrderStatusType = (s: string): TagProps['type'] => {
  const typeMap: Record<string, NonNullable<TagProps['type']>> = {
    paid: 'success',
    pending: 'warning',
    cancelled: 'error',
    refunded: 'info',
    completed: 'success'
  }
  return typeMap[s] || 'default'
}

const getOrderStatusText = (s: string) => ({ paid: '已支付', pending: '待支付', cancelled: '已取消', refunded: '已退款', completed: '已完成' }[s] || s)

const formatFullTime = (time: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

const formatRelativeTime = (time: string) => {
  if (!time) return '-'
  const diff = Date.now() - new Date(time).getTime()
  const minute = 60 * 1000
  const hour = 60 * minute
  const day = 24 * hour

  if (diff < hour) {
    const minutes = Math.max(1, Math.floor(diff / minute))
    return `${minutes} 分钟前`
  }
  if (diff < day) {
    return `${Math.floor(diff / hour)} 小时前`
  }
  return `${Math.floor(diff / day)} 天前`
}

const goToUserSubscription = (user: any) => {
  router.push({ path: '/admin/subscriptions', query: { search: user.email || user.username || String(user.id) } })
}

const goToOrder = (order: any) => {
  router.push({ path: '/admin/orders', query: { order_no: order.order_no } })
}

// 下拉刷新（App 原生感）
const { distance: pullDistance, refreshing: pullRefreshing, onTouchStart: pullTouchStart, onTouchMove: pullTouchMove, onTouchEnd: pullTouchEnd } =
  usePullRefresh(loadDashboard)

// 系统监控 + 签到统计
const monitoring = ref<any>({})
const checkinStats = ref<any>({})

const loadMonitoring = async () => {
  try { const res: any = await getMonitoring(); monitoring.value = res.data || {} } catch {}
}
const loadCheckinStats = async () => {
  try { const res: any = await getCheckInStats(); checkinStats.value = res.data || {} } catch {}
}

onMounted(() => {
  loadDashboard()
  loadMonitoring()
  loadCheckinStats()
})
</script>

<style scoped>
.admin-dashboard {
  position: relative;
}

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

.admin-dashboard {
  padding: 24px;
}

.welcome-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.welcome-text h2 { margin: 0; font-size: 24px; font-weight: 700; }
.welcome-text p { margin: 4px 0 0; color: var(--text-color-secondary); }

.metric-card {
  padding: 20px;
  min-height: 132px;
  border-radius: 16px;
  color: white;
  position: relative;
  overflow: hidden;
  box-shadow: 0 4px 12px rgba(0,0,0,0.05);
  transition: transform 0.12s ease;
}


.metric-primary { background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%); }
.metric-success { background: linear-gradient(135deg, #10b981 0%, #059669 100%); }
.metric-warning { background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%); }
.metric-info { background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%); }

.metric-label { font-size: 14px; opacity: 0.9; }
.metric-value { font-size: 28px; font-weight: 700; margin: 4px 0; }
.metric-sub { font-size: 12px; opacity: 0.8; }
.metric-icon { position: absolute; right: -10px; bottom: -10px; opacity: 0.2; transform: rotate(-15deg); }

.glass-card {
  border-radius: 14px;
  background: color-mix(in srgb, var(--bg-color, #fff) 85%, transparent);
  backdrop-filter: blur(10px);
}

.chart-shell {
  min-height: 320px;
  border-radius: 14px;
  background: linear-gradient(180deg, rgba(59, 130, 246, 0.04), rgba(59, 130, 246, 0));
}

.activity-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.activity-item {
  width: 100%;
  border: 0;
  background: var(--bg-page-color);
  border-radius: 12px;
  padding: 14px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  text-align: left;
  transition: background-color 0.2s ease, transform 0.2s ease;
}

.activity-item:hover {
  background: #eef4ff;
  transform: translateY(-1px);
}

.activity-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.activity-title {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
  word-break: break-all;
}

.activity-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  color: #6b7280;
  font-size: 12px;
}

.activity-side {
  flex-shrink: 0;
  text-align: right;
}

.activity-time {
  color: #6b7280;
  font-size: 12px;
}

.activity-relative {
  color: #111827;
  font-size: 12px;
  font-weight: 600;
  margin-top: 4px;
}

.amount { font-weight: 600; color: var(--text-color); }

@media (max-width: 767px) {
  .admin-dashboard {
    padding: 12px;
  }

  .welcome-section {
    display: grid;
    gap: 12px;
    margin-bottom: 16px;
  }

  .welcome-text h2 {
    font-size: 20px;
    line-height: 1.25;
  }

  .welcome-text p {
    font-size: 13px;
    line-height: 1.5;
  }

  .welcome-action,
  .welcome-action .n-button {
    width: 100%;
  }

  .metric-card {
    min-height: 116px;
    padding: 14px;
  }

  .metric-label {
    font-size: 13px;
  }

  .metric-value {
    font-size: 22px;
    line-height: 1.15;
    word-break: break-all;
  }

  .metric-icon {
    right: -14px;
    bottom: -14px;
  }

  .chart-shell {
    min-height: 240px;
  }

  .activity-item {
    align-items: flex-start;
    flex-direction: column;
    border-radius: 8px;
    padding: 12px;
  }

  .activity-side {
    width: 100%;
    display: flex;
    justify-content: space-between;
    gap: 12px;
    text-align: left;
  }

  .activity-relative {
    margin-top: 0;
  }
}

.monitor-item { padding: 8px 4px; text-align: center; }
.monitor-value { font-size: 22px; font-weight: 700; color: var(--text-color); line-height: 1.3; }
.monitor-label { font-size: 12px; color: var(--text-color-secondary, #888); margin-top: 2px; }
.checkin-stats { padding: 4px 0; }
.checkin-stat-main { text-align: center; padding: 8px 0; }
.checkin-today { font-size: 26px; font-weight: 700; color: var(--primary-color); }
.checkin-today span { font-size: 13px; font-weight: 400; color: var(--text-color-secondary); }
.checkin-reward { font-size: 13px; color: var(--text-color-secondary, #666); margin-top: 6px; }
.checkin-sub { display: flex; flex-direction: column; gap: 6px; font-size: 13px; color: var(--text-color-secondary, #666); }

.monitor-item { padding: 8px 4px; text-align: center; }
.monitor-value { font-size: 22px; font-weight: 700; color: var(--text-color); line-height: 1.3; }
.monitor-label { font-size: 12px; color: var(--text-color-secondary, #888); margin-top: 2px; }
.checkin-stats { padding: 4px 0; }
.checkin-stat-main { text-align: center; padding: 8px 0; }
.checkin-today { font-size: 26px; font-weight: 700; color: var(--primary-color); }
.checkin-today span { font-size: 13px; font-weight: 400; color: var(--text-color-secondary); }
.checkin-reward { font-size: 13px; color: var(--text-color-secondary, #666); margin-top: 6px; }
.checkin-sub { display: flex; flex-direction: column; gap: 6px; font-size: 13px; color: var(--text-color-secondary, #666); }
</style>