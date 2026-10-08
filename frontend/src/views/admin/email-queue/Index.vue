<template>
  <div class="admin-email-queue-page admin-page-shell">
    <!-- Stats Cards -->
    <n-grid :cols="4" :x-gap="16" :y-gap="16" responsive="screen" :item-responsive="true" style="margin-bottom: 24px">
      <n-grid-item span="2 m:1">
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

      <n-grid-item span="2 m:1">
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

      <n-grid-item span="2 m:1">
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

      <n-grid-item span="2 m:1">
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
        <!-- Status Filter Tabs（手机端吸顶，滚动时不跟着走） -->
        <div class="queue-toolbar" :class="{ 'app-sticky-toolbar': appStore.isMobile }">
          <n-tabs v-model:value="statusFilter" type="line" @update:value="handleStatusChange">
            <n-tab-pane name="all" tab="全部" />
            <n-tab-pane name="pending" tab="待发送" />
            <n-tab-pane name="sent" tab="已发送" />
            <n-tab-pane name="failed" tab="发送失败" />
          </n-tabs>
        </div>

        <!-- 全选 / 批量操作：公共组件（桌面在表格上方，手机固定在底部标签栏上方） -->
        <BatchSelectBar
          :total="selection.total.value"
          :selected-count="selection.count.value"
          :all-selected="selection.allSelected.value"
          :indeterminate="selection.indeterminate.value"
          label="封邮件"
          @toggle-all="selection.toggleAll"
          @clear="selection.clear"
        >
          <n-button size="small" type="error" :disabled="!selection.count.value" @click="handleBatchDelete">批量删除</n-button>
        </BatchSelectBar>
        <template v-if="!appStore.isMobile">
          <n-data-table
            class="unified-admin-table"
            :columns="columns"
            :data="emails"
            :loading="loading"
            :pagination="false"
            :bordered="false"
            :single-line="false"
            :scroll-x="1200"
            :row-key="(row) => row.id"
            v-model:checked-row-keys="checkedRowKeys"
          />
        </template>

        <!-- Mobile Cards -->
        <template v-else>
          <div class="mobile-card-list">
            <div
              v-for="item in emails"
              :key="item.id"
              class="mobile-card is-selectable"
              :class="{ 'is-selected': selection.isSelected(item) }"
              @click="selection.toggle(item)"
            >
              <div class="card-check" @click.stop>
                <n-checkbox :checked="selection.isSelected(item)" @update:checked="() => selection.toggle(item)" />
              </div>
              <div class="card-header">
                <span class="card-title">{{ item.to_email }}</span>
                <n-tag :type="getStatusType(item.status)" size="small">{{ getStatusText(item.status) }}</n-tag>
              </div>
              <div class="card-body">
                <div class="card-row"><span class="card-label">主题:</span><span>{{ item.subject }}</span></div>
                <div class="card-row"><span class="card-label">重试次数:</span><span>{{ item.retry_count || 0 }}</span></div>
                <div class="card-row"><span class="card-label">创建时间:</span><span>{{ formatFullDateTime(item.created_at) }}</span></div>
                <div class="card-row"><span class="card-label">发送时间:</span><span>{{ formatFullDateTime(item.sent_at) }}</span></div>
              </div>
              <div class="card-actions" @click.stop>
                <n-button size="small" type="info" quaternary @click="handleDetail(item)">
                  <template #icon><n-icon :component="EyeOutline" /></template>
                  详情
                </n-button>
                <n-button v-if="item.status === 'failed'" size="small" type="warning" @click="handleRetry(item)">
                  <template #icon><n-icon :component="RefreshOutline" /></template>
                  重试
                </n-button>
                <n-button size="small" type="error" quaternary @click="handleDelete(item)">
                  <template #icon><n-icon :component="TrashOutline" /></template>
                  删除
                </n-button>
              </div>
            </div>
          </div>
        </template>

        <!-- Pagination -->
        <n-pagination
          v-model:page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :item-count="pagination.itemCount"
          :page-sizes="[10, 20, 50, 100]"
          show-size-picker
          style="margin-top: 16px; justify-content: flex-end"
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </n-space>
    </n-card>

    <!-- Detail Modal -->
    <n-modal v-model:show="showDetail" preset="card" title="邮件详情" style="width: 720px; max-width: 95vw;" :segmented="{ content: 'soft' }">
      <template v-if="detailItem">
        <n-descriptions :column="2" label-placement="left" bordered size="small">
          <n-descriptions-item label="ID">{{ detailItem.id }}</n-descriptions-item>
          <n-descriptions-item label="状态">
            <n-tag :type="getStatusType(detailItem.status)" size="small">{{ getStatusText(detailItem.status) }}</n-tag>
          </n-descriptions-item>
          <n-descriptions-item label="收件人" :span="2">{{ detailItem.to_email }}</n-descriptions-item>
          <n-descriptions-item label="主题" :span="2">{{ detailItem.subject }}</n-descriptions-item>
          <n-descriptions-item label="邮件类型">{{ translateEmailType(detailItem.email_type) || '-' }}</n-descriptions-item>
          <n-descriptions-item label="内容类型">{{ detailItem.content_type === 'html' ? 'HTML' : '纯文本' }}</n-descriptions-item>
          <n-descriptions-item label="重试次数">{{ detailItem.retry_count || 0 }} / {{ detailItem.max_retries || 3 }}</n-descriptions-item>
          <n-descriptions-item label="创建时间">{{ formatFullDateTime(detailItem.created_at) }}</n-descriptions-item>
          <n-descriptions-item label="发送时间">{{ formatFullDateTime(detailItem.sent_at) }}</n-descriptions-item>
          <n-descriptions-item label="更新时间">{{ formatFullDateTime(detailItem.updated_at) }}</n-descriptions-item>
          <n-descriptions-item v-if="detailItem.error_message" label="错误信息" :span="2">
            <n-text type="error" style="word-break: break-all;">{{ detailItem.error_message }}</n-text>
          </n-descriptions-item>
        </n-descriptions>
        <n-divider style="margin: 16px 0 12px;">邮件内容</n-divider>
        <div v-if="detailItem.content_type === 'html' || (detailItem.content && detailItem.content.includes('<'))" class="email-preview">
          <iframe
            :srcdoc="sanitizeHtml(detailItem.content)"
            style="width: 100%; min-height: 400px; border: 1px solid #ddd; border-radius: 4px;"
            sandbox=""
            title="邮件预览"
          />
        </div>
        <n-code v-else :code="detailItem.content || '(空)'" word-wrap style="max-height: 400px; overflow: auto;" />
      </template>
      <template #footer>
        <n-space justify="end">
          <n-button v-if="detailItem && detailItem.status === 'failed'" type="warning" @click="showDetail = false; handleRetry(detailItem)">重试发送</n-button>
          <n-button @click="showDetail = false">关闭</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, h, onActivated, onMounted, computed } from 'vue'
