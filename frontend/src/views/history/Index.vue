<template>
  <div class="history-page">
    <n-space vertical :size="appStore.isMobile ? 12 : 24">
      <h1 class="title">登录历史</h1>

      <n-grid :x-gap="12" :y-gap="12" cols="3" responsive="screen">
        <n-gi>
          <div class="stat-card">
            <div class="stat-label">总登录次数</div>
            <div class="stat-value">{{ stats.total }}</div>
          </div>
        </n-gi>
        <n-gi>
          <div class="stat-card">
            <div class="stat-label">不同 IP 数</div>
            <div class="stat-value">{{ stats.uniqueIps }}</div>
          </div>
        </n-gi>
        <n-gi>
          <div class="stat-card">
            <div class="stat-label">最近登录</div>
            <div class="stat-value stat-value-sm">{{ stats.lastLogin }}</div>
          </div>
        </n-gi>
      </n-grid>

      <n-card :bordered="false">
        <!-- Desktop table -->
        <n-data-table v-if="!appStore.isMobile" remote
          :columns="columns"
          :data="records"
          :loading="loading"
          :pagination="pagination"
          :bordered="false"
          :single-line="false"
        />
        <!-- Mobile card list -->
        <div v-else>
          <n-spin :show="loading">
            <div v-if="!loading && records.length === 0" class="mobile-empty">暂无登录记录</div>
            <div v-else class="mobile-card-list">
              <div v-for="(record, idx) in records" :key="idx" class="mobile-card">
                <div class="card-row">
                  <span class="label">时间</span>
                  <span class="value">{{ formatDateTime(record.login_time) }}</span>
                </div>
                <div class="card-row">
                  <span class="label">IP</span>
                  <span class="value mono">{{ record.ip_address }}</span>
                </div>
                <div class="card-row">
                  <span class="label">位置</span>
                  <span class="value">{{ formatLocation(record.location) }}</span>
                </div>
                <div class="card-row">
                  <span class="label">状态</span>
                  <span class="value">
                    <n-tag :type="record.login_status === 'success' ? 'success' : 'error'" size="small" :bordered="false">
                      {{ record.login_status === 'success' ? '成功' : '失败' }}
                    </n-tag>
                  </span>
                </div>
              </div>
            </div>
          </n-spin>
          <n-pagination
            v-if="records.length > 0"
            v-model:page="pagination.page"
            :item-count="pagination.itemCount"
            :page-size="pagination.pageSize"
            style="margin-top: 12px; justify-content: center"
            @update:page="(p: number) => { pagination.page = p; loadHistory() }"
          />
        </div>
      </n-card>
    </n-space>
  </div>
</template>

<script setup lang="tsx">
import { ref, reactive, onMounted, h, computed } from 'vue'
import { NTag, useMessage } from 'naive-ui'
import { getLoginHistory } from '@/api/user'
import { useAppStore } from '@/stores/app'
import { formatLocation } from '@/utils/i18n'
import { formatDateTime } from '@/utils/date'
import { usePageLoading } from '@/composables/usePageLoading'

const appStore = useAppStore()
const message = useMessage()
const { loading, beginLoad, endLoad } = usePageLoading()
const records = ref<any[]>([])

const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  onChange: (page: number) => {
    pagination.page = page
    loadHistory()
  },
  onUpdatePageSize: (pageSize: number) => {
    pagination.pageSize = pageSize
    pagination.page = 1
    loadHistory()
  },
})

const stats = computed(() => {
  const total = pagination.itemCount
  const ips = new Set(records.value.map((r: any) => r.ip_address))
  const last = records.value.length > 0 ? formatDateTime(records.value[0].login_time) : '--'
  return { total, uniqueIps: ips.size, lastLogin: last }
})

const columns = [
  {
    title: '登录时间',
    key: 'login_time',
    width: 180,
    resizable: true,
    sorter: (a: any, b: any) => new Date(a.login_time).getTime() - new Date(b.login_time).getTime(),
    render: (row: any) => formatDateTime(row.login_time),
  },
  { title: 'IP 地址', key: 'ip_address', width: 150, resizable: true },
  { title: '位置', key: 'location', width: 150, resizable: true, render: (row: any) => formatLocation(row.location) },
  {
    title: '设备信息',
    key: 'user_agent',
    ellipsis: { tooltip: true },
    render: (row: any) => row.user_agent || '-',
  },
  {
    title: '状态',
    key: 'login_status',
    width: 100,
    resizable: true,
    render: (row: any) => {
      const success = row.login_status === 'success'
      return h(NTag, { type: success ? 'success' : 'error', size: 'small', bordered: false }, { default: () => success ? '成功' : '失败' })
    },
  },
]

const loadHistory = async () => {
  beginLoad(records.value.length > 0)
  try {
    const res = await getLoginHistory({ page: pagination.page, page_size: pagination.pageSize })
    records.value = res.data?.items || []
    pagination.itemCount = res.data?.total || 0
  } catch (error: any) {
    message.error(error.message || '获取登录历史失败')
  } finally {
    endLoad()
  }
}

onMounted(() => {
  loadHistory()
})
</script>

<style scoped>
/* 手机端横向留白由全局统一（10px）：页面根容器不再叠加左右 padding */
.history-page {
  padding: 8px 0 12px;
}

.title {
  font-size: 20px;
  font-weight: 600;
  margin: 0;
  background: var(--brand-gradient);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.stat-card {
  padding: 12px 14px;
  border-radius: 16px;
  background: var(--bg-color, #fff);
  border: 1px solid color-mix(in srgb, var(--border-color, #e2e8f0) 70%, transparent);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04), 0 6px 16px rgba(15, 23, 42, 0.04);
}

.stat-label {
  font-size: 13px;
  color: var(--text-color-secondary);
  margin-bottom: 6px;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
}

.stat-value-sm {
  font-size: 16px;
  word-break: break-word;
}

.mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; word-break: break-all; }

@media (max-width: 767px) {
  /* 手机端左右留白由全局统一（10px），根容器只保留纵向节奏 */
  .history-page { padding: 8px 0 12px !important; }
  /* 统计卡：手机端三列并排，字小一号避免换行 */
  .stat-card { padding: 12px 14px; }
  .stat-value { font-size: 22px; }
  .stat-value-sm { font-size: 14px; }
  /* 分页按钮默认 28×28，手指点不准：手机端撑到 40px（父级已 flex-wrap，不会横向撑破） */
  .history-page :deep(.n-pagination .n-pagination-item) { min-width: 40px; height: 40px; }
}
</style>
