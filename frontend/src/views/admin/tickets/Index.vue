<template>
  <div class="tickets-container">
    <n-card title="工单管理">
      <n-space vertical :size="16">
        <n-space>
          <n-select
            v-model:value="filters.status"
            placeholder="状态筛选"
            clearable
            style="width: 150px"
            :options="statusOptions"
            @update:value="handleSearch"
          />
          <n-select
            v-model:value="filters.priority"
            placeholder="优先级筛选"
            clearable
            style="width: 150px"
            :options="priorityOptions"
            @update:value="handleSearch"
          />
        </n-space>

        <template v-if="!appStore.isMobile">
          <n-data-table
            remote
            :columns="columns"
            :data="tickets"
            :loading="loading"
            :pagination="pagination"
            :bordered="false"
            @update:page="(p: number) => { pagination.page = p; loadTickets() }"
            @update:page-size="(ps: number) => { pagination.pageSize = ps; pagination.page = 1; loadTickets() }"
          />
        </template>

        <template v-else>
          <n-spin :show="loading">
            <div v-if="tickets.length === 0" style="text-align: center; padding: 40px 0; color: #999;">
              暂无数据
            </div>
            <div v-else class="mobile-card-list">
              <div v-for="ticket in tickets" :key="ticket.id" class="mobile-card">
                <div class="card-header">
                  <div class="card-title">{{ ticket.title }}</div>
                  <n-tag :type="getStatusTagType(ticket.status)" size="small">
                    {{ getStatusText(ticket.status) }}
                  </n-tag>
                </div>
                <div class="card-body">
                  <div class="card-row">
                    <span class="card-label">用户ID</span>
                    <span>{{ ticket.user_id }}</span>
                  </div>
                  <div class="card-row">
                    <span class="card-label">优先级</span>
                    <n-tag :type="getPriorityTagType(ticket.priority)" size="small">
                      {{ getPriorityText(ticket.priority) }}
                    </n-tag>
                  </div>
                  <div class="card-row">
                    <span class="card-label">创建时间</span>
                    <span>{{ formatDate(ticket.created_at) }}</span>
                  </div>
                </div>
                <div class="card-actions">
                  <n-button size="small" type="primary" @click="handleViewDetail(ticket.id)">查看详情</n-button>
                </div>
              </div>
            </div>
          </n-spin>
        </template>
      </n-space>
    </n-card>

    <n-modal
      v-model:show="showDetailModal"
      preset="card"
      title="工单详情"
      :style="appStore.isMobile ? 'width: 95%; max-width: 800px' : 'width: 800px'"
      :segmented="{ content: 'soft' }"
    >
      <n-spin :show="detailLoading">
        <div v-if="currentTicket" class="ticket-detail">
          <n-descriptions :column="2" bordered>
            <n-descriptions-item label="工单编号">
              {{ currentTicket.ticket_no }}
            </n-descriptions-item>
            <n-descriptions-item label="用户ID">
              {{ currentTicket.user_id }}
            </n-descriptions-item>
            <n-descriptions-item label="标题">
              {{ currentTicket.title }}
            </n-descriptions-item>
            <n-descriptions-item label="类型">
              <n-tag :type="getTypeTagType(currentTicket.type)">
                {{ getTypeText(currentTicket.type) }}
              </n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="优先级">
              <n-tag :type="getPriorityTagType(currentTicket.priority)">
                {{ getPriorityText(currentTicket.priority) }}
              </n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="状态">
              <n-tag :type="getStatusTagType(currentTicket.status)">
                {{ getStatusText(currentTicket.status) }}
              </n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="创建时间" :span="2">
              {{ formatDate(currentTicket.created_at) }}
            </n-descriptions-item>
          </n-descriptions>

          <n-divider>对话记录</n-divider>

          <div class="chat-container">
            <div
              v-for="reply in currentTicket.replies"
              :key="reply.id"
              :class="['chat-message', reply.is_admin ? 'admin' : 'user']"
            >
              <div class="message-header">
                <n-tag :type="reply.is_admin ? 'success' : 'info'" size="small">
                  {{ reply.is_admin ? '管理员' : '用户' }}
                </n-tag>
                <span class="message-time">{{ formatDate(reply.created_at) }}</span>
              </div>
              <div class="message-content">{{ reply.content }}</div>
            </div>
          </div>

          <n-divider>回复工单</n-divider>

          <n-space vertical :size="12">
            <n-input
              v-model:value="replyContent"
              type="textarea"
              placeholder="输入回复内容..."
              :rows="4"
            />
            <n-space>
              <n-select
                v-model:value="updateStatus"
                placeholder="更新状态"
                style="width: 150px"
                :options="statusOptions"
              />
              <n-button type="primary" @click="handleReply" :loading="replyLoading">
                发送回复
              </n-button>
            </n-space>
          </n-space>
        </div>
      </n-spin>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, h, onMounted } from 'vue'
