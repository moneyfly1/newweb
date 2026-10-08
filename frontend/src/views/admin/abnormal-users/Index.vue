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

        <!-- 全选 / 批量操作：公共组件（桌面在表格上方，手机固定在底部标签栏上方） -->
        <BatchSelectBar
          :total="selection.total.value"
          :selected-count="selection.count.value"
          :all-selected="selection.allSelected.value"
          :indeterminate="selection.indeterminate.value"
          :disabled="loading"
          label="个异常用户"
          @toggle-all="selection.toggleAll"
          @clear="selection.clear"
        >
          <n-button size="small" type="success" :disabled="!selection.count.value" @click="handleBatchEnable">批量解封</n-button>
          <n-button size="small" type="error" :disabled="!selection.count.value" @click="handleBatchDisable">批量封禁</n-button>
        </BatchSelectBar>

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
            :row-key="(row) => row.user_id"
            v-model:checked-row-keys="checkedRowKeys"
          />
        </template>

        <template v-else>
          <div class="mobile-card-list">
            <div
              v-for="row in users"
              :key="row.user_id"
              class="mobile-card is-selectable"
              :class="{ 'is-selected': selection.isSelected(row) }"
              @click="selection.toggle(row)"
            >
              <div class="card-check" @click.stop>
                <n-checkbox :checked="selection.isSelected(row)" @update:checked="() => selection.toggle(row)" />
              </div>
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
              <div class="card-actions" @click.stop>
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
import { ref, h, computed, onActivated, onMounted } from 'vue'
import { NButton, NTag, NSpace, NIcon, useMessage, useDialog } from 'naive-ui'
import { SearchOutline, RefreshOutline, PersonOutline } from '@vicons/ionicons5'
import { useRouter } from 'vue-router'
import { getAbnormalUsers, batchUserAction } from '@/api/admin'
import { useTable } from '@/composables/useTable'
import { useBatchSelection } from '@/composables/useBatchSelection'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'
import { formatFullDateTime } from '@/utils/date'
import BatchSelectBar from '@/components/BatchSelectBar.vue'

const message = useMessage()
const dialog = useDialog()
const router = useRouter()
const appStore = useAppStore()
const userStore = useUserStore()

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

// 全选 / 多选：全站统一实现（桌面表格与手机卡片共用同一份选择状态）。
// 异常用户行没有 id 字段，唯一标识是 user_id。
const selection = useBatchSelection(() => users.value, { getId: (row) => row.user_id })
// Naive 表格要的是数组，这里做一层桥接，保证桌面表格与批量栏状态一致
const checkedRowKeys = computed({
  get: () => [...selection.selectedKeys.value],
  set: (keys) => { selection.selectedKeys.value = new Set(keys) },
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
  { type: 'selection' },
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

/**
 * 批量解封 / 批量封禁：走后端批量接口 batchUserAction（action = enable / disable），
 * 一次请求完成，不再逐条 loop。确认框写明「将影响 N 项」，结束后汇总实际影响条数。
 */
const runBatchUserAction = ({ title, action, rows, content, type = 'warning', positiveText = '确定' }) => {
  if (!rows.length) return
  dialog[type]({
    title,
    content,
    positiveText,
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        const res = await batchUserAction({ user_ids: rows.map(r => r.user_id), action })
        const affected = Number(res?.data?.affected ?? rows.length)
        const skipped = rows.length - affected
        if (skipped > 0) message.warning(`成功 ${affected} 项 / 失败 ${skipped} 项（管理员账号不可${action === 'disable' ? '封禁' : '解封'}）`)
        else message.success(`成功 ${affected} 项`)
        selection.clear()
        await loadData()
      } catch (e) {
        message.error(e?.response?.data?.message || e.message || `${title}失败`)
      }
    }
  })
}

const handleBatchEnable = () => {
  const rows = selection.selectedRows.value
  if (!rows.length) return
  runBatchUserAction({
    title: '批量解封',
    action: 'enable',
    rows,
    type: 'warning',
    content: `将影响 ${rows.length} 项：选中用户恢复为正常状态，其未过期订阅同步恢复。`,
  })
}

const handleBatchDisable = () => {
  const all = selection.selectedRows.value
  if (!all.length) return
  // 不允许封禁当前登录账号（后端也会跳过管理员，这里先挡一层并明确告知）
  const rows = all.filter(r => r.user_id !== userStore.userInfo?.id)
  if (!rows.length) {
    message.warning('选中的都是当前登录账号，无法封禁')
    return
  }
  if (rows.length < all.length) message.info(`已排除当前登录账号 ${all.length - rows.length} 项`)
  runBatchUserAction({
    title: '批量封禁',
    action: 'disable',
    rows,
    type: 'error',
    positiveText: '封禁',
    content: `将影响 ${rows.length} 项：选中用户立即无法登录，其订阅同步停用。`,
  })
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

/* 可选中卡片：左侧给复选框留位（全局样式已做，这里兜底，防止被其他层叠规则盖掉） */
.mobile-card.is-selectable { padding-left: 44px !important; }
@media (max-width: 767px) {
  /* 左右留白由全局统一给（mobile-app-ui.css + admin-mobile.css），页面不再自带内边距 */
  .list-pagination { justify-content: center; }
}
</style>
