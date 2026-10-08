<template>
  <div class="announcements-container">
    <n-card :title="appStore.isMobile ? undefined : '公告管理'" :bordered="false">
      <template v-if="!appStore.isMobile" #header-extra>
        <n-button type="primary" @click="handleCreate">
          发布公告
        </n-button>
      </template>

      <div v-if="appStore.isMobile" class="mobile-toolbar app-sticky-toolbar">
        <div class="mobile-toolbar-title">公告管理</div>
        <n-button size="small" type="primary" @click="handleCreate">发布公告</n-button>
      </div>

      <!-- 全选 / 批量操作：公共组件（桌面在表格上方，手机固定在底部标签栏上方） -->
      <BatchSelectBar
        :total="selection.total.value"
        :selected-count="selection.count.value"
        :all-selected="selection.allSelected.value"
        :indeterminate="selection.indeterminate.value"
        label="条公告"
        @toggle-all="selection.toggleAll"
        @clear="selection.clear"
      >
        <n-button size="small" type="success" :disabled="!selection.count.value" @click="handleBatchEnable">批量启用</n-button>
        <n-button size="small" type="warning" :disabled="!selection.count.value" @click="handleBatchDisable">批量禁用</n-button>
        <n-button size="small" type="error" :disabled="!selection.count.value" @click="handleBatchDelete">批量删除</n-button>
      </BatchSelectBar>

      <template v-if="!appStore.isMobile">
        <n-data-table
          remote
          :columns="columns"
          :data="tableData"
          :loading="loading"
          :pagination="pagination"
          :bordered="false"
          :row-key="(row: any) => row.id"
          v-model:checked-row-keys="checkedRowKeys"
          @update:sorter="handleSorterChange"
        />
      </template>

      <template v-else>
        <n-spin :show="loading">
          <div v-if="tableData.length === 0" style="text-align: center; padding: 40px 0; color: var(--text-color-secondary);">
            暂无数据
          </div>
          <div v-else class="mobile-card-list">
            <div
              v-for="item in tableData"
              :key="item.id"
              class="mobile-card is-selectable"
              :class="{ 'is-selected': selection.isSelected(item) }"
              @click="selection.toggle(item)"
            >
              <div class="card-check" @click.stop>
                <n-checkbox :checked="selection.isSelected(item)" @update:checked="() => selection.toggle(item)" />
              </div>
              <div class="card-header">
                <div class="card-title">{{ item.title }}</div>
                <n-tag :type="item.is_active ? 'success' : 'default'" size="small">
                  {{ item.is_active ? '启用' : '禁用' }}
                </n-tag>
              </div>
              <div class="card-body">
                <div class="card-row">
                  <span class="card-label">类型</span>
                  <component :is="getTypeTag(item.type)" />
                </div>
                <div class="card-row">
                  <span class="card-label">创建时间</span>
                  <span>{{ item.created_at }}</span>
                </div>
              </div>
              <div class="card-actions" @click.stop>
                <n-button size="small" type="primary" @click="handleEdit(item)">编辑</n-button>
                <n-popconfirm @positive-click="handleDelete(item.id)">
                  <template #trigger>
                    <n-button size="small" type="error">删除</n-button>
                  </template>
                  确定删除此公告吗？
                </n-popconfirm>
              </div>
            </div>
          </div>
        </n-spin>
        <n-pagination
          v-if="tableData.length > 0"
          v-model:page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :item-count="pagination.itemCount"
          :page-sizes="pagination.pageSizes || [10, 20, 50, 100]"
          style="margin-top: 16px; justify-content: flex-end"
          @update:page="(p: number) => { pagination.page = p; loadData() }"
          @update:page-size="(ps: number) => { pagination.pageSize = ps; pagination.page = 1; loadData() }"
        />
      </template>
    </n-card>

    <common-drawer
      v-model:show="showDrawer"
      :title="modalTitle"
      :width="600"
      show-footer
      :loading="submitting"
      @confirm="handleSubmit"
      @cancel="showDrawer = false"
    >
      <n-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-placement="top"
      >
        <n-form-item label="标题" path="title">
          <n-input v-model:value="formData.title" placeholder="请输入公告标题" />
        </n-form-item>

        <n-form-item label="内容" path="content">
          <n-input
            v-model:value="formData.content"
            type="textarea"
            placeholder="请输入公告内容"
            :rows="6"
          />
        </n-form-item>

        <n-form-item label="类型" path="type">
          <n-select
            v-model:value="formData.type"
            :options="typeOptions"
            placeholder="请选择公告类型"
          />
        </n-form-item>

        <n-form-item label="状态" path="is_active">
          <n-switch v-model:value="formData.is_active">
            <template #checked>启用</template>
            <template #unchecked>禁用</template>
          </n-switch>
        </n-form-item>
      </n-form>
    </common-drawer>
  </div>
</template>

<script setup lang="tsx">
import { ref, computed, h, onActivated, onMounted } from 'vue'
import {
  NCard,
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NSelect,
  NSwitch,
  NSpace,
  NTag,
  NPopconfirm,
  NSpin,
  useMessage,
  useDialog,
  type DataTableColumns,
} from 'naive-ui'
import { listAnnouncements, createAnnouncement, updateAnnouncement, deleteAnnouncement } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { useTable } from '@/composables/useTable'
import { useBatchSelection } from '@/composables/useBatchSelection'
import CommonDrawer from '@/components/CommonDrawer.vue'
import BatchSelectBar from '@/components/BatchSelectBar.vue'

const appStore = useAppStore()

