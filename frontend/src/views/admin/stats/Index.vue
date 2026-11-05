<template>
  <div class="stats-container">
    <n-spin :show="loading">
      <n-space vertical :size="20">
        <!-- Filters -->
        <n-card :bordered="false">
          <n-space align="center" :wrap="true" class="filters-toolbar app-sticky-toolbar">
            <div class="filter-period">
              <n-radio-group v-model:value="period" @update:value="loadFinancialReport">
                <n-radio-button value="day">日</n-radio-button>
                <n-radio-button value="week">周</n-radio-button>
                <n-radio-button value="month">月</n-radio-button>
              </n-radio-group>
            </div>
            <n-button v-if="appStore.isMobile" size="small" secondary @click="showMoreFilters = !showMoreFilters">
              {{ showMoreFilters ? '收起' : '筛选' }}
            </n-button>
            <div class="filter-range" v-show="!appStore.isMobile || showMoreFilters">
              <n-date-picker
                v-model:value="dateRange"
                type="daterange"
                clearable
                :shortcuts="dateShortcuts"
                @update:value="onDateRangeChange"
              />
            </div>
            <div class="filter-export" v-show="!appStore.isMobile || showMoreFilters">
              <n-button @click="handleExport" :loading="exporting">导出 CSV</n-button>
            </div>
          </n-space>
        </n-card>

        <n-tabs v-model:value="activeTab" type="line" animated display-directive="if" class="stats-tabs">
          <n-tab-pane name="overview" tab="概览">
            <n-space vertical :size="16">
              <n-card :bordered="false" class="stats-card">
                <div class="metric-section">
                  <div class="metric-section-title">财务</div>
                  <div class="kpi-grid">
                    <div class="kpi-tile">
                      <span class="kpi-label">总收入</span>
                      <span class="kpi-value">{{ formatCurrency(summary.total_revenue) }}</span>
                    </div>
                    <div class="kpi-tile">
                      <span class="kpi-label">总订单</span>
                      <span class="kpi-value">{{ summary.total_orders }}</span>
                    </div>
                    <div class="kpi-tile">
                      <span class="kpi-label">已支付</span>
                      <span class="kpi-value">{{ summary.paid_orders }}</span>
                    </div>
                    <div class="kpi-tile">
                      <span class="kpi-label">平均客单价</span>
                      <span class="kpi-value">{{ formatCurrency(summary.average_order_amount) }}</span>
                    </div>
                    <div class="kpi-tile">
                      <span class="kpi-label">充值总额</span>
                      <span class="kpi-value">{{ formatCurrency(summary.total_recharge) }}</span>
                    </div>
                    <div class="kpi-tile">
                      <span class="kpi-label">新用户</span>
                      <span class="kpi-value">{{ summary.new_users }}</span>
                    </div>
                  </div>
                </div>

                <div class="metric-divider"></div>

                <div class="metric-section">
                  <div class="metric-section-title">用户</div>
                  <div class="kpi-grid kpi-grid--compact">
                    <div class="kpi-tile">
                      <span class="kpi-label">总用户</span>
                      <span class="kpi-value">{{ userStats.total_users || 0 }}</span>
                    </div>
                    <div class="kpi-tile">
                      <span class="kpi-label">活跃用户</span>
                      <span class="kpi-value">{{ userStats.active_users || 0 }}</span>
                    </div>
                    <div class="kpi-tile">
                      <span class="kpi-label">今日新增</span>
                      <span class="kpi-value">{{ userStats.today_new_users || 0 }}</span>
                    </div>
                    <div class="kpi-tile">
                      <span class="kpi-label">付费用户</span>
                      <span class="kpi-value">{{ userStats.paid_users || 0 }}</span>
                    </div>
                  </div>
                </div>
              </n-card>

              <n-card :bordered="false" class="stats-card">
                <template #header>
                  <div class="card-title-row">
                    <span class="card-title">收入趋势</span>
                    <span class="card-hint">单位：元</span>
                  </div>
                </template>
                <template v-if="revenueChart.length > 0">
                  <div class="mobile-chart">
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
                </template>
                <n-empty v-else description="暂无数据" />
              </n-card>
            </n-space>
          </n-tab-pane>

          <n-tab-pane name="payment" tab="收入与支付">
            <n-space vertical :size="16">
              <n-card :bordered="false" class="stats-card">
                <template #header>
                  <div class="card-title-row">
                    <span class="card-title">支付方式分布</span>
                    <span class="card-hint">近 {{ payDays }} 天</span>
                  </div>
                </template>
                <div v-if="payMethodStats.length > 0" class="pay-dist">
                  <payment-pie-chart v-if="!appStore.isMobile" :data="payPieData" class="pay-dist-chart" />
                  <div class="pay-dist-list">
                    <div v-for="(item, index) in payMethodStats" :key="index" class="pay-dist-row">
                      <div class="pay-dist-head">
                        <span class="pay-dist-name">{{ paymentMethodText(item.payment_method) }}</span>
                        <span class="pay-dist-amount">{{ formatCurrency(item.total_amount) }}</span>
                      </div>
                      <n-progress
                        type="line"
                        :percentage="Math.round(((item.total_amount || 0) / payMaxAmount) * 100)"
                        :show-indicator="false"
                        :height="6"
                        :border-radius="3"
                        :color="getColor(index)"
                      />
                      <div class="pay-dist-meta">{{ item.order_count }} 笔 · 成功率 {{ item.success_rate ? item.success_rate.toFixed(1) : '0' }}%</div>
                    </div>
                  </div>
                </div>
                <n-empty v-else description="暂无支付数据" />
              </n-card>

              <n-card :bordered="false" class="stats-card">
                <template #header>
                  <div class="card-title-row">
                    <span class="card-title">套餐销售排行</span>
                    <span class="card-hint">按销售额排序</span>
                  </div>
                </template>
                <div v-if="packageStats.length > 0" class="pkg-list">
                  <div v-for="(item, index) in packageStats" :key="index" class="pkg-row">
                    <span class="pkg-rank" :class="{ 'is-top': index < 3 }">{{ index + 1 }}</span>
                    <div class="pkg-main">
                      <div class="pkg-name">{{ item.package_name || '未知套餐' }}</div>
                      <n-progress
                        type="line"
                        :percentage="Math.round(((item.count || 0) / maxPackageCount) * 100)"
                        :show-indicator="false"
                        :height="6"
                        :border-radius="3"
                        :color="'var(--primary-color)'"
                      />
                    </div>
                    <div class="pkg-meta">
                      <span class="pkg-count">{{ item.count }} 单</span>
                      <span class="pkg-amount">{{ formatCurrency(item.amount) }}</span>
                    </div>
                  </div>
                </div>
                <n-empty v-else description="暂无数据" />
              </n-card>

              <n-card :bordered="false" class="stats-card">
                <template #header>
                  <div class="card-title-row">
                    <span class="card-title">支付方式对比</span>
                    <n-radio-group v-model:value="payDays" size="small" @update:value="loadPaymentAnalysis">
                      <n-radio-button value="7">7天</n-radio-button>
                      <n-radio-button value="30">30天</n-radio-button>
                      <n-radio-button value="90">90天</n-radio-button>
                    </n-radio-group>
                  </div>
                </template>
                <n-data-table
                  v-if="payComparisons.length > 0"
                  :columns="payCompareColumns"
                  :data="payComparisons"
                  :bordered="false"
                  size="small"
                  :pagination="false"
                />
                <n-empty v-else description="暂无对比数据" />
              </n-card>

              <n-card :bordered="false" class="stats-card">
                <template #header>
                  <div class="card-title-row">
                    <span class="card-title">订单高峰时段</span>
                    <n-select
                      v-model:value="payAnalysisMethod"
                      :options="payAnalysisOptions"
                      size="small"
                      clearable
                      placeholder="全部支付方式"
                      style="width: 170px"
                      @update:value="loadPayAnalysis"
                    />
                  </div>
                </template>
                <div v-if="payHourly.length > 0" class="hourly-grid">
                  <div v-for="h in payHourly" :key="h.hour" class="hourly-item" :title="`${h.hour}时：${h.order_count} 单 · 成功率 ${h.success_rate ? h.success_rate.toFixed(1) : '0'}%`">
                    <div class="hourly-bar" :style="{ height: hourlyHeight(h.order_count) }"></div>
                    <span class="hourly-label">{{ h.hour }}时</span>
                  </div>
                </div>
                <n-empty v-else description="请选择支付方式查看时段分布" style="padding: 20px 0" />
              </n-card>
            </n-space>
          </n-tab-pane>

          <n-tab-pane name="users" tab="用户与地区">
            <n-space vertical :size="16">
        <!-- Top Users -->
        <n-card title="消费排行 TOP 10" :bordered="false">
          <div v-if="topUsers.length > 0 && appStore.isMobile" class="mobile-stat-list top-user-list">
            <div v-for="(item, index) in topUsers" :key="index" class="mobile-stat-item top-user-item" style="cursor: pointer" @click="userCurrent = item; userDrawer = true">
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
            :row-props="userRowProps"
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
                <span class="region-detail">{{ formatRegionDetail(item) }}</span>
                <span class="region-bar-wrap">
                  <span class="region-bar" :style="{ width: Math.round((item.count / maxRegionCount) * 100) + '%', background: getColor(index) }"></span>
                </span>
                <span class="region-count-sm">{{ item.count }}</span>
              </div>
            </div>
            <n-empty v-else description="暂无地区数据" style="padding: 40px 0" />
          </n-spin>
        </n-card>
            </n-space>
          </n-tab-pane>

          <n-tab-pane name="devices" tab="设备分析">
          <!-- 设备分析（对齐 myweb：按客户端/系统分布 + 设备明细） -->
          <!-- 按用户 / 按分组（对齐 myweb：一个客户多设备多客户端、客户端搜索、升级提醒） -->
          <n-card title="按用户 / 按分组" :bordered="false" style="margin-top: 16px">
            <device-usage-section />
          </n-card>
          </n-tab-pane>
        </n-tabs>
      </n-space>
    </n-spin>

    <!-- 设备详情：统一走抽屉，表格里只留摘要字段 -->
    <!-- 用户消费详情抽屉 -->
    <n-drawer v-model:show="userDrawer" :width="appStore.isMobile ? '100%' : 400" placement="right">
      <n-drawer-content title="用户消费详情" closable>
        <n-descriptions v-if="userCurrent" :column="1" label-placement="left" bordered size="small">
          <n-descriptions-item label="用户名">{{ userCurrent.username || '-' }}</n-descriptions-item>
          <n-descriptions-item label="用户 ID">{{ userCurrent.user_id }}</n-descriptions-item>
          <n-descriptions-item label="消费总额">{{ formatCurrency(userCurrent.total_spent) }}</n-descriptions-item>
          <n-descriptions-item label="订单数">{{ userCurrent.order_count }}</n-descriptions-item>
          <n-descriptions-item label="客单价">{{ formatCurrency(userCurrent.order_count ? userCurrent.total_spent / userCurrent.order_count : 0) }}</n-descriptions-item>
        </n-descriptions>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>


