<template>
  <div class="logs-container">
    <n-card title="系统日志" :bordered="false">
      <n-tabs type="line" animated @update:value="handleTabChange">
        <n-tab-pane name="audit" tab="审计日志">
          <template v-if="!appStore.isMobile">
            <n-data-table
              remote
              :columns="auditColumns"
              :data="auditData"
              :loading="auditLoading"
              :pagination="auditPagination"
              :bordered="false"
              @update:sorter="handleAuditSorterChange"
            />
          </template>
          <template v-else>
            <div class="mobile-card-list">
              <div v-for="item in auditData" :key="item.id" class="mobile-card">
                <div class="card-header">
                  <span class="card-title">ID: {{ item.id }}</span>
                </div>
                <div class="card-body">
                  <div class="card-row"><span class="card-label">管理员ID:</span><span>{{ item.user_id }}</span></div>
                  <div class="card-row"><span class="card-label">操作:</span><span>{{ item.action_type }}</span></div>
                  <div class="card-row"><span class="card-label">目标类型:</span><span>{{ item.resource_type }}</span></div>
                  <div class="card-row"><span class="card-label">目标ID:</span><span>{{ item.resource_id }}</span></div>
                  <div class="card-row"><span class="card-label">IP地址:</span><span>{{ item.ip_address }}</span></div>
                  <div class="card-row"><span class="card-label">创建时间:</span><span>{{ item.created_at }}</span></div>
                </div>
              </div>
            </div>
            <n-pagination
              v-model:page="auditPagination.page"
              v-model:page-size="auditPagination.pageSize"
              :item-count="auditPagination.itemCount"
              :page-sizes="auditPagination.pageSizes"
              show-size-picker
              style="margin-top: 16px; justify-content: center"
            />
          </template>
        </n-tab-pane>

        <n-tab-pane name="login" tab="登录日志">
          <template v-if="!appStore.isMobile">
            <n-data-table
              remote
              :columns="loginColumns"
              :data="loginData"
              :loading="loginLoading"
              :pagination="loginPagination"
              :bordered="false"
              @update:sorter="handleLoginSorterChange"
            />
          </template>
          <template v-else>
            <div class="mobile-card-list">
              <div v-for="item in loginData" :key="item.id" class="mobile-card">
                <div class="card-header">
                  <span class="card-title">ID: {{ item.id }}</span>
                  <n-tag :type="item.login_status === 'success' ? 'success' : 'error'" size="small">{{ item.login_status }}</n-tag>
                </div>
                <div class="card-body">
                  <div class="card-row"><span class="card-label">用户ID:</span><span>{{ item.user_id }}</span></div>
                  <div class="card-row"><span class="card-label">IP地址:</span><span>{{ item.ip_address }}</span></div>
                  <div class="card-row"><span class="card-label">位置:</span><span>{{ item.location }}</span></div>
                  <div class="card-row"><span class="card-label">设备:</span><span>{{ item.user_agent }}</span></div>
                  <div class="card-row"><span class="card-label">登录时间:</span><span>{{ item.login_time }}</span></div>
                </div>
              </div>
            </div>
            <n-pagination
              v-model:page="loginPagination.page"
              v-model:page-size="loginPagination.pageSize"
              :item-count="loginPagination.itemCount"
              :page-sizes="loginPagination.pageSizes"
              show-size-picker
              style="margin-top: 16px; justify-content: center"
            />
          </template>
        </n-tab-pane>

        <n-tab-pane name="registration" tab="注册日志">
          <template v-if="!appStore.isMobile">
            <n-data-table
              remote
              :columns="registrationColumns"
              :data="registrationData"
              :loading="registrationLoading"
              :pagination="registrationPagination"
              :bordered="false"
              @update:sorter="handleRegistrationSorterChange"
            />
          </template>
          <template v-else>
            <div class="mobile-card-list">
              <div v-for="item in registrationData" :key="item.id" class="mobile-card">
                <div class="card-header">
                  <span class="card-title">ID: {{ item.id }}</span>
                </div>
                <div class="card-body">
                  <div class="card-row"><span class="card-label">用户ID:</span><span>{{ item.user_id }}</span></div>
                  <div class="card-row"><span class="card-label">IP地址:</span><span>{{ item.ip_address }}</span></div>
                  <div class="card-row"><span class="card-label">邀请码:</span><span>{{ item.invite_code }}</span></div>
                  <div class="card-row"><span class="card-label">创建时间:</span><span>{{ item.created_at }}</span></div>
                </div>
              </div>
            </div>
            <n-pagination
              v-model:page="registrationPagination.page"
              v-model:page-size="registrationPagination.pageSize"
              :item-count="registrationPagination.itemCount"
              :page-sizes="registrationPagination.pageSizes"
              show-size-picker
              style="margin-top: 16px; justify-content: center"
            />
          </template>
        </n-tab-pane>

        <n-tab-pane name="subscription" tab="订阅日志">
          <template v-if="!appStore.isMobile">
            <n-data-table
              remote
              :columns="subscriptionColumns"
              :data="subscriptionData"
              :loading="subscriptionLoading"
              :pagination="subscriptionPagination"
              :bordered="false"
              @update:sorter="handleSubscriptionSorterChange"
            />
          </template>
          <template v-else>
            <div class="mobile-card-list">
              <div v-for="item in subscriptionData" :key="item.id" class="mobile-card">
                <div class="card-header">
                  <span class="card-title">ID: {{ item.id }}</span>
                </div>
                <div class="card-body">
                  <div class="card-row"><span class="card-label">用户ID:</span><span>{{ item.user_id }}</span></div>
                  <div class="card-row"><span class="card-label">操作:</span><span>{{ item.action_type }}</span></div>
                  <div class="card-row"><span class="card-label">详情:</span><span>{{ item.description }}</span></div>
                  <div class="card-row"><span class="card-label">创建时间:</span><span>{{ item.created_at }}</span></div>
                </div>
              </div>
            </div>
            <n-pagination
              v-model:page="subscriptionPagination.page"
              v-model:page-size="subscriptionPagination.pageSize"
              :item-count="subscriptionPagination.itemCount"
              :page-sizes="subscriptionPagination.pageSizes"
              show-size-picker
              style="margin-top: 16px; justify-content: center"
            />
          </template>
        </n-tab-pane>

        <n-tab-pane name="balance" tab="余额日志">
          <template v-if="!appStore.isMobile">
            <n-data-table
              remote
              :columns="balanceColumns"
              :data="balanceData"
              :loading="balanceLoading"
              :pagination="balancePagination"
              :bordered="false"
              @update:sorter="handleBalanceSorterChange"
            />
          </template>
          <template v-else>
            <div class="mobile-card-list">
              <div v-for="item in balanceData" :key="item.id" class="mobile-card">
                <div class="card-header">
                  <span class="card-title">ID: {{ item.id }}</span>
                </div>
                <div class="card-body">
                  <div class="card-row"><span class="card-label">用户ID:</span><span>{{ item.user_id }}</span></div>
                  <div class="card-row"><span class="card-label">类型:</span><span>{{ item.change_type }}</span></div>
                  <div class="card-row"><span class="card-label">金额:</span><span>{{ item.amount }}</span></div>
                  <div class="card-row"><span class="card-label">余额:</span><span>{{ item.balance_after }}</span></div>
                  <div class="card-row"><span class="card-label">备注:</span><span>{{ item.description }}</span></div>
                  <div class="card-row"><span class="card-label">创建时间:</span><span>{{ item.created_at }}</span></div>
                </div>
              </div>
            </div>
            <n-pagination
              v-model:page="balancePagination.page"
              v-model:page-size="balancePagination.pageSize"
              :item-count="balancePagination.itemCount"
              :page-sizes="balancePagination.pageSizes"
              show-size-picker
              style="margin-top: 16px; justify-content: center"
            />
          </template>
        </n-tab-pane>

        <n-tab-pane name="commission" tab="佣金日志">
          <template v-if="!appStore.isMobile">
            <n-data-table
              remote
              :columns="commissionColumns"
              :data="commissionData"
              :loading="commissionLoading"
              :pagination="commissionPagination"
              :bordered="false"
              @update:sorter="handleCommissionSorterChange"
            />
          </template>
          <template v-else>
            <div class="mobile-card-list">
              <div v-for="item in commissionData" :key="item.id" class="mobile-card">
                <div class="card-header">
                  <span class="card-title">ID: {{ item.id }}</span>
                </div>
                <div class="card-body">
                  <div class="card-row"><span class="card-label">用户ID:</span><span>{{ item.inviter_id }}</span></div>
                  <div class="card-row"><span class="card-label">来源用户ID:</span><span>{{ item.invitee_id }}</span></div>
                  <div class="card-row"><span class="card-label">金额:</span><span>{{ item.amount }}</span></div>
                  <div class="card-row"><span class="card-label">类型:</span><span>{{ item.commission_type }}</span></div>
                  <div class="card-row"><span class="card-label">创建时间:</span><span>{{ item.created_at }}</span></div>
                </div>
              </div>
            </div>
            <n-pagination
              v-model:page="commissionPagination.page"
              v-model:page-size="commissionPagination.pageSize"
              :item-count="commissionPagination.itemCount"
              :page-sizes="commissionPagination.pageSizes"
              show-size-picker
              style="margin-top: 16px; justify-content: center"
            />
          </template>
        </n-tab-pane>

        <n-tab-pane name="system" tab="系统日志">
          <n-space style="margin-bottom: 12px">
            <n-select v-model:value="systemLevelFilter" :options="levelOptions" placeholder="级别" clearable style="width: 120px" @update:value="loadSystemLogs" />
            <n-select v-model:value="systemModuleFilter" :options="moduleOptions" placeholder="模块" clearable style="width: 140px" @update:value="loadSystemLogs" />
          </n-space>
          <template v-if="!appStore.isMobile">
            <n-data-table
              remote
              :columns="systemColumns"
              :data="systemData"
              :loading="systemLoading"
              :pagination="systemPagination"
              :bordered="false"
              @update:sorter="handleSystemSorterChange"
            />
          </template>
          <template v-else>
            <div class="mobile-card-list">
              <div v-for="item in systemData" :key="item.id" class="mobile-card">
                <div class="card-header">
                  <n-tag :type="item.level === 'error' ? 'error' : item.level === 'warn' ? 'warning' : 'info'" size="small">{{ item.level }}</n-tag>
                  <span style="font-size: 12px; color: #999">{{ item.created_at }}</span>
                </div>
                <div class="card-body">
                  <div class="card-row"><span class="card-label">模块:</span><span>{{ item.module }}</span></div>
                  <div class="card-row"><span class="card-label">消息:</span><span>{{ item.message }}</span></div>
                  <div v-if="item.detail" class="card-row"><span class="card-label">详情:</span><span>{{ item.detail }}</span></div>
                </div>
              </div>
            </div>
            <n-pagination
              v-model:page="systemPagination.page"
              v-model:page-size="systemPagination.pageSize"
              :item-count="systemPagination.itemCount"
              :page-sizes="systemPagination.pageSizes"
              show-size-picker
              style="margin-top: 16px; justify-content: center"
            />
          </template>
        </n-tab-pane>
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup lang="tsx">
import { ref, reactive, h, onMounted } from 'vue'
import { NCard, NTabs, NTabPane, NDataTable, NTag, NPagination, NSpace, NSelect, useMessage, type DataTableColumns } from 'naive-ui'
import { getAuditLogs, getLoginLogs, getRegistrationLogs, getSubscriptionLogs, getBalanceLogs, getCommissionLogs, getSystemLogs } from '@/api/admin'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const message = useMessage()
const currentTab = ref('audit')

