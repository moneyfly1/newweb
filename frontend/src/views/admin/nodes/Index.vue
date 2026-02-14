<template>
  <div class="nodes-container">
    <n-card title="节点管理">
      <template #header-extra>
        <n-space>
          <n-button @click="handleRefresh" :loading="refreshing">
            <template #icon>
              <n-icon><RefreshOutline /></n-icon>
            </template>
            刷新列表
          </n-button>
          <n-button type="info" @click="showImportLinksModal = true">
            <template #icon>
              <n-icon><LinkOutline /></n-icon>
            </template>
            导入链接
          </n-button>
          <n-button type="primary" @click="showImportSubModal = true">
            <template #icon>
              <n-icon><CloudDownloadOutline /></n-icon>
            </template>
            导入订阅
          </n-button>
          <n-button
            v-if="checkedRowKeys.length > 0"
            type="error"
            @click="handleBatchDelete"
          >
            <template #icon>
              <n-icon><TrashOutline /></n-icon>
            </template>
            批量删除 ({{ checkedRowKeys.length }})
          </n-button>
        </n-space>
      </template>

      <n-data-table
        remote
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :pagination="pagination"
        :bordered="false"
        :row-key="(row) => row.id"
        @update:checked-row-keys="handleCheck"
        @update:page="(p) => { pagination.page = p; fetchData() }"
        @update:page-size="(ps) => { pagination.pageSize = ps; pagination.page = 1; fetchData() }"
      />
    </n-card>

    <!-- Import Subscription Modal -->
    <n-modal
      v-model:show="showImportSubModal"
      title="导入订阅"
      preset="card"
      style="width: 600px"
      :mask-closable="false"
    >
      <n-form label-placement="top">
        <n-form-item label="订阅链接">
          <n-input
            v-model:value="subscriptionUrl"
            placeholder="请输入订阅链接 (Clash/V2Ray)"
            type="text"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 12px">
          <n-button @click="showImportSubModal = false">取消</n-button>
          <n-button
            type="primary"
            @click="handleImportSubscription"
            :loading="importing"
            :disabled="!subscriptionUrl.trim()"
          >
            导入
          </n-button>
        </div>
      </template>
    </n-modal>

    <!-- Import Links Modal -->
    <n-modal
      v-model:show="showImportLinksModal"
      title="导入链接"
      preset="card"
      style="width: 600px"
      :mask-closable="false"
    >
      <n-form label-placement="top">
        <n-form-item label="节点链接">
          <n-input
            v-model:value="nodeLinks"
            placeholder="请输入节点链接，每行一个&#10;支持: vmess://, vless://, trojan://, ss://, 等"
            type="textarea"
            :rows="10"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 12px">
          <n-button @click="showImportLinksModal = false">取消</n-button>
          <n-button
            type="primary"
            @click="handleImportLinks"
            :loading="importing"
            :disabled="!nodeLinks.trim()"
          >
            导入
          </n-button>
        </div>
      </template>
    </n-modal>

    <!-- Edit Modal -->
    <n-modal
      v-model:show="showEditModal"
      title="编辑节点"
      preset="card"
      style="width: 600px"
      :mask-closable="false"
    >
      <n-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-placement="left"
        label-width="100"
      >
        <n-form-item label="节点名称" path="name">
          <n-input v-model:value="formData.name" placeholder="请输入节点名称" />
        </n-form-item>
        <n-form-item label="地区" path="region">
          <n-input v-model:value="formData.region" placeholder="如: 香港, 美国, 日本" />
        </n-form-item>
        <n-form-item label="排序" path="order_index">
          <n-input-number v-model:value="formData.order_index" :min="0" style="width: 100%" />
        </n-form-item>
        <n-form-item label="启用" path="is_active">
          <n-switch v-model:value="formData.is_active" />
        </n-form-item>
        <n-form-item label="备注" path="description">
          <n-input
            v-model:value="formData.description"
            type="textarea"
            placeholder="请输入备注信息"
            :rows="3"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 12px">
          <n-button @click="showEditModal = false">取消</n-button>
          <n-button type="primary" @click="handleSubmit" :loading="submitting">确定</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, reactive, h, onMounted } from 'vue'
import { NButton, NTag, NSpace, NIcon, NSwitch, useMessage, useDialog } from 'naive-ui'
import {
  CloudDownloadOutline,
  LinkOutline,
  RefreshOutline,
  CreateOutline,
  TrashOutline
} from '@vicons/ionicons5'
import { listAdminNodes, updateNode, deleteNode, importNodes } from '@/api/admin'

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const submitting = ref(false)
const importing = ref(false)
const refreshing = ref(false)
const showImportSubModal = ref(false)
const showImportLinksModal = ref(false)
const showEditModal = ref(false)
const tableData = ref([])
const formRef = ref(null)
const editId = ref(null)
const checkedRowKeys = ref([])

const subscriptionUrl = ref('')
const nodeLinks = ref('')

const formData = reactive({
  name: '',
  region: '',
  is_active: true,
  order_index: 0,
  description: ''
})

const rules = {
  name: { required: true, message: '请输入节点名称', trigger: 'blur' }
}

