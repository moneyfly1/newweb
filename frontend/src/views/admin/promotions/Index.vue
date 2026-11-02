<template>
  <div class="promotions-container admin-page-shell">
    <n-card :title="'促销活动管理'" :bordered="false">
      <template #header-extra>
        <n-button type="primary" @click="handleAdd">创建活动</n-button>
      </template>
      <n-data-table
        :columns="columns"
        :data="list"
        :loading="loading"
        :pagination="{ pageSize: 20 }"
        size="small"
      />
    </n-card>

    <n-modal v-model:show="showModal" preset="card" :title="editing ? '编辑活动' : '创建活动'" style="width: 560px">
      <n-form :model="form" label-placement="left" label-width="90">
        <n-form-item label="名称"><n-input v-model:value="form.name" placeholder="如：双十一大促" /></n-form-item>
        <n-form-item label="类型">
          <n-select v-model:value="form.type" :options="typeOptions" />
        </n-form-item>
        <n-form-item label="折扣类型">
          <n-select v-model:value="form.discount_type" :options="discountTypeOptions" />
        </n-form-item>
        <n-form-item label="折扣值">
          <n-input-number v-model:value="form.discount_value" :step="0.1" :precision="2" style="width: 100%" />
          <template #feedback>percentage：0.9=九折；fixed：直接减免金额</template>
        </n-form-item>
        <n-form-item label="最低金额">
          <n-input-number v-model:value="form.min_amount" :min="0" :precision="2" style="width: 100%" />
        </n-form-item>
        <n-form-item label="最多减免">
          <n-input-number v-model:value="form.max_discount" :min="0" :precision="2" style="width: 100%" />
        </n-form-item>
        <n-form-item label="开始时间">
          <n-date-picker v-model:value="form.start_ts" type="datetime" style="width: 100%" />
        </n-form-item>
        <n-form-item label="结束时间">
          <n-date-picker v-model:value="form.end_ts" type="datetime" style="width: 100%" />
        </n-form-item>
        <n-form-item label="描述">
          <n-input v-model:value="form.description" type="textarea" :rows="2" />
        </n-form-item>
        <n-form-item label="启用">
          <n-switch v-model:value="form.is_active" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showModal = false">取消</n-button>
          <n-button type="primary" :loading="saving" @click="handleSave">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, onMounted, reactive, ref } from 'vue'
import { NButton, NSpace, NTag, useMessage } from 'naive-ui'
import request from '@/utils/request'
import { formatDateTime } from '@/utils/format'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const showModal = ref(false)
const editing = ref(false)
const editingId = ref(0)
const list = ref<any[]>([])

const typeOptions = [
  { label: '限时折扣', value: 'flash_sale' },
  { label: '新用户', value: 'new_user' },
  { label: '召回', value: 'recall' },
  { label: '会员日', value: 'member_day' },
]
const discountTypeOptions = [
  { label: '折扣率（percentage）', value: 'percentage' },
  { label: '固定减免（fixed）', value: 'fixed' },
]

const form = reactive<any>({
  name: '', type: 'flash_sale', discount_type: 'percentage', discount_value: 0.9,
  min_amount: 0, max_discount: 0, description: '', is_active: true,
  start_ts: Date.now(), end_ts: Date.now() + 7 * 24 * 3600 * 1000,
})

function fmt(ts: number) {
  const d = new Date(ts)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}

async function load() {
  loading.value = true
  try {
    const res: any = await request.get('/admin/promotions')
    list.value = res || []
  } finally {
    loading.value = false
  }
}

function handleAdd() {
  editing.value = false
  editingId.value = 0
  Object.assign(form, {
    name: '', type: 'flash_sale', discount_type: 'percentage', discount_value: 0.9,
    min_amount: 0, max_discount: 0, description: '', is_active: true,
    start_ts: Date.now(), end_ts: Date.now() + 7 * 24 * 3600 * 1000,
  })
  showModal.value = true
}

function handleEdit(row: any) {
  editing.value = true
  editingId.value = row.id
  Object.assign(form, {
    name: row.name, type: row.type, discount_type: row.discount_type,
    discount_value: row.discount_value, min_amount: row.min_amount,
    max_discount: row.max_discount, description: row.description || '',
    is_active: row.is_active,
    start_ts: new Date(row.start_time).getTime(),
    end_ts: new Date(row.end_time).getTime(),
  })
  showModal.value = true
}

async function handleSave() {
  if (!form.name) { message.warning('请填写活动名称'); return }
  saving.value = true
  try {
    const payload = {
      ...form,
      start_time: fmt(form.start_ts),
      end_time: fmt(form.end_ts),
    }
    if (editing.value) {
      await request.put(`/admin/promotions/${editingId.value}`, payload)
    } else {
      await request.post('/admin/promotions', payload)
    }
    message.success('保存成功')
    showModal.value = false
    await load()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleDelete(row: any) {
  try {
    await request.delete(`/admin/promotions/${row.id}`)
    message.success('已删除')
    await load()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '删除失败')
  }
}

const columns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '名称', key: 'name', ellipsis: { tooltip: true } },
  { title: '类型', key: 'type', width: 100 },
  { title: '折扣', key: 'discount_value', width: 100,
    render: (row: any) => row.discount_type === 'percentage' ? `${row.discount_value}×` : `-¥${row.discount_value}` },
  { title: '时间', key: 'start_time', width: 200,
    render: (row: any) => `${formatDateTime(row.start_time)} ~ ${formatDateTime(row.end_time)}` },
  { title: '状态', key: 'is_active', width: 80,
    render: (row: any) => h(NTag, { type: row.is_active ? 'success' : 'default', size: 'small' },
      { default: () => row.is_active ? '启用' : '停用' }) },
  { title: '操作', key: 'actions', width: 140,
    render: (row: any) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, { size: 'tiny', onClick: () => handleEdit(row) }, { default: () => '编辑' }),
        h(NButton, { size: 'tiny', type: 'error', secondary: true, onClick: () => handleDelete(row) }, { default: () => '删除' }),
      ],
    }) },
]

onMounted(load)
</script>
