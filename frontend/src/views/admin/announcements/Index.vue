<template>
  <div class="announcements-container">
    <n-card :title="appStore.isMobile ? undefined : '公告管理'" :bordered="false">
      <template v-if="!appStore.isMobile" #header-extra>
        <n-button type="primary" @click="handleCreate">
          发布公告
        </n-button>
      </template>

      <div v-if="appStore.isMobile" class="mobile-toolbar">
        <div class="mobile-toolbar-title">公告管理</div>
        <n-button size="small" type="primary" @click="handleCreate">发布公告</n-button>
      </div>

      <template v-if="!appStore.isMobile">
        <n-data-table
          remote
          :columns="columns"
          :data="tableData"
          :loading="loading"
          :pagination="pagination"
          :bordered="false"
          @update:page="(p) => { pagination.page = p; fetchData() }"
          @update:page-size="(ps) => { pagination.pageSize = ps; pagination.page = 1; fetchData() }"
        />
      </template>

      <template v-else>
        <n-spin :show="loading">
          <div v-if="tableData.length === 0" style="text-align: center; padding: 40px 0; color: #999;">
            暂无数据
          </div>
          <div v-else class="mobile-card-list">
            <div v-for="item in tableData" :key="item.id" class="mobile-card">
              <div class="card-header">
                <div class="card-title">{{ item.title }}</div>
                <n-tag :type="item.is_active ? 'success' : 'default'" size="small">
                  {{ item.is_active ? '启用' : '禁用' }}
                </n-tag>
              </div>
              <div class="card-body">
                <div class="card-row">
                  <span class="card-label">类型</span>
                  <component :is="getTypeTag(item.type)" />
                </div>
                <div class="card-row">
                  <span class="card-label">创建时间</span>
                  <span>{{ item.created_at }}</span>
                </div>
              </div>
              <div class="card-actions">
                <n-button size="small" type="primary" @click="handleEdit(item)">编辑</n-button>
                <n-popconfirm @positive-click="handleDelete(item.id)">
                  <template #trigger>
                    <n-button size="small" type="error">删除</n-button>
                  </template>
                  确定删除此公告吗？
                </n-popconfirm>
              </div>
            </div>
          </div>
        </n-spin>
      </template>
    </n-card>

    <n-modal
      v-model:show="showModal"
      :title="modalTitle"
      preset="card"
      :style="appStore.isMobile ? 'width: 95%; max-width: 600px' : 'width: 600px'"
      :mask-closable="false"
    >
      <n-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-placement="top"
      >
        <n-form-item label="标题" path="title">
          <n-input v-model:value="formData.title" placeholder="请输入公告标题" />
        </n-form-item>

        <n-form-item label="内容" path="content">
          <n-input
            v-model:value="formData.content"
            type="textarea"
            placeholder="请输入公告内容"
            :rows="6"
          />
        </n-form-item>

        <n-form-item label="类型" path="type">
          <n-select
            v-model:value="formData.type"
            :options="typeOptions"
            placeholder="请选择公告类型"
          />
        </n-form-item>

        <n-form-item label="状态" path="is_active">
          <n-switch v-model:value="formData.is_active">
            <template #checked>启用</template>
            <template #unchecked>禁用</template>
          </n-switch>
        </n-form-item>
      </n-form>

      <template #footer>
        <n-space justify="end">
          <n-button @click="showModal = false">取消</n-button>
          <n-button type="primary" @click="handleSubmit" :loading="submitting">
            确定
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="tsx">
import { ref, reactive, h, onMounted } from 'vue'
import {
  NCard,
  NButton,
  NDataTable,
  NModal,
  NForm,
  NFormItem,
  NInput,
  NSelect,
  NSwitch,
  NSpace,
  NTag,
  NPopconfirm,
  NSpin,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { listAnnouncements, createAnnouncement, updateAnnouncement, deleteAnnouncement } from '@/api/admin'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const message = useMessage()
const formRef = ref()
const loading = ref(false)
const submitting = ref(false)
const showModal = ref(false)
const modalTitle = ref('发布公告')
const tableData = ref<any[]>([])
const isEdit = ref(false)

const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  onChange: (page: number) => {
    pagination.page = page
    loadData()
  },
  onUpdatePageSize: (pageSize: number) => {
    pagination.pageSize = pageSize
    pagination.page = 1
    loadData()
  },
})

