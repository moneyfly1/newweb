<template>
  <div class="admin-users-page">
    <n-card :title="appStore.isMobile ? undefined : '用户管理'" :bordered="false" class="page-card">
      <n-space vertical :size="16">
        <!-- Desktop Header -->
        <template v-if="!appStore.isMobile">
          <n-space justify="space-between" align="center" style="width: 100%">
            <n-space>
              <n-input
                v-model:value="searchQuery"
                placeholder="搜索邮箱或用户名"
                clearable
                style="width: 260px"
                @keyup.enter="handleSearch"
              >
                <template #prefix>
                  <n-icon :component="SearchOutline" />
                </template>
              </n-input>
              <n-select
                v-model:value="statusFilter"
                placeholder="状态筛选"
                clearable
                style="width: 140px"
                :options="statusOptions"
                @update:value="handleSearch"
              />
              <n-button type="info" @click="handleSearch">
                <template #icon><n-icon :component="SearchOutline" /></template>
                搜索
              </n-button>
            </n-space>
            <n-space>
              <n-button type="primary" @click="openCreateModal">
                <template #icon><n-icon :component="AddOutline" /></template>
                新增用户
              </n-button>
              <n-button @click="handleExportCSV">
                <template #icon><n-icon :component="DownloadOutline" /></template>
                导出CSV
              </n-button>
              <n-button @click="triggerImportCSV">
                <template #icon><n-icon :component="CloudUploadOutline" /></template>
                导入CSV
              </n-button>
              <input ref="importFileInput" type="file" accept=".csv" style="display:none" @change="handleImportCSV" />
              <n-button @click="fetchUsers">
                <template #icon><n-icon :component="RefreshOutline" /></template>
                刷新
              </n-button>
            </n-space>
          </n-space>
        </template>

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
                style="flex: 1"
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
              <input ref="importFileInput" type="file" accept=".csv" style="display:none" @change="handleImportCSV" />
            </div>
          </div>
        </div>

        <!-- Batch operations -->
        <n-space v-if="checkedRowKeys.length > 0" align="center">
          <span style="color: #666">已选择 {{ checkedRowKeys.length }} 项</span>
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
          <div v-if="loading" style="text-align: center; padding: 40px;">
            <n-spin size="medium" />
          </div>
          <div v-else-if="users.length === 0" style="text-align: center; padding: 40px; color: #999;">
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
                  <span>¥{{ (row.balance ?? 0).toFixed(2) }}</span>
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
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </n-space>
    </n-card>

    <!-- Create/Edit User Modal -->
    <n-modal v-model:show="showEditModal" preset="card" :title="isCreate ? '新增用户' : '编辑用户'" :style="{ width: appStore.isMobile ? '95%' : '520px' }">
      <n-form ref="formRef" :model="editForm" :rules="formRulesComputed" label-placement="left" label-width="80" style="margin-top: 8px">
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
          <n-input-number v-model:value="editForm.balance" placeholder="请输入余额" :min="0" :precision="2" style="width: 100%" />
        </n-form-item>
        <n-form-item label="管理员" path="is_admin">
          <n-switch v-model:value="editForm.is_admin" />
        </n-form-item>
        <n-form-item label="启用" path="is_active">
          <n-switch v-model:value="editForm.is_active" />
        </n-form-item>
        <n-form-item label="备注" path="notes">
          <n-input v-model:value="editForm.notes" type="textarea" placeholder="备注信息" :rows="3" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showEditModal = false">取消</n-button>
          <n-button type="primary" :loading="saving" @click="handleSaveUser">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- User Detail Drawer -->
    <n-drawer v-model:show="showDetailDrawer" :width="appStore.isMobile ? '100%' : 780" placement="right" closable>
      <n-drawer-content :title="'用户详情 - ' + (userDetail.username || userDetail.email || '')" closable>
        <n-descriptions bordered :column="appStore.isMobile ? 1 : 2" label-placement="left" size="small">
          <n-descriptions-item label="ID">{{ userDetail.id }}</n-descriptions-item>
          <n-descriptions-item label="用户名">{{ userDetail.username || '-' }}</n-descriptions-item>
          <n-descriptions-item label="邮箱">{{ userDetail.email || '-' }}</n-descriptions-item>
          <n-descriptions-item label="余额">¥{{ (userDetail.balance ?? 0).toFixed(2) }}</n-descriptions-item>
          <n-descriptions-item label="状态">
            <n-tag :type="userDetail.is_active ? 'success' : 'error'" size="small">{{ userDetail.is_active ? '激活' : '禁用' }}</n-tag>
            <n-tag v-if="userDetail.is_admin" type="warning" size="small" style="margin-left:4px">管理员</n-tag>
          </n-descriptions-item>
          <n-descriptions-item label="等级">{{ userDetail.level_name || '无' }}</n-descriptions-item>
          <n-descriptions-item label="注册时间">{{ fmtDate(userDetail.created_at) }}</n-descriptions-item>
          <n-descriptions-item label="最后登录">{{ fmtDate(userDetail.last_login) }}</n-descriptions-item>
        </n-descriptions>

        <!-- Subscription -->
        <n-divider>订阅信息</n-divider>
        <template v-if="userDetail.subscription">
          <n-descriptions bordered :column="2" label-placement="left" size="small">
            <n-descriptions-item label="套餐">{{ userDetail.package_name || '-' }}</n-descriptions-item>
            <n-descriptions-item label="状态">
              <n-tag :type="subStatusType(userDetail.subscription.status)" size="small">{{ subStatusText(userDetail.subscription.status) }}</n-tag>
              <n-tag v-if="!userDetail.subscription.is_active" type="error" size="small" style="margin-left:4px">已停用</n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="设备">{{ userDetail.subscription.current_devices || 0 }} / {{ userDetail.subscription.device_limit || 0 }}</n-descriptions-item>
            <n-descriptions-item label="到期时间">{{ fmtDate(userDetail.subscription.expire_time) }}</n-descriptions-item>
          </n-descriptions>
          <div v-if="userDetail.subscription_urls" style="margin-top:8px">
            <div class="url-row"><span class="url-label">通用</span><code class="url-text">{{ userDetail.subscription_urls.universal_url }}</code></div>
            <div class="url-row"><span class="url-label">Clash</span><code class="url-text">{{ userDetail.subscription_urls.clash_url }}</code></div>
          </div>
        </template>
        <n-empty v-else description="暂无订阅" size="small" />

        <!-- Tabs for records -->
        <n-tabs type="line" style="margin-top:16px" animated>
          <n-tab-pane name="orders" tab="订单记录">
            <n-data-table v-if="(userDetail.recent_orders||[]).length" :columns="orderCols" :data="userDetail.recent_orders" :bordered="false" size="small" :max-height="240" />
            <n-empty v-else description="暂无订单" size="small" />
          </n-tab-pane>
          <n-tab-pane name="devices" tab="设备记录">
            <n-data-table v-if="(userDetail.devices||[]).length" :columns="deviceCols" :data="userDetail.devices" :bordered="false" size="small" :max-height="240" />
            <n-empty v-else description="暂无设备" size="small" />
          </n-tab-pane>
          <n-tab-pane name="logins" tab="登录历史">
            <n-data-table v-if="(userDetail.login_history||[]).length" :columns="loginCols" :data="userDetail.login_history" :bordered="false" size="small" :max-height="240" />
            <n-empty v-else description="暂无记录" size="small" />
          </n-tab-pane>
          <n-tab-pane name="resets" tab="重置记录">
            <n-data-table v-if="(userDetail.resets||[]).length" :columns="resetCols" :data="userDetail.resets" :bordered="false" size="small" :max-height="240" />
            <n-empty v-else description="暂无记录" size="small" />
          </n-tab-pane>
          <n-tab-pane name="balance" tab="余额变动">
            <n-data-table v-if="(userDetail.balance_logs||[]).length" :columns="balanceCols" :data="userDetail.balance_logs" :bordered="false" size="small" :max-height="240" />
            <n-empty v-else description="暂无记录" size="small" />
          </n-tab-pane>
          <n-tab-pane name="recharge" tab="充值记录">
            <n-data-table v-if="(userDetail.recharge_records||[]).length" :columns="rechargeCols" :data="userDetail.recharge_records" :bordered="false" size="small" :max-height="240" />
            <n-empty v-else description="暂无记录" size="small" />
          </n-tab-pane>
        </n-tabs>
      </n-drawer-content>
    </n-drawer>

    <!-- Reset Password Modal -->
    <n-modal v-model:show="showResetPwdModal" preset="card" title="重置密码" :style="{ width: appStore.isMobile ? '95%' : '420px' }">
      <n-form ref="resetPwdFormRef" :model="resetPwdForm" :rules="resetPwdRules" label-placement="left" label-width="80">
        <n-form-item label="新密码" path="password">
          <n-input v-model:value="resetPwdForm.password" type="password" show-password-on="click" placeholder="请输入新密码" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showResetPwdModal = false">取消</n-button>
          <n-button type="primary" :loading="resettingPwd" @click="handleResetPassword">确认重置</n-button>
        </n-space>
      </template>
    </n-modal>

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
  listUsers, getUser, updateUser, deleteUser, toggleUserActive,
  createUser, resetUserPassword, deleteUserDevice,
  batchUserAction, exportUsersCSV, importUsersCSV
} from '@/api/admin'
import { listUserLevels } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'