// Sort states for each tab
const auditSortState = ref({ sort: 'id', order: 'desc' })
const loginSortState = ref({ sort: 'id', order: 'desc' })
const registrationSortState = ref({ sort: 'id', order: 'desc' })
const subscriptionSortState = ref({ sort: 'id', order: 'desc' })
const balanceSortState = ref({ sort: 'id', order: 'desc' })
const commissionSortState = ref({ sort: 'id', order: 'desc' })
const systemSortState = ref({ sort: 'id', order: 'desc' })

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
  { title: 'ID', key: 'id', width: 80, resizable: true, sorter: 'default' },
  { title: '管理员ID', key: 'user_id', width: 100, resizable: true },
  { title: '操作', key: 'action_type', width: 150, resizable: true },
  { title: '目标类型', key: 'resource_type', width: 120, resizable: true },
  { title: '目标ID', key: 'resource_id', width: 100, resizable: true },
  { title: 'IP地址', key: 'ip_address', width: 140, resizable: true },
  { title: '创建时间', key: 'created_at', width: 180, resizable: true, sorter: (a: any, b: any) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime() },
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
  { title: 'ID', key: 'id', width: 80, resizable: true, sorter: 'default' },
  { title: '用户ID', key: 'user_id', width: 100, resizable: true },
  { title: 'IP地址', key: 'ip_address', width: 140, resizable: true },
  { title: '位置', key: 'location', ellipsis: { tooltip: true } },
  { title: '设备', key: 'user_agent', width: 150, resizable: true, ellipsis: { tooltip: true } },
  {
    title: '状态',
    key: 'login_status',
    width: 100,
    resizable: true,
    render: (row: any) =>
      h(NTag, { type: row.login_status === 'success' ? 'success' : 'error' }, { default: () => row.login_status }),
  },
  { title: '创建时间', key: 'login_time', width: 180, resizable: true, sorter: (a: any, b: any) => new Date(a.login_time).getTime() - new Date(b.login_time).getTime() },
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
  { title: 'ID', key: 'id', width: 80, resizable: true, sorter: 'default' },
  { title: '用户ID', key: 'user_id', width: 100, resizable: true },
  { title: 'IP地址', key: 'ip_address', width: 140, resizable: true },
  { title: '邀请码', key: 'invite_code', width: 150, resizable: true },
  { title: '创建时间', key: 'created_at', width: 180, resizable: true, sorter: (a: any, b: any) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime() },
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
  { title: 'ID', key: 'id', width: 80, resizable: true, sorter: 'default' },
  { title: '用户ID', key: 'user_id', width: 100, resizable: true },
  { title: '操作', key: 'action_type', width: 150, resizable: true },
  { title: '详情', key: 'description', ellipsis: { tooltip: true } },
  { title: '创建时间', key: 'created_at', width: 180, resizable: true, sorter: (a: any, b: any) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime() },
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
  { title: 'ID', key: 'id', width: 80, resizable: true, sorter: 'default' },
  { title: '用户ID', key: 'user_id', width: 100, resizable: true },
  { title: '类型', key: 'change_type', width: 120, resizable: true },
  { title: '金额', key: 'amount', width: 120, resizable: true },
  { title: '余额', key: 'balance_after', width: 120, resizable: true },
  { title: '备注', key: 'description', ellipsis: { tooltip: true } },
  { title: '创建时间', key: 'created_at', width: 180, resizable: true, sorter: (a: any, b: any) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime() },
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
  { title: 'ID', key: 'id', width: 80, resizable: true, sorter: 'default' },
  { title: '用户ID', key: 'inviter_id', width: 100, resizable: true },
  { title: '来源用户ID', key: 'invitee_id', width: 120, resizable: true },
  { title: '金额', key: 'amount', width: 120, resizable: true },
  { title: '类型', key: 'commission_type', width: 120, resizable: true },
  { title: '创建时间', key: 'created_at', width: 180, resizable: true, sorter: (a: any, b: any) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime() },
]

