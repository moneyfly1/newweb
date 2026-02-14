<template>
  <div class="logs-container">
    <n-card title="系统日志" :bordered="false">
      <n-tabs type="line" animated @update:value="handleTabChange">
        <n-tab-pane name="audit" tab="审计日志">
          <n-data-table
            remote
            :columns="auditColumns"
            :data="auditData"
            :loading="auditLoading"
            :pagination="auditPagination"
            :bordered="false"
          />
        </n-tab-pane>

        <n-tab-pane name="login" tab="登录日志">
          <n-data-table
            remote
            :columns="loginColumns"
            :data="loginData"
            :loading="loginLoading"
            :pagination="loginPagination"
            :bordered="false"
          />
        </n-tab-pane>

        <n-tab-pane name="registration" tab="注册日志">
          <n-data-table
            remote
            :columns="registrationColumns"
            :data="registrationData"
            :loading="registrationLoading"
            :pagination="registrationPagination"
            :bordered="false"
          />
        </n-tab-pane>

        <n-tab-pane name="subscription" tab="订阅日志">
          <n-data-table
            remote
            :columns="subscriptionColumns"
            :data="subscriptionData"
            :loading="subscriptionLoading"
            :pagination="subscriptionPagination"
            :bordered="false"
          />
        </n-tab-pane>

        <n-tab-pane name="balance" tab="余额日志">
          <n-data-table
            remote
            :columns="balanceColumns"
            :data="balanceData"
            :loading="balanceLoading"
            :pagination="balancePagination"
            :bordered="false"
          />
        </n-tab-pane>

        <n-tab-pane name="commission" tab="佣金日志">
          <n-data-table
            remote
            :columns="commissionColumns"
            :data="commissionData"
            :loading="commissionLoading"
            :pagination="commissionPagination"
            :bordered="false"
          />
        </n-tab-pane>
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup lang="tsx">
import { ref, reactive, h, onMounted } from 'vue'
import { NCard, NTabs, NTabPane, NDataTable, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { getAuditLogs, getLoginLogs, getRegistrationLogs, getSubscriptionLogs, getBalanceLogs, getCommissionLogs } from '@/api/admin'

const message = useMessage()
const currentTab = ref('audit')

// Audit logs
const auditLoading = ref(false)
const auditData = ref<any[]>([])
const auditPagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  onChange: (page: number) => {
    auditPagination.page = page
    loadAuditLogs()
  },
  onUpdatePageSize: (pageSize: number) => {
    auditPagination.pageSize = pageSize
    auditPagination.page = 1
    loadAuditLogs()
  },
})

const auditColumns: DataTableColumns = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '管理员ID', key: 'user_id', width: 100 },
  { title: '操作', key: 'action_type', width: 150 },
  { title: '目标类型', key: 'resource_type', width: 120 },
  { title: '目标ID', key: 'resource_id', width: 100 },
  { title: 'IP地址', key: 'ip_address', width: 140 },
  { title: '创建时间', key: 'created_at', width: 180 },
]

// Login logs
const loginLoading = ref(false)
const loginData = ref<any[]>([])
const loginPagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  onChange: (page: number) => {
    loginPagination.page = page
    loadLoginLogs()
  },
  onUpdatePageSize: (pageSize: number) => {
    loginPagination.pageSize = pageSize
    loginPagination.page = 1
    loadLoginLogs()
  },
})

const loginColumns: DataTableColumns = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '用户ID', key: 'user_id', width: 100 },
  { title: 'IP地址', key: 'ip_address', width: 140 },
  { title: '位置', key: 'location', ellipsis: { tooltip: true } },
  { title: '设备', key: 'user_agent', width: 150, ellipsis: { tooltip: true } },
  {
    title: '状态',
    key: 'login_status',
    width: 100,
    render: (row: any) =>
      h(NTag, { type: row.login_status === 'success' ? 'success' : 'error' }, { default: () => row.login_status }),
  },
  { title: '创建时间', key: 'login_time', width: 180 },
]

// Registration logs
const registrationLoading = ref(false)
const registrationData = ref<any[]>([])
const registrationPagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  onChange: (page: number) => {
    registrationPagination.page = page
    loadRegistrationLogs()
  },
  onUpdatePageSize: (pageSize: number) => {
    registrationPagination.pageSize = pageSize
    registrationPagination.page = 1
    loadRegistrationLogs()
  },
})

const registrationColumns: DataTableColumns = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '用户ID', key: 'user_id', width: 100 },
  { title: 'IP地址', key: 'ip_address', width: 140 },
  { title: '邀请码', key: 'invite_code', width: 150 },
  { title: '创建时间', key: 'created_at', width: 180 },
]

// Subscription logs
const subscriptionLoading = ref(false)
const subscriptionData = ref<any[]>([])
const subscriptionPagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  onChange: (page: number) => {
    subscriptionPagination.page = page
    loadSubscriptionLogs()
  },
  onUpdatePageSize: (pageSize: number) => {
    subscriptionPagination.pageSize = pageSize
    subscriptionPagination.page = 1
    loadSubscriptionLogs()
  },
})