const message = useMessage()
const dialog = useDialog()
const appStore = useAppStore()
const userStore = useUserStore()

// State
const loading = ref(false)
const saving = ref(false)
const resettingPwd = ref(false)
const users = ref([])
const searchQuery = ref('')
const statusFilter = ref(null)
const currentPage = ref(1)
const pageSize = ref(20)
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
const showEditModal = ref(false)
const showResetPwdModal = ref(false)
const showDetailDrawer = ref(false)
const isCreate = ref(false)
const formRef = ref(null)
const resetPwdFormRef = ref(null)
const resetPwdTargetId = ref(null)
const userDetail = ref({})

const editForm = reactive({
  id: null,
  username: '',
  email: '',
  password: '',
  balance: 0,
  is_admin: false,
  is_active: true,
  notes: ''
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
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码至少6个字符', trigger: 'blur' }
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
    render: (row) => `¥${(row.balance ?? 0).toFixed(2)}`
  },
  {
    title: '等级',
    key: 'level',
    width: 90, resizable: true,
    render: (row) => row.level_name || row.level || '无'
  },
  {
    title: '状态',
    key: 'is_active',
    width: 80, resizable: true,
    render: (row) => h(NTag, { type: row.is_active ? 'success' : 'error', size: 'small' }, { default: () => row.is_active ? '激活' : '禁用' })
  },
  {
    title: '管理员',
    key: 'is_admin',
    width: 80, resizable: true,
    render: (row) => row.is_admin ? h(NTag, { type: 'warning', size: 'small' }, { default: () => '管理员' }) : '-'
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
    width: 80,
    fixed: 'right',
    render: (row) => h(
      NDropdown,
      {
        trigger: 'click',
        options: actionOptions(row),
        onSelect: (key) => handleAction(key, row)
      },
      { default: () => h(NButton, { size: 'small', quaternary: true }, { icon: () => h(NIcon, { component: EllipsisVertical }) }) }
    )
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
}

const openCreateModal = () => {
  resetEditForm()
  isCreate.value = true
  showEditModal.value = true
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
  showEditModal.value = true
}

// Save user
const handleSaveUser = async () => {
  try {
    await formRef.value?.validate()
  } catch { return }
  saving.value = true
  try {
    if (isCreate.value) {
      await createUser({
        username: editForm.username,
        email: editForm.email,
        password: editForm.password,
        balance: editForm.balance,
        is_admin: editForm.is_admin,
        is_active: editForm.is_active,
        notes: editForm.notes
      })
      message.success('用户创建成功')
    } else {
      await updateUser(editForm.id, {
        username: editForm.username,
        email: editForm.email,
        balance: editForm.balance,
        is_admin: editForm.is_admin,
        is_active: editForm.is_active,
        notes: editForm.notes
      })
      message.success('用户更新成功')
    }
    showEditModal.value = false
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
  showResetPwdModal.value = true
}

const handleResetPassword = async () => {
  try {
    await resetPwdFormRef.value?.validate()
  } catch { return }
  resettingPwd.value = true
  try {
    await resetUserPassword(resetPwdTargetId.value, { password: resetPwdForm.password })
    message.success('密码重置成功')
    showResetPwdModal.value = false
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

// View detail
const handleViewDetail = async (row) => {
  try {
    const res = await getUser(row.id)
    const d = res.data
    // Flatten: spread user fields + attach nested data
    userDetail.value = {
      ...d.user,
      subscription: d.subscription || null,
      subscription_urls: d.subscription_urls || {},
      package_name: d.package_name || '',
      recent_orders: d.recent_orders || [],
      devices: d.devices || [],
      resets: d.resets || [],
      balance_logs: d.balance_logs || [],
      login_history: d.login_history || [],
      recharge_records: d.recharge_records || [],
    }
    showDetailDrawer.value = true
  } catch (e) {
    message.error('获取用户详情失败')
  }
}

const fmtDate = (d) => d ? new Date(d).toLocaleString('zh-CN') : '-'

const handleDeleteDevice = (device) => {
  dialog.warning({
    title: '确认删除设备',
    content: `确定要删除设备 ${device.device_name || device.software_name || '未知设备'} 吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteUserDevice(userDetail.value.id, device.id)
        message.success('设备已删除')
        // Refresh detail
        await handleViewDetail(userDetail.value)
      } catch (error) {
        message.error('删除设备失败：' + (error.message || '未知错误'))
      }
    }
  })
}
const subStatusType = (s) => ({ active: 'success', expiring: 'warning', expired: 'error' }[s] || 'default')
const subStatusText = (s) => ({ active: '活跃', expiring: '即将到期', expired: '已过期', disabled: '已禁用' }[s] || s || '-')

const orderCols = [
  { title: '订单号', key: 'order_no', width: 180, ellipsis: { tooltip: true } },
  { title: '金额', key: 'final_amount', width: 90, render: (r) => `¥${(r.final_amount ?? r.amount ?? 0).toFixed(2)}` },
  { title: '状态', key: 'status', width: 80 },
  { title: '时间', key: 'created_at', width: 160, render: (r) => fmtDate(r.created_at) }
]
const deviceCols = [
  { title: '设备名', key: 'device_name', ellipsis: { tooltip: true }, render: (r) => r.device_name || r.software_name || '未知设备' },
  { title: 'IP', key: 'ip_address', width: 130, render: (r) => r.ip_address || '-' },
  { title: '最后活跃', key: 'last_access', width: 160, render: (r) => fmtDate(r.last_access || r.updated_at) },
  {
    title: '操作', key: 'actions', width: 80,
    render: (r) => h(NButton, { size: 'small', type: 'error', secondary: true, onClick: () => handleDeleteDevice(r) }, { default: () => '删除' })
  }
]
const loginCols = [
  { title: 'IP', key: 'ip_address', width: 130, render: (r) => r.ip_address || '-' },
  { title: '位置', key: 'location', width: 100, render: (r) => r.location || '-' },
  { title: 'UA', key: 'user_agent', ellipsis: { tooltip: true }, render: (r) => r.user_agent || '-' },
  { title: '状态', key: 'login_status', width: 70, render: (r) => h(NTag, { type: r.login_status === 'success' ? 'success' : 'error', size: 'small' }, { default: () => r.login_status === 'success' ? '成功' : '失败' }) },
  { title: '时间', key: 'login_time', width: 160, render: (r) => fmtDate(r.login_time) }
]
const resetCols = [
  { title: '操作者', key: 'reset_by', width: 100, render: (r) => r.reset_by || '-' },
  { title: '类型', key: 'reset_type', width: 80 },
  { title: '原因', key: 'reason', ellipsis: { tooltip: true } },
  { title: '时间', key: 'created_at', width: 160, render: (r) => fmtDate(r.created_at) }
]
const balanceCols = [
  { title: '类型', key: 'change_type', width: 90 },
  { title: '金额', key: 'amount', width: 90, render: (r) => `¥${(r.amount ?? 0).toFixed(2)}` },
  { title: '变动后', key: 'balance_after', width: 90, render: (r) => `¥${(r.balance_after ?? 0).toFixed(2)}` },
  { title: '说明', key: 'description', ellipsis: { tooltip: true }, render: (r) => r.description || '-' },
  { title: '时间', key: 'created_at', width: 160, render: (r) => fmtDate(r.created_at) }
]
const rechargeCols = [
  { title: '金额', key: 'amount', width: 90, render: (r) => `¥${(r.amount ?? 0).toFixed(2)}` },
  { title: '方式', key: 'payment_method', width: 100 },
  { title: '状态', key: 'status', width: 80 },
  { title: '时间', key: 'created_at', width: 160, render: (r) => fmtDate(r.created_at) }
]

onMounted(() => { fetchUsers() })
</script>

<style scoped>
.admin-users-page {
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

@media (max-width: 767px) {
  .admin-users-page { padding: 8px; }
}

.mobile-toolbar {
  margin-bottom: 12px;
}

.mobile-toolbar-title {
  font-size: 17px;
  font-weight: 600;
  margin-bottom: 10px;
  color: var(--text-color, #333);
}

.mobile-toolbar-controls {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.mobile-toolbar-row {
  display: flex;
  gap: 8px;
  align-items: center;
}
.url-row { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.url-label { font-size: 12px; color: var(--text-color-secondary, #666); min-width: 40px; }
.url-text { font-size: 12px; word-break: break-all; color: var(--text-color, #333); background: rgba(0,0,0,0.03); padding: 2px 6px; border-radius: 3px; }

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
</style>
