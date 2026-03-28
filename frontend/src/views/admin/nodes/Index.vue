<template>
  <div class="nodes-container admin-page-shell">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">节点管理</h2>
        <p class="page-subtitle">管理所有接入系统的节点，支持链接导入、批量测试及状态监控</p>
      </div>
      <div class="header-right">
        <n-space>
          <n-input
            v-model:value="searchQuery"
            placeholder="搜索节点名称 / 地区 / 协议"
            clearable
            style="width: 250px"
            @keyup.enter="handleSearch"
          >
            <template #prefix><n-icon :component="SearchOutline" /></template>
          </n-input>
          <n-button secondary @click="handleRefresh" :loading="loading">
            <template #icon><n-icon><refresh-outline /></n-icon></template>
            刷新
          </n-button>
          <n-button type="info" secondary @click="showImportLinksDrawer = true">
            <template #icon><n-icon><link-outline /></n-icon></template>
            导入链接
          </n-button>
          <n-button type="primary" secondary @click="showImportSubDrawer = true">
            <template #icon><n-icon><cloud-download-outline /></n-icon></template>
            导入订阅
          </n-button>
        </n-space>
      </div>
    </div>

    <transition name="fade">
      <div v-if="checkedRowKeys.length > 0" class="batch-bar">
        <div class="batch-info">已选择 {{ checkedRowKeys.length }} 个节点</div>
        <n-space>
          <n-button size="small" type="success" secondary @click="handleBatchAction('enable')">批量启用</n-button>
          <n-button size="small" type="warning" secondary @click="handleBatchAction('disable')">批量禁用</n-button>
          <n-button size="small" type="info" secondary @click="handleBatchTest">批量测速</n-button>
          <n-button size="small" type="info" secondary @click="handleBatchAction('online')">批量上线</n-button>
          <n-button size="small" type="error" ghost @click="handleBatchDelete">批量删除</n-button>
        </n-space>
      </div>
    </transition>

    <n-card :bordered="false" class="main-card">
      <n-data-table
        remote
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :pagination="pagination"
        :bordered="false"
        :single-line="false"
        :row-key="(row: any) => row.id"
        :scroll-x="appStore.isMobile ? 980 : 1200"
        class="unified-admin-table"
        @update:checked-row-keys="handleCheck"
        @update:sorter="handleSorterChange"
        @update:page="(p: number) => { pagination.page = p; fetchData() }"
        @update:page-size="(ps: number) => { pagination.pageSize = ps; pagination.page = 1; fetchData() }"
      />
    </n-card>

    <common-drawer v-model:show="showImportSubDrawer" title="从订阅导入" :width="500" show-footer :loading="importing" @confirm="handleImportSubscription">
      <n-form label-placement="top">
        <n-form-item label="订阅链接 (Clash/V2Ray/Trojan)">
          <n-input v-model:value="subscriptionUrl" type="textarea" :rows="3" placeholder="https://example.com/sub?token=xxx" />
        </n-form-item>
        <n-alert type="info" :bordered="false">系统将自动解析订阅中的节点并同步到当前列表。</n-alert>
      </n-form>
    </common-drawer>

    <common-drawer v-model:show="showImportLinksDrawer" title="批量导入节点链接" :width="600" show-footer :loading="importing" @confirm="handleImportLinks">
      <n-form label-placement="top">
        <n-form-item label="节点链接列表">
          <n-input v-model:value="nodeLinks" type="textarea" :rows="12" placeholder="vmess://...&#10;vless://...&#10;trojan://..." />
        </n-form-item>
      </n-form>
    </common-drawer>

    <common-drawer v-model:show="showEditDrawer" title="节点属性编辑" :width="500" show-footer :loading="submitting" @confirm="handleSubmit">
      <n-form ref="formRef" :model="formData" :rules="rules" label-placement="left" label-width="100">
        <n-form-item label="名称" path="name"><n-input v-model:value="formData.name" /></n-form-item>
        <n-form-item label="地区" path="region"><n-input v-model:value="formData.region" /></n-form-item>
        <n-form-item label="排序权重" path="order_index"><n-input-number v-model:value="formData.order_index" :min="0" /></n-form-item>
        <n-form-item label="服务状态" path="is_active"><n-switch v-model:value="formData.is_active" /></n-form-item>
        <n-form-item label="备注"><n-input v-model:value="formData.description" type="textarea" :rows="3" /></n-form-item>
      </n-form>
    </common-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, h, onMounted } from 'vue'
import { NButton, NTag, NSpace, NIcon, NSwitch, useMessage, useDialog, type DataTableColumns, type FormInst, type TagProps } from 'naive-ui'
import {
  CloudDownloadOutline, LinkOutline, RefreshOutline,
  SpeedometerOutline, GlobeOutline, ShieldCheckmarkOutline, SearchOutline
} from '@vicons/ionicons5'
import { listAdminNodes, updateNode, deleteNode, importNodes, batchNodeAction, testNode } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import CommonDrawer from '@/components/CommonDrawer.vue'

