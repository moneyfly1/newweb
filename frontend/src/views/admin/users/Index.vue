<template>
  <div class="admin-users-page">
    <n-card title="用户管理" :bordered="false" class="page-card">
      <n-space vertical :size="16">
        <!-- Header area -->
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
            <n-button @click="fetchUsers">
              <template #icon><n-icon :component="RefreshOutline" /></template>
              刷新
            </n-button>
          </n-space>
        </n-space>

        <!-- Batch operations -->
        <n-space v-if="checkedRowKeys.length > 0" align="center">
          <span style="color: #666">已选择 {{ checkedRowKeys.length }} 项</span>
          <n-button size="small" type="success" @click="handleBatchEnable">批量启用</n-button>
          <n-button size="small" type="warning" @click="handleBatchDisable">批量禁用</n-button>
          <n-button size="small" type="error" @click="handleBatchDelete">批量删除</n-button>
        </n-space>

        <!-- Data table -->
        <n-data-table
          :columns="columns"
          :data="users"
          :loading="loading"
          :pagination="false"
          :bordered="false"
          :single-line="false"
          :row-key="(row) => row.id"
          :checked-row-keys="checkedRowKeys"
          @update:checked-row-keys="handleCheck"
        />

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
    <n-modal v-model:show="showEditModal" preset="card" :title="isCreate ? '新增用户' : '编辑用户'" style="width: 520px">
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

    <!-- Reset Password Modal -->
    <n-modal v-model:show="showResetPwdModal" preset="card" title="重置密码" style="width: 420px">
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
  </div>
</template>

<script setup>
import { ref, reactive, h, onMounted, computed } from 'vue'
import { NButton, NTag, NSpace, NIcon, NDropdown, useMessage, useDialog } from 'naive-ui'
import { SearchOutline, AddOutline, RefreshOutline, EllipsisVertical } from '@vicons/ionicons5'
import {
  listUsers, getUser, updateUser, deleteUser, toggleUserActive,
  createUser, resetUserPassword
} from '@/api/admin'

const message = useMessage()
const dialog = useDialog()

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

// Modals
const showEditModal = ref(false)
const showResetPwdModal = ref(false)
const isCreate = ref(false)
const formRef = ref(null)
const resetPwdFormRef = ref(null)
const resetPwdTargetId = ref(null)

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
  { label: '编辑', key: 'edit' },
  { label: row.is_active ? '禁用' : '启用', key: 'toggle' },
  { label: '重置密码', key: 'resetPwd' },
  { label: '删除', key: 'delete' }
]
// Table columns
const columns = [
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 70, sorter: 'default' },
  { title: '用户名', key: 'username', ellipsis: { tooltip: true }, width: 130 },
  { title: '邮箱', key: 'email', ellipsis: { tooltip: true }, width: 200 },
  {
    title: '余额',
    key: 'balance',
    width: 100,
    sorter: (a, b) => a.balance - b.balance,
    render: (row) => `¥${(row.balance ?? 0).toFixed(2)}`
  },
  {
    title: '等级',
    key: 'level',
    width: 90,
    render: (row) => row.level_name || row.level || '无'
  },
  {
    title: '状态',
    key: 'is_active',
    width: 80,
    render: (row) => h(NTag, { type: row.is_active ? 'success' : 'error', size: 'small' }, { default: () => row.is_active ? '激活' : '禁用' })
  },
  {
    title: '管理员',
    key: 'is_admin',
    width: 80,
    render: (row) => row.is_admin ? h(NTag, { type: 'warning', size: 'small' }, { default: () => '管理员' }) : '-'
  },
  {
    title: '注册时间',
    key: 'created_at',
    width: 170,
    render: (row) => row.created_at ? new Date(row.created_at).toLocaleString('zh-CN') : '-'
  },
  {
    title: '最后登录',
    key: 'last_login',
    width: 170,
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

// Action dispatcher
const handleAction = (key, row) => {
  switch (key) {
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
        message.error('操作失败：' + (error.message || '未知错误'))
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
        const inactive = users.value.filter(u => checkedRowKeys.value.includes(u.id) && !u.is_active)
        await Promise.all(inactive.map(u => toggleUserActive(u.id)))
        message.success(`已启用 ${inactive.length} 个用户`)
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
        const active = users.value.filter(u => checkedRowKeys.value.includes(u.id) && u.is_active)
        await Promise.all(active.map(u => toggleUserActive(u.id)))
        message.success(`已禁用 ${active.length} 个用户`)
        checkedRowKeys.value = []
        fetchUsers()
      } catch (error) {
        message.error('批量禁用失败：' + (error.message || '未知错误'))
      }
    }
  })
}

const handleBatchDelete = () => {
  dialog.error({
    title: '批量删除',
    content: `确定要删除选中的 ${checkedRowKeys.value.length} 个用户吗？此操作不可恢复！`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await Promise.all(checkedRowKeys.value.map(id => deleteUser(id)))
        message.success(`已删除 ${checkedRowKeys.value.length} 个用户`)
        checkedRowKeys.value = []
        fetchUsers()
      } catch (error) {
        message.error('批量删除失败：' + (error.message || '未知错误'))
      }
    }
  })
}

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
</style>
