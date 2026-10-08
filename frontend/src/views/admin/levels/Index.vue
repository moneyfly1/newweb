<template>
  <div class="levels-container">
    <n-card :title="appStore.isMobile ? undefined : '用户等级管理'">
      <template v-if="!appStore.isMobile" #header-extra>
        <n-button type="primary" @click="handleAdd">
          添加等级
        </n-button>
      </template>

      <div v-if="appStore.isMobile" class="mobile-toolbar app-sticky-toolbar">
        <div class="mobile-toolbar-title">用户等级管理</div>
        <n-button size="small" type="primary" @click="handleAdd">添加等级</n-button>
      </div>

      <!-- 全选 / 批量操作：公共组件（桌面在表格上方，手机固定在底部标签栏上方） -->
      <BatchSelectBar
        :total="selection.total.value"
        :selected-count="selection.count.value"
        :all-selected="selection.allSelected.value"
        :indeterminate="selection.indeterminate.value"
        label="个等级"
        @toggle-all="selection.toggleAll"
        @clear="selection.clear"
      >
        <n-button size="small" type="error" :disabled="!selection.count.value" @click="handleBatchDelete">批量删除</n-button>
      </BatchSelectBar>

      <template v-if="!appStore.isMobile">
        <n-data-table
          :columns="columns"
          :data="levels"
          :loading="loading"
          :bordered="false"
          :row-key="(row: any) => row.id"
          v-model:checked-row-keys="checkedRowKeys"
        />
      </template>

      <template v-else>
        <div class="mobile-card-list">
          <div
            v-for="row in levels"
            :key="row.id"
            class="mobile-card is-selectable"
            :class="{ 'is-selected': selection.isSelected(row) }"
            @click="selection.toggle(row)"
          >
            <div class="card-check" @click.stop>
              <n-checkbox :checked="selection.isSelected(row)" @update:checked="() => selection.toggle(row)" />
            </div>
            <div class="card-header">
              <span class="card-title">{{ row.level_name }}</span>
              <n-tag :type="row.is_active ? 'success' : 'default'" size="small">
                {{ row.is_active ? '启用' : '禁用' }}
              </n-tag>
            </div>
            <div class="card-body">
              <div class="card-row">
                <span class="card-label">折扣率</span>
                <span>{{ row.discount_rate }}%</span>
              </div>
              <div class="card-row">
                <span class="card-label">最低消费</span>
                <span>¥{{ formatAmount(row.min_consumption) }}</span>
              </div>
              <div class="card-row" v-if="row.benefits">
                <span class="card-label">权益说明</span>
                <span style="text-align: right; flex: 1; margin-left: 8px;">{{ row.benefits }}</span>
              </div>
            </div>
            <div class="card-actions" @click.stop>
              <n-button size="small" type="primary" @click="handleEdit(row)">编辑</n-button>
              <n-button size="small" type="error" @click="handleDelete(row.id)">删除</n-button>
            </div>
          </div>
        </div>
      </template>

      <n-pagination
        v-if="pagination.itemCount > pagination.pageSize"
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :item-count="pagination.itemCount"
        :page-sizes="[10, 20, 50]"
        show-size-picker
        style="margin-top: 16px; justify-content: flex-end"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <common-drawer
      v-model:show="showDrawer"
      :title="isEdit ? '编辑等级' : '添加等级'"
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
        label-placement="left"
        label-width="100"
      >
        <n-form-item label="等级名称" path="level_name">
          <n-input v-model:value="formData.level_name" placeholder="请输入等级名称" />
        </n-form-item>

        <n-form-item label="等级数值" path="level_order">
          <n-input-number
            v-model:value="formData.level_order"
            placeholder="请输入等级数值"
            :min="0"
            style="width: 100%"
          />
        </n-form-item>

        <n-form-item label="折扣率" path="discount_rate">
          <n-input-number
            v-model:value="formData.discount_rate"
            placeholder="0-100，100表示无折扣"
            :min="0"
            :max="100"
            style="width: 100%"
          >
            <template #suffix>%</template>
          </n-input-number>
        </n-form-item>

        <n-form-item label="最低消费" path="min_consumption">
          <n-input-number
            v-model:value="formData.min_consumption"
            placeholder="请输入最低消费金额"
            :min="0"
            style="width: 100%"
          >
            <template #suffix>元</template>
          </n-input-number>
        </n-form-item>

        <n-form-item label="权益说明" path="benefits">
          <n-input
            v-model:value="formData.benefits"
            type="textarea"
            placeholder="请输入权益说明，每行一个权益"
            :rows="4"
          />
        </n-form-item>

        <n-form-item label="是否启用" path="is_active">
          <n-switch v-model:value="formData.is_active" />
        </n-form-item>
      </n-form>
    </common-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, h, onActivated, onMounted } from 'vue'
import { NButton, NTag, NSpace, useMessage, useDialog } from 'naive-ui'
import { listUserLevels, createUserLevel, updateUserLevel, deleteUserLevel } from '@/api/admin'
import { useTable } from '@/composables/useTable'
import { useBatchSelection } from '@/composables/useBatchSelection'
import { useAppStore } from '@/stores/app'
import { formatAmount, formatCurrency } from '@/utils/amount'
import CommonDrawer from '@/components/CommonDrawer.vue'
import BatchSelectBar from '@/components/BatchSelectBar.vue'

const message = useMessage()
const dialog = useDialog()
const appStore = useAppStore()

const submitting = ref(false)
const showDrawer = ref(false)
const isEdit = ref(false)
const formRef = ref()

