<template>
  <div class="admin-users-page admin-page-shell">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">用户管理</h2>
        <p class="page-subtitle">管理系统所有用户，支持高级搜索、余额调整、等级设置及批量操作</p>
      </div>
      <div class="header-right">
        <n-space>
          <n-input
            v-model:value="searchQuery"
            placeholder="搜索邮箱/用户名/订阅地址"
            clearable
            class="search-input"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <n-icon :component="SearchOutline" />
            </template>
          </n-input>
          <n-select
            v-model:value="statusFilter"
            placeholder="状态"
            clearable
            class="status-select"
            :options="statusOptions"
            @update:value="handleSearch"
          />
          <n-button type="primary" @click="openCreateModal">
            <template #icon><n-icon :component="AddOutline" /></template>
            新增用户
          </n-button>
          <n-button @click="fetchUsers" secondary>
            <template #icon><n-icon :component="RefreshOutline" /></template>
            刷新
          </n-button>
        </n-space>
      </div>
    </div>

    <n-card :bordered="false" class="page-card admin-main-card">
      <n-space vertical :size="16">

        <!-- Mobile Header -->
        <div v-if="appStore.isMobile" class="mobile-toolbar">
          <div class="mobile-toolbar-title">用户管理</div>
          <div class="mobile-toolbar-controls">
            <n-input
              v-model:value="searchQuery"
              placeholder="搜索邮箱或用户名"
              clearable
              size="small"
              @keyup.enter="handleSearch"
            >
              <template #prefix>
                <n-icon :component="SearchOutline" />
              </template>
            </n-input>
            <div class="mobile-toolbar-row">
              <n-select
                v-model:value="statusFilter"
                placeholder="状态筛选"
                clearable
                size="small"
                class="flex-1"
                :options="statusOptions"
                @update:value="handleSearch"
              />
              <n-button size="small" type="info" @click="handleSearch">
                <template #icon><n-icon :component="SearchOutline" /></template>
              </n-button>
              <n-button size="small" @click="fetchUsers">
                <template #icon><n-icon :component="RefreshOutline" /></template>
              </n-button>
            </div>
            <div class="mobile-toolbar-row">
              <n-button size="small" type="primary" @click="openCreateModal">
                <template #icon><n-icon :component="AddOutline" /></template>
                新增
              </n-button>
              <n-button size="small" @click="handleExportCSV">
                <template #icon><n-icon :component="DownloadOutline" /></template>
                导出
              </n-button>
              <n-button size="small" @click="triggerImportCSV">
                <template #icon><n-icon :component="CloudUploadOutline" /></template>
                导入
              </n-button>
              <input ref="importFileInput" type="file" accept=".csv" class="hidden-input" @change="handleImportCSV" />
            </div>
          </div>
        </div>

        <!-- Batch operations -->
        <n-space v-if="checkedRowKeys.length > 0" align="center">
          <span class="batch-selected-text">已选择 {{ checkedRowKeys.length }} 项</span>
          <n-button size="small" type="success" @click="handleBatchEnable">批量启用</n-button>
          <n-button size="small" type="warning" @click="handleBatchDisable">批量禁用</n-button>
          <n-popconfirm @positive-click="doBatchDelete">
            <template #trigger>
              <n-button size="small" type="error">批量删除</n-button>
            </template>
            确定要删除选中的 {{ checkedRowKeys.length }} 个用户吗？此操作不可恢复！
          </n-popconfirm>
          <n-button size="small" @click="openSetLevelModal">设置等级</n-button>
        </n-space>

        <!-- Data table (Desktop) -->
        <template v-if="!appStore.isMobile">
          <n-data-table
            class="unified-admin-table"
            :columns="columns"
            :data="users"
            :loading="loading"
            :pagination="false"
            :bordered="false"
            :single-line="false"
            :row-key="(row) => row.id"
            :checked-row-keys="checkedRowKeys"
            :row-props="getRowProps"
            @update:checked-row-keys="handleCheck"
          />
        </template>

        <!-- Mobile card layout -->
        <template v-else>
          <div v-if="loading" class="loading-center">
            <n-spin size="medium" />
          </div>
          <div v-else-if="users.length === 0" class="empty-state">
            暂无数据
          </div>
          <div v-else class="mobile-card-list">
            <div v-for="row in users" :key="row.id" class="mobile-card">
              <div class="card-header">
                <div class="card-title">{{ row.username }}</div>
                <n-space :size="4">
                  <n-tag :type="row.is_active ? 'success' : 'error'" size="small">
                    {{ row.is_active ? '激活' : '禁用' }}
                  </n-tag>
                  <n-tag v-if="row.is_admin" type="warning" size="small">管理员</n-tag>
                </n-space>
              </div>
              <div class="card-body">
                <div class="card-row">
                  <span class="card-label">邮箱</span>
                  <span>{{ row.email }}</span>
                </div>
                <div class="card-row">
                  <span class="card-label">余额</span>
                  <span>{{ formatCurrency(row.balance) }}</span>
                </div>
                <div class="card-row">
                  <span class="card-label">等级</span>
                  <span>{{ row.level_name || row.level || '无' }}</span>
                </div>
                <div class="card-row">
                  <span class="card-label">注册时间</span>
                  <span>{{ row.created_at ? new Date(row.created_at).toLocaleString('zh-CN') : '-' }}</span>
                </div>
              </div>
              <div class="card-actions">
                <n-button size="small" @click="handleViewDetail(row)">详情</n-button>
                <n-button size="small" @click="handleEdit(row)">编辑</n-button>
                <n-button size="small" @click="handleToggleActive(row)">
                  {{ row.is_active ? '禁用' : '启用' }}
                </n-button>
                <n-dropdown
                  trigger="click"
                  :options="[
                    { label: '重置密码', key: 'resetPwd' },
                    { label: '删除', key: 'delete' }
                  ]"
                  @select="(key) => handleAction(key, row)"
                >
                  <n-button size="small" quaternary>
                    <template #icon>
                      <n-icon :component="EllipsisVertical" />
                    </template>
                  </n-button>
                </n-dropdown>
              </div>
            </div>
          </div>
        </template>

        <n-pagination
          v-model:page="currentPage"
          v-model:page-size="pageSize"
          :page-count="totalPages"
          :page-sizes="[10, 20, 50, 100]"
          show-size-picker
          style="margin-top: 16px; justify-content: flex-end"
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </n-space>
    </n-card>

    <!-- Create/Edit User Drawer -->
    <common-drawer
      v-model:show="showEditDrawer"
      :title="isCreate ? '新增用户' : '编辑用户'"
      :width="520"
      show-footer
      :loading="saving"
      @confirm="handleSaveUser"
      @cancel="showEditDrawer = false"
    >
      <n-form ref="formRef" :model="editForm" :rules="formRulesComputed" label-placement="left" label-width="80" class="form-spacing">
        <n-form-item label="用户名" path="username">
          <n-input v-model:value="editForm.username" placeholder="请输入用户名" />
        </n-form-item>
        <n-form-item label="邮箱" path="email">
          <n-input v-model:value="editForm.email" placeholder="请输入邮箱" />
        </n-form-item>
        <n-form-item v-if="isCreate" label="密码" path="password">
          <n-input v-model:value="editForm.password" type="password" show-password-on="click" placeholder="请输入密码" />
        </n-form-item>
        <n-form-item label="余额" path="balance">
          <n-input-number v-model:value="editForm.balance" placeholder="请输入余额" :min="0" :precision="2" class="full-width" />
        </n-form-item>
        <n-form-item label="管理员" path="is_admin">
          <n-switch v-model:value="editForm.is_admin" />
        </n-form-item>
        <n-form-item label="启用" path="is_active">
          <n-switch v-model:value="editForm.is_active" />
        </n-form-item>
        <n-form-item label="到期时间" path="expire_time">
          <n-date-picker
            v-model:value="editForm.expire_time"
            type="datetime"
            clearable
            class="full-width"
            placeholder="选择到期时间"
          />
        </n-form-item>
        <n-form-item label="设备数量" path="device_limit">
          <n-space vertical class="full-width">
            <n-input-number
              v-model:value="editForm.device_limit"
              :min="1"
              :max="1000"
              class="full-width"
              placeholder="设备数量限制"
            />
            <n-space :size="8">
              <n-button size="tiny" secondary type="info" @click="editForm.device_limit = (editForm.device_limit || 0) + 1">+1</n-button>
              <n-button size="tiny" secondary type="info" @click="editForm.device_limit = (editForm.device_limit || 0) + 2">+2</n-button>
              <n-button size="tiny" secondary type="info" @click="editForm.device_limit = (editForm.device_limit || 0) + 5">+5</n-button>
              <n-button size="tiny" secondary type="info" @click="editForm.device_limit = (editForm.device_limit || 0) + 10">+10</n-button>
            </n-space>
          </n-space>
        </n-form-item>
        <n-form-item label="备注" path="notes">
          <n-input v-model:value="editForm.notes" type="textarea" placeholder="备注信息" :rows="3" />
        </n-form-item>
      </n-form>
    </common-drawer>

    <UserDetailDrawer ref="userDetailDrawerRef" />

    <!-- Reset Password Drawer -->
    <common-drawer
      v-model:show="showResetPwdDrawer"
      title="重置密码"
      :width="420"
      show-footer
      :loading="resettingPwd"
      @confirm="handleResetPassword"
      @cancel="showResetPwdDrawer = false"
    >
      <n-form ref="resetPwdFormRef" :model="resetPwdForm" :rules="resetPwdRules" label-placement="left" label-width="80">
        <n-form-item label="新密码" path="password">
          <n-input v-model:value="resetPwdForm.password" type="password" show-password-on="click" placeholder="请输入新密码" />
        </n-form-item>
      </n-form>
    </common-drawer>

    <!-- Set Level Modal -->
    <n-modal v-model:show="showSetLevelModal" preset="card" title="设置用户等级" :style="{ width: appStore.isMobile ? '95%' : '420px' }">
      <n-form label-placement="left" label-width="80">
        <n-form-item label="等级">
          <n-select
            v-model:value="selectedLevelId"
            placeholder="请选择等级"
            :options="userLevels.map(l => ({ label: l.level_name || l.name, value: l.id }))"
            clearable
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showSetLevelModal = false">取消</n-button>
          <n-button type="primary" @click="handleBatchSetLevel">确认</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Import Result Modal -->
    <n-modal v-model:show="showImportResultModal" preset="card" title="导入结果" :style="{ width: appStore.isMobile ? '95%' : '520px' }">
      <n-space vertical :size="12">
        <n-space>
          <n-tag type="info">总计: {{ importResult.total }}</n-tag>
          <n-tag type="success">导入: {{ importResult.imported }}</n-tag>
          <n-tag type="warning">跳过: {{ importResult.skipped }}</n-tag>
        </n-space>
        <div v-if="importResult.errors && importResult.errors.length > 0">
          <div style="font-weight: 600; margin-bottom: 4px;">错误详情:</div>
          <n-scrollbar style="max-height: 200px">
            <div v-for="(err, idx) in importResult.errors" :key="idx" style="font-size: 13px; color: #d03050; padding: 2px 0;">
              {{ err }}
            </div>
          </n-scrollbar>
        </div>
      </n-space>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showImportResultModal = false">关闭</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, reactive, h, onMounted, computed } from 'vue'
