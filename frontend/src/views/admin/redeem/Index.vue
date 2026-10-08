<template>
  <div class="redeem-container admin-page-shell">
    <n-card :title="appStore.isMobile ? undefined : '兑换码管理'" :bordered="false" class="admin-main-card">
      <template v-if="!appStore.isMobile" #header-extra>
        <n-button type="primary" @click="handleGenerate">
          批量生成
        </n-button>
      </template>

      <!-- Mobile toolbar：吸顶工具条；不用全局 .mobile-toolbar-row（它会把行内按钮按网格拉成整行宽而溢出屏幕） -->
      <div v-if="appStore.isMobile" class="app-sticky-toolbar redeem-toolbar">
        <div class="redeem-toolbar__title">兑换码管理</div>
        <div class="redeem-toolbar__actions">
          <n-button size="small" type="primary" @click="handleGenerate">批量生成</n-button>
        </div>
      </div>

      <!-- 全选 / 批量操作：公共组件（桌面在列表上方，手机固定在底部标签栏上方） -->
      <BatchSelectBar
        :total="selection.total.value"
        :selected-count="selection.count.value"
        :all-selected="selection.allSelected.value"
        :indeterminate="selection.indeterminate.value"
        label="个兑换码"
        @toggle-all="selection.toggleAll"
        @clear="selection.clear"
      >
        <n-button size="small" type="error" :disabled="!selection.count.value" @click="handleBatchDelete">
          批量删除
        </n-button>
      </BatchSelectBar>

      <template v-if="!appStore.isMobile">
        <n-data-table
          class="unified-admin-table"
          remote
          :columns="columns"
          :data="codes"
          :loading="loading"
          :pagination="pagination"
          :bordered="false"
          :row-key="(row: any) => row.id"
          v-model:checked-row-keys="checkedRowKeys"
          @update:sorter="handleSorterChange"
        />
      </template>

      <template v-else>
        <div v-if="codes.length === 0" class="mobile-empty">暂无数据</div>
        <div v-else class="mobile-card-list">
          <div
            v-for="row in codes"
            :key="row.id"
            class="mobile-card is-selectable"
            :class="{ 'is-selected': selection.isSelected(row) }"
            @click="selection.toggle(row)"
          >
            <div class="card-check" @click.stop>
              <n-checkbox :checked="selection.isSelected(row)" @update:checked="() => selection.toggle(row)" />
            </div>
            <div class="card-header">
              <span class="card-title">{{ row.code }}</span>
              <n-tag :type="row.type === 'balance' ? 'success' : 'info'" size="small">
                {{ row.type === 'balance' ? '余额' : '套餐' }}
              </n-tag>
            </div>
            <div class="card-body">
              <div class="card-row">
                <span class="card-label">类型</span>
                <span>{{ row.type === 'balance' ? formatCurrency(row.value) : `套餐#${row.value}` }}</span>
              </div>
              <div class="card-row">
                <span class="card-label">状态</span>
                <n-tag :type="row.status === 'unused' ? 'success' : row.status === 'used' ? 'default' : 'warning'" size="small">
                  {{ row.status === 'unused' ? '未使用' : row.status === 'used' ? '已使用' : '已过期' }}
                </n-tag>
              </div>
              <div class="card-row">
                <span class="card-label">使用次数</span>
                <span>{{ row.used_count || 0 }} / {{ row.max_uses || 1 }}</span>
              </div>
              <div class="card-row">
                <span class="card-label">创建时间</span>
                <span>{{ formatFullDateTime(row.created_at) }}</span>
              </div>
            </div>
            <div class="card-actions" @click.stop>
              <n-button size="small" @click="copyCode(row.code)">复制</n-button>
              <n-button size="small" type="error" :disabled="row.used_count > 0" @click="handleDelete(row.id)">删除</n-button>
            </div>
          </div>
        </div>

        <n-pagination
          class="list-pagination"
          v-model:page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :item-count="pagination.itemCount"
          :page-sizes="pagination.pageSizes"
          show-size-picker
          @update:page="loadCodes"
          @update:page-size="(ps: number) => { pagination.pageSize = ps; pagination.page = 1; loadCodes() }"
        />
      </template>
    </n-card>

    <common-drawer
      v-model:show="showGenerateDrawer"
      title="批量生成兑换码"
      :width="500"
      show-footer
      :loading="submitting"
      @confirm="handleSubmit"
      @cancel="showGenerateDrawer = false"
    >
      <n-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-placement="left"
        label-width="100"
      >
        <n-form-item label="类型" path="type">
          <n-select
            v-model:value="formData.type"
            placeholder="请选择类型"
            :options="typeOptions"
          />
        </n-form-item>

        <n-form-item label="数值" path="value">
          <n-input-number
            v-model:value="formData.value"
            :placeholder="formData.type === 'balance' ? '充值金额（元）' : '套餐ID'"
            :min="1"
            style="width: 100%"
          />
        </n-form-item>

        <n-form-item label="生成数量" path="quantity">
          <n-input-number
            v-model:value="formData.quantity"
            placeholder="请输入生成数量"
            :min="1"
            :max="100"
            style="width: 100%"
          />
        </n-form-item>

        <n-alert type="info" style="margin-top: 12px">
          {{ formData.type === 'balance' ? `将生成 ${formData.quantity} 个面值为 ${formData.value} 元的余额兑换码` : `将生成 ${formData.quantity} 个套餐ID为 ${formData.value} 的套餐兑换码` }}
        </n-alert>
      </n-form>
    </common-drawer>

    <n-modal
      v-model:show="showCodesModal"
      preset="card"
      title="生成的兑换码"
      :style="appStore.isMobile ? 'width: 95vw; max-width: 600px' : 'width: 600px'"
      :segmented="{ content: 'soft' }"
    >
      <n-alert type="success" style="margin-bottom: 16px">
        成功生成 {{ generatedCodes.length }} 个兑换码，请及时保存
      </n-alert>
      
      <n-space vertical :size="8">
        <div
          v-for="(code, index) in generatedCodes"
          :key="index"
          class="code-item"
        >
          <n-text code>{{ code }}</n-text>
          <n-button
            text
            size="small"
            @click="copyCode(code)"
          >
            复制
          </n-button>
        </div>
      </n-space>

      <template #footer>
        <n-space justify="end">
          <n-button @click="copyAllCodes">复制全部</n-button>
          <n-button type="primary" @click="showCodesModal = false">关闭</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, h, onActivated, onMounted } from 'vue'
