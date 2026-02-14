<template>
  <div class="admin-email-queue-page">
    <!-- Stats Cards -->
    <n-grid :cols="4" :x-gap="16" :y-gap="16" responsive="screen" style="margin-bottom: 24px">
      <n-grid-item span="4 s:2 m:1">
        <n-card class="stat-card stat-card-blue" :bordered="false">
          <div class="stat-content">
            <div class="stat-icon">
              <n-icon :size="28">
                <MailOutline />
              </n-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">总邮件数</div>
              <div class="stat-value">{{ stats.total }}</div>
            </div>
          </div>
        </n-card>
      </n-grid-item>

      <n-grid-item span="4 s:2 m:1">
        <n-card class="stat-card stat-card-orange" :bordered="false">
          <div class="stat-content">
            <div class="stat-icon">
              <n-icon :size="28">
                <TimeOutline />
              </n-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">待发送</div>
              <div class="stat-value">{{ stats.pending }}</div>
            </div>
          </div>
        </n-card>
      </n-grid-item>

      <n-grid-item span="4 s:2 m:1">
        <n-card class="stat-card stat-card-green" :bordered="false">
          <div class="stat-content">
            <div class="stat-icon">
              <n-icon :size="28">
                <CheckmarkCircleOutline />
              </n-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">已发送</div>
              <div class="stat-value">{{ stats.sent }}</div>
            </div>
          </div>
        </n-card>
      </n-grid-item>

      <n-grid-item span="4 s:2 m:1">
        <n-card class="stat-card stat-card-red" :bordered="false">
          <div class="stat-content">
            <div class="stat-icon">
              <n-icon :size="28">
                <CloseCircleOutline />
              </n-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">发送失败</div>
              <div class="stat-value">{{ stats.failed }}</div>
            </div>
          </div>
        </n-card>
      </n-grid-item>
    </n-grid>

    <!-- Main Card -->
    <n-card title="邮件队列" :bordered="false" class="page-card">
      <n-space vertical :size="16">
        <!-- Status Filter Tabs -->
        <n-tabs v-model:value="statusFilter" type="line" @update:value="handleStatusChange">
          <n-tab-pane name="all" tab="全部" />
          <n-tab-pane name="pending" tab="待发送" />
          <n-tab-pane name="sent" tab="已发送" />
          <n-tab-pane name="failed" tab="发送失败" />
        </n-tabs>

        <!-- Data Table -->
        <n-data-table
          :columns="columns"
          :data="emails"
          :loading="loading"
          :pagination="false"
          :bordered="false"
          :single-line="false"
          :scroll-x="1200"
        />

        <!-- Pagination -->
        <n-pagination
          v-model:page="currentPage"
          v-model:page-size="pageSize"
          :page-count="totalPages"
          :page-sizes="[10, 20, 50, 100]"
          show-size-picker
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </n-space>
    </n-card>
  </div>
</template>

<script setup>
import { ref, h, onMounted, computed } from 'vue'
import { NButton, NTag, NSpace, NIcon, useMessage, useDialog } from 'naive-ui'
import {
  MailOutline,
  TimeOutline,
  CheckmarkCircleOutline,
  CloseCircleOutline,
  RefreshOutline,
  TrashOutline
} from '@vicons/ionicons5'
import { listEmailQueue, retryEmail, deleteEmail } from '@/api/admin'

const message = useMessage()
const dialog = useDialog()

// State
const loading = ref(false)
const emails = ref([])
const statusFilter = ref('all')
const currentPage = ref(1)
const pageSize = ref(20)
const totalPages = ref(0)
const totalCount = ref(0)

// Stats
const stats = computed(() => {
  return {
    total: totalCount.value,
    pending: emails.value.filter(e => e.status === 'pending').length,
    sent: emails.value.filter(e => e.status === 'sent').length,
    failed: emails.value.filter(e => e.status === 'failed').length
  }
})

// Status helpers
const getStatusType = (status) => {
  const typeMap = {
    pending: 'warning',
    sent: 'success',
    failed: 'error'
  }
  return typeMap[status] || 'default'
}

const getStatusText = (status) => {
  const textMap = {
    pending: '待发送',
    sent: '已发送',
    failed: '发送失败'
  }
  return textMap[status] || status
}

