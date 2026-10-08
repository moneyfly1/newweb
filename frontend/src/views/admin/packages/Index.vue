<template>
  <div class="admin-packages-page admin-page-shell" @touchstart.passive="pullTouchStart" @touchmove.passive="pullTouchMove" @touchend.passive="pullTouchEnd">
    <transition name="fade">
      <div v-if="pullDistance > 0 || pullRefreshing" class="pull-indicator" :style="{ transform: `translate(-50%, ${Math.min(pullDistance, 70) - 40}px)` }">
        <n-spin v-if="pullRefreshing" size="small" />
        <span v-else>{{ pullDistance >= 55 ? '释放刷新' : '下拉刷新' }}</span>
      </div>
    </transition>
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">套餐管理</h2>
        <p class="page-subtitle">定义和管理订阅套餐，包含价格、时长、特性及设备数限制等配置</p>
      </div>
      <div class="header-right">
        <n-space>
          <n-button type="primary" @click="handleCreate">
            <template #icon><n-icon :component="AddOutline" /></template>
            新建套餐
          </n-button>
        </n-space>
      </div>
    </div>

    <!-- 统一搜索筛选工具栏（SearchFilterBar 组件，桌面单行不换行；手机端吸顶） -->
    <div class="app-sticky-toolbar mobile-sticky-toolbar">
      <search-filter-bar
        v-model:values="filterValues"
        :filters="filterConfig"
        search-placeholder="搜索套餐名称 / 描述"
        @search="handleSearch"
      />
    </div>

    <n-card :bordered="false" class="page-card admin-main-card">

      <!-- 全选 / 批量操作：公共组件（桌面在列表上方，手机固定在底部标签栏上方）
           注意：后端没有套餐批量接口，这里的批量启用/禁用沿用原实现（逐条调用 updatePackage） -->
      <BatchSelectBar
        :total="selection.total.value"
        :selected-count="selection.count.value"
        :all-selected="selection.allSelected.value"
        :indeterminate="selection.indeterminate.value"
        label="个套餐"
        @toggle-all="selection.toggleAll"
        @clear="selection.clear"
      >
        <n-button size="small" type="success" :disabled="!selection.count.value" @click="handleBatchEnable">批量启用</n-button>
        <n-button size="small" type="warning" :disabled="!selection.count.value" @click="handleBatchDisable">批量禁用</n-button>
      </BatchSelectBar>

      <n-space vertical :size="16">
        <template v-if="!appStore.isMobile">
          <n-data-table
            class="unified-admin-table"
            :columns="columns"
            :data="packages"
            :loading="loading"
            :pagination="false"
            :bordered="false"
            :single-line="false"
            :row-key="(row) => row.id"
            v-model:checked-row-keys="checkedRowKeys"
          />
        </template>

        <template v-else>
          <n-spin :show="loading">
            <div v-if="packages.length === 0" class="mobile-empty">
              暂无数据
            </div>
            <div v-else class="mobile-card-list">
              <div
                v-for="pkg in packages"
                :key="pkg.id"
                class="mobile-card is-selectable"
                :class="{ 'is-selected': selection.isSelected(pkg) }"
                @click="selection.toggle(pkg)"
              >
                <div class="card-check" @click.stop>
                  <n-checkbox :checked="selection.isSelected(pkg)" @update:checked="() => selection.toggle(pkg)" />
                </div>
                <div class="card-header">
                  <div class="card-title">{{ pkg.name }}</div>
                  <n-tag :type="pkg.is_active ? 'success' : 'default'" size="small">
                    {{ pkg.is_active ? '启用' : '禁用' }}
                  </n-tag>
                </div>
                <div class="card-body">
                  <div class="card-row">
                    <span class="card-label">价格</span>
                    <span class="card-value price-text">{{ formatCurrency(pkg.price) }}</span>
                  </div>
                  <div class="card-row">
                    <span class="card-label">有效期</span>
                    <span class="card-value">{{ pkg.duration_days }} 天</span>
                  </div>
                </div>
                <div class="card-actions" @click.stop>
                  <n-button size="small" type="primary" @click="handleEdit(pkg)">编辑</n-button>
                  <n-button size="small" type="error" @click="handleDelete(pkg)">删除</n-button>
                </div>
              </div>
            </div>
          </n-spin>
        </template>

        <n-pagination
          v-model:page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :item-count="pagination.itemCount"
          :page-sizes="[10, 20, 50, 100]"
          show-size-picker
          style="margin-top: 16px; justify-content: flex-end"
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </n-space>
    </n-card>

    <common-drawer
      v-model:show="showEditDrawer"
      :title="isCreating ? '新建套餐' : '编辑套餐'"
      :width="600"
      show-footer
      :loading="saving"
      @confirm="handleSavePackage"
      @cancel="showEditDrawer = false"
    >
      <n-form
        ref="formRef"
        :model="editForm"
        :rules="formRules"
        label-placement="left"
        label-width="120"
        style="margin-top: 20px"
      >
        <n-form-item label="套餐名称" path="name">
          <n-input v-model:value="editForm.name" placeholder="请输入套餐名称" />
        </n-form-item>
        <n-form-item label="套餐描述" path="description">
          <n-input
            v-model:value="editForm.description"
            type="textarea"
            placeholder="请输入套餐描述"
            :rows="3"
          />
        </n-form-item>
        <n-form-item label="价格（元）" path="price">
          <n-input-number
            v-model:value="editForm.price"
            placeholder="请输入价格"
            :min="0"
            :precision="2"
            style="width: 100%"
          >
            <template #prefix>¥</template>
          </n-input-number>
        </n-form-item>
        <n-form-item label="有效期（天）" path="duration_days">
          <n-input-number
            v-model:value="editForm.duration_days"
            placeholder="请输入有效期天数"
            :min="1"
            style="width: 100%"
          />
        </n-form-item>
        <n-form-item label="设备数量限制" path="device_limit">
          <n-input-number
            v-model:value="editForm.device_limit"
            placeholder="请输入设备数量限制"
            :min="1"
            style="width: 100%"
          />
        </n-form-item>
        <n-form-item label="排序顺序" path="sort_order">
          <n-input-number
            v-model:value="editForm.sort_order"
            placeholder="数字越小越靠前"
            :min="0"
            style="width: 100%"
          />
        </n-form-item>
        <n-form-item label="特性列表" path="features">
          <n-input
            v-model:value="editForm.features"
            type="textarea"
            placeholder="每行一个特性，如：不限速&#10;专属节点&#10;优先客服"
            :rows="3"
          />
        </n-form-item>
        <n-form-item label="启用状态" path="is_active">
          <n-switch v-model:value="editForm.is_active">
            <template #checked>启用</template>
            <template #unchecked>禁用</template>
          </n-switch>
        </n-form-item>
        <n-form-item label="推荐套餐" path="is_featured">
          <n-switch v-model:value="editForm.is_featured">
            <template #checked>推荐</template>
            <template #unchecked>不推荐</template>
          </n-switch>
        </n-form-item>
      </n-form>
    </common-drawer>
  </div>
