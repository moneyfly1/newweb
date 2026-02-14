<template>
  <div class="admin-dashboard">
    <n-space vertical :size="24">
      <!-- Stats Cards -->
      <n-grid :cols="2" :x-gap="16" :y-gap="16" responsive="screen" :item-responsive="true">
        <n-grid-item span="2 s:3 l:1">
          <n-card class="stat-card stat-card-blue" :bordered="false">
            <div class="stat-content">
              <div class="stat-icon">
                <n-icon :size="32">
                  <PeopleOutline />
                </n-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">总用户数</div>
                <div class="stat-value">{{ stats.total_users || 0 }}</div>
              </div>
            </div>
          </n-card>
        </n-grid-item>

        <n-grid-item span="2 s:3 l:1">
          <n-card class="stat-card stat-card-green" :bordered="false">
            <div class="stat-content">
              <div class="stat-icon">
                <n-icon :size="32">
                  <CheckmarkCircleOutline />
                </n-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">活跃订阅</div>
                <div class="stat-value">{{ stats.active_subscriptions || 0 }}</div>
              </div>
            </div>
          </n-card>
        </n-grid-item>

        <n-grid-item span="2 s:3 l:1">
          <n-card class="stat-card stat-card-orange" :bordered="false">
            <div class="stat-content">
              <div class="stat-icon">
                <n-icon :size="32">
                  <TrendingUpOutline />
                </n-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">今日收入</div>
                <div class="stat-value">¥{{ stats.today_revenue || 0 }}</div>
              </div>
            </div>
          </n-card>
        </n-grid-item>

        <n-grid-item span="2 s:3 l:1">
          <n-card class="stat-card stat-card-purple" :bordered="false">
            <div class="stat-content">
              <div class="stat-icon">
                <n-icon :size="32">
                  <WalletOutline />
                </n-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">本月收入</div>
                <div class="stat-value">¥{{ stats.month_revenue || 0 }}</div>
              </div>
            </div>
          </n-card>
        </n-grid-item>

        <n-grid-item span="2 s:3 l:1">
          <n-card class="stat-card stat-card-red" :bordered="false">
            <div class="stat-content">
              <div class="stat-icon">
                <n-icon :size="32">
                  <CartOutline />
                </n-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">待处理订单</div>
                <div class="stat-value">{{ stats.pending_orders || 0 }}</div>
              </div>
            </div>
          </n-card>
        </n-grid-item>

        <n-grid-item span="2 s:3 l:1">
          <n-card class="stat-card stat-card-cyan" :bordered="false">
            <div class="stat-content">
              <div class="stat-icon">
                <n-icon :size="32">
                  <ChatbubbleEllipsesOutline />
                </n-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">待处理工单</div>
                <div class="stat-value">{{ stats.pending_tickets || 0 }}</div>
              </div>
            </div>
          </n-card>
        </n-grid-item>
      </n-grid>

      <!-- Charts Row -->
      <n-grid :cols="1" :x-gap="16" :y-gap="16" responsive="screen">
        <n-grid-item span="1 m:2">
          <n-card title="收入趋势" :bordered="false" class="chart-card">
            <div class="chart-placeholder">
              <n-icon :size="48" :depth="3">
                <BarChartOutline />
              </n-icon>
              <div class="placeholder-text">图表加载中...</div>
            </div>
          </n-card>
        </n-grid-item>

        <n-grid-item span="1 m:2">
          <n-card title="用户增长" :bordered="false" class="chart-card">
            <div class="chart-placeholder">
              <n-icon :size="48" :depth="3">
                <StatsChartOutline />
              </n-icon>
              <div class="placeholder-text">图表加载中...</div>
            </div>
          </n-card>
        </n-grid-item>
      </n-grid>

      <!-- Data Tables Row -->
      <n-grid :cols="1" :x-gap="16" :y-gap="16" responsive="screen">
        <n-grid-item span="1 m:2">
          <n-card title="最近订单" :bordered="false" class="data-card">
            <n-table :single-line="false" size="small">
              <thead>
                <tr>
                  <th>订单号</th>
                  <th>金额</th>
                  <th>状态</th>
                  <th>时间</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="!recentOrders.length">
                  <td colspan="4" style="text-align: center; color: #999;">暂无数据</td>
                </tr>
                <tr v-for="order in recentOrders" :key="order.order_no">
                  <td>{{ order.order_no }}</td>
                  <td>¥{{ order.amount }}</td>
                  <td>
                    <n-tag
                      :type="getOrderStatusType(order.status)"
                      size="small"
                      :bordered="false"
                    >
                      {{ getOrderStatusText(order.status) }}
                    </n-tag>
                  </td>
                  <td>{{ formatTime(order.created_at) }}</td>
                </tr>
              </tbody>
            </n-table>
          </n-card>
        </n-grid-item>

        <n-grid-item span="1 m:2">
          <n-card title="待处理工单" :bordered="false" class="data-card">
            <n-list v-if="pendingTickets.length" hoverable clickable>
              <n-list-item v-for="ticket in pendingTickets" :key="ticket.ticket_no">
                <template #prefix>
                  <n-icon :size="20">
                    <DocumentTextOutline />
                  </n-icon>
                </template>
                <n-thing>
                  <template #header>
                    <div class="ticket-header">
                      <span>{{ ticket.title }}</span>
                      <n-tag
                        :type="getPriorityType(ticket.priority)"
                        size="small"
                        :bordered="false"
                      >
                        {{ getPriorityText(ticket.priority) }}
                      </n-tag>
                    </div>
                  </template>
                  <template #description>
                    <div class="ticket-meta">
                      <span>工单号: {{ ticket.ticket_no }}</span>
                      <span>{{ formatTime(ticket.created_at) }}</span>
                    </div>
                  </template>
                </n-thing>
              </n-list-item>
            </n-list>
            <n-empty v-else description="暂无待处理工单" size="small" style="padding: 40px 0;" />
          </n-card>
        </n-grid-item>
      </n-grid>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import {
  PeopleOutline,
  CheckmarkCircleOutline,
  TrendingUpOutline,
  WalletOutline,
  CartOutline,
  ChatbubbleEllipsesOutline,
  BarChartOutline,
  StatsChartOutline,
  DocumentTextOutline
} from '@vicons/ionicons5'
import { getAdminDashboard } from '@/api/admin'

