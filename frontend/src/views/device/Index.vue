<template>
  <div class="device-page">
    <n-card title="设备管理" :bordered="false">
      <template #header-extra>
        <n-button @click="fetchDevices" :loading="loading">
          <template #icon>
            <n-icon><svg viewBox="0 0 24 24"><path fill="currentColor" d="M17.65 6.35A7.958 7.958 0 0 0 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08A5.99 5.99 0 0 1 12 18c-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/></svg></n-icon>
          </template>
          刷新
        </n-button>
      </template>

      <!-- 全选 / 批量操作：公共组件（桌面在列表上方，手机端自动变成固定在底部的操作栏） -->
      <BatchSelectBar
        :total="selection.total.value"
        :selected-count="selection.count.value"
        :all-selected="selection.allSelected.value"
        :indeterminate="selection.indeterminate.value"
        label="台设备"
        @toggle-all="selection.toggleAll"
        @clear="selection.clear"
      >
        <n-button size="small" type="error" :disabled="!selection.count.value" @click="handleBatchDelete">
          批量删除
        </n-button>
      </BatchSelectBar>

      <n-spin :show="loading">
        <n-empty v-if="!loading && devices.length === 0" description="暂无设备记录">
          <template #extra>
            <n-text depth="3">当前账号未绑定任何设备</n-text>
          </template>
        </n-empty>

        <template v-else>
          <!-- Desktop table -->
          <n-data-table v-if="!appStore.isMobile"
            :columns="columns"
            :data="devices"
            :bordered="false"
            :single-line="false"
            :pagination="false"
            :row-key="rowKey"
            v-model:checked-row-keys="checkedRowKeys"
          />
          <!-- Mobile card list -->
          <div v-else class="mobile-card-list">
            <div
              v-for="device in devices"
              :key="device.id"
              class="mobile-card is-selectable"
              :class="{ 'is-selected': selection.isSelected(device) }"
              @click="selection.toggle(device)"
            >
              <div class="card-check" @click.stop>
                <n-checkbox :checked="selection.isSelected(device)" @update:checked="() => selection.toggle(device)" />
              </div>
              <div class="card-row">
                <span class="label">设备名称</span>
                <span class="value">
                  {{ device.device_name || device.software_name || '未知设备' }}
                  <n-tag :type="device.is_online ? 'success' : 'default'" size="tiny" :bordered="false">
                    {{ device.is_online ? '在线' : '离线' }}
                  </n-tag>
                </span>
              </div>
              <div class="card-row">
                <span class="label">客户端</span>
                <span class="value">{{ device.software_name || '未知' }}</span>
              </div>
              <div class="card-row">
                <span class="label">系统</span>
                <span class="value">{{ device.os_name || '-' }}</span>
              </div>
              <div class="card-row">
                <span class="label">IP 地址</span>
                <span class="value" style="font-family: monospace;">{{ device.ip_address || '-' }}</span>
              </div>
              <div class="card-row">
                <span class="label">地区</span>
                <span class="value">{{ formatLocation(device.region) }}</span>
              </div>
              <div class="card-row" @click.stop>
                <span class="label">备注</span>
                <n-input
                  :value="device.remark || ''"
                  size="small"
                  placeholder="添加备注..."
                  style="max-width: 160px; text-align: right"
                  @update:value="(val: string) => device.remark = val"
                  @blur="saveRemark(device)"
                  @keyup.enter="($event.target as HTMLInputElement)?.blur()"
                />
              </div>
              <div class="card-row">
                <span class="label">最后访问</span>
                <span class="value">{{ formatFullDateTime(device.last_access) }}</span>
              </div>
              <div class="card-actions" @click.stop>
                <n-button size="small" type="error" @click="handleDelete(device.id)">删除</n-button>
              </div>
            </div>
          </div>
        </template>
      </n-spin>
      <n-pagination
        v-if="totalDevices > pageSize"
        v-model:page="currentPage"
        v-model:page-size="pageSize"
        :item-count="totalDevices"
        :page-sizes="[10, 20, 50]"
        show-size-picker
        style="margin-top: 16px; justify-content: flex-end"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <common-drawer
      v-model:show="showDeleteModal"
      title="确认删除"
      :width="appStore.isMobile ? '100%' : 400"
      show-footer
      @confirm="handleConfirmDelete"
      @cancel="showDeleteModal = false"
    >
      <div>确定要删除此设备吗？删除后该设备将无法继续使用订阅。</div>
    </common-drawer>
  </div>