<script setup lang="ts">
import { ref, reactive, computed, h, onMounted, defineAsyncComponent } from 'vue'
import { useMessage, NTag } from 'naive-ui'
import request from '@/utils/request'
import { getFinancialReport, exportFinancialReport, getRegionStats, getPaymentStats, getPaymentMethodComparison, getPaymentAnalysis, getAdminUserStats } from '@/api/admin'
import { formatCountryOnly } from '@/utils/format'
import DeviceUsageSection from './DeviceUsageSection.vue'
import { useAppStore } from '@/stores/app'
import { formatCurrency } from '@/utils/amount'
const PaymentPieChart = defineAsyncComponent(() => import('../components/PaymentPieChart.vue'))
import { formatDateTime } from '@/utils/format'

const appStore = useAppStore()
const message = useMessage()
const activeTab = ref('overview')
const showMoreFilters = ref(false)

const userDrawer = ref(false)
const userCurrent = ref<any>(null)
const userRowProps = (row: any) => ({
  style: 'cursor: pointer',
  onClick: () => { userCurrent.value = row; userDrawer.value = true },
})

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
// 地区展示统一走 formatCountryOnly：国家/地区显示中文（含港澳台口径），
// 省市有值且不是 '-' 时追加显示，避免出现「United States · - · -」这类英文口径。
const formatRegionDetail = (item: { country?: string; province?: string; city?: string }) => {
  const parts: string[] = []
  const country = formatCountryOnly(item.country, '')
  if (country) parts.push(country)
  for (const v of [item.province, item.city]) {
    const text = (v || '').trim()
    if (text && text !== '-' && !parts.includes(text)) parts.push(text)
  }
  return parts.length ? parts.join(' · ') : '-'
}

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
const maxPackageCount = computed(() => Math.max(1, ...packageStats.value.map((s: any) => Number(s.count) || 0)))
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
.bar-text { color: #fff; font-size: 12px; white-space: nowrap; overflow: hidden; }
.chart-orders { width: 60px; font-size: 12px; color: var(--text-color-secondary); text-align: right; flex-shrink: 0; }
.legend { display: flex; align-items: center; font-size: 13px; color: var(--text-color-secondary); }
.legend-dot { width: 10px; height: 10px; border-radius: 2px; margin-right: 4px; display: inline-block; }
.method-item { margin-bottom: 12px; }
.method-info { display: flex; justify-content: space-between; margin-bottom: 4px; font-size: 14px; }
.method-name { font-weight: 500; }
.method-detail { color: var(--text-color-secondary); font-size: 13px; }
.device-dist-title { font-size: 13px; font-weight: 600; margin-bottom: 8px; }
.device-dist-row { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.device-dist-name { width: 110px; font-size: 12px; color: var(--text-color-secondary); flex-shrink: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.device-dist-row :deep(.n-progress) { flex: 1; }
.device-dist-count { width: 44px; text-align: right; font-size: 12px; flex-shrink: 0; }
.device-empty { font-size: 12px; color: var(--text-color-disabled); margin: 0; }
.device-detail-head { display: flex; align-items: center; justify-content: space-between; margin: 16px 0 8px; gap: 8px; flex-wrap: wrap; }
.region-list-compact { max-height: 320px; overflow-y: auto; }
.region-list-compact.expanded { max-height: none; }
.region-row-compact {
  display: flex; align-items: center; gap: 8px; padding: 6px 0; border-bottom: 1px solid #f8f8f8;
  font-size: 13px;
}
.region-rank-sm {
  width: 20px; height: 20px; border-radius: 50%; background: var(--bg-page-color);
  display: flex; align-items: center; justify-content: center;
  font-size: 12px; font-weight: 600; color: var(--text-color-secondary); flex-shrink: 0;
}
.region-detail { flex: 0 0 180px; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-color); }
.region-bar-wrap { flex: 1; height: 8px; background: var(--bg-page-color); border-radius: 4px; min-width: 40px; }
.region-bar { height: 8px; border-radius: 4px; display: block; transition: width 0.3s; }
.region-count-sm { width: 40px; text-align: right; font-weight: 600; color: var(--text-color); flex-shrink: 0; font-size: 13px; }
@media (max-width: 767px) {
  /* 左右留白由全局统一给（mobile-app-ui.css + admin-mobile.css），页面不再自带内边距 */
  .filters-toolbar { flex-direction: column; align-items: stretch !important; gap: 12px; }
  /* 卡片内容区本来就有 12px 内边距，工具条不再叠加全局的 -12px 出血，保持左右对称 */
  .filters-toolbar.app-sticky-toolbar { margin-left: 0; margin-right: 0; }
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
  .region-rank-sm { width: 20px; height: 20px; font-size: 12px; }
  .mobile-stat-list { display: flex; flex-direction: column; gap: 10px; }
  .mobile-stat-item {
    padding: 12px 14px;
    border: 1px solid color-mix(in srgb, var(--border-color, #e5e7eb) 60%, transparent);
    border-radius: 16px;
    background: var(--bg-color);
    box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
  }
  .mobile-stat-main { font-size: 14px; font-weight: 600; color: var(--text-color); }
  .mobile-stat-sub { margin-top: 4px; font-size: 12px; color: var(--text-color-secondary); }
  .mobile-stat-meta { margin-top: 8px; display: flex; justify-content: space-between; align-items: center; font-size: 13px; color: var(--text-color-secondary); }
}

.hourly-header { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 12px; }
.hourly-title { font-size: 14px; font-weight: 600; color: var(--text-color); }
.hourly-grid { display: flex; align-items: flex-end; gap: 4px; overflow-x: auto; padding-bottom: 4px; }
.hourly-item { display: flex; flex-direction: column; align-items: center; gap: 4px; flex-shrink: 0; }
.hourly-bar { width: 14px; border-radius: 4px 4px 0 0; background: var(--primary-color, #4f46e5); opacity: 0.75; min-height: 4px; transition: height 0.3s; }
.hourly-label { font-size: 12px; color: var(--text-color-secondary, #888); }

.pay-stats-wrap { display: flex; flex-direction: column; gap: 8px; }
.pay-stats-wrap .method-list { display: flex; flex-direction: column; gap: 10px; }

/* 数据分析页：标签化组织，避免所有区块堆在一屏 */
.stats-tabs :deep(.n-tabs-nav) { margin-bottom: 4px; }
.stats-tabs :deep(.n-tab-pane) { padding-top: 12px; }
@media (max-width: 767px) {
  .stats-tabs :deep(.n-tabs-tab) { font-size: 13px; padding: 0 7px; }
  /* 窄屏（iPhone 12~15 的 390px）再收一档，四个标签刚好铺满不裁切 */
  .stats-tabs :deep(.n-tabs-tab-wrapper) { flex: 1 1 0; min-width: 0; }
  .stats-tabs :deep(.n-tabs-tab) { justify-content: center; }
  /* 4 个标签在 390px 宽屏放不下，允许横向滚动而不是把最后一个切成半个字 */
  .stats-tabs :deep(.n-tabs-nav-scroll-content) { overflow-x: auto; scrollbar-width: none; }
  .stats-tabs :deep(.n-tabs-nav-scroll-content)::-webkit-scrollbar { display: none; }
  .stats-tabs :deep(.n-tabs-nav) { overflow: hidden; }
}
/* 空态收进卡片，不再占满整屏 */
.stats-card :deep(.n-empty) { padding: 12px 0; }
.stats-card :deep(.n-empty__icon) { width: 40px; height: 40px; font-size: 40px; }

/* ===== 数据分析：统一卡片、指标块与支付/套餐列表 ===== */
.stats-card :deep(.n-card__content) { padding: 16px; }
.card-title-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.card-title { font-size: 15px; font-weight: 600; }
.card-hint { font-size: 12px; color: var(--text-color-secondary); }

.metric-section-title { font-size: 12px; font-weight: 600; letter-spacing: .5px; color: var(--text-color-secondary); margin-bottom: 10px; }
.metric-divider { height: 1px; background: var(--border-color, #e5e7eb); margin: 16px 0; }

.kpi-grid { display: grid; gap: 10px; grid-template-columns: repeat(2, minmax(0, 1fr)); }
.kpi-grid--compact { grid-template-columns: repeat(2, minmax(0, 1fr)); }
@media (min-width: 768px) {
  .kpi-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .kpi-grid--compact { grid-template-columns: repeat(4, minmax(0, 1fr)); }
}
@media (min-width: 1200px) {
  .kpi-grid { grid-template-columns: repeat(6, minmax(0, 1fr)); }
}
.kpi-tile {
  display: flex; flex-direction: column; gap: 6px;
  padding: 14px 14px 12px; border-radius: 10px;
  background: var(--n-color, #fff);
  border: 1px solid var(--border-color, #e5e7eb);
  min-width: 0;
}
.kpi-label { font-size: 12px; color: var(--text-color-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.kpi-value { font-size: 20px; font-weight: 600; line-height: 1.25; word-break: break-all; }
@media (max-width: 767px) { .kpi-value { font-size: 17px; } }

.pay-dist { display: grid; gap: 20px; grid-template-columns: 240px 1fr; align-items: center; }
.pay-dist-chart { width: 240px; }
.pay-dist-list { display: flex; flex-direction: column; gap: 14px; min-width: 0; }
.pay-dist-row { display: flex; flex-direction: column; gap: 4px; }
.pay-dist-head { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; }
.pay-dist-name { font-size: 13px; font-weight: 500; }
.pay-dist-amount { font-size: 13px; font-weight: 600; }
.pay-dist-meta { font-size: 12px; color: var(--text-color-secondary); }
@media (max-width: 767px) {
  .pay-dist { grid-template-columns: 1fr; gap: 16px; }
  .pay-dist-chart { width: 100%; }
}

.pkg-list { display: flex; flex-direction: column; gap: 12px; }
.pkg-row { display: grid; grid-template-columns: 24px 1fr auto; gap: 12px; align-items: center; }
.pkg-rank { width: 24px; height: 24px; border-radius: 6px; display: inline-flex; align-items: center; justify-content: center; font-size: 12px; color: var(--text-color-secondary); background: var(--bg-page-color, #f1f5f9); }
.pkg-rank.is-top { color: #fff; background: var(--primary-color, #4f46e5); }
.pkg-main { min-width: 0; display: flex; flex-direction: column; gap: 6px; }
.pkg-name { font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pkg-meta { display: flex; flex-direction: column; align-items: flex-end; gap: 2px; }
.pkg-count { font-size: 12px; color: var(--text-color-secondary); }
.pkg-amount { font-size: 13px; font-weight: 600; }
@media (max-width: 767px) {
  .pkg-row { grid-template-columns: 24px 1fr; }
  .pkg-meta { grid-column: 2; flex-direction: row; justify-content: space-between; align-items: center; }
}
</style>
