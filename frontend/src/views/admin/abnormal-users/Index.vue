<template>
  <div class="abnormal-users-page">
    <n-card title="异常用户检测" :bordered="false" class="page-card">
      <n-space vertical :size="16">
        <!-- Header area -->
        <n-space justify="space-between" align="center" style="width: 100%">
          <n-space>
            <n-select
              v-model:value="typeFilter"
              placeholder="异常类型筛选"
              clearable
              style="width: 200px"
              :options="typeOptions"
              @update:value="handleSearch"
            />
            <n-button type="info" @click="handleSearch">
              <template #icon><n-icon :component="SearchOutline" /></template>
              搜索
            </n-button>
          </n-space>
          <n-button @click="fetchAbnormalUsers">
            <template #icon><n-icon :component="RefreshOutline" /></template>
            刷新
          </n-button>
        </n-space>

        <!-- Data table -->
        <template v-if="!appStore.isMobile">
          <n-data-table
            :columns="columns"
            :data="users"
            :loading="loading"
            :pagination="false"
            :bordered="false"
            :single-line="false"
          />
        </template>

        <template v-else>
          <div class="mobile-card-list">
            <div v-for="row in users" :key="row.user_id" class="mobile-card">
              <div class="card-header">
                <span class="card-title">{{ row.username }}</span>
                <n-tag :type="getTypeTag(row.abnormal_type).type" size="small">
                  {{ getTypeTag(row.abnormal_type).label }}
                </n-tag>
              </div>
              <div class="card-body">
                <div class="card-row">
                  <span class="card-label">邮箱</span>
                  <span style="overflow: hidden; text-overflow: ellipsis;">{{ row.email }}</span>
                </div>
                <div class="card-row">
                  <span class="card-label">异常原因</span>
                  <span style="text-align: right; flex: 1; margin-left: 8px;">{{ row.details }}</span>
                </div>
                <div class="card-row">
                  <span class="card-label">最后活跃</span>
                  <span>{{ row.last_active ? new Date(row.last_active).toLocaleString('zh-CN') : '-' }}</span>
                </div>
              </div>
              <div class="card-actions">
                <n-button size="small" type="primary" @click="handleViewUser(row.user_id)">
                  <template #icon><n-icon><PersonOutline /></n-icon></template>
                  查看用户
                </n-button>
              </div>
            </div>
          </div>
        </template>

        <n-alert v-if="users.length === 0 && !loading" type="info" title="暂无异常用户">
          当前没有检测到异常用户
        </n-alert>
      </n-space>
    </n-card>
  </div>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { NButton, NTag, NSpace, NIcon, useMessage } from 'naive-ui'
import { SearchOutline, RefreshOutline, PersonOutline } from '@vicons/ionicons5'
import { useRouter } from 'vue-router'
import { getAbnormalUsers } from '@/api/admin'
import { useAppStore } from '@/stores/app'

const message = useMessage()
const router = useRouter()
const appStore = useAppStore()

// State
const loading = ref(false)
const users = ref([])
const typeFilter = ref(null)

const typeOptions = [
  { label: '全部', value: null },
  { label: '订阅重置过多', value: 'excessive_resets' },
  { label: '设备数超限', value: 'device_limit_exceeded' },
  { label: '可疑登录', value: 'suspicious_logins' }
]

// Type tag mapping
const getTypeTag = (type) => {
  const typeMap = {
    excessive_resets: { label: '订阅重置过多', type: 'warning' },
    device_limit_exceeded: { label: '设备数超限', type: 'error' },
    suspicious_logins: { label: '可疑登录', type: 'info' }
  }
  return typeMap[type] || { label: type, type: 'default' }
}

// Table columns
const columns = [
  { title: 'User ID', key: 'user_id', width: 80, resizable: true, sorter: 'default' },
  { title: '用户名', key: 'username', ellipsis: { tooltip: true }, width: 150, resizable: true },
  { title: '邮箱', key: 'email', ellipsis: { tooltip: true }, width: 220, resizable: true },
  {
    title: '异常类型',
    key: 'abnormal_type',
    width: 150,
    resizable: true,
    render: (row) => {
      const tag = getTypeTag(row.abnormal_type)
      return h(NTag, { type: tag.type, size: 'small' }, { default: () => tag.label })
    }
  },
  {
    title: '详情',
    key: 'details',
    ellipsis: { tooltip: true },
    width: 200,
    resizable: true
  },
  {
    title: '最后活跃',
    key: 'last_active',
    width: 170,
    resizable: true,
    render: (row) => row.last_active ? new Date(row.last_active).toLocaleString('zh-CN') : '-'
  },
  {
    title: '操作',
    key: 'actions',
    width: 120,
    fixed: 'right',
    render: (row) => h(
      NButton,
      {
        size: 'small',
        type: 'primary',
        onClick: () => handleViewUser(row.user_id)
      },
      {
        icon: () => h(NIcon, { component: PersonOutline }),
        default: () => '查看用户'
      }
    )
  }
]

// Fetch abnormal users
const fetchAbnormalUsers = async () => {
  loading.value = true
  try {
    const params = {
      type: typeFilter.value || undefined
    }
    const response = await getAbnormalUsers(params)
    users.value = response.data.users || []
  } catch (error) {
    message.error('获取异常用户列表失败：' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  fetchAbnormalUsers()
}

const handleViewUser = (userId) => {
  router.push(`/admin/users/${userId}`)
}

onMounted(() => {
  fetchAbnormalUsers()
})
</script>

<style scoped>
.abnormal-users-page {
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
  white-space: nowrap;
  margin-right: 8px;
}

.card-actions {
  display: flex;
  gap: 8px;
  padding: 10px 14px;
  border-top: 1px solid #f0f0f0;
  flex-wrap: wrap;
}

@media (max-width: 767px) {
  .abnormal-users-page { padding: 8px; }
}
</style>
