<template>
  <div class="custom-nodes-container">
    <n-card :title="appStore.isMobile ? undefined : '专线节点管理'">
      <template v-if="!appStore.isMobile" #header-extra>
        <n-space>
          <n-button type="primary" @click="showImportModal = true">
            <template #icon><n-icon><CloudUploadOutline /></n-icon></template>
            导入链接
          </n-button>
          <n-button type="error" :disabled="checkedRowKeys.length === 0" @click="handleBatchDelete">
            批量删除 ({{ checkedRowKeys.length }})
          </n-button>
        </n-space>
      </template>

      <div v-if="appStore.isMobile" class="mobile-toolbar">
        <div class="mobile-toolbar-title">专线节点管理</div>
        <div class="mobile-toolbar-row">
          <n-button size="small" type="primary" @click="showImportModal = true">
            <template #icon><n-icon><CloudUploadOutline /></n-icon></template>
            导入链接
          </n-button>
          <n-button size="small" type="error" :disabled="checkedRowKeys.length === 0" @click="handleBatchDelete">
            批量删除 ({{ checkedRowKeys.length }})
          </n-button>
        </div>
      </div>

      <template v-if="!appStore.isMobile">
        <n-data-table
          remote
          :columns="columns"
          :data="tableData"
          :loading="loading"
          :pagination="pagination"
          :bordered="false"
          :row-key="(row) => row.id"
          v-model:checked-row-keys="checkedRowKeys"
          @update:sorter="handleSorterChange"
          @update:page="(p) => { pagination.page = p; fetchData() }"
          @update:page-size="(ps) => { pagination.pageSize = ps; pagination.page = 1; fetchData() }"
        />
      </template>

      <template v-else>
        <div class="mobile-card-list">
          <div v-for="row in tableData" :key="row.id" class="mobile-card">
            <div class="card-header">
              <span class="card-title">{{ row.display_name }}</span>
              <n-tag :type="protocolColorMap[row.protocol] || 'default'" size="small">
                {{ row.protocol.toUpperCase() }}
              </n-tag>
            </div>
            <div class="card-body">
              <div class="card-row">
                <span class="card-label">节点名称</span>
                <span>{{ row.name }}</span>
              </div>
              <div class="card-row">
                <span class="card-label">服务器</span>
                <span>{{ row.domain }}:{{ row.port }}</span>
              </div>
              <div class="card-row">
                <span class="card-label">状态</span>
                <n-switch :value="row.is_active" @update:value="(value) => handleToggleActive(row, value)" />
              </div>
              <div class="card-row">
                <span class="card-label">过期时间</span>
                <span>{{ row.expire_time ? formatDate(row.expire_time) : '-' }}</span>
              </div>
            </div>
            <div class="card-actions">
              <n-button size="small" type="primary" @click="handleEdit(row)">
                <template #icon><n-icon><CreateOutline /></n-icon></template>
                编辑
              </n-button>
              <n-button size="small" type="info" @click="handleAssign(row)">
                <template #icon><n-icon><PeopleOutline /></n-icon></template>
                分配
              </n-button>
              <n-button size="small" @click="handleViewLink(row)">
                <template #icon><n-icon><LinkOutline /></n-icon></template>
                链接
              </n-button>
              <n-button size="small" type="error" @click="handleDelete(row)">
                <template #icon><n-icon><TrashOutline /></n-icon></template>
                删除
              </n-button>
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
          @update:page="fetchData"
          @update:page-size="(ps) => { pagination.pageSize = ps; pagination.page = 1; fetchData() }"
        />
      </template>
    </n-card>

    <!-- Create/Edit Modal -->
    <n-modal
      v-model:show="showEditModal"
      :title="editId ? '编辑专线节点' : '创建专线节点'"
      preset="card"
      :style="appStore.isMobile ? 'width: 95vw; max-width: 700px' : 'width: 700px'"
      :mask-closable="false"
    >
      <n-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-placement="left"
        label-width="120"
      >
        <n-form-item label="节点名称" path="name">
          <n-input v-model:value="formData.name" placeholder="请输入节点名称（内部标识）" />
        </n-form-item>

        <n-form-item label="显示名称" path="display_name">
          <n-input v-model:value="formData.display_name" placeholder="请输入显示名称（用户可见）" />
        </n-form-item>

        <n-form-item label="协议" path="protocol">
          <n-select
            v-model:value="formData.protocol"
            placeholder="请选择协议"
            :options="protocolOptions"
          />
        </n-form-item>

        <n-form-item label="域名/IP" path="domain">
          <n-input v-model:value="formData.domain" placeholder="请输入域名或IP地址" />
        </n-form-item>

        <n-form-item label="端口" path="port">
          <n-input-number v-model:value="formData.port" :min="1" :max="65535" style="width: 100%" placeholder="请输入端口号" />
        </n-form-item>

        <n-form-item label="配置信息" path="config">
          <n-input
            v-model:value="formData.config"
            type="textarea"
            placeholder="请输入节点配置信息（JSON格式）"
            :rows="6"
          />
        </n-form-item>

        <n-form-item label="启用状态" path="is_active">
          <n-switch v-model:value="formData.is_active" />
        </n-form-item>

        <n-form-item label="过期时间" path="expire_time">
          <n-date-picker
            v-model:value="formData.expire_time"
            type="datetime"
            clearable
            style="width: 100%"
            placeholder="选择过期时间（可选）"
          />
        </n-form-item>

        <n-form-item label="跟随用户过期" path="follow_user_expire">
          <n-switch v-model:value="formData.follow_user_expire" />
          <n-text depth="3" style="margin-left: 12px; font-size: 12px">
            启用后，节点将在用户订阅过期时自动失效
          </n-text>
        </n-form-item>
      </n-form>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 12px">
          <n-button @click="showEditModal = false">取消</n-button>
          <n-button type="primary" @click="handleSubmit" :loading="submitting">确定</n-button>
        </div>
      </template>
    </n-modal>

    <!-- Assign Modal -->
    <n-modal
      v-model:show="showAssignModal"
      title="分配节点给用户"
      preset="card"
      :style="appStore.isMobile ? 'width: 95vw; max-width: 600px' : 'width: 600px'"
      :mask-closable="false"
    >
      <n-form label-placement="top">
        <n-form-item label="选择用户">
          <n-select
            v-model:value="assignUserIds"
            multiple
            filterable
            placeholder="请选择要分配的用户"
            :options="userOptions"
            :loading="loadingUsers"
          />
        </n-form-item>
        <n-alert type="info" style="margin-top: 12px">
          选中的用户将可以使用此专线节点
        </n-alert>
      </n-form>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 12px">
          <n-button @click="showAssignModal = false">取消</n-button>
          <n-button type="primary" @click="handleAssignSubmit" :loading="assigning">确定</n-button>
        </div>
      </template>
    </n-modal>

    <!-- Import Links Modal -->
    <n-modal
      v-model:show="showImportModal"
      title="导入节点链接"
      preset="card"
      :style="appStore.isMobile ? 'width: 95vw; max-width: 600px' : 'width: 600px'"
      :mask-closable="false"
    >
      <n-form label-placement="top">
        <n-form-item label="节点链接">
          <n-input
            v-model:value="importLinks"
            type="textarea"
            placeholder="每行一个节点链接，支持 vmess:// vless:// trojan:// ss://"
            :rows="8"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 12px">
          <n-button @click="showImportModal = false">取消</n-button>
          <n-button type="primary" @click="handleImportSubmit" :loading="importing">导入</n-button>
        </div>
      </template>
    </n-modal>

    <!-- View Link Modal -->
    <n-modal
      v-model:show="showLinkModal"
      title="节点链接"
      preset="card"
      :style="appStore.isMobile ? 'width: 95vw; max-width: 600px' : 'width: 600px'"
    >
      <n-form label-placement="top">
        <n-form-item :label="linkData.name">
          <n-input
            :value="linkData.link"
            type="textarea"
            readonly
            :rows="4"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 12px">
          <n-button @click="handleCopyLink">复制链接</n-button>
          <n-button @click="showLinkModal = false">关闭</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, reactive, h, onMounted } from 'vue'
