<template>
  <div class="admin-dashboard">
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
            <div class="metric-sub">付费率: {{ calculateConversion() }}%</div>
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
            <v-chart :option="revenueChartOption" autoresize style="height: 320px;" />
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card title="待办任务" :bordered="false" class="glass-card shadow-sm">
            <n-list hoverable clickable>
              <n-list-item @click="$router.push('/admin/orders?status=pending')">
                <template #prefix><n-icon :size="20" color="#f0a020"><cart-outline /></n-icon></template>
                <n-thing title="待支付订单" :description="`${stats.pending_orders || 0} 个订单正在等待用户支付`" />
              </n-list-item>
              <n-list-item @click="$router.push('/admin/tickets')">
                <template #prefix><n-icon :size="20" color="#18a058"><chatbubble-ellipses-outline /></n-icon></template>
                <n-thing title="待处理工单" :description="`${stats.pending_tickets || 0} 个工单需要管理员回复`" />
              </n-list-item>
              <n-list-item @click="$router.push('/admin/abnormal-users')">
                <template #prefix><n-icon :size="20" color="#d03050"><alert-circle-outline /></n-icon></template>
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
                    <span class="amount">¥{{ (order.final_amount || order.amount || 0).toFixed(2) }}</span>
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
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useMessage, type TagProps } from 'naive-ui'
import {
  PeopleOutline, CheckmarkCircleOutline, TrendingUpOutline, WalletOutline,
  CartOutline, ChatbubbleEllipsesOutline, RefreshOutline, AlertCircleOutline
} from '@vicons/ionicons5'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent, VisualMapComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { useRouter } from 'vue-router'
import { getAdminDashboard } from '@/api/admin'
import { useAppStore } from '@/stores/app'

use([CanvasRenderer, BarChart, GridComponent, TooltipComponent, LegendComponent, VisualMapComponent])

const router = useRouter()
const appStore = useAppStore()
const message = useMessage()

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

const calculateConversion = () => {
  if (!stats.value.total_users) return 0
  return ((stats.value.active_subscriptions / stats.value.total_users) * 100).toFixed(1)
}

const revenueChartOption = computed(() => ({
  tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
  grid: { left: '4%', right: '4%', top: '10%', bottom: '10%', containLabel: true },
  xAxis: { type: 'category', data: revenueTrend.value.map(d => d.date.slice(5)), axisLine: { lineStyle: { color: '#eee' } }, axisLabel: { color: '#999' } },
  yAxis: { type: 'value', splitLine: { lineStyle: { type: 'dashed', color: '#f5f5f5' } }, axisLabel: { color: '#999' } },
  series: [{
    name: '收入',
    type: 'bar',
    data: revenueTrend.value.map(d => d.value),
    itemStyle: {
      color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: '#3b82f6' }, { offset: 1, color: '#60a5fa' }] },
      borderRadius: [6, 6, 0, 0]
    },
    barMaxWidth: 16
  }]
}))

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

onMounted(() => loadDashboard())
</script>

<style scoped>
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
.welcome-text p { margin: 4px 0 0; color: #666; }

.metric-card {
  padding: 20px;
  border-radius: 16px;
  color: white;
  position: relative;
  overflow: hidden;
  box-shadow: 0 4px 12px rgba(0,0,0,0.05);
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
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(10px);
}

.activity-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.activity-item {
  width: 100%;
  border: 0;
  background: #f8fafc;
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

.amount { font-weight: 600; color: #333; }

@media (max-width: 767px) {
  .admin-dashboard { padding: 12px; }
  .welcome-section { flex-direction: column; align-items: flex-start; gap: 16px; }
  .metric-value { font-size: 20px; }
  .activity-item { align-items: flex-start; flex-direction: column; }
  .activity-side { text-align: left; }
}
</style>