const message = useMessage()
const dialog = useDialog()
const appStore = useAppStore()

const loading = ref(false)
const submitting = ref(false)
const importing = ref(false)
const showImportSubDrawer = ref(false)
const showImportLinksDrawer = ref(false)
const showEditDrawer = ref(false)
const tableData = ref<any[]>([])
const formRef = ref<FormInst | null>(null)
const editId = ref<number | null>(null)
const checkedRowKeys = ref<number[]>([])
const subscriptionUrl = ref('')
const nodeLinks = ref('')
const searchQuery = ref('')

const sortState = ref({ sort: 'order_index', order: 'asc' })
const pagination = reactive({ page: 1, pageSize: 20, itemCount: 0, showSizePicker: true, pageSizes: [20, 50, 100] })

const formData = reactive({ name: '', region: '', is_active: true, order_index: 0, description: '' })
const rules = { name: { required: true, message: '请输入节点名称' } }

const protocolColorMap: Record<string, NonNullable<TagProps['type']>> = { vmess: 'info', vless: 'success', trojan: 'warning', hysteria2: 'error' }
const statusColorMap: Record<string, NonNullable<TagProps['type']>> = { online: 'success', offline: 'error' }

const columns: DataTableColumns<any> = [
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 70, sorter: 'default' },
  {
    title: '节点名称',
    key: 'name',
    minWidth: 220,
    render: (row: any) => h('div', { class: 'cell-block' }, [
      h('div', { class: 'cell-inline' }, [
        h(NIcon, { component: ShieldCheckmarkOutline, class: 'node-icon', style: { color: row.is_active ? '#18a058' : '#d03050' } }),
        h('span', { class: 'cell-title' }, row.name)
      ]),
      h('div', { class: 'cell-sub' }, row.description || '暂无备注')
    ])
  },
  {
    title: '协议',
    key: 'type',
    width: 100,
    render: (row: any) => h(NTag, { type: protocolColorMap[row.type] || 'default', size: 'small', round: true, bordered: false }, { default: () => row.type?.toUpperCase() || '-' })
  },
  {
    title: '地区',
    key: 'region',
    width: 130,
    render: (row: any) => h('div', { class: 'cell-inline left-text' }, [
      h(NIcon, { component: GlobeOutline, size: 14, class: 'inline-icon' }),
      h('span', row.region || '-')
    ])
  },
  {
    title: '来源',
    key: 'source_index',
    width: 120,
    render: (row: any) => {
      if (row.source_index && row.source_index > 0) {
        return h(NTag, { type: 'info', size: 'small' }, { default: () => `订阅 #${row.source_index}` })
      }
      return h(NTag, { type: 'default', size: 'small' }, { default: () => '手动添加' })
    }
  },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: (row: any) => h(NTag, { type: statusColorMap[row.status] || 'default', size: 'small', ghost: true }, { default: () => row.status === 'online' ? '在线' : '离线' })
  },
  {
    title: '延迟',
    key: 'latency',
    width: 100,
    render: (row: any) => h('div', { class: row.latency > 0 ? 'latency-value left-text' : 'latency-offline left-text' }, [
      h(NIcon, { component: SpeedometerOutline, size: 14 }),
      h('span', row.latency > 0 ? `${row.latency}ms` : '-')
    ])
  },
  {
    title: '启用',
    key: 'is_active',
    width: 80,
    render: (row: any) => h('div', { class: 'left-text' }, [
      h(NSwitch, { size: 'small', value: row.is_active, onUpdateValue: (v) => handleToggleActive(row, v) })
    ])
  },
  { title: '排序', key: 'order_index', width: 80, sorter: 'default' },
  {
    title: '操作',
    key: 'actions',
    width: 170,
    fixed: 'right',
    render: (row: any) => h(NSpace, { justify: 'start' }, {
      default: () => [
        h(NButton, { size: 'tiny', quaternary: true, onClick: () => handleTest(row) }, { default: () => '测试' }),
        h(NButton, { size: 'tiny', type: 'primary', quaternary: true, onClick: () => handleEdit(row) }, { default: () => '编辑' }),
        h(NButton, { size: 'tiny', type: 'error', quaternary: true, onClick: () => handleDelete(row) }, { default: () => '删除' })
      ]
    })
  }
]