import { usePageLoading } from '@/composables/usePageLoading'
import { NButton, NTag, NSpace, useMessage, useDialog } from 'naive-ui'
import { listRedeemCodes, createRedeemCodes, deleteRedeemCode } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { useBatchSelection } from '@/composables/useBatchSelection'
import BatchSelectBar from '@/components/BatchSelectBar.vue'
import { copyToClipboard as clipboardCopy } from '@/utils/clipboard'
import { formatFullDateTime } from '@/utils/date'
import { formatCurrency } from '@/utils/amount'
import CommonDrawer from '@/components/CommonDrawer.vue'

const message = useMessage()
const dialog = useDialog()
const appStore = useAppStore()

const { loading, beginLoad, endLoad } = usePageLoading()
const submitting = ref(false)
const codes = ref<any[]>([])
const showGenerateDrawer = ref(false)
const showCodesModal = ref(false)
const generatedCodes = ref<string[]>([])
const formRef = ref()
const sortState = ref({ sort: 'id', order: 'desc' })

// 全选 / 多选：全站统一实现（桌面表格与手机卡片共用同一份选择状态）
const selection = useBatchSelection(() => codes.value)
// Naive 的表格要的是数组，这里做一层桥接，保证两边状态一致
const checkedRowKeys = computed({
  get: () => [...selection.selectedKeys.value],
  set: (keys) => { selection.selectedKeys.value = new Set(keys) }
})