import { NButton, NTag, NSpace, NIcon, NSwitch, useMessage, useDialog } from 'naive-ui'
import {
  CreateOutline,
  TrashOutline,
  PeopleOutline,
  CloudUploadOutline,
  LinkOutline
} from '@vicons/ionicons5'
import {
  listCustomNodes,
  createCustomNode,
  updateCustomNode,
  deleteCustomNode,
  assignCustomNode,
  listUsers,
  importCustomNodeLinks,
  batchDeleteCustomNodes,
  getCustomNodeLink
} from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { copyToClipboard as clipboardCopy } from '@/utils/clipboard'

const message = useMessage()
const dialog = useDialog()
const appStore = useAppStore()

const loading = ref(false)
const submitting = ref(false)
const assigning = ref(false)
const loadingUsers = ref(false)
const showEditModal = ref(false)
const showAssignModal = ref(false)
const tableData = ref([])
const formRef = ref(null)
const editId = ref(null)
const assignNodeId = ref(null)
const assignUserIds = ref([])
const userOptions = ref([])
const showImportModal = ref(false)
const showLinkModal = ref(false)
const importing = ref(false)
const importLinks = ref('')
const checkedRowKeys = ref([])
const linkData = reactive({ link: '', name: '', protocol: '' })
const sortState = ref({ sort: 'id', order: 'desc' })