</template>

<script setup>
import { ref, reactive, h, onActivated, onMounted, computed } from 'vue'
import { NButton, NTag, NSpace, NIcon, NSpin, useMessage, useDialog } from 'naive-ui'
import { AddOutline } from '@vicons/ionicons5'
import { listAdminPackages, createPackage, updatePackage, deletePackage } from '@/api/admin'
import { useTable } from '@/composables/useTable'
import { useBatchSelection } from '@/composables/useBatchSelection'
import BatchSelectBar from '@/components/BatchSelectBar.vue'
import { usePullRefresh } from '@/composables/usePullRefresh'
import { useAppStore } from '@/stores/app'
import { formatCurrency } from '@/utils/amount'
import CommonDrawer from '@/components/CommonDrawer.vue'
import SearchFilterBar from '@/components/SearchFilterBar.vue'

const appStore = useAppStore()

const message = useMessage()
const dialog = useDialog()

const searchQuery = ref('')

// 统一筛选工具栏状态（值与原 refs 同步，保持业务逻辑不变）
const filterValues = reactive({
  search: '',
})
const filterConfig = []

// 统一表格状态（含搜索参数）
const { loading, tableData: packages, pagination, loadData, reload } = useTable(listAdminPackages, {
  getParams: () => ({ search: searchQuery.value || undefined }),
})
const fetchPackages = loadData

// 全选 / 多选：唯一的选择状态，桌面表格与手机卡片共用（公共组件 useBatchSelection + BatchSelectBar）
const selection = useBatchSelection(() => packages.value)