const formData = reactive({
  type: 'balance',
  value: 10,
  quantity: 1
})

const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
  onChange: (page: number) => {
    pagination.page = page
    loadCodes()
  },
  onUpdatePageSize: (pageSize: number) => {
    pagination.pageSize = pageSize
    pagination.page = 1
    loadCodes()
  }
})

const typeOptions = [
  { label: '余额充值', value: 'balance' },
  { label: '套餐兑换', value: 'package' }
]

const rules = {
  type: { required: true, message: '请选择类型', trigger: 'change' },
  value: { required: true, type: 'number', message: '请输入数值', trigger: 'blur' },
  quantity: { required: true, type: 'number', message: '请输入生成数量', trigger: 'blur' }
}

const columns = [
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 60, resizable: true, sorter: 'default' },
  {
    title: '兑换码',
    key: 'code',
    width: 200,
    resizable: true,
    render: (row: any) => {
      return h(NSpace, { align: 'center' }, {
        default: () => [
          h('code', { style: { fontSize: '13px' } }, row.code),
          h(NButton, {
            text: true,
            size: 'small',
            onClick: () => copyCode(row.code)
          }, { default: () => '复制' })
        ]
      })
    }
  },
  {
    title: '类型',
    key: 'type',
    width: 100,
    resizable: true,
    render: (row: any) => h(NTag, { type: row.type === 'balance' ? 'success' : 'info' }, { default: () => row.type === 'balance' ? '余额' : '套餐' })
  },
  {
    title: '数值',
    key: 'value',
    width: 100,
    resizable: true,
    render: (row: any) => row.type === 'balance' ? formatCurrency(row.value) : `套餐#${row.value}`
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    resizable: true,
    render: (row: any) => {
      const statusMap: Record<string, { text: string; type: any }> = {
        unused: { text: '未使用', type: 'success' },
        used: { text: '已使用', type: 'default' },
        expired: { text: '已过期', type: 'warning' }
      }
      const status = statusMap[row.status] || { text: row.status, type: 'default' }
      return h(NTag, { type: status.type }, { default: () => status.text })
    }
  },
  { title: '使用次数', key: 'used_count', width: 100, resizable: true, render: (row: any) => `${row.used_count || 0} / ${row.max_uses || 1}` },
  { title: '创建时间', key: 'created_at', width: 160, resizable: true, render: (row: any) => formatFullDateTime(row.created_at) },
  {
    title: '操作',
    key: 'actions',
    width: 100,
    fixed: 'right' as const,
    render: (row: any) => {
      return h(NButton, {
        size: 'small',
        type: 'error',
        disabled: row.used_count > 0,
        onClick: () => handleDelete(row.id)
      }, { default: () => '删除' })
    }
  }
]

const loadCodes = async () => {
  beginLoad(codes.value.length > 0)
  try {
    const res = await listRedeemCodes({
      page: pagination.page,
      page_size: pagination.pageSize,
      sort: sortState.value.sort,
      order: sortState.value.order,
    })
    codes.value = res.data.items || []
    pagination.itemCount = res.data.total || 0
  } catch (error: any) {
    message.error(error.message || '加载兑换码列表失败')
  } finally {
    endLoad()
  }
}

const handleSorterChange = (sorter: any) => {
  if (sorter && sorter.columnKey && sorter.order) {
    sortState.value.sort = sorter.columnKey
    sortState.value.order = sorter.order === 'ascend' ? 'asc' : 'desc'
  } else {
    sortState.value.sort = 'id'
    sortState.value.order = 'desc'
  }
  pagination.page = 1
  loadCodes()
}

