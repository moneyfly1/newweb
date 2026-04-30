<template>
  <div class="stats-container">
    <n-spin :show="loading">
      <n-space vertical :size="20">
        <!-- Filters -->
        <n-card :bordered="false">
          <n-space align="center" :wrap="true" class="filters-toolbar">
            <div class="filter-period">
              <n-radio-group v-model:value="period" @update:value="loadFinancialReport">
                <n-radio-button value="day">日</n-radio-button>
                <n-radio-button value="week">周</n-radio-button>
                <n-radio-button value="month">月</n-radio-button>
              </n-radio-group>
            </div>
            <div class="filter-range">
              <n-date-picker
                v-model:value="dateRange"
                type="daterange"
                clearable
                :shortcuts="dateShortcuts"
                @update:value="onDateRangeChange"
              />
            </div>
            <div class="filter-export">
              <n-button @click="handleExport" :loading="exporting">导出 CSV</n-button>
            </div>
          </n-space>
        </n-card>

        <!-- Summary Cards -->
        <n-card title="财务概览" :bordered="false" class="summary-card">
          <n-grid :cols="appStore.isMobile ? 2 : 6" :x-gap="16" :y-gap="16" class="summary-grid">
            <n-gi>
              <n-statistic label="总收入" :value="summary.total_revenue">
                <template #prefix>¥</template>
              </n-statistic>
            </n-gi>
            <n-gi>
              <n-statistic label="总订单" :value="summary.total_orders" />
            </n-gi>
            <n-gi>
              <n-statistic label="已支付" :value="summary.paid_orders" />
            </n-gi>
            <n-gi>
              <n-statistic label="平均客单价" :value="summary.average_order_amount">
                <template #prefix>¥</template>
              </n-statistic>
            </n-gi>
            <n-gi>
              <n-statistic label="充值总额" :value="summary.total_recharge">
                <template #prefix>¥</template>
              </n-statistic>
            </n-gi>
            <n-gi>
              <n-statistic label="新用户" :value="summary.new_users" />
            </n-gi>
          </n-grid>
        </n-card>
        <!-- Revenue Chart (bar visualization) -->
        <n-card title="收入趋势" :bordered="false">
          <div v-if="revenueChart.length > 0">
            <div v-for="(item, index) in revenueChart" :key="index" class="chart-row">
              <div class="chart-meta">
                <div class="chart-label">{{ item.date }}</div>
                <div class="chart-orders">{{ item.orders }} 单</div>
              </div>
              <div class="chart-bars">
                <div class="bar-group">
                  <div class="bar revenue-bar" :style="{ width: barWidth(item.revenue, maxRevenue) + '%' }">
                    <span v-if="item.revenue > 0" class="bar-text">{{ formatCurrency(item.revenue) }}</span>
                  </div>
                </div>
                <div class="bar-group">
                  <div class="bar recharge-bar" :style="{ width: barWidth(item.recharge, maxRevenue) + '%' }">
                    <span v-if="item.recharge > 0" class="bar-text">{{ formatCurrency(item.recharge) }}</span>
                  </div>
                </div>
              </div>
            </div>
            <n-space class="chart-legend" style="margin-top: 12px" :size="16">
              <span class="legend"><span class="legend-dot" style="background: #18a058"></span>收入</span>
              <span class="legend"><span class="legend-dot" style="background: #2080f0"></span>充值</span>
            </n-space>
          </div>
          <n-empty v-else description="暂无数据" />
        </n-card>

        <n-grid :cols="appStore.isMobile ? 1 : 2" :x-gap="16" :y-gap="16">
          <!-- Payment Method Stats -->
          <n-gi>
            <n-card title="支付方式分布" :bordered="false">
              <div v-if="paymentMethodStats.length > 0">
                <div v-for="(item, index) in paymentMethodStats" :key="index" class="method-item">
                  <div class="method-info">
                    <span class="method-name">{{ item.method || '未知' }}</span>
                    <span class="method-detail">{{ item.count }} 笔 / {{ formatCurrency(item.amount) }}</span>
                  </div>
                  <n-progress
                    type="line"
                    :percentage="Math.round((item.amount / maxPaymentAmount) * 100)"
                    :show-indicator="false"
                    :height="8"
                    :border-radius="4"
                    :color="getColor(index)"
                  />
                </div>
              </div>
              <n-empty v-else description="暂无数据" />
            </n-card>
          </n-gi>
          <!-- Package Stats -->
          <n-gi>
            <n-card title="套餐销售排行" :bordered="false">
              <div v-if="packageStats.length > 0 && appStore.isMobile" class="mobile-stat-list">
                <div v-for="(item, index) in packageStats" :key="index" class="mobile-stat-item">
                  <div class="mobile-stat-main">{{ item.package_name }}</div>
                  <div class="mobile-stat-meta">
                    <span>销量 {{ item.count }}</span>
                    <strong>{{ formatCurrency(item.amount) }}</strong>
                  </div>
                </div>
              </div>
              <n-data-table
                v-else-if="packageStats.length > 0"
                :columns="packageColumns"
                :data="packageStats"
                :bordered="false"
                size="small"
                :pagination="false"
              />
              <n-empty v-else description="暂无数据" />
            </n-card>
          </n-gi>
        </n-grid>

        <!-- Top Users -->
        <n-card title="消费排行 TOP 10" :bordered="false">
          <div v-if="topUsers.length > 0 && appStore.isMobile" class="mobile-stat-list top-user-list">
            <div v-for="(item, index) in topUsers" :key="index" class="mobile-stat-item top-user-item">
              <div class="mobile-stat-main">#{{ index + 1 }} {{ item.username || '未知用户' }}</div>
              <div class="mobile-stat-sub">用户ID：{{ item.user_id }}</div>
              <div class="mobile-stat-meta">
                <span>{{ item.order_count }} 单</span>
                <strong>{{ formatCurrency(item.total_spent) }}</strong>
              </div>
            </div>
          </div>
          <n-data-table
            v-else-if="topUsers.length > 0"
            :columns="topUserColumns"
            :data="topUsers"
            :bordered="false"
            size="small"
            :pagination="false"
          />
          <n-empty v-else description="暂无数据" />
        </n-card>

        <!-- Region Stats (preserved from original) -->
        <n-card title="用户地区分布" :bordered="false">
          <n-spin :show="regionLoading">
            <div v-if="regionStats.length > 0">
              <div class="region-header">
                <span class="region-rank">#</span>
                <span class="region-col">国家</span>
                <span class="region-col">省份</span>
                <span class="region-col">城市</span>
                <span class="region-count">用户数</span>
              </div>
              <div v-for="(item, index) in regionStats" :key="index" class="region-item">
                <div class="region-info">
                  <span class="region-rank">{{ index + 1 }}</span>
                  <span class="region-col">{{ item.country || '-' }}</span>
                  <span class="region-col">{{ item.province || '-' }}</span>
                  <span class="region-col">{{ item.city || '-' }}</span>
                  <span class="region-count">{{ item.count }} 人</span>
                </div>
                <n-progress
                  type="line"
                  :percentage="Math.round((item.count / maxRegionCount) * 100)"
                  :show-indicator="false"
                  :height="8"
                  :border-radius="4"
                  :color="getColor(index)"
                />
              </div>
            </div>
            <n-empty v-else description="暂无地区数据" style="padding: 40px 0" />
          </n-spin>
        </n-card>
      </n-space>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, h, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { getFinancialReport, exportFinancialReport, getRegionStats } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { formatCurrency } from '@/utils/amount'