const subscriptionColumns: DataTableColumns = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '用户ID', key: 'user_id', width: 100 },
  { title: '操作', key: 'action_type', width: 150 },
  { title: '详情', key: 'description', ellipsis: { tooltip: true } },
  { title: '创建时间', key: 'created_at', width: 180 },
]

// Balance logs
const balanceLoading = ref(false)
const balanceData = ref<any[]>([])
const balancePagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  onChange: (page: number) => {
    balancePagination.page = page
    loadBalanceLogs()
  },
  onUpdatePageSize: (pageSize: number) => {
    balancePagination.pageSize = pageSize
    balancePagination.page = 1
    loadBalanceLogs()
  },
})

const balanceColumns: DataTableColumns = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '用户ID', key: 'user_id', width: 100 },
  { title: '类型', key: 'change_type', width: 120 },
  { title: '金额', key: 'amount', width: 120 },
  { title: '余额', key: 'balance_after', width: 120 },
  { title: '备注', key: 'description', ellipsis: { tooltip: true } },
  { title: '创建时间', key: 'created_at', width: 180 },
]

// Commission logs
const commissionLoading = ref(false)
const commissionData = ref<any[]>([])
const commissionPagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  onChange: (page: number) => {
    commissionPagination.page = page
    loadCommissionLogs()
  },
  onUpdatePageSize: (pageSize: number) => {
    commissionPagination.pageSize = pageSize
    commissionPagination.page = 1
    loadCommissionLogs()
  },
})

const commissionColumns: DataTableColumns = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '用户ID', key: 'inviter_id', width: 100 },
  { title: '来源用户ID', key: 'invitee_id', width: 120 },
  { title: '金额', key: 'amount', width: 120 },
  { title: '类型', key: 'commission_type', width: 120 },
  { title: '创建时间', key: 'created_at', width: 180 },
]

const loadAuditLogs = async () => {
  auditLoading.value = true
  try {
    const res = await getAuditLogs({
      page: auditPagination.page,
      page_size: auditPagination.pageSize,
    })
    auditData.value = res.data?.items || res.data || []
    auditPagination.itemCount = res.data?.total || 0
  } catch (error: any) {
    message.error(error.message || '加载失败')
  } finally {
    auditLoading.value = false
  }
}

const loadLoginLogs = async () => {
  loginLoading.value = true
  try {
    const res = await getLoginLogs({
      page: loginPagination.page,
      page_size: loginPagination.pageSize,
    })
    loginData.value = res.data?.items || res.data || []
    loginPagination.itemCount = res.data?.total || 0
  } catch (error: any) {
    message.error(error.message || '加载失败')
  } finally {
    loginLoading.value = false
  }
}

const loadRegistrationLogs = async () => {
  registrationLoading.value = true
  try {
    const res = await getRegistrationLogs({
      page: registrationPagination.page,
      page_size: registrationPagination.pageSize,
    })
    registrationData.value = res.data?.items || res.data || []
    registrationPagination.itemCount = res.data?.total || 0
  } catch (error: any) {
    message.error(error.message || '加载失败')
  } finally {
    registrationLoading.value = false
  }
}

const loadSubscriptionLogs = async () => {
  subscriptionLoading.value = true
  try {
    const res = await getSubscriptionLogs({
      page: subscriptionPagination.page,
      page_size: subscriptionPagination.pageSize,
    })
    subscriptionData.value = res.data?.items || res.data || []
    subscriptionPagination.itemCount = res.data?.total || 0
  } catch (error: any) {
    message.error(error.message || '加载失败')
  } finally {
    subscriptionLoading.value = false
  }
}

const loadBalanceLogs = async () => {
  balanceLoading.value = true
  try {
    const res = await getBalanceLogs({
      page: balancePagination.page,
      page_size: balancePagination.pageSize,
    })
    balanceData.value = res.data?.items || res.data || []
    balancePagination.itemCount = res.data?.total || 0
  } catch (error: any) {
    message.error(error.message || '加载失败')
  } finally {
    balanceLoading.value = false
  }
}

const loadCommissionLogs = async () => {
  commissionLoading.value = true
  try {
    const res = await getCommissionLogs({
      page: commissionPagination.page,
      page_size: commissionPagination.pageSize,
    })
    commissionData.value = res.data?.items || res.data || []
    commissionPagination.itemCount = res.data?.total || 0
  } catch (error: any) {
    message.error(error.message || '加载失败')
  } finally {
    commissionLoading.value = false
  }
}

const handleTabChange = (value: string) => {
  currentTab.value = value
  switch (value) {
    case 'audit':
      if (auditData.value.length === 0) loadAuditLogs()
      break
    case 'login':
      if (loginData.value.length === 0) loadLoginLogs()
      break
    case 'registration':
      if (registrationData.value.length === 0) loadRegistrationLogs()
      break
    case 'subscription':
      if (subscriptionData.value.length === 0) loadSubscriptionLogs()
      break
    case 'balance':
      if (balanceData.value.length === 0) loadBalanceLogs()
      break
    case 'commission':
      if (commissionData.value.length === 0) loadCommissionLogs()
      break
  }
}

onMounted(() => {
  loadAuditLogs()
})
</script>

<style scoped>
.logs-container {
  padding: 20px;
}

@media (max-width: 767px) {
  .logs-container { padding: 8px; }
}
</style>