// 桌面表格仍需数组形式的勾选键，用 computed 桥接，保证与手机端卡片状态一致
const checkedRowKeys = computed({
  get: () => [...selection.selectedKeys.value],
  set: (keys) => { selection.selectedKeys.value = new Set(keys) },
})

// 下拉刷新（App 原生感）——此前只 import 未解构，导致移动端下拉刷新失效
const { distance: pullDistance, refreshing: pullRefreshing, onTouchStart: pullTouchStart, onTouchMove: pullTouchMove, onTouchEnd: pullTouchEnd } =
  usePullRefresh(loadData)

const showEditDrawer = ref(false)
const isCreating = ref(false)
const saving = ref(false)
const formRef = ref(null)
const editForm = reactive({
  id: null,
  name: '',
  description: '',
  price: 0,
  duration_days: 30,
  device_limit: 3,
  features: '',
  is_active: true,
  is_featured: false,
  sort_order: 0
})

const formRules = {
  name: [
    { required: true, message: '请输入套餐名称', trigger: 'blur' }
  ],
  price: [
    { required: true, message: '请输入价格', trigger: 'blur', type: 'number' }
  ],
  duration_days: [
    { required: true, message: '请输入有效期天数', trigger: 'blur', type: 'number' }
  ],
  device_limit: [
    { required: true, message: '请输入设备数量限制', trigger: 'blur', type: 'number' }
  ],
  sort_order: [
    { required: true, message: '请输入排序顺序', trigger: 'blur', type: 'number' }
  ]
}

const columns = [
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 80, resizable: true, sorter: 'default' },
  { title: '套餐名称', key: 'name', ellipsis: { tooltip: true } },
  {
    title: '价格',
    key: 'price',
    width: 120,
    resizable: true,
    sorter: (a, b) => a.price - b.price,
    render: (row) => h(
      'span',
      { style: 'color: var(--success-color); font-weight: 600' },
      formatCurrency(row.price)
    )
  },
  {
    title: '有效期',
    key: 'duration_days',
    width: 100,
    resizable: true,
    sorter: (a, b) => a.duration_days - b.duration_days,
    render: (row) => `${row.duration_days} 天`
  },
  {
    title: '设备限制',
    key: 'device_limit',
    width: 100,
    resizable: true,
    render: (row) => `${row.device_limit} 台`
  },
  {
    title: '状态',
    key: 'is_active',
    width: 100,
    resizable: true,
    render: (row) => h(
      NTag,
      { type: row.is_active ? 'success' : 'default', size: 'small' },
      { default: () => row.is_active ? '启用' : '禁用' }
    )
  },
  { title: '排序', key: 'sort_order', width: 80, resizable: true },
  {
    title: '操作',
    key: 'actions',
    width: 160,
    fixed: 'right',
    render: (row) => h(
      NSpace,
      {},
      {
        default: () => [
          h(
            NButton,
            {
              size: 'small',
              type: 'primary',
              onClick: () => handleEdit(row)
            },
            { default: () => '编辑' }
          ),
          h(
            NButton,
            {
              size: 'small',
              type: 'error',
              onClick: () => handleDelete(row)
            },
            { default: () => '删除' }
          )
        ]
      }
    )
  }
]

const handleSearch = () => {
  searchQuery.value = filterValues.search || ''
  reload()
}

const handlePageChange = (page) => {
  pagination.page = page
  fetchPackages()
}

const handlePageSizeChange = (size) => {
  pagination.pageSize = size
  pagination.page = 1
  fetchPackages()
}

const resetForm = () => {
  editForm.id = null
  editForm.name = ''
  editForm.description = ''
  editForm.price = 0
  editForm.duration_days = 30
  editForm.device_limit = 3
  editForm.features = ''
  editForm.is_active = true
  editForm.is_featured = false
  editForm.sort_order = 0
}

const handleCreate = () => {
  resetForm()
  isCreating.value = true
  showEditDrawer.value = true
}

const handleEdit = (row) => {
  editForm.id = row.id
  editForm.name = row.name
  editForm.description = row.description || ''
  editForm.price = row.price
  editForm.duration_days = row.duration_days
  editForm.device_limit = row.device_limit
  // features is stored as JSON array string, convert to newline-separated for editing
  let feat = ''
  if (row.features) {
    try { feat = JSON.parse(row.features).join('\n') } catch { feat = row.features }
  }
  editForm.features = feat
  editForm.is_active = row.is_active
  editForm.is_featured = row.is_featured || false
  editForm.sort_order = row.sort_order
  isCreating.value = false
  showEditDrawer.value = true
}