const loadAuditLogs = async () => {
  auditLoading.value = true
  try {
    const res = await getAuditLogs({
      page: auditPagination.page,
      page_size: auditPagination.pageSize,
      sort: auditSortState.value.sort,
      order: auditSortState.value.order,
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
      sort: loginSortState.value.sort,
      order: loginSortState.value.order,
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
      sort: registrationSortState.value.sort,
      order: registrationSortState.value.order,
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
      sort: subscriptionSortState.value.sort,
      order: subscriptionSortState.value.order,
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
      sort: balanceSortState.value.sort,
      order: balanceSortState.value.order,
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
      sort: commissionSortState.value.sort,
      order: commissionSortState.value.order,
    })
    commissionData.value = res.data?.items || res.data || []
    commissionPagination.itemCount = res.data?.total || 0
  } catch (error: any) {
    message.error(error.message || '加载失败')
  } finally {
    commissionLoading.value = false
  }
}

// System logs
const systemLoading = ref(false)
const systemData = ref<any[]>([])
const systemLevelFilter = ref<string | null>(null)
const systemModuleFilter = ref<string | null>(null)
const levelOptions = [
  { label: 'info', value: 'info' },
  { label: 'warn', value: 'warn' },
  { label: 'error', value: 'error' },
]
const moduleOptions = [
  { label: '调度器', value: 'scheduler' },
  { label: '邮件', value: 'email' },
  { label: '通知', value: 'notify' },
  { label: '支付', value: 'payment' },
  { label: '系统', value: 'system' },
]
const systemPagination = reactive({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  onChange: (page: number) => {
    systemPagination.page = page
    loadSystemLogs()
  },
  onUpdatePageSize: (pageSize: number) => {
    systemPagination.pageSize = pageSize
    systemPagination.page = 1
    loadSystemLogs()
  },
})