const fetchData = async () => {
  loading.value = true
  try {
    const res = await listAdminNodes({
      page: pagination.page,
      page_size: pagination.pageSize,
      sort: sortState.value.sort,
      order: sortState.value.order,
      search: searchQuery.value || undefined
    })
    tableData.value = res.data.items || []
    pagination.itemCount = res.data.total || 0
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.page = 1
  fetchData()
}

const handleSorterChange = (sorter: any) => {
  sortState.value.sort = sorter.columnKey || 'order_index'
  sortState.value.order = sorter.order === 'ascend' ? 'asc' : 'desc'
  fetchData()
}

const handleCheck = (keys: number[]) => { checkedRowKeys.value = keys }

const handleToggleActive = async (row: any, v: boolean) => {
  try {
    await updateNode(row.id, { is_active: v })
    message.success(`${v ? '启用' : '禁用'}成功`)
    row.is_active = v
  } catch {}
}

const handleTest = async (row: any) => {
  message.info(`正在测试节点 ${row.name}...`)
  try {
    const res = await testNode(row.id)
    row.latency = res.data.latency
    row.status = res.data.status
    if (res.data.status === 'online') {
      message.success(`${row.name} 在线，TCP 延迟 ${row.latency}ms`)
    } else {
      message.warning(`${row.name} 离线：当前测速仅检测服务器到节点 ${res.data.address || ''} 的 TCP 连通性`)
    }
  } catch {}
}

const handleBatchTest = async () => {
  if (checkedRowKeys.value.length === 0) return
  loading.value = true
  try {
    const targets = tableData.value.filter((row: any) => checkedRowKeys.value.includes(row.id))
    await Promise.all(targets.map((row: any) => handleTest(row)))
    message.success(`已完成 ${targets.length} 个节点测速`)
  } finally {
    loading.value = false
  }
}

const handleBatchAction = async (action: string) => {
  try {
    const res = await batchNodeAction({ ids: checkedRowKeys.value, action })
    message.success(`批量处理完成, 影响 ${res.data.affected} 个节点`)
    checkedRowKeys.value = []
    fetchData()
  } catch {}
}

const handleBatchDelete = () => {
  dialog.error({
    title: '危险操作',
    content: `确认彻底删除这 ${checkedRowKeys.value.length} 个节点吗？`,
    positiveText: '确认删除',
    onPositiveClick: () => handleBatchAction('delete')
  })
}

const handleEdit = (row: any) => {
  editId.value = row.id
  Object.assign(formData, { name: row.name, region: row.region, is_active: row.is_active, order_index: row.order_index, description: row.description || '' })
  showEditDrawer.value = true
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
    await updateNode(editId.value as number, formData)
    message.success('节点信息已更新')
    showEditDrawer.value = false
    fetchData()
  } catch {}
}

const handleDelete = (row: any) => {
  dialog.warning({
    title: '确认删除',
    content: `节点 "${row.name}" 删除后不可恢复。`,
    positiveText: '确定',
    onPositiveClick: async () => {
      await deleteNode(row.id)
      message.success('已删除')
      fetchData()
    }
  })
}

const handleImportSubscription = async () => {
  if (!subscriptionUrl.value) return message.warning('请输入链接')
  importing.value = true
  try {
    await importNodes({ type: 'subscription', url: subscriptionUrl.value })
    message.success('导入任务已提交')
    showImportSubDrawer.value = false
    fetchData()
  } finally { importing.value = false }
}

const handleImportLinks = async () => {
  if (!nodeLinks.value) return message.warning('请输入链接内容')
  importing.value = true
  try {
    await importNodes({ type: 'links', links: nodeLinks.value })
    message.success('批量导入成功')
    showImportLinksDrawer.value = false
    fetchData()
  } finally { importing.value = false }
}

const handleRefresh = () => fetchData()

onMounted(() => fetchData())
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px; }
.page-title { margin: 0; font-size: 24px; font-weight: 700; color: var(--n-title-text-color); }
.page-subtitle { margin: 4px 0 0; color: #888; font-size: 14px; }
.main-card { border-radius: 12px; box-shadow: 0 4px 16px rgba(0,0,0,0.05); }

.unified-admin-table :deep(.n-data-table-th),
.unified-admin-table :deep(.n-data-table-td) {
  text-align: left;
}
.unified-admin-table :deep(.n-data-table-td__content) {
  justify-content: flex-start;
  text-align: left;
}

.batch-bar {
  position: sticky; top: 0; z-index: 10;
  display: flex; justify-content: space-between; align-items: center;
  padding: 12px 20px; margin-bottom: 16px;
  background: #3b82f6; color: white; border-radius: 12px;
  box-shadow: 0 8px 24px rgba(59, 130, 246, 0.3);
}
.batch-info { font-weight: 600; }

.cell-block { display: flex; flex-direction: column; align-items: flex-start; gap: 4px; text-align: left; }
.cell-inline { display: flex; align-items: center; gap: 6px; justify-content: flex-start; text-align: left; }
.cell-title { font-weight: 600; color: #1f2937; }
.cell-sub { font-size: 12px; color: #64748b; }
.inline-icon { color: #94a3b8; }
.node-icon { opacity: 0.85; }
.left-text { display: flex; justify-content: flex-start; align-items: center; }

.latency-value { color: #18a058; font-weight: 600; font-family: monospace; }
.latency-offline { color: #d03050; gap: 4px; opacity: 0.65; }

@media (max-width: 767px) {
  .admin-page-shell { padding: 12px; }
  .page-header { flex-direction: column; align-items: flex-start; gap: 16px; }
  .batch-bar { flex-direction: column; gap: 12px; }
}

.fade-enter-active, .fade-leave-active { transition: all 0.3s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; transform: translateY(-20px); }
</style>
