<template>
  <div class="custom-nodes-container admin-page-shell">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">专线节点管理</h2>
        <p class="page-subtitle">管理独享专线节点，支持按用户分配、批量导入、状态监控及过期自动管理</p>
      </div>
      <div class="header-right">
        <n-space>
          <n-input
            v-model:value="searchKeyword"
            clearable
            placeholder="搜索邮箱/域名/节点名称"
            style="width: 250px"
            @keyup.enter="handleSearch"
          >
            <template #prefix><n-icon :component="SearchOutline" /></template>
          </n-input>
          <n-dropdown
            trigger="click"
            :options="[
              { label: '批量分配', key: 'assign', icon: () => h(NIcon, null, { default: () => h(PeopleOutline) }), disabled: checkedRowKeys.length === 0 },
              { label: '批量删除', key: 'delete', icon: () => h(NIcon, null, { default: () => h(TrashOutline) }), disabled: checkedRowKeys.length === 0 }
            ]"
            @select="(key) => key === 'assign' ? handleBatchAssign() : handleBatchDelete()"
          >
            <n-button secondary :disabled="checkedRowKeys.length === 0">批量操作 ({{ checkedRowKeys.length }})</n-button>
          </n-dropdown>
          <n-button type="primary" @click="showImportDrawer = true">
            <template #icon><n-icon><CloudUploadOutline /></n-icon></template>
            导入链接
          </n-button>
          <n-button @click="handleResetSearch" secondary>
            <template #icon><n-icon><RefreshOutline /></n-icon></template>
            刷新
          </n-button>
        </n-space>
      </div>
    </div>

    <n-card :bordered="false" class="admin-main-card">
      <div v-if="appStore.isMobile" class="mobile-toolbar">
        <div class="mobile-toolbar-search">
          <n-input
            v-model:value="searchKeyword"
            clearable
            placeholder="搜索节点名称/域名"
            @keyup.enter="handleSearch"
          >
            <template #prefix><n-icon :component="SearchOutline" /></template>
          </n-input>
          <n-button type="info" @click="handleSearch">搜索</n-button>
        </div>
        <div class="mobile-toolbar-row">
          <n-button size="small" type="primary" @click="showImportDrawer = true">导入</n-button>
          <n-button size="small" secondary @click="handleResetSearch">刷新</n-button>
        </div>
      </div>

      <template v-if="!appStore.isMobile">
        <n-data-table
          class="unified-admin-table"
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
                <span>{{ row.expire_time ? formatFullDateTime(row.expire_time) : '-' }}</span>
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
          style="margin-top: 16px; justify-content: flex-end"
          @update:page="fetchData"
          @update:page-size="(ps) => { pagination.pageSize = ps; pagination.page = 1; fetchData() }"
        />
      </template>
    </n-card>

    <!-- Create/Edit Drawer -->
    <common-drawer
      v-model:show="showEditDrawer"
      :title="editId ? '编辑专线节点' : '创建专线节点'"
      :width="700"
      show-footer
      :loading="submitting"
      @confirm="handleSubmit"
      @cancel="showEditDrawer = false"
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
    </common-drawer>

    <!-- Assign Drawer -->
    <common-drawer
      v-model:show="showAssignDrawer"
      title="分配节点给用户"
      :width="600"
      show-footer
      :loading="assigning"
      @confirm="handleAssignSubmit"
      @cancel="showAssignDrawer = false"
    >
      <n-form label-placement="top">
        <n-form-item label="选择用户">
          <n-select
            v-model:value="assignUserIds"
            multiple
            remote
            filterable
            clearable
            placeholder="输入邮箱/用户名搜索并选择用户"
            :options="userOptions"
            :loading="loadingUsers"
            :show-arrow="true"
            @search="handleUserSearch"
          />
        </n-form-item>
        <n-form-item label="专线独立到期时间（可选）">
          <n-date-picker
            v-model:value="assignExpiresAt"
            type="datetime"
            clearable
            style="width: 100%"
            placeholder="不设置则跟随订阅到期时间"
          />
        </n-form-item>
        <n-form-item label="显示模式">
          <n-switch v-model:value="assignDedicatedOnly">
            <template #checked>
              只显示专线节点
            </template>
            <template #unchecked>
              显示全部节点
            </template>
          </n-switch>
        </n-form-item>
        <n-form-item label="限制设备数量">
          <n-switch v-model:value="assignLimitDevices">
            <template #checked>
              跟随系统限制
            </template>
            <template #unchecked>
              不限制设备数量
            </template>
          </n-switch>
        </n-form-item>
        <n-alert type="info" style="margin-top: 12px">
          <template v-if="assignDedicatedOnly && !assignLimitDevices">
            用户订阅<b>只显示专线节点</b>，且<b>不限制设备数量</b>。适合独享专线 VIP 用户。
          </template>
          <template v-else-if="assignDedicatedOnly && assignLimitDevices">
            用户订阅<b>只显示专线节点</b>，设备数量<b>跟随系统限制</b>。
          </template>
          <template v-else-if="!assignDedicatedOnly && !assignLimitDevices">
            专线节点<b>附加到公共节点列表</b>中，且<b>不限制设备数量</b>。
          </template>
          <template v-else>
            专线节点<b>附加到公共节点列表</b>中，设备数量<b>跟随系统限制</b>。
          </template>
        </n-alert>
      </n-form>
    </common-drawer>

    <!-- Import Links Drawer -->
    <common-drawer
      v-model:show="showImportDrawer"
      title="导入节点链接"
      :width="600"
      show-footer
      :loading="importing"
      @confirm="handleImportSubmit"
      @cancel="showImportDrawer = false"
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
    </common-drawer>

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
import { usePageLoading } from '@/composables/usePageLoading'
import { NButton, NTag, NSpace, NIcon, NSwitch, useMessage } from 'naive-ui'
import {
  CreateOutline,
  TrashOutline,
  PeopleOutline,
  CloudUploadOutline,
  LinkOutline,
  SearchOutline,
  RefreshOutline
} from '@vicons/ionicons5'
import {
  listCustomNodes,
  createCustomNode,
  updateCustomNode,
  deleteCustomNode,
  assignCustomNode,
  batchAssignCustomNodes,
  listUsers,
  importCustomNodeLinks,
  batchDeleteCustomNodes,
  getCustomNodeLink
} from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { copyToClipboard as clipboardCopy } from '@/utils/clipboard'
import { formatFullDateTime } from '@/utils/date'
import CommonDrawer from '@/components/CommonDrawer.vue'

