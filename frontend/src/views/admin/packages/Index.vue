<template>
  <div class="admin-packages-page">
    <n-card title="套餐管理" :bordered="false" class="page-card">
      <template #header-extra>
        <n-button type="primary" @click="handleCreate">
          <template #icon>
            <n-icon :component="AddOutline" />
          </template>
          新建套餐
        </n-button>
      </template>

      <n-space vertical :size="16">
        <template v-if="!appStore.isMobile">
          <n-data-table
            :columns="columns"
            :data="packages"
            :loading="loading"
            :pagination="false"
            :bordered="false"
            :single-line="false"
          />
        </template>

        <template v-else>
          <n-spin :show="loading">
            <div v-if="packages.length === 0" style="text-align: center; padding: 40px 0; color: #999;">
              暂无数据
            </div>
            <div v-else class="mobile-card-list">
              <div v-for="pkg in packages" :key="pkg.id" class="mobile-card">
                <div class="card-header">
                  <div class="card-title">{{ pkg.name }}</div>
                  <n-tag :type="pkg.is_active ? 'success' : 'default'" size="small">
                    {{ pkg.is_active ? '启用' : '禁用' }}
                  </n-tag>
                </div>
                <div class="card-body">
                  <div class="card-row">
                    <span class="card-label">价格</span>
                    <span style="color: #18a058; font-weight: 600;">¥{{ pkg.price.toFixed(2) }}</span>
                  </div>
                  <div class="card-row">
                    <span class="card-label">有效期</span>
                    <span>{{ pkg.duration_days }} 天</span>
                  </div>
                  <div class="card-row">
                    <span class="card-label">流量</span>
                    <span>{{ pkg.traffic_limit && pkg.traffic_limit > 0 ? (pkg.traffic_limit / (1024 * 1024 * 1024)).toFixed(0) + ' GB' : '不限' }}</span>
                  </div>
                </div>
                <div class="card-actions">
                  <n-button size="small" type="primary" @click="handleEdit(pkg)">编辑</n-button>
                  <n-button size="small" type="error" @click="handleDelete(pkg)">删除</n-button>
                </div>
              </div>
            </div>
          </n-spin>
        </template>

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

    <n-modal
      v-model:show="showEditModal"
      preset="dialog"
      :title="isCreating ? '新建套餐' : '编辑套餐'"
      :positive-text="'保存'"
      :negative-text="'取消'"
      :style="appStore.isMobile ? 'width: 95%; max-width: 600px' : 'width: 600px'"
      @positive-click="handleSavePackage"
    >
      <n-form
        ref="formRef"
        :model="editForm"
        :rules="formRules"
        label-placement="left"
        label-width="120"
        style="margin-top: 20px"
      >
        <n-form-item label="套餐名称" path="name">
          <n-input v-model:value="editForm.name" placeholder="请输入套餐名称" />
        </n-form-item>
        <n-form-item label="套餐描述" path="description">
          <n-input
            v-model:value="editForm.description"
            type="textarea"
            placeholder="请输入套餐描述"
            :rows="3"
          />
        </n-form-item>
        <n-form-item label="价格（元）" path="price">
          <n-input-number
            v-model:value="editForm.price"
            placeholder="请输入价格"
            :min="0"
            :precision="2"
            style="width: 100%"
          >
            <template #prefix>¥</template>
          </n-input-number>
        </n-form-item>
        <n-form-item label="有效期（天）" path="duration_days">
          <n-input-number
            v-model:value="editForm.duration_days"
            placeholder="请输入有效期天数"
            :min="1"
            style="width: 100%"
          />
        </n-form-item>
        <n-form-item label="设备数量限制" path="device_limit">
          <n-input-number
            v-model:value="editForm.device_limit"
            placeholder="请输入设备数量限制"
            :min="1"
            style="width: 100%"
          />
        </n-form-item>
        <n-form-item label="排序顺序" path="sort_order">
          <n-input-number
            v-model:value="editForm.sort_order"
            placeholder="数字越小越靠前"
            :min="0"
            style="width: 100%"
          />
        </n-form-item>
        <n-form-item label="流量限制" path="traffic_limit">
          <n-input-number
            v-model:value="editForm.traffic_limit"
            placeholder="0 表示不限制，单位：字节"
            :min="0"
            style="width: 100%"
          >
            <template #suffix>字节（0=不限）</template>
          </n-input-number>
        </n-form-item>
        <n-form-item label="速率限制" path="speed_limit">
          <n-input-number
            v-model:value="editForm.speed_limit"
            placeholder="0 表示不限制，单位：Mbps"
            :min="0"
            style="width: 100%"
          >
            <template #suffix>Mbps（0=不限）</template>
          </n-input-number>
        </n-form-item>
        <n-form-item label="特性列表" path="features">
          <n-input
            v-model:value="editForm.features"
            type="textarea"
            placeholder="每行一个特性，如：不限速&#10;专属节点&#10;优先客服"
            :rows="3"
          />
        </n-form-item>
        <n-form-item label="启用状态" path="is_active">
          <n-switch v-model:value="editForm.is_active">
            <template #checked>启用</template>
            <template #unchecked>禁用</template>
          </n-switch>
        </n-form-item>
        <n-form-item label="推荐套餐" path="is_featured">
          <n-switch v-model:value="editForm.is_featured">
            <template #checked>推荐</template>
            <template #unchecked>不推荐</template>
          </n-switch>
        </n-form-item>
      </n-form>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, reactive, h, onMounted } from 'vue'