const handleSavePackage = async () => {
  saving.value = true
  try {
    await formRef.value?.validate()

    const data = {
      name: editForm.name,
      description: editForm.description,
      price: editForm.price,
      duration_days: editForm.duration_days,
      device_limit: editForm.device_limit,
      features: editForm.features.trim()
        ? JSON.stringify(editForm.features.trim().split('\n').map(s => s.trim()).filter(Boolean))
        : null,
      is_active: editForm.is_active,
      is_featured: editForm.is_featured,
      sort_order: editForm.sort_order
    }

    if (isCreating.value) {
      await createPackage(data)
      message.success('套餐创建成功')
    } else {
      await updatePackage(editForm.id, data)
      message.success('套餐更新成功')
    }

    showEditDrawer.value = false
    fetchPackages()
  } catch (error) {
    if (error?.errors) return
    message.error((isCreating.value ? '创建' : '更新') + '套餐失败：' + (error.message || '未知错误'))
  } finally {
    saving.value = false
  }
}

const handleDelete = (row) => {
  dialog.error({
    title: '确认删除',
    content: `确定要删除套餐 ${row.name} 吗？此操作不可恢复！`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deletePackage(row.id)
        message.success('套餐删除成功')
        fetchPackages()
      } catch (error) {
        message.error('删除套餐失败：' + (error.message || '未知错误'))
      }
    }
  })
}

// 后端没有套餐批量接口：批量启/禁用沿用的是逐条调用 updatePackage 的既有实现，
// 只是把入口换成统一的批量选择（selection.selectedRows），不做假接口。
const handleBatchEnable = async () => {
  const targets = selection.selectedRows.value
  if (targets.length === 0) return
  try {
    await Promise.all(targets.map((pkg) => updatePackage(pkg.id, { ...pkg, is_active: true })))
    message.success('批量启用成功')
    selection.clear()
    fetchPackages()
  } catch { message.error('批量启用失败') }
}

const handleBatchDisable = async () => {
  const targets = selection.selectedRows.value
  if (targets.length === 0) return
  try {
    await Promise.all(targets.map((pkg) => updatePackage(pkg.id, { ...pkg, is_active: false })))
    message.success('批量禁用成功')
    selection.clear()
    fetchPackages()
  } catch { message.error('批量禁用失败') }
}

onMounted(() => {
  fetchPackages()
})

// KeepAlive 缓存激活时静默刷新数据（不清 loading 遮罩、不重置分页）
onActivated(() => {
  fetchPackages()
})
</script>

<style scoped>

.pull-indicator {
  position: fixed;
  top: 0;
  left: 50%;
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 96px;
  height: 34px;
  padding: 0 14px;
  border-radius: 999px;
  background: var(--bg-color, #fff);
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.12);
  font-size: 12px;
  color: var(--text-color-secondary, #666);
  transition: transform 0.15s ease;
}
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
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

/* 卡片外观、内边距、发丝分隔线、按压反馈、左侧多选框统一由全局 mobile-cards.css 提供，
   页面只保留内容布局（标题截断、价格配色）。 */
.card-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 15px;
  margin-right: 8px;
}

.card-value {
  min-width: 0;
  word-break: break-word;
}
.price-text {
  color: var(--success-color);
  font-weight: 600;
}

.mobile-toolbar { margin-bottom: 12px; }
.mobile-toolbar-title { font-size: 17px; font-weight: 600; margin-bottom: 10px; color: var(--text-color, #333); }

@media (max-width: 767px) {
  /* 页面根容器不再自带内边距：全局已给内容区 10px，页面再加就是白边浪费 */
  .admin-packages-page { padding-left: 0; padding-right: 0; }
  /* 搜索筛选条吸顶：复用全局 .app-sticky-toolbar，只把它的负外边距清零，
     否则工具条会比 10px 内容区各宽出 2px，在 393px 屏幕上就是横向溢出。 */
  .app-sticky-toolbar.mobile-sticky-toolbar {
    margin: 0 0 10px;
    padding-left: 0;
    padding-right: 0;
  }
  .app-sticky-toolbar.mobile-sticky-toolbar :deep(.sf-bar) {
    margin-bottom: 0;
  }
  .mobile-empty {
    text-align: center;
    padding: 40px 0;
    color: var(--text-color-secondary);
  }
}
</style>