</template>

<script setup lang="tsx">
import { ref, h, onMounted, computed } from 'vue'
import { NButton, NTime, NInput, NTag, useMessage, useDialog } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { getSubscriptionDevices, deleteDevice, updateDeviceRemark } from '@/api/subscription'
import { useAppStore } from '@/stores/app'
import { formatLocation } from '@/utils/i18n'
import { formatFullDateTime } from '@/utils/date'
import CommonDrawer from '@/components/CommonDrawer.vue'
import BatchSelectBar from '@/components/BatchSelectBar.vue'
import { useBatchSelection } from '@/composables/useBatchSelection'
import { usePageLoading } from '@/composables/usePageLoading'

interface Device {
  id: number
  device_name: string
  software_name: string
  software_version: string
  os_name: string
  os_version: string
  device_model: string
  device_brand: string
  subscription_type: string
  user_agent: string
  ip_address: string
  region: string
  remark: string
  device_fingerprint: string
  last_access: string
  // is_online 后端按「3 分钟内心跳」或「24 小时内拉过订阅」计算得出
  is_online?: boolean
  created_at: string
  _origRemark?: string
}

const appStore = useAppStore()
const message = useMessage()
const dialog = useDialog()
const { loading, beginLoad, endLoad } = usePageLoading()
const devices = ref<Device[]>([])
const showDeleteModal = ref(false)
const deleteDeviceId = ref<number | null>(null)
const currentPage = ref(1)
const pageSize = ref(10)
const totalDevices = ref(0)

// 全选 / 多选：唯一的选择状态，桌面表格与手机卡片共用（公共组件 useBatchSelection + BatchSelectBar）
const selection = useBatchSelection<Device>(() => devices.value)
const rowKey = (row: Device) => row.id
// 桌面表格仍需数组形式的勾选键，用 computed 桥接，保证与手机端卡片状态一致
const checkedRowKeys = computed<number[]>({
  get: () => [...selection.selectedKeys.value] as number[],
  set: (keys) => { selection.selectedKeys.value = new Set(keys) },
})

const columns: DataTableColumns<Device> = [
  { type: 'selection' },
  {
    title: '设备名称',
    key: 'device_name',
    minWidth: 150,
    render: (row: Device) => {
      return h('span', row.device_name || row.software_name || '未知设备')
    }
  },
  {
    title: '状态',
    key: 'is_online',
    width: 80,
    render: (row: Device) => h(NTag, {
      type: row.is_online ? 'success' : 'default',
      size: 'small',
      bordered: false
    }, { default: () => row.is_online ? '在线' : '离线' })
  },
  {
    title: '备注',
    key: 'remark',
    width: 160,
    render: (row: Device) => {
      return h(NInput, {
        value: row.remark || '',
        size: 'small',
        placeholder: '添加备注...',
        onUpdateValue: (val: string) => { row.remark = val },
        onBlur: () => saveRemark(row),
        onKeyup: (e: KeyboardEvent) => { if (e.key === 'Enter') { (e.target as HTMLInputElement)?.blur() } }
      })
    }
  },
  {
    title: '客户端',
    key: 'software_name',
    width: 120,
    render: (row: Device) => row.software_name || '未知'
  },
  {
    title: '版本',
    key: 'software_version',
    width: 80,
    render: (row: Device) => row.software_version || '-'
  },
  {
    title: '系统',
    key: 'os_name',
    width: 80,
    render: (row: Device) => row.os_name || '-'
  },
  {
    title: '设备型号',
    key: 'device_model',
    width: 130,
    render: (row: Device) => row.device_model || '-'
  },
  {
    title: 'IP 地址',
    key: 'ip_address',
    width: 140,
    resizable: true,
    render: (row: Device) => row.ip_address || '-'
  },
  {
    title: '地区',
    key: 'region',
    width: 120,
    resizable: true,
    render: (row: Device) => formatLocation(row.region)
  },
  {
    title: '最后访问',
    key: 'last_access',
    width: 180,
    resizable: true,
    render: (row: Device) => {
      if (!row.last_access) return '-'
      return h(NTime, { time: new Date(row.last_access), type: 'relative' })
    }
  },
  {
    title: '添加时间',
    key: 'created_at',
    width: 180,
    resizable: true,
    render: (row: Device) => {
      return h(NTime, { time: new Date(row.created_at), format: 'yyyy-MM-dd HH:mm:ss' })
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 80,
    render: (row: Device) => {
      return h(
        NButton,
        {
          size: 'small',
          type: 'error',
          secondary: true,
          onClick: () => handleDelete(row.id)
        },
        { default: () => '删除' }
      )
    }
  }
]

const fetchDevices = async () => {
  beginLoad(devices.value.length > 0)
  try {
    const res = await getSubscriptionDevices({ page: currentPage.value, page_size: pageSize.value })
    const data = res.data
    if (Array.isArray(data)) {
      devices.value = data
      totalDevices.value = data.length
    } else {
      devices.value = data?.items || []
      totalDevices.value = data?.total || 0
    }
    // 记录原始备注，供失焦保存时做变更检测
    devices.value.forEach((d: any) => { d._origRemark = d.remark || '' })
  } catch (error: any) {
    message.error(error.message || '获取设备列表失败')
  } finally {
    endLoad()
  }
}

const handlePageChange = (page: number) => {
  currentPage.value = page
  fetchDevices()
}

const handlePageSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
  fetchDevices()
}