const message = useMessage()
const dialog = useDialog()
const formRef = ref()
const submitting = ref(false)
const showDrawer = ref(false)
const modalTitle = ref('发布公告')
const isEdit = ref(false)

// 统一表格状态（loading / data / 分页 / 排序）
const { loading, tableData, pagination, loadData, handleSorterChange } =
  useTable(listAnnouncements)

// 全选 / 多选：全站统一实现（桌面表格与手机卡片共用同一份选择状态）
const selection = useBatchSelection(() => tableData.value)
// Naive 表格要的是数组，这里做一层桥接，保证两边状态一致
const checkedRowKeys = computed({
  get: () => [...selection.selectedKeys.value],
  set: (keys) => { selection.selectedKeys.value = new Set(keys) },
})

const formData = ref({
  id: 0,
  title: '',
  content: '',
  type: 'info',
  is_active: true,
})

const rules = {
  title: { required: true, message: '请输入标题', trigger: 'blur' },
  content: { required: true, message: '请输入内容', trigger: 'blur' },
  type: { required: true, message: '请选择类型', trigger: 'change' },
}

const typeOptions = [
  { label: '信息', value: 'info' },
  { label: '警告', value: 'warning' },
  { label: '成功', value: 'success' },
]

const getTypeTag = (type: string) => {
  const typeMap: Record<string, any> = {
    info: { type: 'info', text: '信息' },
    warning: { type: 'warning', text: '警告' },
    success: { type: 'success', text: '成功' },
  }
  const config = typeMap[type] || typeMap.info
  return h(NTag, { type: config.type }, { default: () => config.text })
}

const columns: DataTableColumns = [
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 80, resizable: true, sorter: 'default' },
  { title: '标题', key: 'title', ellipsis: { tooltip: true } },
  {
    title: '类型',
    key: 'type',
    width: 100,
    resizable: true,
    render: (row: any) => getTypeTag(row.type),
  },
  {
    title: '状态',
    key: 'is_active',
    width: 100,
    resizable: true,
    render: (row: any) =>
      h(NTag, { type: row.is_active ? 'success' : 'default' }, { default: () => (row.is_active ? '启用' : '禁用') }),
  },
  { title: '创建时间', key: 'created_at', width: 180, resizable: true, sorter: (a: any, b: any) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime() },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    render: (row: any) =>
      h(NSpace, {}, () => [
        h(
          NButton,
          { size: 'small', onClick: () => handleEdit(row) },
          { default: () => '编辑' }
        ),
        h(
          NPopconfirm,
          {
            onPositiveClick: () => handleDelete(row.id),
          },
          {
            default: () => '确定删除此公告吗？',
            trigger: () =>
              h(NButton, { size: 'small', type: 'error' }, { default: () => '删除' }),
          }
        ),
      ]),
  },
]

const handleCreate = () => {
  isEdit.value = false
  modalTitle.value = '发布公告'
  formData.value = {
    id: 0,
    title: '',
    content: '',
    type: 'info',
    is_active: true,
  }
  showDrawer.value = true
}

const handleEdit = (row: any) => {
  isEdit.value = true
  modalTitle.value = '编辑公告'
  formData.value = { ...row }
  showDrawer.value = true
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  submitting.value = true
  try {
    if (isEdit.value) {
      await updateAnnouncement(formData.value.id, formData.value)
    } else {
      await createAnnouncement(formData.value)
    }
    message.success(isEdit.value ? '更新成功' : '创建成功')
    showDrawer.value = false
    loadData()
  } catch (error: any) {
    message.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (id: number) => {
  try {
    await deleteAnnouncement(id)
    message.success('删除成功')
    loadData()
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

const handleBatchEnable = async () => {
  const rows = selection.selectedRows.value
  if (!rows.length) return
  try {
    await Promise.all(rows.map(row => updateAnnouncement(row.id, { ...row, is_active: true })))
    message.success('批量启用成功')
    selection.clear()
    loadData()
  } catch { message.error('批量启用失败') }
}

const handleBatchDisable = async () => {
  const rows = selection.selectedRows.value
  if (!rows.length) return
  try {
    await Promise.all(rows.map(row => updateAnnouncement(row.id, { ...row, is_active: false })))
    message.success('批量禁用成功')
    selection.clear()
    loadData()
  } catch { message.error('批量禁用失败') }
}

const handleBatchDelete = async () => {
  const rows = selection.selectedRows.value
  if (!rows.length) return
  dialog.warning({
    title: '批量删除',
    content: `确定要删除选中的 ${rows.length} 个公告吗？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await Promise.all(rows.map(row => deleteAnnouncement(row.id)))
        message.success('批量删除成功')
        selection.clear()
        loadData()
      } catch { message.error('批量删除失败') }
    }
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
.announcements-container {
  padding: 20px;
}

@media (max-width: 767px) {
  /* 手机端左右留白由全局布局统一给（10px），页面根容器不再自带左右 padding；
     卡片样式统一走全局 .mobile-card（App 风格），页面不再覆盖。 */
  .announcements-container { padding: 8px 0; }
}

/* 吸顶工具栏：全局 .app-sticky-toolbar 用负边距对齐卡片内边距，
   而带 .mobile-card-list 的卡片内容区左右内边距是 0，负边距会把工具条撑出卡片。 */
.mobile-toolbar.app-sticky-toolbar {
  margin-left: 0;
  margin-right: 0;
}

/* 可选中卡片左侧给复选框留位 */

.mobile-toolbar { margin-bottom: 12px; }
.mobile-toolbar-title { font-size: 17px; font-weight: 600; margin-bottom: 10px; color: var(--text-color, #333); }
</style>
