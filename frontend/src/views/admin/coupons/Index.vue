<template>
  <div class="coupons-container">
    <n-card title="优惠券管理">
      <template #header-extra>
        <n-button type="primary" @click="handleAdd">
          <template #icon>
            <n-icon><AddOutline /></n-icon>
          </template>
          创建优惠券
        </n-button>
      </template>

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
    </n-card>

    <n-modal
      v-model:show="showModal"
      :title="modalTitle"
      preset="card"
      style="width: 700px"
      :mask-closable="false"
    >
      <n-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-placement="left"
        label-width="120"
      >
        <n-form-item label="优惠券代码" path="code">
          <n-input v-model:value="formData.code" placeholder="请输入优惠券代码" />
        </n-form-item>
        <n-form-item label="优惠券名称" path="name">
          <n-input v-model:value="formData.name" placeholder="请输入优惠券名称" />
        </n-form-item>
        <n-form-item label="描述" path="description">
          <n-input
            v-model:value="formData.description"
            type="textarea"
            placeholder="请输入优惠券描述"
            :rows="2"
          />
        </n-form-item>
        <n-form-item label="优惠类型" path="type">
          <n-select v-model:value="formData.type" :options="typeOptions" />
        </n-form-item>
        <n-form-item label="优惠值" path="discount_value">
          <n-input-number
            v-model:value="formData.discount_value"
            :min="0"
            style="width: 100%"
            :placeholder="getDiscountPlaceholder()"
          />
        </n-form-item>
        <n-form-item label="最低消费金额" path="min_amount">
          <n-input-number
            v-model:value="formData.min_amount"
            :min="0"
            style="width: 100%"
            placeholder="0表示无限制"
          />
        </n-form-item>
        <n-form-item label="最大优惠金额" path="max_discount">
          <n-input-number
            v-model:value="formData.max_discount"
            :min="0"
            style="width: 100%"
            placeholder="0表示无限制"
          />
        </n-form-item>
        <n-form-item label="生效时间" path="valid_from">
          <n-date-picker
            v-model:value="formData.valid_from"
            type="datetime"
            style="width: 100%"
            placeholder="请选择生效时间"
          />
        </n-form-item>
        <n-form-item label="失效时间" path="valid_until">
          <n-date-picker
            v-model:value="formData.valid_until"
            type="datetime"
            style="width: 100%"
            placeholder="请选择失效时间"
          />
        </n-form-item>
        <n-form-item label="发行总量" path="total_quantity">
          <n-input-number
            v-model:value="formData.total_quantity"
            :min="1"
            style="width: 100%"
            placeholder="请输入发行总量"
          />
        </n-form-item>
        <n-form-item label="单用户限用次数" path="max_uses_per_user">
          <n-input-number
            v-model:value="formData.max_uses_per_user"
            :min="1"
            style="width: 100%"
            placeholder="请输入单用户限用次数"
          />
        </n-form-item>
        <n-form-item label="状态" path="status">
          <n-select v-model:value="formData.status" :options="statusOptions" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 12px">
          <n-button @click="showModal = false">取消</n-button>
          <n-button type="primary" @click="handleSubmit" :loading="submitting">确定</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, reactive, h, onMounted } from 'vue'
import { NButton, NTag, NSpace, NIcon, NTooltip, useMessage, useDialog } from 'naive-ui'
import { AddOutline, CreateOutline, TrashOutline, CopyOutline } from '@vicons/ionicons5'
import { listAdminCoupons, createCoupon, updateCoupon, deleteCoupon } from '@/api/admin'

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const submitting = ref(false)
const showModal = ref(false)
const modalTitle = ref('创建优惠券')
const tableData = ref([])
const formRef = ref(null)
const isEdit = ref(false)
const editId = ref(null)

const typeOptions = [
  { label: '折扣（百分比）', value: 'discount' },
  { label: '固定金额', value: 'fixed' },
  { label: '免费天数', value: 'free_days' }
]

const statusOptions = [
  { label: '启用', value: 'active' },
  { label: '禁用', value: 'inactive' },
  { label: '已过期', value: 'expired' }
]

const formData = reactive({
  code: '',
  name: '',
  description: '',
  type: 'discount',
  discount_value: 0,
  min_amount: 0,
  max_discount: 0,
  valid_from: null,
  valid_until: null,
  total_quantity: 100,
  max_uses_per_user: 1,
  status: 'active'
})

const rules = {
  code: { required: true, message: '请输入优惠券代码', trigger: 'blur' },
  name: { required: true, message: '请输入优惠券名称', trigger: 'blur' },
  type: { required: true, message: '请选择优惠类型', trigger: 'change' },
  discount_value: { required: true, type: 'number', message: '请输入优惠值', trigger: 'blur' },
  total_quantity: { required: true, type: 'number', message: '请输入发行总量', trigger: 'blur' },
  max_uses_per_user: { required: true, type: 'number', message: '请输入单用户限用次数', trigger: 'blur' },
  status: { required: true, message: '请选择状态', trigger: 'change' }
}

const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
})

const getDiscountPlaceholder = () => {
  switch (formData.type) {
    case 'discount':
      return '输入折扣百分比，如：20表示8折'
    case 'fixed':
      return '输入固定优惠金额'
    case 'free_days':
      return '输入免费天数'
    default:
      return '请输入优惠值'
  }
}

const copyToClipboard = (text) => {
  navigator.clipboard.writeText(text).then(() => {
    message.success('已复制到剪贴板')
  }).catch(() => {
    message.error('复制失败')
  })
}

const formatDateTime = (timestamp) => {
  if (!timestamp) return '-'
  const date = new Date(timestamp)
  return date.toLocaleString('zh-CN', { hour12: false })
}

