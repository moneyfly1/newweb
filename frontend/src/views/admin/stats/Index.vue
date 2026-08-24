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
        <!-- 用户概览 -->
        <n-card title="用户概览" :bordered="false" class="summary-card">
          <n-grid :cols="appStore.isMobile ? 2 : 4" :x-gap="16" :y-gap="16" class="summary-grid">
            <n-gi><n-statistic label="总用户" :value="userStats.total_users || 0" /></n-gi>
            <n-gi><n-statistic label="活跃用户" :value="userStats.active_users || 0" /></n-gi>
            <n-gi><n-statistic label="今日新增" :value="userStats.today_new_users || 0" /></n-gi>
            <n-gi><n-statistic label="付费用户" :value="userStats.paid_users || 0" /></n-gi>
          </n-grid>
        </n-card>
        <!-- Revenue Chart (bar visualization) -->
        <n-card title="收入趋势" :bordered="false">
          <revenue-trend-chart v-if="revenueChart.length > 0 && !appStore.isMobile" :data="revenueChart" />
          <div v-if="revenueChart.length > 0 && appStore.isMobile">
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
              <span class="legend"><span class="legend-dot" style="background: var(--success-color)"></span>收入</span>
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
              <payment-pie-chart v-if="packageStats.length > 0 && !appStore.isMobile" :data="packagePieData" />
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

        <!-- Region Stats -->
        <n-card title="用户地区分布" :bordered="false">
          <template #header-extra>
            <n-button size="tiny" text @click="showAllRegions = !showAllRegions">
              {{ showAllRegions ? '收起' : `查看全部 (${regionStats.length})` }}
            </n-button>
          </template>
          <n-spin :show="regionLoading">
            <div v-if="regionStats.length > 0" class="region-list-compact" :class="{ expanded: showAllRegions }">
              <div v-for="(item, index) in displayRegions" :key="index" class="region-row-compact">
                <span class="region-rank-sm">{{ index + 1 }}</span>
                <span class="region-detail">{{ item.country || '-' }} · {{ item.province || '-' }} · {{ item.city || '-' }}</span>
                <span class="region-bar-wrap">
                  <span class="region-bar" :style="{ width: Math.round((item.count / maxRegionCount) * 100) + '%', background: getColor(index) }"></span>
                </span>
                <span class="region-count-sm">{{ item.count }}</span>
              </div>
            </div>
            <n-empty v-else description="暂无地区数据" style="padding: 40px 0" />
          </n-spin>
        </n-card>

        <!-- 支付分析（此前后端已实现、前端未接入） -->
        <n-card title="支付分析" :bordered="false">
          <template #header-extra>
            <n-radio-group v-model:value="payDays" size="small" @update:value="loadPaymentAnalysis">
              <n-radio-button value="7">7天</n-radio-button>
              <n-radio-button value="30">30天</n-radio-button>
              <n-radio-button value="90">90天</n-radio-button>
            </n-radio-group>
          </template>
          <n-grid :cols="appStore.isMobile ? 1 : 2" :x-gap="16" :y-gap="16">
            <n-gi>
              <n-card title="支付方式统计" size="small" :bordered="false">
                <div v-if="payMethodStats.length > 0" class="pay-stats-wrap">
                  <!-- 桌面端：饼图占比 -->
                  <payment-pie-chart v-if="!appStore.isMobile" :data="payPieData" />
                  <div class="method-list">
                    <div v-for="(item, index) in payMethodStats" :key="index" class="method-item">
                      <div class="method-info">
                        <span class="method-name">{{ paymentMethodText(item.payment_method) }}</span>
                        <span class="method-detail">{{ item.order_count }} 笔 / {{ formatCurrency(item.total_amount) }} · 成功率 {{ item.success_rate ? item.success_rate.toFixed(1) : '0' }}%</span>
                      </div>
                      <n-progress
                        type="line"
                        :percentage="Math.round(((item.total_amount || 0) / payMaxAmount) * 100)"
                        :show-indicator="false"
                        :height="8"
                        :border-radius="4"
                        :color="getColor(index)"
                      />
                    </div>
                  </div>
                </div>
                <n-empty v-else description="暂无支付数据" />
              </n-card>
            </n-gi>
            <n-gi>
              <n-card title="支付方式对比" size="small" :bordered="false">
                <div v-if="payComparisons.length > 0">
                  <n-data-table
                    :columns="payCompareColumns"
                    :data="payComparisons"
                    :bordered="false"
                    size="small"
                    :pagination="false"
                    :max-height="260"
                  />
                </div>
                <n-empty v-else description="暂无对比数据" />
              </n-card>
            </n-gi>
          </n-grid>
          <n-divider style="margin: 12px 0" />
          <div class="hourly-header">
            <span class="hourly-title">订单高峰时段</span>
            <n-select
              v-model:value="payAnalysisMethod"
              :options="payAnalysisOptions"
              size="small"
              clearable
              placeholder="选择支付方式"
              style="width: 160px"
              @update:value="loadPayAnalysis"
            />
          </div>
          <div v-if="payHourly.length > 0" class="hourly-grid">
            <div v-for="h in payHourly" :key="h.hour" class="hourly-item" :title="`${h.hour}时：${h.order_count} 单 · 成功率 ${h.success_rate ? h.success_rate.toFixed(1) : '0'}%`">
              <div class="hourly-bar" :style="{ height: hourlyHeight(h.order_count) }"></div>
              <span class="hourly-label">{{ h.hour }}时</span>
            </div>
          </div>
          <n-empty v-else-if="payAnalysisLoaded" description="该时段无订单" style="padding: 20px 0" />
        </n-card>
      </n-space>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, h, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { getFinancialReport, exportFinancialReport, getRegionStats, getPaymentStats, getPaymentMethodComparison, getPaymentAnalysis, getAdminUserStats } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { formatCurrency } from '@/utils/amount'