const appStore = useAppStore()
const message = useMessage()
const loading = ref(false)
const exporting = ref(false)
const regionLoading = ref(false)
const period = ref('month')
const dateRange = ref<[number, number] | null>(null)

const summary = ref({
  total_revenue: 0, total_orders: 0, paid_orders: 0,
  refunded_orders: 0, average_order_amount: 0,
  total_recharge: 0, total_recharge_count: 0,
  new_users: 0, new_subscriptions: 0,
})
const revenueChart = ref<any[]>([])
const paymentMethodStats = ref<any[]>([])
const packageStats = ref<any[]>([])
const topUsers = ref<any[]>([])
const regionStats = ref<Array<{ country: string; province: string; city: string; count: number }>>([])

const dateShortcuts = {
  '最近7天': () => {
    const e = Date.now()
    return [e - 6 * 86400000, e] as [number, number]
  },
  '最近30天': () => {
    const e = Date.now()
    return [e - 29 * 86400000, e] as [number, number]
  },
  '最近90天': () => {
    const e = Date.now()
    return [e - 89 * 86400000, e] as [number, number]
  },
}
const colors = ['#18a058', '#2080f0', '#f0a020', '#d03050', '#8a2be2', '#36ad6a', '#4098fc', '#f2c97d', '#e88080', '#a78bfa']
const getColor = (index: number) => colors[index % colors.length]