const formData = reactive({
  name: '',
  display_name: '',
  protocol: 'vmess',
  domain: '',
  port: 443,
  config: '',
  is_active: true,
  expire_time: null,
  follow_user_expire: false
})

const rules = {
  name: { required: true, message: '请输入节点名称', trigger: 'blur' },
  display_name: { required: true, message: '请输入显示名称', trigger: 'blur' },
  protocol: { required: true, message: '请选择协议', trigger: 'change' },
  domain: { required: true, message: '请输入域名或IP', trigger: 'blur' },
  port: { required: true, type: 'number', message: '请输入端口号', trigger: 'blur' }
}

const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
})

const protocolOptions = [
  { label: 'VMess', value: 'vmess' },
  { label: 'VLESS', value: 'vless' },
  { label: 'Trojan', value: 'trojan' },
  { label: 'Shadowsocks', value: 'ss' },
  { label: 'Hysteria2', value: 'hysteria2' }
]

const protocolColorMap = {
  vmess: 'info',
  vless: 'success',
  trojan: 'warning',
  ss: 'default',
  hysteria2: 'error'
}

const columns = [
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 80, resizable: true, sorter: 'default' },
  { title: '节点名称', key: 'name', ellipsis: { tooltip: true }, minWidth: 150 },
  { title: '显示名称', key: 'display_name', ellipsis: { tooltip: true }, minWidth: 150 },
  {
    title: '协议',
    key: 'protocol',
    width: 120,
    resizable: true,
    render: (row) => {
      const type = protocolColorMap[row.protocol] || 'default'
      return h(NTag, { type }, { default: () => row.protocol.toUpperCase() })
    }
  },
  { title: '域名', key: 'domain', ellipsis: { tooltip: true }, minWidth: 180 },
  { title: '端口', key: 'port', width: 100, resizable: true },
  {
    title: '状态',
    key: 'is_active',
    width: 100,
    resizable: true,
    render: (row) => {
      return h(NSwitch, {
        value: row.is_active,
        onUpdateValue: (value) => handleToggleActive(row, value)
      })
    }
  },
  {
    title: '过期时间',
    key: 'expire_time',
    width: 160,
    resizable: true,
    render: (row) => row.expire_time ? formatDate(row.expire_time) : '-'
  },
  {
    title: '操作',
    key: 'actions',
    width: 280,
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
            type: 'info',
            text: true,
            onClick: () => handleAssign(row)
          }, { default: () => '分配', icon: () => h(NIcon, {}, { default: () => h(PeopleOutline) }) }),
          h(NButton, {
            size: 'small',
            text: true,
            onClick: () => handleViewLink(row)
          }, { default: () => '链接', icon: () => h(NIcon, {}, { default: () => h(LinkOutline) }) }),
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

const formatDate = (date) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
}

const fetchData = async () => {
  loading.value = true
  try {
    const res = await listCustomNodes({
      page: pagination.page,
      page_size: pagination.pageSize,
      sort: sortState.value.sort,
      order: sortState.value.order,
    })
    tableData.value = res.data.items || []
    pagination.itemCount = res.data.total || 0
  } catch (error) {
    message.error(error.message || '获取专线节点列表失败')
  } finally {
    loading.value = false
  }
}

const handleSorterChange = (sorter) => {
  if (sorter && sorter.columnKey && sorter.order) {
    sortState.value.sort = sorter.columnKey
    sortState.value.order = sorter.order === 'ascend' ? 'asc' : 'desc'
  } else {
    sortState.value.sort = 'id'
    sortState.value.order = 'desc'
  }
  pagination.page = 1
  fetchData()
}

const fetchUsers = async () => {
  loadingUsers.value = true
  try {
    const res = await listUsers({ page: 1, page_size: 1000 })
    userOptions.value = (res.data.items || []).map(user => ({
      label: `${user.email} (ID: ${user.id})`,
      value: user.id
    }))
  } catch (error) {
    message.error(error.message || '获取用户列表失败')
  } finally {
    loadingUsers.value = false
  }
}

