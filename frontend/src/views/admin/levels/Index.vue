<template>
  <div class="levels-container">
    <n-card title="用户等级管理">
      <template #header-extra>
        <n-button type="primary" @click="handleAdd">
          添加等级
        </n-button>
      </template>

      <template v-if="!appStore.isMobile">
        <n-data-table
          :columns="columns"
          :data="levels"
          :loading="loading"
          :bordered="false"
        />
      </template>

      <template v-else>
        <div class="mobile-card-list">
          <div v-for="row in levels" :key="row.id" class="mobile-card">
            <div class="card-header">
              <span class="card-title">{{ row.level_name }}</span>
              <n-tag :type="row.is_active ? 'success' : 'default'" size="small">
                {{ row.is_active ? '启用' : '禁用' }}
              </n-tag>
            </div>
            <div class="card-body">
              <div class="card-row">
                <span class="card-label">折扣率</span>
                <span>{{ row.discount_rate }}%</span>
              </div>
              <div class="card-row">
                <span class="card-label">最低消费</span>
                <span>¥{{ row.min_consumption || 0 }}</span>
              </div>
              <div class="card-row" v-if="row.benefits">
                <span class="card-label">权益说明</span>
                <span style="text-align: right; flex: 1; margin-left: 8px;">{{ row.benefits }}</span>
              </div>
            </div>
            <div class="card-actions">
              <n-button size="small" type="primary" @click="handleEdit(row)">编辑</n-button>
              <n-button size="small" type="error" @click="handleDelete(row.id)">删除</n-button>
            </div>
          </div>
        </div>
      </template>
    </n-card>

    <n-modal
      v-model:show="showModal"
      preset="card"
      :title="isEdit ? '编辑等级' : '添加等级'"
      :style="appStore.isMobile ? 'width: 95vw; max-width: 600px' : 'width: 600px'"
      :segmented="{ content: 'soft' }"
    >
      <n-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-placement="left"
        label-width="100"
      >
        <n-form-item label="等级名称" path="level_name">
          <n-input v-model:value="formData.level_name" placeholder="请输入等级名称" />
        </n-form-item>

        <n-form-item label="等级数值" path="level_order">
          <n-input-number
            v-model:value="formData.level_order"
            placeholder="请输入等级数值"
            :min="0"
            style="width: 100%"
          />
        </n-form-item>

        <n-form-item label="折扣率" path="discount_rate">
          <n-input-number
            v-model:value="formData.discount_rate"
            placeholder="0-100，100表示无折扣"
            :min="0"
            :max="100"
            style="width: 100%"
          >
            <template #suffix>%</template>
          </n-input-number>
        </n-form-item>

        <n-form-item label="最低消费" path="min_consumption">
          <n-input-number
            v-model:value="formData.min_consumption"
            placeholder="请输入最低消费金额"
            :min="0"
            style="width: 100%"
          >
            <template #suffix>元</template>
          </n-input-number>
        </n-form-item>

        <n-form-item label="权益说明" path="benefits">
          <n-input
            v-model:value="formData.benefits"
            type="textarea"
            placeholder="请输入权益说明，每行一个权益"
            :rows="4"
          />
        </n-form-item>

        <n-form-item label="是否启用" path="is_active">
          <n-switch v-model:value="formData.is_active" />
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

<script setup lang="ts">
import { ref, reactive, h, onMounted } from 'vue'
import { NButton, NTag, NSpace, useMessage, useDialog } from 'naive-ui'
import { listUserLevels, createUserLevel, updateUserLevel, deleteUserLevel } from '@/api/admin'
import { useAppStore } from '@/stores/app'

const message = useMessage()
const dialog = useDialog()
const appStore = useAppStore()

const loading = ref(false)
const submitting = ref(false)
const levels = ref<any[]>([])
const showModal = ref(false)
const isEdit = ref(false)
const formRef = ref()

const formData = reactive({
  id: 0,
  level_name: '',
  level_order: 0,
  discount_rate: 100,
  min_consumption: 0,
  benefits: '',
  is_active: true
})

const rules = {
  level_name: { required: true, message: '请输入等级名称', trigger: 'blur' },
  level_order: { required: true, type: 'number', message: '请输入等级数值', trigger: 'blur' },
  discount_rate: { required: true, type: 'number', message: '请输入折扣率', trigger: 'blur' }
}

const columns = [
  { title: 'ID', key: 'id', width: 60, resizable: true, sorter: 'default' },
  { title: '等级名称', key: 'level_name', width: 120, resizable: true },
  { title: '等级数值', key: 'level_order', width: 100, resizable: true },
  {
    title: '折扣率',
    key: 'discount_rate',
    width: 100,
    resizable: true,
    render: (row: any) => `${row.discount_rate}%`
  },
  {
    title: '最低消费',
    key: 'min_consumption',
    width: 120,
    resizable: true,
    render: (row: any) => `¥${row.min_consumption || 0}`
  },
  {
    title: '权益说明',
    key: 'benefits',
    width: 200,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '状态',
    key: 'is_active',
    width: 80,
    resizable: true,
    render: (row: any) => h(NTag, { type: row.is_active ? 'success' : 'default' }, { default: () => row.is_active ? '启用' : '禁用' })
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    fixed: 'right' as const,
    render: (row: any) => {
      return h(NSpace, {}, {
        default: () => [
          h(NButton, { size: 'small', onClick: () => handleEdit(row) }, { default: () => '编辑' }),
          h(NButton, {
            size: 'small',
            type: 'error',
            onClick: () => handleDelete(row.id)
          }, { default: () => '删除' })
        ]
      })
    }
  }
]

const loadLevels = async () => {
  loading.value = true
  try {
    const res = await listUserLevels()
    levels.value = res.data.items || res.data || []
  } catch (error: any) {
    message.error(error.message || '加载等级列表失败')
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  formData.id = 0
  formData.level_name = ''
  formData.level_order = 0
  formData.discount_rate = 100
  formData.min_consumption = 0
  formData.benefits = ''
  formData.is_active = true
}

const handleAdd = () => {
  resetForm()
  isEdit.value = false
  showModal.value = true
}

const handleEdit = (row: any) => {
  formData.id = row.id
  formData.level_name = row.level_name
  formData.level_order = row.level_order
  formData.discount_rate = row.discount_rate
  formData.min_consumption = row.min_consumption || 0
  formData.benefits = row.benefits || ''
  formData.is_active = row.is_active
  isEdit.value = true
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
    const data: any = { ...formData }
    delete data.id

    if (isEdit.value) {
      await updateUserLevel(formData.id, data)
      message.success('更新成功')
    } else {
      await createUserLevel(data)
      message.success('创建成功')
    }

    showModal.value = false
    await loadLevels()
  } catch (error: any) {
    message.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleDelete = (id: number) => {
  dialog.warning({
    title: '确认删除',
    content: '确定要删除这个等级吗？此操作不可恢复。',
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteUserLevel(id)
        message.success('删除成功')
        await loadLevels()
      } catch (error: any) {
        message.error(error.message || '删除失败')
      }
    }
  })
}

onMounted(() => {
  loadLevels()
})
</script>

<style scoped>
.levels-container {
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
  .levels-container { padding: 8px; }
}
</style>