import { NButton, NTag, NSpace, NIcon, NText, useMessage, useDialog } from 'naive-ui'
import {
  MailOutline,
  TimeOutline,
  CheckmarkCircleOutline,
  CloseCircleOutline,
  RefreshOutline,
  TrashOutline,
  EyeOutline
} from '@vicons/ionicons5'
import { listEmailQueue, retryEmail, deleteEmail } from '@/api/admin'
import { useTable } from '@/composables/useTable'
import { useBatchSelection } from '@/composables/useBatchSelection'
import { useAppStore } from '@/stores/app'
import { translateEmailType } from '@/utils/i18n'
import { formatFullDateTime } from '@/utils/date'
import DOMPurify from 'dompurify'
import BatchSelectBar from '@/components/BatchSelectBar.vue'

const appStore = useAppStore()

const message = useMessage()
const dialog = useDialog()

// State
const statusFilter = ref('all')
const showDetail = ref(false)
const detailItem = ref(null)

// 统一表格状态（含状态筛选）
const { loading, tableData: emails, pagination, loadData, reload } = useTable(listEmailQueue, {
  getParams: () => ({ status: statusFilter.value === 'all' ? undefined : statusFilter.value }),
})
const fetchEmails = loadData
// 全选 / 多选：全站统一实现（桌面表格与手机卡片共用同一份选择状态）
const selection = useBatchSelection(() => emails.value)
// Naive 表格要的是数组，这里做一层桥接，保证两边状态一致
const checkedRowKeys = computed({
  get: () => [...selection.selectedKeys.value],
  set: (keys) => { selection.selectedKeys.value = new Set(keys) },
})

