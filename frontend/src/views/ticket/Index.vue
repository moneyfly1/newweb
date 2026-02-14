<template>
  <div class="ticket-container">
    <n-card :bordered="false" class="header-card">
      <div class="header">
        <h2>我的工单</h2>
        <n-button type="primary" @click="showCreateModal = true">
          <template #icon>
            <n-icon><AddOutline /></n-icon>
          </template>
          新建工单
        </n-button>
      </div>
    </n-card>

    <n-card :bordered="false" class="table-card">
      <!-- Desktop table -->
      <n-data-table v-if="!appStore.isMobile"
        :columns="columns"
        :data="tickets"
        :loading="loading"
        :pagination="pagination"
        :bordered="false"
      />
      <!-- Mobile card list -->
      <div v-else>
        <n-spin :show="loading">
          <div v-if="!loading && tickets.length === 0" class="mobile-empty">暂无工单</div>
          <div v-for="ticket in tickets" :key="ticket.id" class="mobile-card">
            <div class="card-header-row">
              <span class="card-title">{{ ticket.title }}</span>
              <n-tag :type="{ pending: 'warning', processing: 'info', resolved: 'success', closed: 'default' }[ticket.status] || 'default'" size="small">
                {{ { pending: '待处理', processing: '处理中', resolved: '已解决', closed: '已关闭' }[ticket.status] || ticket.status }}
              </n-tag>
            </div>
            <div class="card-row">
              <span class="label">工单编号</span>
              <span class="value">{{ ticket.ticket_no }}</span>
            </div>
            <div class="card-row">
              <span class="label">类型</span>
              <span class="value">{{ getTypeText(ticket.type) }}</span>
            </div>
            <div class="card-row">
              <span class="label">创建时间</span>
              <span class="value">{{ ticket.created_at }}</span>
            </div>
            <div class="card-actions">
              <n-button type="primary" @click="router.push('/tickets/' + ticket.id)">查看详情</n-button>
            </div>
          </div>
        </n-spin>
        <n-pagination
          v-if="tickets.length > 0"
          v-model:page="pagination.page"
          :item-count="pagination.itemCount"
          :page-size="pagination.pageSize"
          style="margin-top: 16px; justify-content: center;"
          @update:page="(p) => { pagination.page = p; loadTickets() }"
        />
      </div>
    </n-card>

    <n-modal
      v-model:show="showCreateModal"
      preset="card"
      title="新建工单"
      style="width: 600px; border-radius: 12px"
      :mask-closable="false"
    >
      <n-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-placement="top"
      >
        <n-form-item label="工单标题" path="title">
          <n-input
            v-model:value="formData.title"
            placeholder="请输入工单标题"
            maxlength="100"
            show-count
          />
        </n-form-item>
        <n-form-item label="工单类型" path="type">
          <n-select
            v-model:value="formData.type"
            :options="typeOptions"
            placeholder="请选择工单类型"
          />
        </n-form-item>
        <n-form-item label="问题描述" path="content">
          <n-input
            v-model:value="formData.content"
            type="textarea"
            placeholder="请详细描述您遇到的问题"
            :rows="6"
            maxlength="2000"
            show-count
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="modal-footer">
          <n-button @click="showCreateModal = false">取消</n-button>
          <n-button type="primary" @click="handleCreate" :loading="submitting">
            提交工单
          </n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, reactive, h, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NTag, NIcon, useMessage } from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import { listTickets, createTicket } from '@/api/ticket'
import { useAppStore } from '@/stores/app'

const router = useRouter()
const message = useMessage()
const appStore = useAppStore()

const loading = ref(false)
const submitting = ref(false)
const showCreateModal = ref(false)
const tickets = ref([])
const formRef = ref(null)

const formData = reactive({
  title: '',
  type: '',
  content: ''
})

const typeOptions = [
  { label: '技术问题', value: 'technical' },
  { label: '账单问题', value: 'billing' },
  { label: '账户问题', value: 'account' },
  { label: '其他问题', value: 'other' }
]

const rules = {
  title: [
    { required: true, message: '请输入工单标题', trigger: 'blur' }
  ],
  type: [
    { required: true, message: '请选择工单类型', trigger: 'change' }
  ],
  content: [
    { required: true, message: '请输入问题描述', trigger: 'blur' }
  ]
}

const pagination = reactive({
  page: 1,
  pageSize: 10,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  onChange: (page) => {
    pagination.page = page
    loadTickets()
  },
  onUpdatePageSize: (pageSize) => {
    pagination.pageSize = pageSize
    pagination.page = 1
    loadTickets()
  }
})

const getStatusTag = (status) => {
  const statusMap = {
    pending: { type: 'warning', text: '待处理' },
    processing: { type: 'info', text: '处理中' },
    resolved: { type: 'success', text: '已解决' },
    closed: { type: 'default', text: '已关闭' }
  }
  const config = statusMap[status] || { type: 'default', text: status }
  return h(NTag, { type: config.type }, { default: () => config.text })
}

const getPriorityTag = (priority) => {
  const priorityMap = {
    low: { type: 'default', text: '低' },
    normal: { type: 'info', text: '普通' },
    high: { type: 'warning', text: '高' },
    urgent: { type: 'error', text: '紧急' }
  }
  const config = priorityMap[priority] || { type: 'default', text: priority }
  return h(NTag, { type: config.type }, { default: () => config.text })
}

const getTypeText = (type) => {
  const typeMap = {
    technical: '技术问题',
    billing: '账单问题',
    account: '账户问题',
    other: '其他问题'
  }
  return typeMap[type] || type
}

const columns = [
  {
    title: '工单编号',
    key: 'ticket_no',
    width: 150
  },
  {
    title: '标题',
    key: 'title',
    ellipsis: {
      tooltip: true
    }
  },
  {
    title: '类型',
    key: 'type',
    width: 120,
    render: (row) => h(NTag, {}, { default: () => getTypeText(row.type) })
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row) => getStatusTag(row.status)
  },
  {
    title: '优先级',
    key: 'priority',
    width: 100,
    render: (row) => getPriorityTag(row.priority)
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 180
  },
  {
    title: '操作',
    key: 'actions',
    width: 100,
    render: (row) => {
      return h(
        NButton,
        {
          text: true,
          type: 'primary',
          onClick: () => router.push(`/tickets/${row.id}`)
        },
        { default: () => '查看' }
      )
    }
  }
]

const loadTickets = async () => {
  loading.value = true
  try {
    const res = await listTickets({
      page: pagination.page,
      page_size: pagination.pageSize
    })
    tickets.value = res.data.items || []
    pagination.itemCount = res.data.total || 0
  } catch (error) {
    message.error(error.message || '加载工单列表失败')
  } finally {
    loading.value = false
  }
}

const handleCreate = async () => {
  try {
    await formRef.value?.validate()
    submitting.value = true
    await createTicket(formData)
    message.success('工单创建成功')
    showCreateModal.value = false
    Object.assign(formData, { title: '', type: '', content: '' })
    loadTickets()
  } catch (error) {
    if (error.message) {
      message.error(error.message || '创建工单失败')
    }
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  loadTickets()
})
</script>

<style scoped>
.ticket-container {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.header-card {
  margin-bottom: 20px;
  border-radius: 12px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
}

.table-card {
  border-radius: 12px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

@media (max-width: 767px) {
  .ticket-container { padding: 0; }
}
</style>