import { NButton, NTag, NSpace, NIcon, NSpin, useMessage, useDialog } from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import { listAdminPackages, createPackage, updatePackage, deletePackage } from '@/api/admin'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const packages = ref([])
const currentPage = ref(1)
const pageSize = ref(20)
const totalPages = ref(0)

const showEditModal = ref(false)
const isCreating = ref(false)
const formRef = ref(null)
const editForm = reactive({
  id: null,
  name: '',
  description: '',
  price: 0,
  duration_days: 30,
  device_limit: 3,
  traffic_limit: 0,
  speed_limit: 0,
  features: '',
  is_active: true,
  is_featured: false,
  sort_order: 0
})

const formRules = {
  name: [
    { required: true, message: '请输入套餐名称', trigger: 'blur' }
  ],
  price: [
    { required: true, message: '请输入价格', trigger: 'blur', type: 'number' }
  ],
  duration_days: [
    { required: true, message: '请输入有效期天数', trigger: 'blur', type: 'number' }
  ],
  device_limit: [
    { required: true, message: '请输入设备数量限制', trigger: 'blur', type: 'number' }
  ],
  sort_order: [
    { required: true, message: '请输入排序顺序', trigger: 'blur', type: 'number' }
  ]
}

const columns = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '套餐名称', key: 'name', ellipsis: { tooltip: true } },
  {
    title: '价格',
    key: 'price',
    width: 120,
    render: (row) => h(
      'span',
      { style: 'color: #18a058; font-weight: 600' },
      `¥${row.price.toFixed(2)}`
    )
  },
  {
    title: '有效期',
    key: 'duration_days',
    width: 100,
    render: (row) => `${row.duration_days} 天`
  },
  {
    title: '设备限制',
    key: 'device_limit',
    width: 100,
    render: (row) => `${row.device_limit} 台`
  },
  {
    title: '流量',
    key: 'traffic_limit',
    width: 100,
    render: (row) => {
      if (!row.traffic_limit || row.traffic_limit === 0) return '不限'
      const gb = row.traffic_limit / (1024 * 1024 * 1024)
      return `${gb.toFixed(0)} GB`
    }
  },
  {
    title: '状态',
    key: 'is_active',
    width: 100,
    render: (row) => h(
      NTag,
      { type: row.is_active ? 'success' : 'default', size: 'small' },
      { default: () => row.is_active ? '启用' : '禁用' }
    )
  },
  { title: '排序', key: 'sort_order', width: 80 },
  {
    title: '操作',
    key: 'actions',
    width: 160,
    fixed: 'right',
    render: (row) => h(
      NSpace,
      {},
      {
        default: () => [
          h(
            NButton,
            {
              size: 'small',
              type: 'primary',
              onClick: () => handleEdit(row)
            },
            { default: () => '编辑' }
          ),
          h(
            NButton,
            {
              size: 'small',
              type: 'error',
              onClick: () => handleDelete(row)
            },
            { default: () => '删除' }
          )
        ]
      }
    )
  }
]

