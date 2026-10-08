<template>
  <div class="abnormal-users-page admin-page-shell">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">异常用户检测</h2>
        <p class="page-subtitle">自动识别可疑行为，包括多设备共享、频繁重置订阅及异常登录尝试</p>
      </div>
      <div class="header-right">
        <n-space>
          <n-select
            v-model:value="typeFilter"
            placeholder="异常类型筛选"
            clearable
            style="width: 180px"
            :options="typeOptions"
            @update:value="handleSearch"
          />
          <n-button @click="loadData" secondary>
            <template #icon><n-icon :component="RefreshOutline" /></template>
            刷新
          </n-button>
        </n-space>
      </div>
    </div>

    <n-card :bordered="false" class="page-card admin-main-card">

      <!-- Mobile toolbar：吸顶工具条（筛选 + 检测）。不用全局 .mobile-toolbar-row：
           它会把行内控件按网格拉成整行宽，两个控件并排就会顶出屏幕。 -->
      <div v-if="appStore.isMobile" class="app-sticky-toolbar abnormal-toolbar">
        <n-select class="abnormal-toolbar__select" v-model:value="typeFilter" placeholder="异常类型" clearable size="small" :options="typeOptions" @update:value="handleSearch" />
        <n-button size="small" type="info" @click="handleSearch">检测</n-button>
      </div>

      <n-space vertical :size="16">

        <!-- Data table -->
        <template v-if="!appStore.isMobile">
          <n-data-table
            class="unified-admin-table"
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
                  <span>{{ formatFullDateTime(row.last_active) }}</span>
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

        <n-pagination
          v-if="pagination.itemCount > pagination.pageSize"
          class="list-pagination"
          v-model:page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :item-count="pagination.itemCount"
          :page-sizes="[10, 20, 50]"
          show-size-picker
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </n-space>
    </n-card>
  </div>
</template>

<script setup>
import { ref, h, onActivated, onMounted } from 'vue'
import { NButton, NTag, NSpace, NIcon, useMessage } from 'naive-ui'
import { SearchOutline, RefreshOutline, PersonOutline } from '@vicons/ionicons5'
import { useRouter } from 'vue-router'
import { getAbnormalUsers } from '@/api/admin'
import { useTable } from '@/composables/useTable'
import { useAppStore } from '@/stores/app'
import { formatFullDateTime } from '@/utils/date'

const message = useMessage()
const router = useRouter()
const appStore = useAppStore()

// State
const typeFilter = ref(null)

// 统一表格状态（getAbnormalUsers 返回 data.users 非标格式，用 fetcher 包装适配）
const abnormalFetcher = async (params) => {
  const res = await getAbnormalUsers(params)
  const data = res.data || {}
  return { data: { items: data.users || data.items || [], total: data.total || 0 } }
}
const { loading, tableData: users, pagination, loadData, reload } = useTable(abnormalFetcher, {
  getParams: () => ({ type: typeFilter.value || undefined }),
})

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
    render: (row) => formatFullDateTime(row.last_active)
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

const handleSearch = () => { reload() }
const handlePageChange = (page) => { pagination.page = page; loadData() }
const handlePageSizeChange = (size) => { pagination.pageSize = size; pagination.page = 1; loadData() }

const handleViewUser = (userId) => {
  router.push({ name: 'AdminUsers', query: { userId } })
}

onMounted(() => {
  loadData()
})

// KeepAlive 缓存激活时静默刷新数据（不清 loading 遮罩、不重置分页）
onActivated(() => {
  loadData()
})
</script>

<style scoped>
:deep(.n-data-table .n-data-table-th) {
  font-weight: 600;
}

/* 分页：桌面靠右，手机居中 */
.list-pagination { margin-top: 16px; justify-content: flex-end; }

/* 手机端工具条：筛选下拉自适应 + 检测按钮 */
/* 卡片内容区在手机端无左右内边距，工具条不再用全局的 -12px 出血，避免越过卡片边界 */
.abnormal-toolbar { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; margin-left: 0; margin-right: 0; }
.abnormal-toolbar :deep(.abnormal-toolbar__select),
.abnormal-toolbar :deep(.n-select) { flex: 1 1 120px; min-width: 0; }

/* 说明：卡片样式（.mobile-card / .card-header / .card-row…）一律交给全局
   mobile-cards.css + admin-mobile.css，页面不再自己覆盖，避免手机端又变回「网页表格」。 */
@media (max-width: 767px) {
  /* 左右留白由全局统一给（mobile-app-ui.css + admin-mobile.css），页面不再自带内边距 */
  .list-pagination { justify-content: center; }
}
</style>