const message = useMessage()

const stats = ref({
  total_users: 0,
  active_subscriptions: 0,
  today_revenue: 0,
  month_revenue: 0,
  pending_orders: 0,
  pending_tickets: 0
})

const recentOrders = ref<any[]>([])
const pendingTickets = ref<any[]>([])

const loadDashboard = async () => {
  try {
    const res = await getAdminDashboard()
    const data = res.data || res
    stats.value = {
      total_users: data.total_users || 0,
      active_subscriptions: data.active_subscriptions || 0,
      today_revenue: data.today_revenue || 0,
      month_revenue: data.month_revenue || 0,
      pending_orders: data.pending_orders || 0,
      pending_tickets: data.pending_tickets || 0
    }
    recentOrders.value = data.recent_orders || []
    pendingTickets.value = data.pending_ticket_list || []
  } catch (error: any) {
    message.error(error.message || '加载仪表盘数据失败')
  }
}

const getOrderStatusType = (status: string) => {
  const map: Record<string, any> = {
    paid: 'success',
    pending: 'warning',
    cancelled: 'error',
    refunded: 'info'
  }
  return map[status] || 'default'
}

const getOrderStatusText = (status: string) => {
  const map: Record<string, string> = {
    paid: '已支付',
    pending: '待支付',
    cancelled: '已取消',
    refunded: '已退款'
  }
  return map[status] || status
}

const getPriorityType = (priority: string) => {
  const map: Record<string, any> = {
    high: 'error',
    medium: 'warning',
    low: 'info'
  }
  return map[priority] || 'default'
}

const getPriorityText = (priority: string) => {
  const map: Record<string, string> = {
    high: '高',
    medium: '中',
    low: '低'
  }
  return map[priority] || priority
}

const formatTime = (time: string) => {
  if (!time) return '-'
  const date = new Date(time)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 7) return `${days}天前`
  
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

onMounted(() => {
  loadDashboard()
})
</script>

<style scoped>
.admin-dashboard {
  padding: 24px;
  min-height: 100vh;
  background: #f5f7fa;
}

.stat-card {
  border-radius: 12px;
  transition: all 0.3s ease;
  cursor: pointer;
  overflow: hidden;
  position: relative;
}

.stat-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  opacity: 0.1;
  transition: opacity 0.3s ease;
}

.stat-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

.stat-card:hover::before {
  opacity: 0.15;
}

.stat-card-blue::before {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.stat-card-green::before {
  background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
}

.stat-card-orange::before {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.stat-card-purple::before {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.stat-card-red::before {
  background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);
}

.stat-card-cyan::before {
  background: linear-gradient(135deg, #30cfd0 0%, #330867 100%);
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
  position: relative;
  z-index: 1;
}

.stat-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.9);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.stat-card-blue .stat-icon {
  color: #667eea;
}

.stat-card-green .stat-icon {
  color: #11998e;
}

.stat-card-orange .stat-icon {
  color: #f5576c;
}

.stat-card-purple .stat-icon {
  color: #4facfe;
}

.stat-card-red .stat-icon {
  color: #fa709a;
}

.stat-card-cyan .stat-icon {
  color: #30cfd0;
}

.stat-info {
  flex: 1;
}

.stat-label {
  font-size: 14px;
  color: #666;
  margin-bottom: 4px;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #333;
}

.chart-card,
.data-card {
  border-radius: 12px;
  transition: all 0.3s ease;
}

.chart-card:hover,
.data-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
}

.chart-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 280px;
  color: #999;
}

.placeholder-text {
  margin-top: 16px;
  font-size: 14px;
}

.ticket-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.ticket-meta {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 12px;
  color: #999;
}

:deep(.n-card-header) {
  font-weight: 600;
  font-size: 16px;
}

:deep(.n-table) {
  font-size: 13px;
}

:deep(.n-table th) {
  font-weight: 600;
  background: #fafafa;
}

:deep(.n-list-item) {
  padding: 12px 0;
}

@media (max-width: 768px) {
  .admin-dashboard {
    padding: 16px;
  }

  .stat-value {
    font-size: 24px;
  }

  .stat-icon {
    width: 48px;
    height: 48px;
  }

  .chart-placeholder {
    height: 200px;
  }
}
</style>
