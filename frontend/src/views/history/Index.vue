<template>
  <div class="history-page">
    <n-space vertical :size="24">
      <h1 class="title">登录历史</h1>

      <n-grid :x-gap="16" :y-gap="16" cols="1 s:3" responsive="screen">
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
        <n-data-table v-if="!appStore.isMobile"
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
            <div v-for="(record, idx) in records" :key="idx" class="mobile-card">
              <div class="card-row">
                <span class="label">时间</span>
                <span class="value">{{ formatDateTime(record.login_time) }}</span>
              </div>
              <div class="card-row">
                <span class="label">IP</span>
                <span class="value" style="font-family: monospace;">{{ record.ip_address }}</span>
              </div>
              <div class="card-row">
                <span class="label">位置</span>
                <span class="value">{{ record.location || '-' }}</span>
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
          </n-spin>
          <n-pagination
            v-if="records.length > 0"
            v-model:page="pagination.page"
            :item-count="pagination.itemCount"
            :page-size="pagination.pageSize"
            style="margin-top: 16px; justify-content: center;"
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

const appStore = useAppStore()
const message = useMessage()
const loading = ref(false)
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

const formatDateTime = (dateStr: string) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit',
  })
}

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
  { title: '位置', key: 'location', width: 150, resizable: true, render: (row: any) => row.location || '-' },
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
  loading.value = true
  try {
    const res = await getLoginHistory({ page: pagination.page, page_size: pagination.pageSize })
    records.value = res.data?.items || []
    pagination.itemCount = res.data?.total || 0
  } catch (error: any) {
    message.error(error.message || '获取登录历史失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadHistory()
})
</script>

<style scoped>
.history-page {
  padding: 24px;
}

.title {
  font-size: 28px;
  font-weight: 600;
  margin: 0;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.stat-card {
  padding: 20px 24px;
  border-radius: 12px;
  background: linear-gradient(135deg, #667eea10, #764ba210);
  border: 1px solid #667eea20;
}

.stat-label {
  font-size: 13px;
  color: #999;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
}

.stat-value-sm {
  font-size: 16px;
}

@media (max-width: 767px) {
  .history-page { padding: 0 12px; }
  .stat-card { padding: 14px 16px; }
}
</style>