// 统一表格状态
const { loading, tableData: levels, pagination, loadData } = useTable(listUserLevels)
const loadLevels = loadData
// 全选 / 多选：全站统一实现（桌面表格与手机卡片共用同一份选择状态）
const selection = useBatchSelection(() => levels.value)
// Naive 表格要的是数组，这里做一层桥接，保证两边状态一致
const checkedRowKeys = computed({
  get: () => [...selection.selectedKeys.value],
  set: (keys) => { selection.selectedKeys.value = new Set(keys) },
})

const formData = reactive({
  id: 0,
  level_name: '',
  level_order: 0,
  discount_rate: 100,
  min_consumption: 0,
  benefits: '',
  is_active: true
})

const rules = {
  level_name: { required: true, message: '请输入等级名称', trigger: 'blur' },
  level_order: { required: true, type: 'number', message: '请输入等级数值', trigger: 'blur' },
  discount_rate: { required: true, type: 'number', message: '请输入折扣率', trigger: 'blur' }
}

const columns = [
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 60, resizable: true, sorter: 'default' },
  { title: '等级名称', key: 'level_name', width: 120, resizable: true },
  { title: '等级数值', key: 'level_order', width: 100, resizable: true },
  {
    title: '折扣率',
    key: 'discount_rate',
    width: 100,
    resizable: true,
    render: (row: any) => `${row.discount_rate}%`
  },
  {
    title: '最低消费',
    key: 'min_consumption',
    width: 120,
    resizable: true,
    render: (row: any) => formatCurrency(row.min_consumption)
  },
  {
    title: '权益说明',
    key: 'benefits',
    width: 200,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '状态',
    key: 'is_active',
    width: 80,
    resizable: true,
    render: (row: any) => h(NTag, { type: row.is_active ? 'success' : 'default' }, { default: () => row.is_active ? '启用' : '禁用' })
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    fixed: 'right' as const,
    render: (row: any) => {
      return h(NSpace, {}, {
        default: () => [
          h(NButton, { size: 'small', onClick: () => handleEdit(row) }, { default: () => '编辑' }),
          h(NButton, {
            size: 'small',
            type: 'error',
            onClick: () => handleDelete(row.id)
          }, { default: () => '删除' })
        ]
      })
    }
  }
]

// 兼容原页面命名：翻页/改页大小后重新加载
const handlePageChange = (page: number) => {
  pagination.page = page
  loadLevels()
}

const handlePageSizeChange = (size: number) => {
  pagination.pageSize = size
  pagination.page = 1
  loadLevels()
}

const resetForm = () => {
  formData.id = 0
  formData.level_name = ''
  formData.level_order = 0
  formData.discount_rate = 100
  formData.min_consumption = 0
  formData.benefits = ''
  formData.is_active = true
}

const handleAdd = () => {
  resetForm()
  isEdit.value = false
  showDrawer.value = true
}

const handleEdit = (row: any) => {
  formData.id = row.id
  formData.level_name = row.level_name
  formData.level_order = row.level_order
  formData.discount_rate = row.discount_rate
  formData.min_consumption = row.min_consumption || 0
  formData.benefits = row.benefits || ''
  formData.is_active = row.is_active
  isEdit.value = true
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
    const data: any = { ...formData }
    delete data.id

    if (isEdit.value) {
      await updateUserLevel(formData.id, data)
      message.success('更新成功')
    } else {
      await createUserLevel(data)
      message.success('创建成功')
    }

    showDrawer.value = false
    await loadLevels()
  } catch (error: any) {
    message.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleDelete = (id: number) => {
  dialog.warning({
    title: '确认删除',
    content: '确定要删除这个等级吗？此操作不可恢复。',
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteUserLevel(id)
        message.success('删除成功')
        await loadLevels()
      } catch (error: any) {
        message.error(error.message || '删除失败')
      }
    }
  })
}

const handleBatchDelete = () => {
  const rows = selection.selectedRows.value
  if (!rows.length) return
  dialog.warning({
    title: '批量删除',
    content: `确定要删除选中的 ${rows.length} 个等级吗？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await Promise.all(rows.map(row => deleteUserLevel(row.id)))
        message.success('批量删除成功')
        selection.clear()
        loadLevels()
      } catch { message.error('批量删除失败') }
    }
  })
}

onMounted(() => {
  loadLevels()
})

// KeepAlive 缓存激活时静默刷新数据（不清 loading 遮罩、不重置分页）
onActivated(() => {
  loadLevels()
})
</script>

<style scoped>
.levels-container {
  padding: 20px;
}

@media (max-width: 767px) {
  /* 手机端左右留白由全局布局统一给（10px），页面根容器不再自带左右 padding；
     卡片样式统一走全局 .mobile-card（App 风格），页面不再覆盖。 */
  .levels-container { padding: 8px 0; }
}

/* 吸顶工具栏：全局 .app-sticky-toolbar 用负边距对齐卡片内边距，
   而带 .mobile-card-list 的卡片内容区左右内边距是 0，负边距会把工具条撑出卡片。 */
.mobile-toolbar.app-sticky-toolbar {
  margin-left: 0;
  margin-right: 0;
}

/* 可选中卡片左侧给复选框留位 */
.mobile-card.is-selectable {
  padding-left: 44px !important;
}

.mobile-toolbar { margin-bottom: 12px; }
.mobile-toolbar-title { font-size: 17px; font-weight: 600; margin-bottom: 10px; color: var(--text-color, #333); }
</style>