const message = useMessage()
const appStore = useAppStore()

const { loading, beginLoad, endLoad } = usePageLoading()
const submitting = ref(false)
const assigning = ref(false)
const loadingUsers = ref(false)
const showEditDrawer = ref(false)
const showAssignDrawer = ref(false)
const tableData = ref([])
const formRef = ref(null)
const editId = ref(null)
const assignNodeId = ref(null)
const assignNodeIds = ref([])
const assignUserIds = ref([])
const assignExpiresAt = ref(null)
const assignDedicatedOnly = ref(false)
const assignLimitDevices = ref(false)
const userOptions = ref([])
const showImportDrawer = ref(false)
const showLinkModal = ref(false)
const importing = ref(false)
const importLinks = ref('')
const checkedRowKeys = ref([])
const linkData = reactive({ link: '', name: '', protocol: '' })
const sortState = ref({ sort: 'id', order: 'desc' })
const searchKeyword = ref('')

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
    render: (row) => row.expire_time ? formatFullDateTime(row.expire_time) : '-'
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

const fetchData = async () => {
  beginLoad(tableData.value.length > 0)
  try {
    const res = await listCustomNodes({
      page: pagination.page,
      page_size: pagination.pageSize,
      sort: sortState.value.sort,
      order: sortState.value.order,
      search: searchKeyword.value.trim()
    })
    tableData.value = res.data.items || []
    pagination.itemCount = res.data.total || 0
  } catch (error) {
    message.error(error.message || '获取专线节点列表失败')
  } finally {
    endLoad()
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

// 加载用户选项（支持远程搜索：有关键词调后端 search，无关键词加载默认前 50 个）
const fetchUsers = async (keyword = '') => {
  loadingUsers.value = true
  try {
    const params = { page: 1, page_size: 50 }
    if (String(keyword).trim()) params.search = String(keyword).trim()
    const res = await listUsers(params)
    const newOptions = (res.data.items || []).map(user => ({
      label: `${user.email}${user.username ? ' · ' + user.username : ''} (ID: ${user.id})`,
      value: user.id
    }))
    // 保留已选中的用户选项（远程搜索切换时已选的不消失）
    const selected = new Set(assignUserIds.value)
    const merged = [...userOptions.value.filter(o => selected.has(o.value)), ...newOptions]
    const seen = new Set()
    userOptions.value = merged.filter(o => (seen.has(o.value) ? false : seen.add(o.value)))
  } catch (error) {
    message.error(error.message || '获取用户列表失败')
  } finally {
    loadingUsers.value = false
  }
}

// 远程搜索用户（输入关键词触发，300ms 防抖）
let userSearchTimer = null
const handleUserSearch = (query) => {
  if (userSearchTimer) clearTimeout(userSearchTimer)
  userSearchTimer = setTimeout(() => fetchUsers(query), 300)
}

const handlePageChange = (page) => {
  pagination.page = page
  fetchData()
}

const handleSearch = () => {
  pagination.page = 1
  fetchData()
}

const handleResetSearch = () => {
  searchKeyword.value = ''
  pagination.page = 1
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
  showEditDrawer.value = true
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
  showEditDrawer.value = true
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    await formRef.value?.validate()

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

    showEditDrawer.value = false
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
    row.is_active = value // 本地更新，避免整表刷新卡顿
    message.success('更新状态成功')
  } catch (error) {
    message.error(error.message || '更新状态失败')
  }
}

const handleDelete = async (row) => {
  try {
    await deleteCustomNode(row.id)
    message.success('删除专线节点成功')
    fetchData()
  } catch (error) {
    message.error(error.message || '删除专线节点失败')
  }
}

const handleAssign = (row) => {
  assignNodeId.value = row.id
  assignNodeIds.value = [row.id]
  assignUserIds.value = []
  assignExpiresAt.value = null
  assignDedicatedOnly.value = false
  assignLimitDevices.value = false
  showAssignDrawer.value = true
  // 每次打开刷新默认用户列表（前 50 个），支持输入关键词远程搜索更多
  fetchUsers()
}

const handleBatchAssign = () => {
  if (checkedRowKeys.value.length === 0) return
  assignNodeId.value = null
  assignNodeIds.value = [...checkedRowKeys.value]
  assignUserIds.value = []
  assignExpiresAt.value = null
  assignDedicatedOnly.value = false
  assignLimitDevices.value = false
  showAssignDrawer.value = true
  // 每次打开刷新默认用户列表（前 50 个），支持输入关键词远程搜索更多
  fetchUsers()
}

const handleAssignSubmit = async () => {
  if (assignUserIds.value.length === 0) {
    message.warning('请至少选择一个用户')
    return
  }

  if (assignNodeIds.value.length === 0) {
    message.warning('请选择要分配的专线节点')
    return
  }

  const expiresAt = assignExpiresAt.value ? new Date(assignExpiresAt.value).toISOString() : null

  assigning.value = true
  try {
    if (assignNodeIds.value.length === 1) {
      await assignCustomNode(assignNodeIds.value[0], {
        user_ids: assignUserIds.value,
        expires_at: expiresAt,
        dedicated_only: assignDedicatedOnly.value,
        limit_devices: assignLimitDevices.value
      })
      message.success('分配节点成功')
    } else {
      const res = await batchAssignCustomNodes({
        ids: assignNodeIds.value,
        user_ids: assignUserIds.value,
        expires_at: expiresAt,
        dedicated_only: assignDedicatedOnly.value,
        limit_devices: assignLimitDevices.value
      })
      const successCount = res.data?.success || 0
      const totalCount = res.data?.total || assignNodeIds.value.length
      if (successCount !== totalCount) {
        message.warning(`部分分配成功：成功 ${successCount} 个，失败 ${totalCount - successCount} 个`)
      } else {
        message.success(`批量分配成功，共 ${successCount} 个节点`)
      }
    }

    showAssignDrawer.value = false
    checkedRowKeys.value = []
    assignNodeId.value = null
    assignNodeIds.value = []
    assignExpiresAt.value = null
    fetchData()
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
    showImportDrawer.value = false
    importLinks.value = ''
    fetchData()
  } catch (error) {
    message.error(error.message || '导入失败')
  } finally {
    importing.value = false
  }
}

const handleBatchDelete = async () => {
  if (checkedRowKeys.value.length === 0) return
  try {
    await batchDeleteCustomNodes({ ids: checkedRowKeys.value })
    message.success('批量删除成功')
    checkedRowKeys.value = []
    fetchData()
  } catch (error) {
    message.error(error.message || '批量删除失败')
  }
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
.desktop-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.desktop-toolbar-search {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
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
.mobile-toolbar-search { display: flex; flex-direction: column; gap: 8px; margin-bottom: 10px; }
.mobile-toolbar-search-actions { display: flex; gap: 8px; }
.mobile-toolbar-row { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
</style>