const maxRevenue = computed(() => {
  if (revenueChart.value.length === 0) return 1
  return Math.max(...revenueChart.value.map(i => Math.max(i.revenue || 0, i.recharge || 0)), 1)
})
const maxPaymentAmount = computed(() => {
  if (paymentMethodStats.value.length === 0) return 1
  return paymentMethodStats.value[0]?.amount || 1
})
const maxRegionCount = computed(() => {
  if (regionStats.value.length === 0) return 1
  return regionStats.value[0]?.count || 1
})

const barWidth = (val: number, max: number) => max > 0 ? Math.max((val / max) * 100, val > 0 ? 2 : 0) : 0

const packageColumns = [
  { title: '套餐', key: 'package_name' },
  { title: '销量', key: 'count', width: 80 },
  { title: '金额', key: 'amount', width: 120, render: (row: any) => h('span', formatCurrency(row.amount)) },
]

const topUserColumns = [
  { title: '排名', key: 'index', width: 60, render: (_: any, index: number) => h('span', `${index + 1}`) },
  { title: '用户ID', key: 'user_id', width: 80 },
  { title: '用户名', key: 'username' },
  { title: '消费总额', key: 'total_spent', width: 120, render: (row: any) => h('span', formatCurrency(row.total_spent)) },
  { title: '订单数', key: 'order_count', width: 80 },
]
const buildParams = () => {
  const params: any = { period: period.value }
  if (dateRange.value) {
    params.start_date = new Date(dateRange.value[0]).toISOString().slice(0, 10)
    params.end_date = new Date(dateRange.value[1]).toISOString().slice(0, 10)
  }
  return params
}

const onDateRangeChange = () => { loadFinancialReport() }

const loadFinancialReport = async () => {
  loading.value = true
  try {
    const [finRes] = await Promise.all([
      getFinancialReport(buildParams()),
      loadRegionStats(),
    ])
    const d = finRes.data
    if (d) {
      summary.value = d.summary || summary.value
      revenueChart.value = d.revenue_chart || []
      paymentMethodStats.value = d.payment_method_stats || []
      packageStats.value = d.package_stats || []
      topUsers.value = d.top_users || []
    }
  } catch (error: any) {
    message.error(error.message || '加载财务报表失败')
  } finally {
    loading.value = false
  }
}

const loadRegionStats = async () => {
  regionLoading.value = true
  try {
    const res = await getRegionStats()
    regionStats.value = res.data || []
  } catch (error: any) {
    message.error(error.message || '加载地区统计失败')
  } finally {
    regionLoading.value = false
  }
}