const pagination = reactive({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
})

const protocolColorMap = {
  vmess: 'info',
  vless: 'success',
  trojan: 'warning',
  ss: 'default',
  ssr: 'default',
  hysteria2: 'error',
  hysteria: 'error'
}

const columns = [
  {
    type: 'selection'
  },
  { title: 'ID', key: 'id', width: 80 },
  { title: '节点名称', key: 'name', ellipsis: { tooltip: true }, minWidth: 200 },
  {
    title: '协议',
    key: 'type',
    width: 120,
    render: (row) => {
      const type = protocolColorMap[row.type] || 'default'
      return h(NTag, { type }, { default: () => (row.type || '').toUpperCase() })
    }
  },
  { title: '地区', key: 'region', width: 120 },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row) => {
      const statusMap = {
        online: { type: 'success', text: '在线' },
        offline: { type: 'error', text: '离线' },
      }
      const status = statusMap[row.status] || { type: 'default', text: row.status || '未知' }
      return h(NTag, { type: status.type, size: 'small' }, { default: () => status.text })
    }
  },
  {
    title: '来源',
    key: 'is_manual',
    width: 80,
    render: (row) => h(NTag, { type: row.is_manual ? 'info' : 'default', size: 'small' }, { default: () => row.is_manual ? '手动' : '订阅' })
  },
  {
    title: '启用',
    key: 'is_active',
    width: 100,
    render: (row) => {
      return h(NSwitch, {
        value: row.is_active,
        onUpdateValue: (value) => handleToggleActive(row, value)
      })
    }
  },
  { title: '排序', key: 'order_index', width: 80 },
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
    const res = await listAdminNodes({
      page: pagination.page,
      page_size: pagination.pageSize
    })
    tableData.value = res.data.items || []
    pagination.itemCount = res.data.total || 0
  } catch (error) {
    message.error(error.message || '获取节点列表失败')
  } finally {
    loading.value = false
  }
}

const handleCheck = (keys) => {
  checkedRowKeys.value = keys
}

const handleImportSubscription = async () => {
  if (!subscriptionUrl.value.trim()) {
    message.warning('请输入订阅链接')
    return
  }

  importing.value = true
  try {
    const res = await importNodes({
      type: 'subscription',
      url: subscriptionUrl.value.trim()
    })
    message.success(`导入完成: 成功 ${res.data.success}/${res.data.total} 个`)
    showImportSubModal.value = false
    subscriptionUrl.value = ''
    fetchData()
  } catch (error) {
    message.error(error.message || '导入失败')
  } finally {
    importing.value = false
  }
}

const handleImportLinks = async () => {
  if (!nodeLinks.value.trim()) {
    message.warning('请输入节点链接')
    return
  }

  importing.value = true
  try {
    const res = await importNodes({
      type: 'links',
      links: nodeLinks.value.trim()
    })
    message.success(`导入完成: 成功 ${res.data.success}/${res.data.total} 个`)
    showImportLinksModal.value = false
    nodeLinks.value = ''
    fetchData()
  } catch (error) {
    message.error(error.message || '导入失败')
  } finally {
    importing.value = false
  }
}

const handleRefresh = async () => {
  refreshing.value = true
  await fetchData()
  refreshing.value = false
  message.success('刷新完成')
}

const resetForm = () => {
  Object.assign(formData, {
    name: '',
    region: '',
    is_active: true,
    order_index: 0,
    description: ''
  })
  formRef.value?.restoreValidation()
}

const handleEdit = (row) => {
  editId.value = row.id
  Object.assign(formData, {
    name: row.name,
    region: row.region,
    is_active: row.is_active,
    order_index: row.order_index,
    description: row.description || ''
  })
  showEditModal.value = true
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
    submitting.value = true

    await updateNode(editId.value, formData)
    message.success('更新节点成功')

    showEditModal.value = false
    fetchData()
  } catch (error) {
    if (error.message) {
      message.error(error.message || '操作失败')
    }
  } finally {
    submitting.value = false
  }
}

const handleToggleActive = async (row, value) => {
  try {
    await updateNode(row.id, { is_active: value })
    message.success('更新成功')
    fetchData()
  } catch (error) {
    message.error(error.message || '更新失败')
  }
}

const handleDelete = (row) => {
  dialog.warning({
    title: '确认删除',
    content: `确定要删除节点 "${row.name}" 吗？此操作不可恢复。`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteNode(row.id)
        message.success('删除节点成功')
        fetchData()
      } catch (error) {
        message.error(error.message || '删除节点失败')
      }
    }
  })
}

const handleBatchDelete = () => {
  dialog.warning({
    title: '确认批量删除',
    content: `确定要删除选中的 ${checkedRowKeys.value.length} 个节点吗？此操作不可恢复。`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await Promise.all(checkedRowKeys.value.map(id => deleteNode(id)))
        message.success('批量删除成功')
        checkedRowKeys.value = []
        fetchData()
      } catch (error) {
        message.error(error.message || '批量删除失败')
      }
    }
  })
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.nodes-container {
  padding: 20px;
}

@media (max-width: 767px) {
  .nodes-container { padding: 8px; }
}
</style>