// 使用 DOMPurify 安全清理 HTML，防止 XSS
const sanitizeHtml = (html) => {
  if (!html) return ''
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: ['h1','h2','h3','h4','h5','h6','p','br','hr','div','span','a','img',
      'table','thead','tbody','tr','td','th','ul','ol','li','strong','em','b','i','u',
      'blockquote','pre','code','center','font','small','big','sub','sup','style'],
    ALLOWED_ATTR: ['href','src','alt','title','style','class','width','height','align',
      'valign','bgcolor','color','border','cellpadding','cellspacing','colspan','rowspan',
      'target','rel'],
    ALLOW_DATA_ATTR: false,
    ADD_ATTR: ['target'],
  })
}

const handleDetail = (row) => {
  detailItem.value = row
  showDetail.value = true
}

// Stats：优先使用后端全队列统计（列表接口附带 stats），回退当前页估算
const queueStats = ref(null)
const stats = computed(() => {
  if (queueStats.value) {
    return {
      total: queueStats.value.pending + queueStats.value.sent + queueStats.value.failed,
      pending: queueStats.value.pending,
      sent: queueStats.value.sent,
      failed: queueStats.value.failed,
    }
  }
  return {
    total: pagination.itemCount,
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
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 80, fixed: 'left', resizable: true, sorter: 'default' },
  {
    title: '收件人',
    key: 'to_email',
    width: 220,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '主题',
    key: 'subject',
    width: 280,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    resizable: true,
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
    resizable: true,
    render: (row) => row.retry_count || 0
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 170,
    resizable: true,
    sorter: (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
    render: (row) => formatFullDateTime(row.created_at)
  },
  {
    title: '发送时间',
    key: 'sent_at',
    width: 170,
    resizable: true,
    sorter: (a, b) => new Date(a.sent_at || 0).getTime() - new Date(b.sent_at || 0).getTime(),
    render: (row) => formatFullDateTime(row.sent_at)
  },
  {
    title: '操作',
    key: 'actions',
    width: 200,
    fixed: 'right',
    render: (row) => h(
      NSpace,
      { size: 4 },
      {
        default: () => [
          h(
            NButton,
            {
              size: 'small',
              type: 'info',
              quaternary: true,
              onClick: () => handleDetail(row)
            },
            {
              icon: () => h(NIcon, { component: EyeOutline }),
              default: () => '详情'
            }
          ),
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

// Handlers
const handleStatusChange = () => {
  reload()
}

const handlePageChange = (page) => {
  pagination.page = page
  fetchEmails()
}

const handlePageSizeChange = (size) => {
  pagination.pageSize = size
  pagination.page = 1
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

const handleBatchDelete = () => {
  const rows = selection.selectedRows.value
  if (!rows.length) return
  dialog.warning({
    title: '批量删除',
    content: `确定要删除选中的 ${rows.length} 封邮件记录吗？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await Promise.all(rows.map(row => deleteEmail(row.id)))
        message.success('批量删除成功')
        selection.clear()
        fetchEmails()
      } catch { message.error('批量删除失败') }
    }
  })
}

onMounted(() => {
  fetchEmails()
  // 单独获取全队列统计（列表接口附带 stats 字段）
  listEmailQueue({ page: 1, page_size: 1 }) .then((res) => {
    if (res?.data?.stats) queueStats.value = res.data.stats
  }).catch(() => {})
})

// KeepAlive 缓存激活时静默刷新数据（不清 loading 遮罩、不重置分页）
onActivated(() => {
  fetchEmails()
})
</script>

<style scoped>
.stat-card-blue::before {
  background: var(--brand-gradient);
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
  color: var(--primary-color);
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
  color: var(--text-color-secondary);
  margin-bottom: 4px;
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-color);
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

/* 状态筛选吸顶：全局 .app-sticky-toolbar 用负边距对齐卡片内边距，
   而带 .mobile-card-list 的卡片内容区左右内边距是 0，负边距会把工具条撑出卡片。 */
.queue-toolbar.app-sticky-toolbar {
  margin-left: 0;
  margin-right: 0;
}


@media (max-width: 767px) {
  /* 手机端左右留白由全局布局统一给（10px），页面根容器不再自带左右 padding；
     卡片样式统一走全局 .mobile-card（App 风格），页面不再覆盖。 */
  .admin-email-queue-page { padding: 8px 0; }
}

.email-preview {
  max-height: 400px;
  overflow: auto;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 16px;
  background: var(--bg-color);
}
</style>