const handleExport = async () => {
  exporting.value = true
  try {
    const res = await exportFinancialReport(buildParams())
    const blob = new Blob([res as any], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `financial_report_${new Date().toISOString().slice(0, 10)}.csv`
    a.click()
    URL.revokeObjectURL(url)
    message.success('导出成功')
  } catch (error: any) {
    message.error(error.message || '导出失败')
  } finally {
    exporting.value = false
  }
}

onMounted(() => { loadFinancialReport() })
</script>

<style scoped>
.stats-container { padding: 20px; }
.filters-toolbar { width: 100%; }
.filter-period,
.filter-range,
.filter-export { display: flex; }
.filter-range { flex: 1; min-width: 280px; }
.filter-range :deep(.n-date-picker) { width: 100%; }
.chart-row { display: flex; align-items: center; margin-bottom: 8px; gap: 8px; }
.chart-meta { display: flex; align-items: center; gap: 8px; width: 158px; flex-shrink: 0; }
.chart-label { width: 90px; font-size: 13px; color: #666; text-align: right; flex-shrink: 0; }
.chart-bars { flex: 1; min-width: 0; }
.bar-group { margin-bottom: 2px; }
.bar {
  height: 18px; border-radius: 4px; display: flex; align-items: center;
  padding: 0 6px; min-width: 0; transition: width 0.3s;
}
.revenue-bar { background: #18a058; }
.recharge-bar { background: #2080f0; }
.bar-text { color: #fff; font-size: 11px; white-space: nowrap; overflow: hidden; }
.chart-orders { width: 60px; font-size: 12px; color: #999; text-align: right; flex-shrink: 0; }
.legend { display: flex; align-items: center; font-size: 13px; color: #666; }
.legend-dot { width: 10px; height: 10px; border-radius: 2px; margin-right: 4px; display: inline-block; }
.method-item { margin-bottom: 12px; }
.method-info { display: flex; justify-content: space-between; margin-bottom: 4px; font-size: 14px; }
.method-name { font-weight: 500; }
.method-detail { color: #999; font-size: 13px; }
.region-header {
  display: flex; align-items: center; margin-bottom: 8px; font-size: 13px;
  color: #999; font-weight: 500; padding-bottom: 8px; border-bottom: 1px solid #f0f0f0;
}
.region-item { margin-bottom: 12px; }
.region-info { display: flex; align-items: center; margin-bottom: 4px; font-size: 14px; }
.region-rank {
  width: 24px; height: 24px; border-radius: 50%; background: #f0f0f0;
  display: flex; align-items: center; justify-content: center;
  font-size: 12px; font-weight: 600; margin-right: 8px; color: #666; flex-shrink: 0;
}
.region-col { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.region-count { color: #999; font-size: 13px; width: 70px; text-align: right; flex-shrink: 0; }
@media (max-width: 767px) {
  .stats-container { padding: 8px; }
  .filters-toolbar { flex-direction: column; align-items: stretch !important; gap: 12px; }
  .filter-period,
  .filter-range,
  .filter-export { width: 100%; }
  .filter-export .n-button { width: 100%; }
  .summary-card :deep(.n-card-header),
  .summary-card :deep(.n-card__content) { padding-left: 14px; padding-right: 14px; }
  .summary-grid { gap: 10px !important; }
  .summary-card :deep(.n-statistic .n-statistic-value) { font-size: 18px; }
  .summary-card :deep(.n-statistic .n-statistic-label) { font-size: 12px; }
  .chart-row { flex-direction: column; align-items: stretch; gap: 6px; }
  .chart-meta { width: 100%; justify-content: space-between; }
  .chart-label { width: auto; font-size: 12px; text-align: left; }
  .chart-bars { width: 100%; }
  .chart-orders { width: auto; font-size: 12px; text-align: right; }
  .bar { height: 16px; }
  .chart-legend { flex-wrap: wrap; }
  .method-info,
  .region-info { flex-wrap: wrap; align-items: flex-start; }
  .method-info { flex-direction: column; gap: 2px; }
  .method-detail { font-size: 12px; }
  .mobile-stat-list { display: flex; flex-direction: column; gap: 10px; }
  .mobile-stat-item { padding: 12px; border: 1px solid #e8e8e8; border-radius: 10px; background: #fff; }
  .mobile-stat-main { font-size: 14px; font-weight: 600; color: #333; }
  .mobile-stat-sub { margin-top: 4px; font-size: 12px; color: #999; }
  .mobile-stat-meta { margin-top: 8px; display: flex; justify-content: space-between; align-items: center; font-size: 13px; color: #666; }
}
</style>