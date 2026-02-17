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
          />
          <!-- Mobile card list -->
          <div v-else class="mobile-card-list">
            <div v-for="device in devices" :key="device.id" class="mobile-card">
              <div class="card-row">
                <span class="label">设备名称</span>
                <span class="value">{{ device.device_name || device.software_name || '未知设备' }}</span>
              </div>
              <div class="card-row">
                <span class="label">IP 地址</span>
                <span class="value" style="font-family: monospace;">{{ device.ip_address || '-' }}</span>
              </div>
              <div class="card-row">
                <span class="label">地区</span>
                <span class="value">{{ device.region || '-' }}</span>
              </div>
              <div class="card-row">
                <span class="label">最后访问</span>
                <span class="value">{{ device.last_access ? new Date(device.last_access).toLocaleString('zh-CN') : '-' }}</span>
              </div>
              <div class="card-actions">
                <n-button size="small" type="error" @click="handleDelete(device.id)">删除</n-button>
              </div>
            </div>
          </div>
        </template>
      </n-spin>
    </n-card>

    <n-modal
      v-model:show="showDeleteModal"
      preset="dialog"
      title="确认删除"
      content="确定要删除此设备吗？删除后该设备将无法继续使用订阅。"
      positive-text="确认删除"
      negative-text="取消"
      @positive-click="handleConfirmDelete"
    />
  </div>
</template>

<script setup lang="tsx">
import { ref, h, onMounted } from 'vue'
import { NButton, NTime, useMessage } from 'naive-ui'
import { getSubscriptionDevices, deleteDevice } from '@/api/subscription'
import { useAppStore } from '@/stores/app'

interface Device {
  id: number
  device_name: string
  software_name: string
  user_agent: string
  ip_address: string
  region: string
  device_fingerprint: string
  last_access: string
  created_at: string
}

const appStore = useAppStore()
const message = useMessage()
const loading = ref(false)
const devices = ref<Device[]>([])
const showDeleteModal = ref(false)
const deleteDeviceId = ref<number | null>(null)

const parseDeviceName = (userAgent: string): string => {
  if (!userAgent) return '未知设备'
  
  // 简单的 UA 解析
  if (userAgent.includes('iPhone')) return 'iPhone'
  if (userAgent.includes('iPad')) return 'iPad'
  if (userAgent.includes('Android')) return 'Android 设备'
  if (userAgent.includes('Windows')) return 'Windows 电脑'
  if (userAgent.includes('Macintosh')) return 'Mac 电脑'
  if (userAgent.includes('Linux')) return 'Linux 设备'
  
  return '未知设备'
}

const columns = [
  {
    title: '设备名称',
    key: 'device_name',
    minWidth: 120,
    render: (row: Device) => {
      const name = row.device_name || row.software_name || parseDeviceName(row.user_agent)
      return h('span', name)
    }
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
    render: (row: Device) => row.region || '-'
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
  loading.value = true
  try {
    const res = await getSubscriptionDevices()
    devices.value = res.data || []
  } catch (error: any) {
    message.error(error.message || '获取设备列表失败')
  } finally {
    loading.value = false
  }
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
    await fetchDevices()
  } catch (error: any) {
    message.error(error.message || '删除设备失败')
  } finally {
    deleteDeviceId.value = null
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
  .device-page { padding: 0; max-width: none; }
}

.mobile-card-list { display: flex; flex-direction: column; gap: 10px; }
.mobile-card { background: var(--bg-color, #fff); border-radius: 10px; box-shadow: 0 1px 4px rgba(0,0,0,0.08); padding: 12px 14px; }
.card-row { display: flex; justify-content: space-between; align-items: center; padding: 4px 0; font-size: 13px; }
.card-row .label { color: var(--text-color-secondary, #999); flex-shrink: 0; }
.card-row .value { text-align: right; word-break: break-all; color: var(--text-color, #333); }
.card-actions { display: flex; gap: 8px; padding-top: 8px; border-top: 1px solid var(--border-color, #f0f0f0); margin-top: 6px; }
</style>