import { NButton, NTag, NSpace, NIcon, NDropdown, NSpin, useMessage, useDialog } from 'naive-ui'
import { SearchOutline, AddOutline, RefreshOutline, EllipsisVertical, DownloadOutline, CloudUploadOutline } from '@vicons/ionicons5'
import {
  listUsers, updateUser, deleteUser, toggleUserActive,
  createUser, resetUserPassword,
  batchUserAction, exportUsersCSV, importUsersCSV, loginAsUser
} from '@/api/admin'
import { listUserLevels } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'
import { useRoute } from 'vue-router'
import { formatCurrency } from '@/utils/amount'
import CommonDrawer from '@/components/CommonDrawer.vue'
import UserDetailDrawer from './components/UserDetailDrawer.vue'
import '@/styles/admin-common.css'

const message = useMessage()
const dialog = useDialog()
const appStore = useAppStore()
const userStore = useUserStore()
const route = useRoute()

// State
const loading = ref(false)
const saving = ref(false)
const resettingPwd = ref(false)
const users = ref([])
const searchQuery = ref('')
const statusFilter = ref(null)
const currentPage = ref(1)
const pageSize = ref(10)
const totalPages = ref(0)
const checkedRowKeys = ref([])

// Import/Export
const importFileInput = ref(null)
const importing = ref(false)
const showImportResultModal = ref(false)
const importResult = ref({ total: 0, imported: 0, skipped: 0, errors: [] })

