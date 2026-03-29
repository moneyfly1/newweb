<template>
  <div class="payment-gateways-container admin-page-shell">
    <n-card title="支付方式管理" :bordered="false" class="admin-main-card">
      <n-spin :show="loading">
        <n-space vertical :size="16">
          <n-alert type="info" title="提示">
            在这里可以查看和测试所有支付方式的配置状态
          </n-alert>

          <n-data-table
            class="unified-admin-table"
            :columns="columns"
            :data="gateways"
            :bordered="false"
          />
        </n-space>
      </n-spin>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import { NButton, NTag, NSpace, useMessage } from 'naive-ui'
import { testPaymentGateway, listPaymentGateways } from '@/api/admin'

const message = useMessage()
const loading = ref(false)
const gateways = ref<any[]>([])
const checkedRowKeys = ref<any[]>([])

const columns = [
  { type: 'selection' },
  {
    title: '支付方式',
    key: 'display_name',
    width: 150,
  },
  {
    title: '标识',
    key: 'name',
    width: 120,
  },
  {
    title: '配置状态',
    key: 'configured',
    width: 120,
    render: (row: any) => {
      return h(NTag, {
        type: row.configured ? 'success' : 'default'
      }, {
        default: () => row.configured ? '已配置' : '未配置'
      })
    }
  },
  {
    title: '验证状态',
    key: 'valid',
    width: 120,
    render: (row: any) => {
      if (!row.configured) {
        return h(NTag, { type: 'default' }, { default: () => '-' })
      }
      return h(NTag, {
        type: row.valid ? 'success' : 'error'
      }, {
        default: () => row.valid ? '有效' : '无效'
      })
    }
  },
  {
    title: '错误信息',
    key: 'error',
    ellipsis: { tooltip: true },
    render: (row: any) => row.error || '-'
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    fixed: 'right' as const,
    render: (row: any) => {
      return h(NSpace, {}, {
        default: () => [
          h(NButton, {
            size: 'small',
            type: 'primary',
            disabled: !row.configured,
            onClick: () => handleTest(row)
          }, { default: () => '测试配置' })
        ]
      })
    }
  }
]

const loadGateways = async () => {
  loading.value = true
  try {
    const res = await listPaymentGateways()
    gateways.value = res.data.gateways || []
  } catch (error: any) {
    message.error(error.message || '加载支付方式失败')
  } finally {
    loading.value = false
  }
}

const handleTest = async (row: any) => {
  loading.value = true
  try {
    const res = await testPaymentGateway(row.name)
    message.success(res.data.message || '测试成功')
    await loadGateways()
  } catch (error: any) {
    message.error(error.message || '测试失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadGateways()
})
</script>

<style scoped>
</style>