const systemColumns: DataTableColumns = [
  { title: 'ID', key: 'id', width: 70, resizable: true, sorter: 'default' },
  {
    title: '级别', key: 'level', width: 80, resizable: true,
    render: (row: any) => h(NTag, { type: row.level === 'error' ? 'error' : row.level === 'warn' ? 'warning' : 'info', size: 'small' }, { default: () => row.level }),
  },
  { title: '模块', key: 'module', width: 100, resizable: true },
  { title: '消息', key: 'message', ellipsis: { tooltip: true } },
  { title: '详情', key: 'detail', width: 200, resizable: true, ellipsis: { tooltip: true } },
  { title: '时间', key: 'created_at', width: 180, resizable: true, sorter: (a: any, b: any) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime() },
]

const loadSystemLogs = async () => {
  systemLoading.value = true
  try {
    const params: any = { page: systemPagination.page, page_size: systemPagination.pageSize, sort: systemSortState.value.sort, order: systemSortState.value.order }
    if (systemLevelFilter.value) params.level = systemLevelFilter.value
    if (systemModuleFilter.value) params.module = systemModuleFilter.value
    const res = await getSystemLogs(params)
    systemData.value = res.data?.items || res.data || []
    systemPagination.itemCount = res.data?.total || 0
  } catch (error: any) {
    message.error(error.message || '加载失败')
  } finally {
    systemLoading.value = false
  }
}