// Levels for batch set_level
const userLevels = ref([])
const showSetLevelModal = ref(false)
const selectedLevelId = ref(null)

// Modals
const showEditDrawer = ref(false)
const showResetPwdDrawer = ref(false)
const isCreate = ref(false)
const formRef = ref(null)
const resetPwdFormRef = ref(null)
const resetPwdTargetId = ref(null)
const userDetailDrawerRef = ref(null)

const editForm = reactive({
  id: null,
  username: '',
  email: '',
  password: '',
  balance: 0,
  is_admin: false,
  is_active: true,
  notes: '',
  expire_time: null,
  device_limit: 5
})

const resetPwdForm = reactive({ password: '' })

const statusOptions = [
  { label: '全部', value: null },
  { label: '激活', value: 'active' },
  { label: '禁用', value: 'inactive' },
  { label: '管理员', value: 'admin' }
]
const formRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效的邮箱地址', trigger: 'blur' }
  ],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  balance: [{ required: true, message: '请输入余额', trigger: 'blur', type: 'number' }]
}

const resetPwdRules = {
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' }
  ]
}

const formRulesComputed = computed(() => {
  const rules = { ...formRules }
  if (!isCreate.value) {
    delete rules.password
  }
  return rules
})

// Action dropdown options builder
const actionOptions = (row) => [
  { label: '查看详情', key: 'detail' },
  { label: '编辑', key: 'edit' },
  { label: row.is_active ? '禁用' : '启用', key: 'toggle' },
  { label: '重置密码', key: 'resetPwd' },
  { label: '删除', key: 'delete' }
]
// Table columns
const columns = [
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 70, sorter: 'default', resizable: true },
  { title: '用户名', key: 'username', ellipsis: { tooltip: true }, width: 130, resizable: true },
  { title: '邮箱', key: 'email', ellipsis: { tooltip: true }, width: 200, resizable: true },
  {
    title: '余额',
    key: 'balance',
    width: 100, resizable: true,
    sorter: (a, b) => a.balance - b.balance,
    render: (row) => formatCurrency(row.balance)
  },
  {
    title: '状态',
    key: 'is_active',
    width: 80, resizable: true,
    render: (row) => h(NTag, { type: row.is_active ? 'success' : 'error', size: 'small' }, { default: () => row.is_active ? '激活' : '禁用' })
  },
  {
    title: '注册时间',
    key: 'created_at',
    width: 170, resizable: true,
    sorter: (a, b) => new Date(a.created_at || 0) - new Date(b.created_at || 0),
    render: (row) => row.created_at ? new Date(row.created_at).toLocaleString('zh-CN') : '-'
  },
  {
    title: '最后登录',
    key: 'last_login',
    width: 170, resizable: true,
    sorter: (a, b) => new Date(a.last_login || 0) - new Date(b.last_login || 0),
    render: (row) => row.last_login ? new Date(row.last_login).toLocaleString('zh-CN') : '-'
  },
  {
    title: '操作',
    key: 'actions',
    width: 210,
    fixed: 'right',
    render: (row) => h('div', { class: 'action-btn-grid' }, [
      h(NButton, { size: 'small', secondary: true, type: 'info', onClick: () => handleAction('detail', row) }, { default: () => '详情' }),
      h(NButton, { size: 'small', type: 'primary', onClick: () => handleAction('edit', row) }, { default: () => '编辑' }),
      h(NButton, { size: 'small', type: row.is_active ? 'warning' : 'success', onClick: () => handleAction('toggle', row) }, { default: () => row.is_active ? '禁用' : '启用' }),
      h(NButton, { size: 'small', secondary: true, type: 'warning', onClick: () => handleAction('resetPwd', row) }, { default: () => '重置' }),
      h(NButton, { size: 'small', type: 'success', onClick: () => handleAction('loginAs', row) }, { default: () => '代登' }),
      h(NButton, { size: 'small', type: 'error', onClick: () => handleAction('delete', row) }, { default: () => '删除' }),
    ])
  }
]
// Fetch users
const fetchUsers = async () => {
  loading.value = true
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value,
      search: searchQuery.value || undefined,
      is_active: statusFilter.value === 'active' ? true : statusFilter.value === 'inactive' ? false : undefined,
      is_admin: statusFilter.value === 'admin' ? true : undefined
    }
    const response = await listUsers(params)
    users.value = response.data.items || []
    totalPages.value = Math.ceil((response.data.total || 0) / pageSize.value)
  } catch (error) {
    message.error('获取用户列表失败：' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

const handleSearch = () => { currentPage.value = 1; fetchUsers() }
const handlePageChange = (page) => { currentPage.value = page; fetchUsers() }
const handlePageSizeChange = (size) => { pageSize.value = size; currentPage.value = 1; fetchUsers() }
const handleCheck = (keys) => { checkedRowKeys.value = keys }

const getRowProps = (row) => ({
  style: 'cursor: pointer',
  onClick: (e) => {
    if (e.target.closest('.n-button, .n-dropdown, .n-checkbox')) return
    handleViewDetail(row)
  }
})

// Action dispatcher
const handleAction = (key, row) => {
  switch (key) {
    case 'detail': handleViewDetail(row); break
    case 'edit': handleEdit(row); break
    case 'toggle': handleToggleActive(row); break
    case 'resetPwd': openResetPwdModal(row); break
    case 'delete': handleDelete(row); break
    case 'loginAs': handleLoginAs(row); break
  }
}

// Create / Edit
const resetEditForm = () => {
  editForm.id = null
  editForm.username = ''
  editForm.email = ''
  editForm.password = ''
  editForm.balance = 0
  editForm.is_admin = false
  editForm.is_active = true
  editForm.notes = ''
  // 默认到期时间延长一年
  const oneYearLater = new Date()
  oneYearLater.setFullYear(oneYearLater.getFullYear() + 1)
  editForm.expire_time = oneYearLater.getTime()
  // 默认设备数量5个
  editForm.device_limit = 5
}

const openCreateModal = () => {
  resetEditForm()
  isCreate.value = true
  showEditDrawer.value = true
}

const handleEdit = (row) => {
  resetEditForm()
  isCreate.value = false
  editForm.id = row.id
  editForm.username = row.username
  editForm.email = row.email
  editForm.balance = row.balance
  editForm.is_admin = row.is_admin
  editForm.is_active = row.is_active
  editForm.notes = row.notes || ''
  editForm.expire_time = row.expire_time ? new Date(row.expire_time).getTime() : null
  editForm.device_limit = row.device_limit || 5
  showEditDrawer.value = true
}

// Save user
const handleSaveUser = async () => {
  try {
    await formRef.value?.validate()
  } catch { return }
  saving.value = true
  try {
    const userData = {
      username: editForm.username,
      email: editForm.email,
      balance: editForm.balance,
      is_admin: editForm.is_admin,
      is_active: editForm.is_active,
      notes: editForm.notes,
      expire_time: editForm.expire_time ? new Date(editForm.expire_time).toISOString() : null,
      device_limit: editForm.device_limit
    }

    if (isCreate.value) {
      userData.password = editForm.password
      await createUser(userData)
      message.success('用户创建成功')
    } else {
      await updateUser(editForm.id, userData)
      message.success('用户更新成功')
    }
    showEditDrawer.value = false
    fetchUsers()
  } catch (error) {
    message.error((isCreate.value ? '创建' : '更新') + '用户失败：' + (error.message || '未知错误'))
  } finally {
    saving.value = false
  }
}

// Toggle active
const handleToggleActive = (row) => {
  if (row.is_active && row.id === userStore.userInfo?.id) {
    message.error('不能禁用自己')
    return
  }
  dialog.warning({
    title: '确认操作',
    content: `确定要${row.is_active ? '禁用' : '启用'}用户 ${row.username} 吗？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await toggleUserActive(row.id)
        message.success(`用户已${row.is_active ? '禁用' : '启用'}`)
        fetchUsers()
      } catch (error) {
        const msg = error?.response?.data?.message || error?.message || '未知错误'
        message.error('操作失败：' + msg)
      }
    }
  })
}

const handleLoginAs = async (row) => {
  try {
    const res = await loginAsUser(row.id)
    const { access_token, user } = res.data
    localStorage.setItem('admin_token', localStorage.getItem('token') || '')
    localStorage.setItem('admin_user', localStorage.getItem('user') || '')
    localStorage.setItem('token', access_token)
    localStorage.setItem('user', JSON.stringify(user))
    window.open('/', '_blank')
  } catch (error) {
    message.error('登录失败')
  }
}

// Delete
const handleDelete = (row) => {
  dialog.error({
    title: '确认删除',
    content: `确定要删除用户 ${row.username} 吗？此操作不可恢复！`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteUser(row.id)
        message.success('用户删除成功')
        fetchUsers()
      } catch (error) {
        message.error('删除用户失败：' + (error.message || '未知错误'))
      }
    }
  })
}
// Reset password
const openResetPwdModal = (row) => {
  resetPwdTargetId.value = row.id
  resetPwdForm.password = ''
  showResetPwdDrawer.value = true
}

const handleResetPassword = async () => {
  try {
    await resetPwdFormRef.value?.validate()
  } catch { return }
  resettingPwd.value = true
  try {
    await resetUserPassword(resetPwdTargetId.value, { password: resetPwdForm.password })
    message.success('密码重置成功')
    showResetPwdDrawer.value = false
  } catch (error) {
    message.error('密码重置失败：' + (error.message || '未知错误'))
  } finally {
    resettingPwd.value = false
  }
}

// Batch operations
const handleBatchEnable = () => {
  dialog.warning({
    title: '批量启用',
    content: `确定要启用选中的 ${checkedRowKeys.value.length} 个用户吗？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        const res = await batchUserAction({ user_ids: checkedRowKeys.value, action: 'enable' })
        message.success(`已启用 ${res.data.affected} 个用户`)
        checkedRowKeys.value = []
        fetchUsers()
      } catch (error) {
        message.error('批量启用失败：' + (error.message || '未知错误'))
      }
    }
  })
}