const formData = ref({
  id: 0,
  title: '',
  content: '',
  type: 'info',
  is_active: true,
})

const rules = {
  title: { required: true, message: '请输入标题', trigger: 'blur' },
  content: { required: true, message: '请输入内容', trigger: 'blur' },
  type: { required: true, message: '请选择类型', trigger: 'change' },
}

const typeOptions = [
  { label: '信息', value: 'info' },
  { label: '警告', value: 'warning' },
  { label: '成功', value: 'success' },
]

const getTypeTag = (type: string) => {
  const typeMap: Record<string, any> = {
    info: { type: 'info', text: '信息' },
    warning: { type: 'warning', text: '警告' },
    success: { type: 'success', text: '成功' },
  }
  const config = typeMap[type] || typeMap.info
  return h(NTag, { type: config.type }, { default: () => config.text })
}

const columns: DataTableColumns = [
  { title: 'ID', key: 'id', width: 80, resizable: true, sorter: 'default' },
  { title: '标题', key: 'title', ellipsis: { tooltip: true } },
  {
    title: '类型',
    key: 'type',
    width: 100,
    resizable: true,
    render: (row: any) => getTypeTag(row.type),
  },
  {
    title: '状态',
    key: 'is_active',
    width: 100,
    resizable: true,
    render: (row: any) =>
      h(NTag, { type: row.is_active ? 'success' : 'default' }, { default: () => (row.is_active ? '启用' : '禁用') }),
  },
  { title: '创建时间', key: 'created_at', width: 180, resizable: true, sorter: (a: any, b: any) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime() },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    render: (row: any) =>
      h(NSpace, {}, () => [
        h(
          NButton,
          { size: 'small', onClick: () => handleEdit(row) },
          { default: () => '编辑' }
        ),
        h(
          NPopconfirm,
          {
            onPositiveClick: () => handleDelete(row.id),
          },
          {
            default: () => '确定删除此公告吗？',
            trigger: () =>
              h(NButton, { size: 'small', type: 'error' }, { default: () => '删除' }),
          }
        ),
      ]),
  },
]

const loadData = async () => {
  loading.value = true
  try {
    const res = await listAnnouncements({
      page: pagination.page,
      page_size: pagination.pageSize,
    })
    tableData.value = res.data?.items || []
    pagination.itemCount = res.data?.total || 0
  } catch (error: any) {
    message.error(error.message || '加载失败')
  } finally {
    loading.value = false
  }
}

const fetchData = loadData

const handleCreate = () => {
  isEdit.value = false
  modalTitle.value = '发布公告'
  formData.value = {
    id: 0,
    title: '',
    content: '',
    type: 'info',
    is_active: true,
  }
  showModal.value = true
}

const handleEdit = (row: any) => {
  isEdit.value = true
  modalTitle.value = '编辑公告'
  formData.value = { ...row }
  showModal.value = true
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  submitting.value = true
  try {
    let res: any
    if (isEdit.value) {
      res = await updateAnnouncement(formData.value.id, formData.value)
    } else {
      res = await createAnnouncement(formData.value)
    }
    message.success(isEdit.value ? '更新成功' : '创建成功')
    showModal.value = false
    loadData()
  } catch (error: any) {
    message.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (id: number) => {
  try {
    const res = await deleteAnnouncement(id)
    message.success('删除成功')
    loadData()
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.announcements-container {
  padding: 20px;
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
  .announcements-container { padding: 8px; }
}
.mobile-toolbar { margin-bottom: 12px; }
.mobile-toolbar-title { font-size: 17px; font-weight: 600; margin-bottom: 10px; color: var(--text-color, #333); }
</style>
