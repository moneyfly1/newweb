<template>
  <div class="redeem-container">
    <n-card title="兑换码管理">
      <template #header-extra>
        <n-button type="primary" @click="handleGenerate">
          批量生成
        </n-button>
      </template>

      <template v-if="!appStore.isMobile">
        <n-data-table
          remote
          :columns="columns"
          :data="codes"
          :loading="loading"
          :pagination="pagination"
          :bordered="false"
          @update:page="(p: number) => { pagination.page = p; loadCodes() }"
          @update:page-size="(ps: number) => { pagination.pageSize = ps; pagination.page = 1; loadCodes() }"
        />
      </template>

      <template v-else>
        <div class="mobile-card-list">
          <div v-for="row in codes" :key="row.id" class="mobile-card">
            <div class="card-header">
              <span class="card-title">{{ row.code }}</span>
              <n-tag :type="row.type === 'balance' ? 'success' : 'info'" size="small">
                {{ row.type === 'balance' ? '余额' : '套餐' }}
              </n-tag>
            </div>
            <div class="card-body">
              <div class="card-row">
                <span class="card-label">类型</span>
                <span>{{ row.type === 'balance' ? `¥${row.value}` : `套餐#${row.value}` }}</span>
              </div>
              <div class="card-row">
                <span class="card-label">状态</span>
                <n-tag :type="row.status === 'unused' ? 'success' : row.status === 'used' ? 'default' : 'warning'" size="small">
                  {{ row.status === 'unused' ? '未使用' : row.status === 'used' ? '已使用' : '已过期' }}
                </n-tag>
              </div>
              <div class="card-row">
                <span class="card-label">使用次数</span>
                <span>{{ row.used_count || 0 }} / {{ row.max_uses || 1 }}</span>
              </div>
              <div class="card-row">
                <span class="card-label">创建时间</span>
                <span>{{ formatDate(row.created_at) }}</span>
              </div>
            </div>
            <div class="card-actions">
              <n-button size="small" @click="copyCode(row.code)">复制</n-button>
              <n-button size="small" type="error" :disabled="row.used_count > 0" @click="handleDelete(row.id)">删除</n-button>
            </div>
          </div>
        </div>

        <n-pagination
          v-model:page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :item-count="pagination.itemCount"
          :page-sizes="pagination.pageSizes"
          show-size-picker
          style="margin-top: 16px; justify-content: center"
          @update:page="loadCodes"
          @update:page-size="(ps: number) => { pagination.pageSize = ps; pagination.page = 1; loadCodes() }"
        />
      </template>
    </n-card>

    <n-modal
      v-model:show="showGenerateModal"
      preset="card"
      title="批量生成兑换码"
      :style="appStore.isMobile ? 'width: 95vw; max-width: 500px' : 'width: 500px'"
      :segmented="{ content: 'soft' }"
    >
      <n-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-placement="left"
        label-width="100"
      >
        <n-form-item label="类型" path="type">
          <n-select
            v-model:value="formData.type"
            placeholder="请选择类型"
            :options="typeOptions"
          />
        </n-form-item>

        <n-form-item label="数值" path="value">
          <n-input-number
            v-model:value="formData.value"
            :placeholder="formData.type === 'balance' ? '充值金额（元）' : '套餐ID'"
            :min="1"
            style="width: 100%"
          />
        </n-form-item>

        <n-form-item label="生成数量" path="quantity">
          <n-input-number
            v-model:value="formData.quantity"
            placeholder="请输入生成数量"
            :min="1"
            :max="100"
            style="width: 100%"
          />
        </n-form-item>

        <n-alert type="info" style="margin-top: 12px">
          {{ formData.type === 'balance' ? `将生成 ${formData.quantity} 个面值为 ${formData.value} 元的余额兑换码` : `将生成 ${formData.quantity} 个套餐ID为 ${formData.value} 的套餐兑换码` }}
        </n-alert>
      </n-form>

      <template #footer>
        <n-space justify="end">
          <n-button @click="showGenerateModal = false">取消</n-button>
          <n-button type="primary" @click="handleSubmit" :loading="submitting">
            生成
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <n-modal
      v-model:show="showCodesModal"
      preset="card"
      title="生成的兑换码"
      :style="appStore.isMobile ? 'width: 95vw; max-width: 600px' : 'width: 600px'"
      :segmented="{ content: 'soft' }"
    >
      <n-alert type="success" style="margin-bottom: 16px">
        成功生成 {{ generatedCodes.length }} 个兑换码，请及时保存
      </n-alert>
      
      <n-space vertical :size="8">
        <div
          v-for="(code, index) in generatedCodes"
          :key="index"
          class="code-item"
        >
          <n-text code>{{ code }}</n-text>
          <n-button
            text
            size="small"
            @click="copyCode(code)"
          >
            复制
          </n-button>
        </div>
      </n-space>

      <template #footer>
        <n-space justify="end">
          <n-button @click="copyAllCodes">复制全部</n-button>
          <n-button type="primary" @click="showCodesModal = false">关闭</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, h, onMounted } from 'vue'