const handleDelete = (id: number) => {
  deleteDeviceId.value = id
  showDeleteModal.value = true
}

const handleConfirmDelete = async () => {
  if (!deleteDeviceId.value) return

  try {
    await deleteDevice(deleteDeviceId.value)
    message.success('设备删除成功')
    showDeleteModal.value = false
    await fetchDevices()
  } catch (error: any) {
    message.error(error.message || '删除设备失败')
  } finally {
    deleteDeviceId.value = null
  }
}

// 后端没有批量解绑接口：按《手机端改造契约》§2.1 第 2 条逐条调单条删除接口，
// 用 Promise.allSettled 保证一条失败不影响其余，最后汇总「成功 X / 失败 Y」。
const handleBatchDelete = () => {
  const rows = selection.selectedRows.value
  if (!rows.length) return
  dialog.warning({
    title: '批量删除设备',
    content: `将影响 ${rows.length} 台设备，删除后这些设备将无法继续使用订阅。此操作不可恢复。`,
    positiveText: '确定删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      const results = await Promise.allSettled(rows.map(d => deleteDevice(d.id)))
      const ok = results.filter(r => r.status === 'fulfilled').length
      const fail = results.length - ok
      if (ok > 0) message.success(`批量删除完成：成功 ${ok} 台`)
      if (fail > 0) message.error(`批量删除部分失败：失败 ${fail} 台`)
      selection.clear()
      await fetchDevices()
    },
  })
}

const saveRemark = async (row: Device) => {
  const newVal = row.remark || ''
  // 未修改则不发请求
  if (newVal === (row._origRemark || '')) return
  try {
    await updateDeviceRemark(row.id, newVal)
    row._origRemark = newVal
    message.success('备注已保存')
  } catch (error: any) {
    message.error(error.message || '保存备注失败')
  }
}

onMounted(() => {
  fetchDevices()
})
</script>

<style scoped>
.device-page {
  padding: 24px;
}

@media (max-width: 767px) {
  /* 契约 §1：页面根容器不再自带左右内边距（全局已给 10px 留白）；
     卡片 16px 圆角由全局统一提供，页面不再自己写 */
  .device-page { padding: 0; max-width: none; }
  /* 可选中卡片：左侧给复选框让出 44px（本页卡片自带 padding，必须 !important 压过它，
     否则复选框会压在「设备名称」上） */
  .device-page :deep(.mobile-card.is-selectable) { padding-left: 44px !important; }
}

.mobile-card-list { display: flex; flex-direction: column; gap: 10px; }
/* 手机端卡片内边距/圆角由全局 .mobile-card 与 token 接管，这里兜底背景与轻阴影 */
.mobile-card { background: var(--bg-color, #fff); box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04), 0 6px 16px rgba(15, 23, 42, 0.04); padding: 12px 14px; }
.card-row { display: flex; justify-content: space-between; align-items: center; padding: 4px 0; font-size: 13px; }
.card-row .label { color: var(--text-color-secondary, #999); flex-shrink: 0; }
.card-row .value { text-align: right; word-break: break-all; color: var(--text-color, #333); }
.card-actions { display: flex; gap: 8px; padding-top: 8px; border-top: 1px solid var(--border-color, #f0f0f0); margin-top: 6px; }
</style>