import PaymentPieChart from '../components/PaymentPieChart.vue'
import RevenueTrendChart from '../components/RevenueTrendChart.vue'

const appStore = useAppStore()
const message = useMessage()
const loading = ref(false)
const exporting = ref(false)
const regionLoading = ref(false)
const showAllRegions = ref(false)
const period = ref('month')
const dateRange = ref<[number, number] | null>(null)

const summary = ref({
  total_revenue: 0, total_orders: 0, paid_orders: 0,
  refunded_orders: 0, average_order_amount: 0,
  total_recharge: 0, total_recharge_count: 0,
  new_users: 0, new_subscriptions: 0,
})
const revenueChart = ref<any[]>([])
const userStats = ref<any>({})
const loadUserStats = async () => {
  try { const res: any = await getAdminUserStats(); userStats.value = res.data || {} } catch {}
}
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
const colors = ['var(--success-color)', '#2080f0', 'var(--warning-color)', 'var(--danger-color)', '#8a2be2', '#36ad6a', '#4098fc', '#f2c97d', '#e88080', '#a78bfa']
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
const displayRegions = computed(() => {
  if (showAllRegions.value) return regionStats.value
  return regionStats.value.slice(0, 10)
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
    const blob = new Blob([res.data || res], { type: 'text/csv;charset=utf-8' })
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

// ===== 支付分析（后端已实现，前端接入）=====
const payDays = ref('30')
const payMethodStats = ref<any[]>([])
const payComparisons = ref<any[]>([])
const payHourly = ref<any[]>([])
const payAnalysisMethod = ref<string | null>(null)
const payAnalysisLoaded = ref(false)
const payAnalysisOptions = computed(() => [
  { label: '支付宝', value: 'alipay' },
  { label: '易支付', value: 'epay' },
  { label: '微信', value: 'wxpay' },
  { label: 'Stripe', value: 'stripe' },
  { label: '码支付', value: 'codepay' },
  { label: '余额', value: 'balance' },
])
const hourlyHeight = (count: number) => {
  const max = Math.max(1, ...payHourly.value.map((h: any) => h.order_count || 0))
  return Math.max(4, Math.round(((count || 0) / max) * 80)) + 'px'
}
const loadPayAnalysis = async () => {
  if (!payAnalysisMethod.value) { payHourly.value = []; payAnalysisLoaded.value = false; return }
  try {
    const res: any = await getPaymentAnalysis({ days: payDays.value, payment_method: payAnalysisMethod.value })
    payHourly.value = (res.data?.hourly_stats || []).filter((h: any) => h.hour >= 0)
  } catch { payHourly.value = [] }
  payAnalysisLoaded.value = true
}
const packagePieData = computed(() =>
  packageStats.value
    .filter((s: any) => Number(s.amount) > 0)
    .map((s: any) => ({ name: s.package_name || '未知', value: Number(s.amount) || 0 }))
)
const payMaxAmount = computed(() => Math.max(1, ...payMethodStats.value.map((s: any) => Number(s.total_amount) || 0)))
const payPieData = computed(() =>
  payMethodStats.value
    .filter((s: any) => Number(s.total_amount) > 0)
    .map((s: any) => ({ name: paymentMethodText(s.payment_method), value: Number(s.total_amount) || 0 }))
)

const paymentMethodText = (m: string) => {
  const map: Record<string, string> = {
    alipay: '支付宝', epay: '易支付', wxpay: '微信', qqpay: 'QQ', codepay: '码支付',
    codepay_alipay: '码支付-支付宝', codepay_wxpay: '码支付-微信', stripe: 'Stripe',
    balance: '余额', admin_manual: '管理员确认', crypto: '加密货币', recharge: '充值',
  }
  return map[m] || m || '未知'
}

const payCompareColumns = [
  { title: '方式', key: 'payment_method', width: 90, render: (row: any) => paymentMethodText(row.payment_method) },
  { title: '总单', key: 'total_orders', width: 60 },
  { title: '成功', key: 'success_orders', width: 60 },
  { title: '成功率', key: 'success_rate', width: 75, render: (row: any) => (row.success_rate ? row.success_rate.toFixed(1) : '0') + '%' },
  { title: '金额', key: 'total_amount', width: 90, render: (row: any) => formatCurrency(row.total_amount) },
  { title: '均时(分)', key: 'average_time', width: 80, render: (row: any) => (row.average_time ? Number(row.average_time).toFixed(1) : '-') },
]

const loadPaymentAnalysis = async () => {
  const [statsRes, compRes] = await Promise.allSettled([
    getPaymentStats({ days: payDays.value }),
    getPaymentMethodComparison({ days: payDays.value }),
  ])
  if (statsRes.status === 'fulfilled') payMethodStats.value = (statsRes.value as any).data?.method_stats || []
  if (compRes.status === 'fulfilled') payComparisons.value = (compRes.value as any).data?.comparisons || []
  if (payAnalysisMethod.value) loadPayAnalysis()
}

onMounted(() => {
  loadFinancialReport()
  loadPaymentAnalysis()
  loadUserStats()
})
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
.chart-label { width: 90px; font-size: 13px; color: var(--text-color-secondary); text-align: right; flex-shrink: 0; }
.chart-bars { flex: 1; min-width: 0; }
.bar-group { margin-bottom: 2px; }
.bar {
  height: 18px; border-radius: 4px; display: flex; align-items: center;
  padding: 0 6px; min-width: 0; transition: width 0.3s;
}
.revenue-bar { background: var(--success-color); }
.recharge-bar { background: #2080f0; }
.bar-text { color: #fff; font-size: 11px; white-space: nowrap; overflow: hidden; }
.chart-orders { width: 60px; font-size: 12px; color: var(--text-color-secondary); text-align: right; flex-shrink: 0; }
.legend { display: flex; align-items: center; font-size: 13px; color: var(--text-color-secondary); }
.legend-dot { width: 10px; height: 10px; border-radius: 2px; margin-right: 4px; display: inline-block; }
.method-item { margin-bottom: 12px; }
.method-info { display: flex; justify-content: space-between; margin-bottom: 4px; font-size: 14px; }
.method-name { font-weight: 500; }
.method-detail { color: var(--text-color-secondary); font-size: 13px; }
.region-list-compact { max-height: 320px; overflow-y: auto; }
.region-list-compact.expanded { max-height: none; }
.region-row-compact {
  display: flex; align-items: center; gap: 8px; padding: 6px 0; border-bottom: 1px solid #f8f8f8;
  font-size: 13px;
}
.region-rank-sm {
  width: 20px; height: 20px; border-radius: 50%; background: var(--bg-page-color);
  display: flex; align-items: center; justify-content: center;
  font-size: 11px; font-weight: 600; color: var(--text-color-secondary); flex-shrink: 0;
}
.region-detail { flex: 0 0 180px; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-color); }
.region-bar-wrap { flex: 1; height: 8px; background: var(--bg-page-color); border-radius: 4px; min-width: 40px; }
.region-bar { height: 8px; border-radius: 4px; display: block; transition: width 0.3s; }
.region-count-sm { width: 40px; text-align: right; font-weight: 600; color: var(--text-color); flex-shrink: 0; font-size: 13px; }
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
  .method-info { flex-direction: column; gap: 2px; }
  .method-detail { font-size: 12px; }
  .region-detail { flex: 0 0 120px; font-size: 12px; }
  .region-row-compact { gap: 4px; padding: 4px 0; }
  .region-rank-sm { width: 16px; height: 16px; font-size: 10px; }
  .mobile-stat-list { display: flex; flex-direction: column; gap: 10px; }
  .mobile-stat-item { padding: 12px; border: 1px solid var(--border-color); border-radius: 10px; background: var(--bg-color); }
  .mobile-stat-main { font-size: 14px; font-weight: 600; color: var(--text-color); }
  .mobile-stat-sub { margin-top: 4px; font-size: 12px; color: var(--text-color-secondary); }
  .mobile-stat-meta { margin-top: 8px; display: flex; justify-content: space-between; align-items: center; font-size: 13px; color: var(--text-color-secondary); }
}

.hourly-header { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 12px; }
.hourly-title { font-size: 14px; font-weight: 600; color: var(--text-color); }
.hourly-grid { display: flex; align-items: flex-end; gap: 4px; overflow-x: auto; padding-bottom: 4px; }
.hourly-item { display: flex; flex-direction: column; align-items: center; gap: 4px; flex-shrink: 0; }
.hourly-bar { width: 14px; border-radius: 4px 4px 0 0; background: var(--primary-color, #4f46e5); opacity: 0.75; min-height: 4px; transition: height 0.3s; }
.hourly-label { font-size: 10px; color: var(--text-color-secondary, #888); }

.pay-stats-wrap { display: flex; flex-direction: column; gap: 8px; }
.pay-stats-wrap .method-list { display: flex; flex-direction: column; gap: 10px; }
</style>