const fetchPackages = async () => {
  loading.value = true
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    const response = await listAdminPackages(params)
    packages.value = response.data.items || []
    totalPages.value = Math.ceil((response.data.total || 0) / pageSize.value)
  } catch (error) {
    message.error('获取套餐列表失败：' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

const handlePageChange = (page) => {
  currentPage.value = page
  fetchPackages()
}

const handlePageSizeChange = (size) => {
  pageSize.value = size
  currentPage.value = 1
  fetchPackages()
}

const resetForm = () => {
  editForm.id = null
  editForm.name = ''
  editForm.description = ''
  editForm.price = 0
  editForm.duration_days = 30
  editForm.device_limit = 3
  editForm.traffic_limit = 0
  editForm.speed_limit = 0
  editForm.features = ''
  editForm.is_active = true
  editForm.is_featured = false
  editForm.sort_order = 0
}

const handleCreate = () => {
  resetForm()
  isCreating.value = true
  showEditModal.value = true
}

const handleEdit = (row) => {
  editForm.id = row.id
  editForm.name = row.name
  editForm.description = row.description || ''
  editForm.price = row.price
  editForm.duration_days = row.duration_days
  editForm.device_limit = row.device_limit
  editForm.traffic_limit = row.traffic_limit || 0
  editForm.speed_limit = row.speed_limit || 0
  // features is stored as JSON array string, convert to newline-separated for editing
  let feat = ''
  if (row.features) {
    try { feat = JSON.parse(row.features).join('\n') } catch { feat = row.features }
  }
  editForm.features = feat
  editForm.is_active = row.is_active
  editForm.is_featured = row.is_featured || false
  editForm.sort_order = row.sort_order
  isCreating.value = false
  showEditModal.value = true
}

const handleSavePackage = async () => {
  try {
    await formRef.value?.validate()
    
    const data = {
      name: editForm.name,
      description: editForm.description,
      price: editForm.price,
      duration_days: editForm.duration_days,
      device_limit: editForm.device_limit,
      traffic_limit: editForm.traffic_limit || 0,
      speed_limit: editForm.speed_limit || 0,
      features: editForm.features.trim()
        ? JSON.stringify(editForm.features.trim().split('\n').map(s => s.trim()).filter(Boolean))
        : null,
      is_active: editForm.is_active,
      is_featured: editForm.is_featured,
      sort_order: editForm.sort_order
    }

    if (isCreating.value) {
      await createPackage(data)
      message.success('套餐创建成功')
    } else {
      await updatePackage(editForm.id, data)
      message.success('套餐更新成功')
    }
    
    showEditModal.value = false
    fetchPackages()
  } catch (error) {
    if (error?.errors) return
    message.error((isCreating.value ? '创建' : '更新') + '套餐失败：' + (error.message || '未知错误'))
  }
}

const handleDelete = (row) => {
  dialog.error({
    title: '确认删除',
    content: `确定要删除套餐 ${row.name} 吗？此操作不可恢复！`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deletePackage(row.id)
        message.success('套餐删除成功')
        fetchPackages()
      } catch (error) {
        message.error('删除套餐失败：' + (error.message || '未知错误'))
      }
    }
  })
}

onMounted(() => {
  fetchPackages()
})
</script>

<style scoped>
.admin-packages-page {
  padding: 20px;
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
  .admin-packages-page { padding: 8px; }
}
</style>