import { NButton, NTag, NSpace, NSpin, useMessage, useDialog } from 'naive-ui'
import { listAdminTickets, getAdminTicket, updateTicket, replyAdminTicket } from '@/api/admin'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const detailLoading = ref(false)
const replyLoading = ref(false)
const tickets = ref<any[]>([])
const currentTicket = ref<any>(null)
const showDetailModal = ref(false)
const replyContent = ref('')
const updateStatus = ref('')

const filters = reactive({
  status: null,
  priority: null,
  page: 1,
  page_size: 10
})

const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  onChange: (page: number) => {
    filters.page = page
    loadTickets()
  },
  onUpdatePageSize: (pageSize: number) => {
    filters.page_size = pageSize
    filters.page = 1
    loadTickets()
  }
})

const statusOptions = [
  { label: '待处理', value: 'pending' },
  { label: '处理中', value: 'processing' },
  { label: '已解决', value: 'resolved' },
  { label: '已关闭', value: 'closed' }
]

const priorityOptions = [
  { label: '低', value: 'low' },
  { label: '中', value: 'normal' },
  { label: '高', value: 'high' },
  { label: '紧急', value: 'urgent' }
]

const columns = [
  { title: 'ID', key: 'id', width: 60, resizable: true, sorter: 'default' },
  { title: '工单编号', key: 'ticket_no', width: 150, resizable: true },
  { title: '用户ID', key: 'user_id', width: 80, resizable: true },
  { title: '标题', key: 'title', ellipsis: { tooltip: true } },
  {
    title: '类型',
    key: 'type',
    width: 100,
    resizable: true,
    render: (row: any) => h(NTag, { type: getTypeTagType(row.type) }, { default: () => getTypeText(row.type) })
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    resizable: true,
    render: (row: any) => h(NTag, { type: getStatusTagType(row.status) }, { default: () => getStatusText(row.status) })
  },
  {
    title: '优先级',
    key: 'priority',
    width: 100,
    resizable: true,
    render: (row: any) => h(NTag, { type: getPriorityTagType(row.priority) }, { default: () => getPriorityText(row.priority) })
  },
  { title: '分配给', key: 'assigned_to', width: 100, resizable: true },
  {
    title: '创建时间',
    key: 'created_at',
    width: 160,
    resizable: true,
    sorter: (a: any, b: any) => new Date(a.created_at || 0).getTime() - new Date(b.created_at || 0).getTime(),
    render: (row: any) => formatDate(row.created_at)
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    fixed: 'right' as const,
    render: (row: any) => {
      return h(NSpace, {}, {
        default: () => [
          h(NButton, { size: 'small', onClick: () => handleViewDetail(row.id) }, { default: () => '查看/回复' }),
          h(NButton, {
            size: 'small',
            type: 'primary',
            onClick: () => handleQuickStatusUpdate(row)
          }, { default: () => '更新状态' })
        ]
      })
    }
  }
]

const getTypeText = (type: string) => {
  const map: Record<string, string> = {
    technical: '技术支持',
    billing: '账单问题',
    account: '账户问题',
    other: '其他'
  }
  return map[type] || type
}

const getTypeTagType = (type: string) => {
  const map: Record<string, any> = {
    technical: 'info',
    billing: 'warning',
    account: 'default',
    other: 'error'
  }
  return map[type] || 'default'
}

const getStatusText = (status: string) => {
  const map: Record<string, string> = {
    pending: '待处理',
    processing: '处理中',
    resolved: '已解决',
    closed: '已关闭'
  }
  return map[status] || status
}