const handlePageChange = (page) => {
  pagination.page = page
  fetchData()
}

const resetForm = () => {
  Object.assign(formData, {
    name: '',
    display_name: '',
    protocol: 'vmess',
    domain: '',
    port: 443,
    config: '',
    is_active: true,
    expire_time: null,
    follow_user_expire: false
  })
  formRef.value?.restoreValidation()
}

const handleCreate = () => {
  editId.value = null
  resetForm()
  showEditModal.value = true
}

const handleEdit = (row) => {
  editId.value = row.id
  Object.assign(formData, {
    name: row.name,
    display_name: row.display_name,
    protocol: row.protocol,
    domain: row.domain,
    port: row.port,
    config: row.config || '',
    is_active: row.is_active,
    expire_time: row.expire_time ? new Date(row.expire_time).getTime() : null,
    follow_user_expire: row.follow_user_expire || false
  })
  showEditModal.value = true
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
    submitting.value = true

    const data = {
      ...formData,
      expire_time: formData.expire_time ? new Date(formData.expire_time).toISOString() : null
    }

    if (editId.value) {
      await updateCustomNode(editId.value, data)
      message.success('更新专线节点成功')
    } else {
      await createCustomNode(data)
      message.success('创建专线节点成功')
    }

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
    await updateCustomNode(row.id, { is_active: value })
    message.success('更新状态成功')
    fetchData()
  } catch (error) {
    message.error(error.message || '更新状态失败')
  }
}

const handleDelete = (row) => {
  dialog.warning({
    title: '确认删除',
    content: `确定要删除专线节点 "${row.display_name}" 吗？此操作不可恢复。`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteCustomNode(row.id)
        message.success('删除专线节点成功')
        fetchData()
      } catch (error) {
        message.error(error.message || '删除专线节点失败')
      }
    }
  })
}

const handleAssign = (row) => {
  assignNodeId.value = row.id
  assignUserIds.value = []
  showAssignModal.value = true
  if (userOptions.value.length === 0) {
    fetchUsers()
  }
}

const handleAssignSubmit = async () => {
  if (assignUserIds.value.length === 0) {
    message.warning('请至少选择一个用户')
    return
  }

  assigning.value = true
  try {
    await assignCustomNode(assignNodeId.value, {
      user_ids: assignUserIds.value
    })
    message.success('分配节点成功')
    showAssignModal.value = false
  } catch (error) {
    message.error(error.message || '分配节点失败')
  } finally {
    assigning.value = false
  }
}

const handleImportSubmit = async () => {
  if (!importLinks.value.trim()) {
    message.warning('请输入节点链接')
    return
  }
  importing.value = true
  try {
    const res = await importCustomNodeLinks({ links: importLinks.value })
    message.success(`导入完成: 成功 ${res.data.success}/${res.data.total} 个`)
    showImportModal.value = false
    importLinks.value = ''
    fetchData()
  } catch (error) {
    message.error(error.message || '导入失败')
  } finally {
    importing.value = false
  }
}

const handleBatchDelete = () => {
  if (checkedRowKeys.value.length === 0) return
  dialog.warning({
    title: '确认批量删除',
    content: `确定要删除选中的 ${checkedRowKeys.value.length} 个专线节点吗？此操作不可恢复。`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await batchDeleteCustomNodes({ ids: checkedRowKeys.value })
        message.success('批量删除成功')
        checkedRowKeys.value = []
        fetchData()
      } catch (error) {
        message.error(error.message || '批量删除失败')
      }
    }
  })
}

const handleViewLink = async (row) => {
  try {
    const res = await getCustomNodeLink(row.id)
    Object.assign(linkData, res.data)
    showLinkModal.value = true
  } catch (error) {
    message.error(error.message || '获取链接失败')
  }
}

const handleCopyLink = async () => {
  if (linkData.link) {
    const ok = await clipboardCopy(linkData.link)
    ok ? message.success('链接已复制') : message.error('复制失败')
  }
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.custom-nodes-container {
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
  .custom-nodes-container { padding: 8px; }
}
.mobile-toolbar { margin-bottom: 12px; }
.mobile-toolbar-title { font-size: 17px; font-weight: 600; margin-bottom: 10px; color: var(--text-color, #333); }
.mobile-toolbar-row { display: flex; gap: 8px; align-items: center; }
</style>