// Table columns
const columns = [
  { title: 'ID', key: 'id', width: 80, fixed: 'left' },
  {
    title: '收件人',
    key: 'to_email',
    width: 220,
    ellipsis: { tooltip: true }
  },
  {
    title: '主题',
    key: 'subject',
    width: 280,
    ellipsis: { tooltip: true }
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row) => h(
      NTag,
      { type: getStatusType(row.status), size: 'small' },
      { default: () => getStatusText(row.status) }
    )
  },
  {
    title: '重试次数',
    key: 'retry_count',
    width: 100,
    render: (row) => row.retry_count || 0
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 170,
    render: (row) => row.created_at ? new Date(row.created_at).toLocaleString('zh-CN') : '-'
  },
  {
    title: '发送时间',
    key: 'sent_at',
    width: 170,
    render: (row) => row.sent_at ? new Date(row.sent_at).toLocaleString('zh-CN') : '-'
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    fixed: 'right',
    render: (row) => h(
      NSpace,
      {},
      {
        default: () => [
          row.status === 'failed' && h(
            NButton,
            {
              size: 'small',
              type: 'warning',
              onClick: () => handleRetry(row)
            },
            {
              icon: () => h(NIcon, { component: RefreshOutline }),
              default: () => '重试'
            }
          ),
          h(
            NButton,
            {
              size: 'small',
              type: 'error',
              quaternary: true,
              onClick: () => handleDelete(row)
            },
            {
              icon: () => h(NIcon, { component: TrashOutline })
            }
          )
        ].filter(Boolean)
      }
    )
  }
]

// Fetch emails
const fetchEmails = async () => {
  loading.value = true
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value,
      status: statusFilter.value === 'all' ? undefined : statusFilter.value
    }
    const response = await listEmailQueue(params)
    emails.value = response.data.items || []
    totalCount.value = response.data.total || 0
    totalPages.value = Math.ceil(totalCount.value / pageSize.value)
  } catch (error) {
    message.error('获取邮件队列失败：' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// Handlers
const handleStatusChange = () => {
  currentPage.value = 1
  fetchEmails()
}

const handlePageChange = (page) => {
  currentPage.value = page
  fetchEmails()
}

const handlePageSizeChange = (size) => {
  pageSize.value = size
  currentPage.value = 1
  fetchEmails()
}

const handleRetry = (row) => {
  dialog.warning({
    title: '确认重试',
    content: `确定要重试发送邮件给 ${row.to_email} 吗？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await retryEmail(row.id)
        message.success('邮件已加入重试队列')
        fetchEmails()
      } catch (error) {
        message.error('重试失败：' + (error.message || '未知错误'))
      }
    }
  })
}

const handleDelete = (row) => {
  dialog.error({
    title: '确认删除',
    content: `确定要删除这封邮件记录吗？此操作不可恢复！`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteEmail(row.id)
        message.success('邮件记录已删除')
        fetchEmails()
      } catch (error) {
        message.error('删除失败：' + (error.message || '未知错误'))
      }
    }
  })
}

onMounted(() => {
  fetchEmails()
})
</script>

<style scoped>
.admin-email-queue-page {
  padding: 20px;
}

.stat-card {
  border-radius: 8px;
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
  opacity: 0.08;
  transition: opacity 0.3s ease;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.stat-card:hover::before {
  opacity: 0.12;
}

.stat-card-blue::before {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.stat-card-orange::before {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.stat-card-green::before {
  background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
}

.stat-card-red::before {
  background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 12px;
  position: relative;
  z-index: 1;
}

.stat-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.9);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.06);
}

.stat-card-blue .stat-icon {
  color: #667eea;
}

.stat-card-orange .stat-icon {
  color: #f5576c;
}

.stat-card-green .stat-icon {
  color: #11998e;
}

.stat-card-red .stat-icon {
  color: #fa709a;
}

.stat-info {
  flex: 1;
}

.stat-label {
  font-size: 13px;
  color: #666;
  margin-bottom: 4px;
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: #333;
}

.page-card {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

:deep(.n-data-table) {
  font-size: 14px;
}

:deep(.n-data-table .n-data-table-th) {
  font-weight: 600;
}

:deep(.n-tabs .n-tabs-tab) {
  font-weight: 500;
}

@media (max-width: 767px) {
  .admin-email-queue-page { padding: 8px; }
}
</style>