const handleGenerate = () => {
  formData.type = 'balance'
  formData.value = 10
  formData.quantity = 1
  showGenerateDrawer.value = true
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  submitting.value = true
  try {
    // 套餐兑换码必须携带 package_id，否则兑换时会把套餐ID当作天数（后端防御见 AdminCreateRedeemCodes）
    const payload: any = { ...formData }
    if (payload.type === 'package') {
      payload.package_id = payload.value
    }
    const res = await createRedeemCodes(payload)
    message.success('生成成功')
    
    generatedCodes.value = res.data.codes || []
    showGenerateDrawer.value = false
    showCodesModal.value = true
    
    await loadCodes()
  } catch (error: any) {
    message.error(error.message || '生成失败')
  } finally {
    submitting.value = false
  }
}

const copyCode = async (code: string) => {
  const ok = await clipboardCopy(code)
  ok ? message.success('复制成功') : message.error('复制失败')
}

const copyAllCodes = async () => {
  const text = generatedCodes.value.join('\n')
  const ok = await clipboardCopy(text)
  ok ? message.success('已复制全部兑换码') : message.error('复制失败')
}

const handleDelete = (id: number) => {
  dialog.warning({
    title: '确认删除',
    content: '确定要删除这个兑换码吗？此操作不可恢复。',
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteRedeemCode(id)
        message.success('删除成功')
        await loadCodes()
      } catch (error: any) {
        message.error(error.message || '删除失败')
      }
    }
  })
}

const handleBatchDelete = () => {
  // 后端没有批量删除接口，按 id 逐个调用（与单条删除同一接口）
  const rows = selection.selectedRows.value
  if (!rows.length) return
  // 与单条删除一致：已使用的兑换码不可删除
  const used = rows.filter((c: any) => c.used_count > 0)
  if (used.length > 0) {
    message.warning(`选中中有 ${used.length} 个已使用的兑换码，不可删除`)
    return
  }
  dialog.warning({
    title: '批量删除',
    content: `确定要删除选中的 ${rows.length} 个兑换码吗？此操作不可恢复。`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await Promise.all(rows.map((r: any) => deleteRedeemCode(r.id)))
        message.success('批量删除成功')
        selection.clear()
        loadCodes()
      } catch { message.error('批量删除失败') }
    }
  })
}

onMounted(() => {
  loadCodes()
})

// KeepAlive 缓存激活时静默刷新数据（不清 loading 遮罩、不重置分页）
onActivated(() => {
  loadCodes()
})
</script>

<style scoped>
.code-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--bg-page-color, #f5f5f5);
  border-radius: 8px;
}

.code-item :deep(.n-text) {
  min-width: 0;
  overflow-wrap: anywhere;
}

/* 分页：桌面靠右，手机居中 */
.list-pagination { margin-top: 16px; justify-content: flex-end; }

/* 手机端工具条：标题 + 操作按钮各占一行，按钮允许换行（不会横向溢出） */
/* 卡片内容区在手机端无左右内边距，工具条不再用全局的 -12px 出血，避免越过卡片边界 */
.redeem-toolbar { display: flex; flex-direction: column; gap: 8px; margin-left: 0; margin-right: 0; }
.redeem-toolbar__title { font-size: 16px; font-weight: 650; color: var(--text-color, #333); }
.redeem-toolbar__actions { display: flex; flex-wrap: wrap; gap: 8px; }
.redeem-toolbar__actions .n-button { flex: 1 1 0; min-width: 0; }

@media (max-width: 767px) {
  /* 左右留白与底部批量栏空间都由全局统一给，页面不再自己加内边距 */
  .list-pagination { justify-content: center; }
  /* 全局 .admin-page-shell .mobile-card 用了 padding:0 !important，
     这里补回左侧复选框的位置，并让选中态可见 */
  .redeem-container :deep(.mobile-card.is-selectable) { padding-left: 44px !important; }
  .redeem-container :deep(.mobile-card.is-selected) {
    border-color: var(--primary-color, #4f46e5) !important;
    background: color-mix(in srgb, var(--primary-color-soft, rgba(102, 126, 234, 0.08)) 70%, var(--bg-color, #fff)) !important;
  }
}
</style>