import { NButton, NTag, NSpace, useMessage, useDialog } from 'naive-ui'
import { listRedeemCodes, createRedeemCodes, deleteRedeemCode } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { copyToClipboard as clipboardCopy } from '@/utils/clipboard'

const message = useMessage()
const dialog = useDialog()
const appStore = useAppStore()

const loading = ref(false)
const submitting = ref(false)
const codes = ref<any[]>([])
const showGenerateModal = ref(false)
const showCodesModal = ref(false)
const generatedCodes = ref<string[]>([])
const formRef = ref()

const formData = reactive({
  type: 'balance',
  value: 10,
  quantity: 1
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [20, 50, 100],
  onChange: (page: number) => {
    pagination.page = page
    loadCodes()
  },
  onUpdatePageSize: (pageSize: number) => {
    pagination.pageSize = pageSize
    pagination.page = 1
    loadCodes()
  }
})

const typeOptions = [
  { label: '余额充值', value: 'balance' },
  { label: '套餐兑换', value: 'package' }
]

const rules = {
  type: { required: true, message: '请选择类型', trigger: 'change' },
  value: { required: true, type: 'number', message: '请输入数值', trigger: 'blur' },
  quantity: { required: true, type: 'number', message: '请输入生成数量', trigger: 'blur' }
}

const columns = [
  { title: 'ID', key: 'id', width: 60, resizable: true, sorter: 'default' },
  {
    title: '兑换码',
    key: 'code',
    width: 200,
    resizable: true,
    render: (row: any) => {
      return h(NSpace, { align: 'center' }, {
        default: () => [
          h('code', { style: { fontSize: '13px' } }, row.code),
          h(NButton, {
            text: true,
            size: 'small',
            onClick: () => copyCode(row.code)
          }, { default: () => '复制' })
        ]
      })
    }
  },
  {
    title: '类型',
    key: 'type',
    width: 100,
    resizable: true,
    render: (row: any) => h(NTag, { type: row.type === 'balance' ? 'success' : 'info' }, { default: () => row.type === 'balance' ? '余额' : '套餐' })
  },
  {
    title: '数值',
    key: 'value',
    width: 100,
    resizable: true,
    render: (row: any) => row.type === 'balance' ? `¥${row.value}` : `套餐#${row.value}`
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    resizable: true,
    render: (row: any) => {
      const statusMap: Record<string, { text: string; type: any }> = {
        unused: { text: '未使用', type: 'success' },
        used: { text: '已使用', type: 'default' },
        expired: { text: '已过期', type: 'warning' }
      }
      const status = statusMap[row.status] || { text: row.status, type: 'default' }
      return h(NTag, { type: status.type }, { default: () => status.text })
    }
  },
  { title: '使用次数', key: 'used_count', width: 100, resizable: true, render: (row: any) => `${row.used_count || 0} / ${row.max_uses || 1}` },
  { title: '创建时间', key: 'created_at', width: 160, resizable: true, render: (row: any) => formatDate(row.created_at) },
  {
    title: '操作',
    key: 'actions',
    width: 100,
    fixed: 'right' as const,
    render: (row: any) => {
      return h(NButton, {
        size: 'small',
        type: 'error',
        disabled: row.used_count > 0,
        onClick: () => handleDelete(row.id)
      }, { default: () => '删除' })
    }
  }
]

const formatDate = (date: string) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
}

const loadCodes = async () => {
  loading.value = true
  try {
    const res = await listRedeemCodes({
      page: pagination.page,
      page_size: pagination.pageSize
    })
    codes.value = res.data.items || []
    pagination.itemCount = res.data.total || 0
  } catch (error: any) {
    message.error(error.message || '加载兑换码列表失败')
  } finally {
    loading.value = false
  }
}

const handleGenerate = () => {
  formData.type = 'balance'
  formData.value = 10
  formData.quantity = 1
  showGenerateModal.value = true
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  submitting.value = true
  try {
    const res = await createRedeemCodes(formData)
    message.success('生成成功')
    
    generatedCodes.value = res.data.codes || []
    showGenerateModal.value = false
    showCodesModal.value = true
    
    await loadCodes()
  } catch (error: any) {
    message.error(error.message || '生成失败')
  } finally {
    submitting.value = false
  }
}

const copyCode = async (code: string) => {
  const ok = await clipboardCopy(code)
  ok ? message.success('复制成功') : message.error('复制失败')
}

const copyAllCodes = async () => {
  const text = generatedCodes.value.join('\n')
  const ok = await clipboardCopy(text)
  ok ? message.success('已复制全部兑换码') : message.error('复制失败')
}

const handleDelete = (id: number) => {
  dialog.warning({
    title: '确认删除',
    content: '确定要删除这个兑换码吗？此操作不可恢复。',
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteRedeemCode(id)
        message.success('删除成功')
        await loadCodes()
      } catch (error: any) {
        message.error(error.message || '删除失败')
      }
    }
  })
}

onMounted(() => {
  loadCodes()
})
</script>

<style scoped>
.redeem-container {
  padding: 20px;
}

.code-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #f5f5f5;
  border-radius: 4px;
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
  font-family: monospace;
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
  .redeem-container { padding: 8px; }
}
</style>