const handleBatchDisable = () => {
  dialog.warning({
    title: '批量禁用',
    content: `确定要禁用选中的 ${checkedRowKeys.value.length} 个用户吗？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        const res = await batchUserAction({ user_ids: checkedRowKeys.value, action: 'disable' })
        message.success(`已禁用 ${res.data.affected} 个用户`)
        checkedRowKeys.value = []
        fetchUsers()
      } catch (error) {
        message.error('批量禁用失败：' + (error.message || '未知错误'))
      }
    }
  })
}

const doBatchDelete = async () => {
  try {
    const res = await batchUserAction({ user_ids: checkedRowKeys.value, action: 'delete' })
    message.success(`已删除 ${res.data.affected} 个用户`)
    checkedRowKeys.value = []
    fetchUsers()
  } catch (error) {
    message.error('批量删除失败：' + (error.message || '未知错误'))
  }
}

// Set level
const fetchLevels = async () => {
  try {
    const res = await listUserLevels({ page: 1, page_size: 100 })
    userLevels.value = res.data.items || res.data || []
  } catch {}
}

const openSetLevelModal = async () => {
  selectedLevelId.value = null
  if (userLevels.value.length === 0) await fetchLevels()
  showSetLevelModal.value = true
}

const handleBatchSetLevel = async () => {
  if (!selectedLevelId.value) {
    message.warning('请选择等级')
    return
  }
  try {
    const res = await batchUserAction({
      user_ids: checkedRowKeys.value,
      action: 'set_level',
      data: { level_id: selectedLevelId.value }
    })
    message.success(`已设置 ${res.data.affected} 个用户的等级`)
    showSetLevelModal.value = false
    checkedRowKeys.value = []
    fetchUsers()
  } catch (error) {
    message.error('设置等级失败：' + (error.message || '未知错误'))
  }
}

// CSV Export
const handleExportCSV = async () => {
  try {
    const params = {
      search: searchQuery.value || undefined,
      is_active: statusFilter.value === 'active' ? 'true' : statusFilter.value === 'inactive' ? 'false' : undefined
    }
    const res = await exportUsersCSV(params)
    const blobData = res.data || res
    // If server returned JSON error as blob, try to parse it
    if (blobData instanceof Blob && blobData.type && blobData.type.includes('application/json')) {
      const text = await blobData.text()
      const json = JSON.parse(text)
      message.error(json.message || '导出失败')
      return
    }
    const blob = blobData instanceof Blob ? blobData : new Blob([blobData], { type: 'text/csv;charset=utf-8' })
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `users_${new Date().toISOString().slice(0, 10)}.csv`
    a.click()
    window.URL.revokeObjectURL(url)
    message.success('导出成功')
  } catch (error) {
    // Handle blob error responses
    if (error?.response?.data instanceof Blob) {
      try {
        const text = await error.response.data.text()
        const json = JSON.parse(text)
        message.error('导出失败：' + (json.message || '未知错误'))
        return
      } catch {}
    }
    message.error('导出失败：' + (error.message || '未知错误'))
  }
}

// CSV Import
const triggerImportCSV = () => {
  importFileInput.value?.click()
}

const handleImportCSV = async (e) => {
  const file = e.target.files?.[0]
  if (!file) return
  importing.value = true
  try {
    const formData = new FormData()
    formData.append('file', file)
    const res = await importUsersCSV(formData)
    importResult.value = res.data
    showImportResultModal.value = true
    fetchUsers()
  } catch (error) {
    message.error('导入失败：' + (error.message || '未知错误'))
  } finally {
    importing.value = false
    // Reset file input so same file can be selected again
    if (importFileInput.value) importFileInput.value.value = ''
  }
}

const handleViewDetail = (row) => {
  userDetailDrawerRef.value?.open(row.id || row.user_id)
}

onMounted(async () => {
  await fetchUsers()
  // 如果 URL 中有 userId 参数，自动打开该用户详情
  const userId = route.query.userId
  if (userId) {
    const user = users.value.find(u => u.id === Number(userId))
    handleViewDetail(user || { id: Number(userId) })
  }
})
</script>

<style scoped>
/* Mobile card styles */
.mobile-card-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.mobile-card {
  background: var(--bg-color, #fff);
  border-radius: 12px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-bottom: 1px solid var(--border-color, #f0f0f0);
}

.card-title {
  font-weight: 600;
  font-size: 14px;
  color: var(--text-color, #333);
}

.card-body {
  padding: 10px 14px;
}

.card-row {
  display: flex;
  justify-content: space-between;
  padding: 4px 0;
  font-size: 13px;
  gap: 8px;
}

.card-row > span:last-child {
  word-break: break-all;
  text-align: right;
  min-width: 0;
  color: var(--text-color, #333);
}

.card-label {
  color: var(--text-color-secondary, #999);
  flex-shrink: 0;
}

.card-actions {
  display: flex;
  gap: 8px;
  padding: 10px 14px;
  border-top: 1px solid var(--border-color, #f0f0f0);
  flex-wrap: wrap;
}
:deep(.action-btn-grid) { display: grid; grid-template-columns: repeat(3, 1fr); gap: 4px; }
</style>