const getStatusTagType = (status: string) => {
  const map: Record<string, any> = {
    pending: 'warning',
    processing: 'info',
    resolved: 'success',
    closed: 'default'
  }
  return map[status] || 'default'
}

const getPriorityText = (priority: string) => {
  const map: Record<string, string> = {
    low: '低',
    normal: '中',
    high: '高',
    urgent: '紧急'
  }
  return map[priority] || priority
}

const getPriorityTagType = (priority: string) => {
  const map: Record<string, any> = {
    low: 'default',
    normal: 'info',
    high: 'warning',
    urgent: 'error'
  }
  return map[priority] || 'default'
}

const formatDate = (date: string) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
}

const loadTickets = async () => {
  loading.value = true
  try {
    const params: any = { ...filters }
    if (!params.status) delete params.status
    if (!params.priority) delete params.priority
    
    const res = await listAdminTickets(params)
    tickets.value = res.data.items || []
    pagination.itemCount = res.data.total || 0
  } catch (error: any) {
    message.error(error.message || '加载工单列表失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  filters.page = 1
  loadTickets()
}

const handleViewDetail = async (id: number) => {
  detailLoading.value = true
  showDetailModal.value = true
  try {
    const res = await getAdminTicket(id)
    currentTicket.value = { ...res.data.ticket, replies: res.data.replies || [] }
    updateStatus.value = res.data.ticket.status
  } catch (error: any) {
    message.error(error.message || '加载工单详情失败')
    showDetailModal.value = false
  } finally {
    detailLoading.value = false
  }
}

const handleReply = async () => {
  if (!replyContent.value.trim()) {
    message.warning('请输入回复内容')
    return
  }

  replyLoading.value = true
  try {
    await replyAdminTicket(currentTicket.value.id, {
      content: replyContent.value,
    })
    message.success('回复成功')
    replyContent.value = ''
    await handleViewDetail(currentTicket.value.id)
    await loadTickets()
  } catch (error: any) {
    message.error(error.message || '回复失败')
  } finally {
    replyLoading.value = false
  }
}

const handleQuickStatusUpdate = (row: any) => {
  dialog.create({
    title: '更新工单状态',
    content: () => {
      const status = ref(row.status)
      return h(NSpace, { vertical: true }, {
        default: () => [
          h('div', {}, `工单编号: ${row.ticket_no}`),
          h('div', {}, `当前状态: ${getStatusText(row.status)}`),
          h('div', { style: { marginTop: '12px' } }, '选择新状态:'),
          h('select', {
            value: status.value,
            onChange: (e: any) => { status.value = e.target.value },
            style: { width: '100%', padding: '8px', borderRadius: '4px', border: '1px solid #e0e0e0' }
          }, statusOptions.map(opt => h('option', { value: opt.value }, opt.label)))
        ]
      })
    },
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await updateTicket(row.id, { status: row.status })
        message.success('状态更新成功')
        await loadTickets()
      } catch (error: any) {
        message.error(error.message || '状态更新失败')
      }
    }
  })
}

onMounted(() => {
  loadTickets()
})
</script>

<style scoped>
.tickets-container {
  padding: 20px;
}

.ticket-detail {
  max-height: 70vh;
  overflow-y: auto;
}

.chat-container {
  max-height: 400px;
  overflow-y: auto;
  padding: 16px;
  background: #f5f5f5;
  border-radius: 8px;
}

.chat-message {
  margin-bottom: 16px;
  padding: 12px;
  border-radius: 8px;
  background: white;
}

.chat-message.admin {
  border-left: 3px solid #18a058;
}

.chat-message.user {
  border-left: 3px solid #2080f0;
}

.message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.message-time {
  font-size: 12px;
  color: #999;
}

.message-content {
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.mobile-card-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.mobile-card {
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.08);
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-bottom: 1px solid #f0f0f0;
}

.card-title {
  font-weight: 600;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  margin-right: 8px;
}

.card-body {
  padding: 10px 14px;
}

.card-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
  font-size: 13px;
}

.card-label {
  color: #999;
}

.card-actions {
  display: flex;
  gap: 8px;
  padding: 10px 14px;
  border-top: 1px solid #f0f0f0;
  flex-wrap: wrap;
}

@media (max-width: 767px) {
  .tickets-container { padding: 8px; }
}
</style>