const handleAuditSorterChange = (sorter: any) => {
  if (sorter && sorter.columnKey && sorter.order) {
    auditSortState.value.sort = sorter.columnKey
    auditSortState.value.order = sorter.order === 'ascend' ? 'asc' : 'desc'
  } else {
    auditSortState.value.sort = 'id'
    auditSortState.value.order = 'desc'
  }
  auditPagination.page = 1
  loadAuditLogs()
}

const handleLoginSorterChange = (sorter: any) => {
  if (sorter && sorter.columnKey && sorter.order) {
    loginSortState.value.sort = sorter.columnKey
    loginSortState.value.order = sorter.order === 'ascend' ? 'asc' : 'desc'
  } else {
    loginSortState.value.sort = 'id'
    loginSortState.value.order = 'desc'
  }
  loginPagination.page = 1
  loadLoginLogs()
}

const handleRegistrationSorterChange = (sorter: any) => {
  if (sorter && sorter.columnKey && sorter.order) {
    registrationSortState.value.sort = sorter.columnKey
    registrationSortState.value.order = sorter.order === 'ascend' ? 'asc' : 'desc'
  } else {
    registrationSortState.value.sort = 'id'
    registrationSortState.value.order = 'desc'
  }
  registrationPagination.page = 1
  loadRegistrationLogs()
}

const handleSubscriptionSorterChange = (sorter: any) => {
  if (sorter && sorter.columnKey && sorter.order) {
    subscriptionSortState.value.sort = sorter.columnKey
    subscriptionSortState.value.order = sorter.order === 'ascend' ? 'asc' : 'desc'
  } else {
    subscriptionSortState.value.sort = 'id'
    subscriptionSortState.value.order = 'desc'
  }
  subscriptionPagination.page = 1
  loadSubscriptionLogs()
}

const handleBalanceSorterChange = (sorter: any) => {
  if (sorter && sorter.columnKey && sorter.order) {
    balanceSortState.value.sort = sorter.columnKey
    balanceSortState.value.order = sorter.order === 'ascend' ? 'asc' : 'desc'
  } else {
    balanceSortState.value.sort = 'id'
    balanceSortState.value.order = 'desc'
  }
  balancePagination.page = 1
  loadBalanceLogs()
}

const handleCommissionSorterChange = (sorter: any) => {
  if (sorter && sorter.columnKey && sorter.order) {
    commissionSortState.value.sort = sorter.columnKey
    commissionSortState.value.order = sorter.order === 'ascend' ? 'asc' : 'desc'
  } else {
    commissionSortState.value.sort = 'id'
    commissionSortState.value.order = 'desc'
  }
  commissionPagination.page = 1
  loadCommissionLogs()
}

const handleSystemSorterChange = (sorter: any) => {
  if (sorter && sorter.columnKey && sorter.order) {
    systemSortState.value.sort = sorter.columnKey
    systemSortState.value.order = sorter.order === 'ascend' ? 'asc' : 'desc'
  } else {
    systemSortState.value.sort = 'id'
    systemSortState.value.order = 'desc'
  }
  systemPagination.page = 1
  loadSystemLogs()
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
    case 'system':
      if (systemData.value.length === 0) loadSystemLogs()
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

.mobile-card-list { display: flex; flex-direction: column; gap: 12px; }
.mobile-card { background: #fff; border-radius: 10px; box-shadow: 0 1px 4px rgba(0,0,0,0.08); overflow: hidden; }
.card-header { display: flex; align-items: center; justify-content: space-between; padding: 12px 14px; border-bottom: 1px solid #f0f0f0; }
.card-title { font-weight: 600; font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.card-body { padding: 10px 14px; }
.card-row { display: flex; justify-content: space-between; align-items: center; padding: 4px 0; font-size: 13px; }
.card-label { color: #999; }
.card-actions { display: flex; gap: 8px; padding: 10px 14px; border-top: 1px solid #f0f0f0; flex-wrap: wrap; }

@media (max-width: 767px) {
  .logs-container { padding: 8px; }
}
</style>