const columns = [
  { title: 'ID', key: 'id', width: 80 },
  {
    title: '优惠券代码',
    key: 'code',
    width: 180,
    render: (row) => {
      return h(NSpace, { align: 'center' }, {
        default: () => [
          h('span', { style: 'font-family: monospace; font-weight: bold' }, row.code),
          h(NTooltip, {}, {
            trigger: () => h(NButton, {
              size: 'small',
              text: true,
              onClick: () => copyToClipboard(row.code)
            }, { icon: () => h(NIcon, {}, { default: () => h(CopyOutline) }) }),
            default: () => '复制代码'
          })
        ]
      })
    }
  },
  { title: '名称', key: 'name', ellipsis: { tooltip: true } },
  {
    title: '类型',
    key: 'type',
    width: 140,
    render: (row) => {
      const typeMap = {
        discount: { type: 'info', text: '折扣' },
        fixed: { type: 'success', text: '固定金额' },
        free_days: { type: 'warning', text: '免费天数' }
      }
      const type = typeMap[row.type] || { type: 'default', text: row.type }
      return h(NTag, { type: type.type }, { default: () => type.text })
    }
  },
  {
    title: '优惠值',
    key: 'discount_value',
    width: 120,
    render: (row) => {
      if (row.type === 'discount') return `${row.discount_value}%`
      if (row.type === 'fixed') return `¥${row.discount_value}`
      if (row.type === 'free_days') return `${row.discount_value}天`
      return row.discount_value
    }
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row) => {
      const statusMap = {
        active: { type: 'success', text: '启用' },
        inactive: { type: 'error', text: '禁用' },
        expired: { type: 'default', text: '已过期' }
      }
      const status = statusMap[row.status] || { type: 'default', text: row.status }
      return h(NTag, { type: status.type }, { default: () => status.text })
    }
  },
  {
    title: '使用情况',
    key: 'usage',
    width: 120,
    render: (row) => `${row.used_quantity || 0} / ${row.total_quantity || 0}`
  },
  {
    title: '生效时间',
    key: 'valid_from',
    width: 180,
    render: (row) => formatDateTime(row.valid_from)
  },
  {
    title: '失效时间',
    key: 'valid_until',
    width: 180,
    render: (row) => formatDateTime(row.valid_until)
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    fixed: 'right',
    render: (row) => {
      return h(NSpace, {}, {
        default: () => [
          h(NButton, {
            size: 'small',
            type: 'primary',
            text: true,
            onClick: () => handleEdit(row)
          }, { default: () => '编辑', icon: () => h(NIcon, {}, { default: () => h(CreateOutline) }) }),
          h(NButton, {
            size: 'small',
            type: 'error',
            text: true,
            onClick: () => handleDelete(row)
          }, { default: () => '删除', icon: () => h(NIcon, {}, { default: () => h(TrashOutline) }) })
        ]
      })
    }
  }
]

const fetchData = async () => {
  loading.value = true
  try {
    const res = await listAdminCoupons({
      page: pagination.page,
      page_size: pagination.pageSize
    })
    tableData.value = res.data.items || []
    pagination.itemCount = res.data.total || 0
  } catch (error) {
    message.error(error.message || '获取优惠券列表失败')
  } finally {
    loading.value = false
  }
}

const handlePageChange = (page) => {
  pagination.page = page
  fetchData()
}

const resetForm = () => {
  Object.assign(formData, {
    code: '',
    name: '',
    description: '',
    type: 'discount',
    discount_value: 0,
    min_amount: 0,
    max_discount: 0,
    valid_from: null,
    valid_until: null,
    total_quantity: 100,
    max_uses_per_user: 1,
    status: 'active'
  })
  formRef.value?.restoreValidation()
}

const handleAdd = () => {
  isEdit.value = false
  modalTitle.value = '创建优惠券'
  resetForm()
  showModal.value = true
}

const handleEdit = (row) => {
  isEdit.value = true
  editId.value = row.id
  modalTitle.value = '编辑优惠券'
  Object.assign(formData, {
    code: row.code,
    name: row.name,
    description: row.description || '',
    type: row.type,
    discount_value: row.discount_value,
    min_amount: row.min_amount || 0,
    max_discount: row.max_discount || 0,
    valid_from: row.valid_from ? new Date(row.valid_from).getTime() : null,
    valid_until: row.valid_until ? new Date(row.valid_until).getTime() : null,
    total_quantity: row.total_quantity,
    max_uses_per_user: row.max_uses_per_user,
    status: row.status
  })
  showModal.value = true
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
    submitting.value = true
    
    const submitData = {
      ...formData,
      valid_from: formData.valid_from ? new Date(formData.valid_from).toISOString() : null,
      valid_until: formData.valid_until ? new Date(formData.valid_until).toISOString() : null
    }
    
    if (isEdit.value) {
      await updateCoupon(editId.value, submitData)
      message.success('更新优惠券成功')
    } else {
      await createCoupon(submitData)
      message.success('创建优惠券成功')
    }
    
    showModal.value = false
    fetchData()
  } catch (error) {
    if (error.message) {
      message.error(error.message || '操作失败')
    }
  } finally {
    submitting.value = false
  }
}

const handleDelete = (row) => {
  dialog.warning({
    title: '确认删除',
    content: `确定要删除优惠券 "${row.name}" (${row.code}) 吗？此操作不可恢复。`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteCoupon(row.id)
        message.success('删除优惠券成功')
        fetchData()
      } catch (error) {
        message.error(error.message || '删除优惠券失败')
      }
    }
  })
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.coupons-container {
  padding: 20px;
}

@media (max-width: 767px) {
  .coupons-container { padding: 8px; }
}
</style>
