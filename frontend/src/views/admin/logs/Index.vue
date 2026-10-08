<template>
  <div class="logs-container admin-page-shell">
    <n-card title="系统日志" :bordered="false" class="admin-main-card">
      <template #header-extra>
        <n-popconfirm @positive-click="handleClearCurrent">
          <template #trigger>
            <n-button type="error" size="small" :loading="clearing">
              <template #icon><n-icon><trash-outline /></n-icon></template>
              清空当前日志
            </n-button>
          </template>
          确定清空「{{ currentTabText }}」的全部记录吗？此操作不可恢复！
        </n-popconfirm>
      </template>
      <n-tabs type="line" animated display-directive="show" @update:value="handleTabChange">
        <n-tab-pane name="audit" tab="审计日志">
          <!-- 全选 / 批量操作：公共组件（桌面在表格上方，手机固定在底部标签栏上方）。
               只读日志没有行内操作，按契约第 2.1 节第 3 条提供「复制所选」。 -->
          <BatchSelectBar
            :total="selections.audit.total.value"
            :selected-count="selections.audit.count.value"
            :all-selected="selections.audit.allSelected.value"
            :indeterminate="selections.audit.indeterminate.value"
            label="条记录"
            @toggle-all="selections.audit.toggleAll"
            @clear="selections.audit.clear"
          >
            <n-button size="small" type="primary" :disabled="!selections.audit.count.value" @click="copyTab('audit')">复制所选</n-button>
          </BatchSelectBar>
          <template v-if="!appStore.isMobile">
            <n-data-table
              remote
              :columns="auditColumns"
              :data="auditData"
              :loading="auditLoading"
              :pagination="auditPagination"
              :bordered="false"
              :scroll-x="1100"
              :single-line="false"
              :row-key="(row: any) => row.id"
              v-model:checked-row-keys="auditCheckedKeys"
              @update:sorter="handleAuditSorterChange"
            />
          </template>
          <template v-else>
            <div class="mobile-card-list">
              <div
                v-for="item in auditData"
                :key="item.id"
                class="mobile-card is-selectable"
                :class="{ 'is-selected': selections.audit.isSelected(item) }"
                @click="selections.audit.toggle(item)"
              >
                <div class="card-check" @click.stop>
                  <n-checkbox :checked="selections.audit.isSelected(item)" @update:checked="() => selections.audit.toggle(item)" />
                </div>
                <div class="card-header">
                  <span class="card-title">ID: {{ item.id }}</span>
                </div>
                <div class="card-body">
                  <div class="card-row"><span class="card-label">管理员ID:</span><span>{{ item.user_id }}</span></div>
                  <div class="card-row"><span class="card-label">操作:</span><span>{{ item.action_type }}</span></div>
                  <div class="card-row"><span class="card-label">描述:</span><span>{{ item.action_description }}</span></div>
                  <div class="card-row"><span class="card-label">目标:</span><span>{{ item.resource_type }} #{{ item.resource_id }}</span></div>
                  <div class="card-row"><span class="card-label">IP:</span><span>{{ item.ip_address }}</span></div>
                  <div class="card-row"><span class="card-label">时间:</span><span>{{ item.created_at }}</span></div>
                </div>
              </div>
            </div>
            <n-pagination
              class="list-pagination"
              v-model:page="auditPagination.page"
              v-model:page-size="auditPagination.pageSize"
              :item-count="auditPagination.itemCount"
              :page-sizes="auditPagination.pageSizes"
              show-size-picker
              @update:page="(p: number) => { auditPagination.page = p; loadAuditLogs() }"
              @update:page-size="(ps: number) => { auditPagination.pageSize = ps; auditPagination.page = 1; loadAuditLogs() }"
            />
          </template>
        </n-tab-pane>

        <n-tab-pane name="login" tab="登录日志">
          <!-- 全选 / 批量操作：公共组件（桌面在表格上方，手机固定在底部标签栏上方）。
               只读日志没有行内操作，按契约第 2.1 节第 3 条提供「复制所选」。 -->
          <BatchSelectBar
            :total="selections.login.total.value"
            :selected-count="selections.login.count.value"
            :all-selected="selections.login.allSelected.value"
            :indeterminate="selections.login.indeterminate.value"
            label="条记录"
            @toggle-all="selections.login.toggleAll"
            @clear="selections.login.clear"
          >
            <n-button size="small" type="primary" :disabled="!selections.login.count.value" @click="copyTab('login')">复制所选</n-button>
          </BatchSelectBar>
          <template v-if="!appStore.isMobile">
            <n-data-table
              remote
              :columns="loginColumns"
              :data="loginData"
              :loading="loginLoading"
              :pagination="loginPagination"
              :bordered="false"
              :row-key="(row: any) => row.id"
              v-model:checked-row-keys="loginCheckedKeys"
              @update:sorter="handleLoginSorterChange"
            />
          </template>
          <template v-else>
            <div class="mobile-card-list">
              <div
                v-for="item in loginData"
                :key="item.id"
                class="mobile-card is-selectable"
                :class="{ 'is-selected': selections.login.isSelected(item) }"
                @click="selections.login.toggle(item)"
              >
                <div class="card-check" @click.stop>
                  <n-checkbox :checked="selections.login.isSelected(item)" @update:checked="() => selections.login.toggle(item)" />
                </div>
                <div class="card-header">
                  <span class="card-title">ID: {{ item.id }}</span>
                  <n-tag :type="item.login_status === 'success' ? 'success' : 'error'" size="small">{{ translateLoginStatus(item.login_status) }}</n-tag>
                </div>
                <div class="card-body">
                  <div class="card-row"><span class="card-label">用户ID:</span><span>{{ item.user_id }}</span></div>
                  <div class="card-row"><span class="card-label">IP地址:</span><span>{{ item.ip_address }}</span></div>
                  <div class="card-row"><span class="card-label">位置:</span><span>{{ formatLocation(item.location) }}</span></div>
                  <div class="card-row"><span class="card-label">设备:</span><span>{{ parseDeviceInfo(item.user_agent) }}</span></div>
                  <div class="card-row"><span class="card-label">登录时间:</span><span>{{ item.login_time }}</span></div>
                </div>
              </div>
            </div>
            <n-pagination
              class="list-pagination"
              v-model:page="loginPagination.page"
              v-model:page-size="loginPagination.pageSize"
              :item-count="loginPagination.itemCount"
              :page-sizes="loginPagination.pageSizes"
              show-size-picker
              @update:page="(p: number) => { loginPagination.page = p; loadLoginLogs() }"
              @update:page-size="(ps: number) => { loginPagination.pageSize = ps; loginPagination.page = 1; loadLoginLogs() }"
            />
          </template>
        </n-tab-pane>

        <n-tab-pane name="registration" tab="注册日志">
          <!-- 全选 / 批量操作：公共组件（桌面在表格上方，手机固定在底部标签栏上方）。
               只读日志没有行内操作，按契约第 2.1 节第 3 条提供「复制所选」。 -->
          <BatchSelectBar
            :total="selections.registration.total.value"
            :selected-count="selections.registration.count.value"
            :all-selected="selections.registration.allSelected.value"
            :indeterminate="selections.registration.indeterminate.value"
            label="条记录"
            @toggle-all="selections.registration.toggleAll"
            @clear="selections.registration.clear"
          >
            <n-button size="small" type="primary" :disabled="!selections.registration.count.value" @click="copyTab('registration')">复制所选</n-button>
          </BatchSelectBar>
          <template v-if="!appStore.isMobile">
            <n-data-table
              remote
              :columns="registrationColumns"
              :data="registrationData"
              :loading="registrationLoading"
              :pagination="registrationPagination"
              :bordered="false"
              :row-key="(row: any) => row.id"
              v-model:checked-row-keys="registrationCheckedKeys"
              @update:sorter="handleRegistrationSorterChange"
            />
          </template>
          <template v-else>
            <div class="mobile-card-list">
              <div
                v-for="item in registrationData"
                :key="item.id"
                class="mobile-card is-selectable"
                :class="{ 'is-selected': selections.registration.isSelected(item) }"
                @click="selections.registration.toggle(item)"
              >
                <div class="card-check" @click.stop>
                  <n-checkbox :checked="selections.registration.isSelected(item)" @update:checked="() => selections.registration.toggle(item)" />
                </div>
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
              class="list-pagination"
              v-model:page="registrationPagination.page"
              v-model:page-size="registrationPagination.pageSize"
              :item-count="registrationPagination.itemCount"
              :page-sizes="registrationPagination.pageSizes"
              show-size-picker
              @update:page="(p: number) => { registrationPagination.page = p; loadRegistrationLogs() }"
              @update:page-size="(ps: number) => { registrationPagination.pageSize = ps; registrationPagination.page = 1; loadRegistrationLogs() }"
            />
          </template>
        </n-tab-pane>

        <n-tab-pane name="subscription" tab="订阅日志">
          <!-- 全选 / 批量操作：公共组件（桌面在表格上方，手机固定在底部标签栏上方）。
               只读日志没有行内操作，按契约第 2.1 节第 3 条提供「复制所选」。 -->
          <BatchSelectBar
            :total="selections.subscription.total.value"
            :selected-count="selections.subscription.count.value"
            :all-selected="selections.subscription.allSelected.value"
            :indeterminate="selections.subscription.indeterminate.value"
            label="条记录"
            @toggle-all="selections.subscription.toggleAll"
            @clear="selections.subscription.clear"
          >
            <n-button size="small" type="primary" :disabled="!selections.subscription.count.value" @click="copyTab('subscription')">复制所选</n-button>
          </BatchSelectBar>
          <template v-if="!appStore.isMobile">
            <n-data-table
              remote
              :columns="subscriptionColumns"
              :data="subscriptionData"
              :loading="subscriptionLoading"
              :pagination="subscriptionPagination"
              :bordered="false"
              :row-key="(row: any) => row.id"
              v-model:checked-row-keys="subscriptionCheckedKeys"
              @update:sorter="handleSubscriptionSorterChange"
            />
          </template>
          <template v-else>
            <div class="mobile-card-list">
              <div
                v-for="item in subscriptionData"
                :key="item.id"
                class="mobile-card is-selectable"
                :class="{ 'is-selected': selections.subscription.isSelected(item) }"
                @click="selections.subscription.toggle(item)"
              >
                <div class="card-check" @click.stop>
                  <n-checkbox :checked="selections.subscription.isSelected(item)" @update:checked="() => selections.subscription.toggle(item)" />
                </div>
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
              class="list-pagination"
              v-model:page="subscriptionPagination.page"
              v-model:page-size="subscriptionPagination.pageSize"
              :item-count="subscriptionPagination.itemCount"
              :page-sizes="subscriptionPagination.pageSizes"
              show-size-picker
              @update:page="(p: number) => { subscriptionPagination.page = p; loadSubscriptionLogs() }"
              @update:page-size="(ps: number) => { subscriptionPagination.pageSize = ps; subscriptionPagination.page = 1; loadSubscriptionLogs() }"
            />
          </template>
        </n-tab-pane>

        <n-tab-pane name="balance" tab="余额日志">
          <!-- 全选 / 批量操作：公共组件（桌面在表格上方，手机固定在底部标签栏上方）。
               只读日志没有行内操作，按契约第 2.1 节第 3 条提供「复制所选」。 -->
          <BatchSelectBar
            :total="selections.balance.total.value"
            :selected-count="selections.balance.count.value"
            :all-selected="selections.balance.allSelected.value"
            :indeterminate="selections.balance.indeterminate.value"
            label="条记录"
            @toggle-all="selections.balance.toggleAll"
            @clear="selections.balance.clear"
          >
            <n-button size="small" type="primary" :disabled="!selections.balance.count.value" @click="copyTab('balance')">复制所选</n-button>
          </BatchSelectBar>
          <template v-if="!appStore.isMobile">
            <n-data-table
              remote
              :columns="balanceColumns"
              :data="balanceData"
              :loading="balanceLoading"
              :pagination="balancePagination"
              :bordered="false"
              :row-key="(row: any) => row.id"
              v-model:checked-row-keys="balanceCheckedKeys"
              @update:sorter="handleBalanceSorterChange"
            />
          </template>
          <template v-else>
            <div class="mobile-card-list">
              <div
                v-for="item in balanceData"
                :key="item.id"
                class="mobile-card is-selectable"
                :class="{ 'is-selected': selections.balance.isSelected(item) }"
                @click="selections.balance.toggle(item)"
              >
                <div class="card-check" @click.stop>
                  <n-checkbox :checked="selections.balance.isSelected(item)" @update:checked="() => selections.balance.toggle(item)" />
                </div>
                <div class="card-header">
                  <span class="card-title">ID: {{ item.id }}</span>
                </div>
                <div class="card-body">
                  <div class="card-row"><span class="card-label">用户ID:</span><span>{{ item.user_id }}</span></div>
                  <div class="card-row"><span class="card-label">类型:</span><span>{{ translateBalanceChangeType(item.change_type) }}</span></div>
                  <div class="card-row"><span class="card-label">金额:</span><span>{{ item.amount }}</span></div>
                  <div class="card-row"><span class="card-label">余额:</span><span>{{ item.balance_after }}</span></div>
                  <div class="card-row"><span class="card-label">备注:</span><span>{{ item.description }}</span></div>
                  <div class="card-row"><span class="card-label">创建时间:</span><span>{{ item.created_at }}</span></div>
                </div>
              </div>
            </div>
            <n-pagination
              class="list-pagination"
              v-model:page="balancePagination.page"
              v-model:page-size="balancePagination.pageSize"
              :item-count="balancePagination.itemCount"
              :page-sizes="balancePagination.pageSizes"
              show-size-picker
              @update:page="(p: number) => { balancePagination.page = p; loadBalanceLogs() }"
              @update:page-size="(ps: number) => { balancePagination.pageSize = ps; balancePagination.page = 1; loadBalanceLogs() }"
            />
          </template>
        </n-tab-pane>

        <n-tab-pane name="commission" tab="佣金日志">
          <!-- 全选 / 批量操作：公共组件（桌面在表格上方，手机固定在底部标签栏上方）。
               只读日志没有行内操作，按契约第 2.1 节第 3 条提供「复制所选」。 -->
          <BatchSelectBar
            :total="selections.commission.total.value"
            :selected-count="selections.commission.count.value"
            :all-selected="selections.commission.allSelected.value"
            :indeterminate="selections.commission.indeterminate.value"
            label="条记录"
            @toggle-all="selections.commission.toggleAll"
            @clear="selections.commission.clear"
          >
            <n-button size="small" type="primary" :disabled="!selections.commission.count.value" @click="copyTab('commission')">复制所选</n-button>
          </BatchSelectBar>
          <template v-if="!appStore.isMobile">
            <n-data-table
              remote
              :columns="commissionColumns"
              :data="commissionData"
              :loading="commissionLoading"
              :pagination="commissionPagination"
              :bordered="false"
              :row-key="(row: any) => row.id"
              v-model:checked-row-keys="commissionCheckedKeys"
              @update:sorter="handleCommissionSorterChange"
            />
          </template>
          <template v-else>
            <div class="mobile-card-list">
              <div
                v-for="item in commissionData"
                :key="item.id"
                class="mobile-card is-selectable"
                :class="{ 'is-selected': selections.commission.isSelected(item) }"
                @click="selections.commission.toggle(item)"
              >
                <div class="card-check" @click.stop>
                  <n-checkbox :checked="selections.commission.isSelected(item)" @update:checked="() => selections.commission.toggle(item)" />
                </div>
                <div class="card-header">
                  <span class="card-title">ID: {{ item.id }}</span>
                </div>
                <div class="card-body">
                  <div class="card-row"><span class="card-label">用户ID:</span><span>{{ item.inviter_id }}</span></div>
                  <div class="card-row"><span class="card-label">来源用户ID:</span><span>{{ item.invitee_id }}</span></div>
                  <div class="card-row"><span class="card-label">金额:</span><span>{{ item.amount }}</span></div>
                  <div class="card-row"><span class="card-label">类型:</span><span>{{ translateCommissionType(item.commission_type) }}</span></div>
                  <div class="card-row"><span class="card-label">创建时间:</span><span>{{ item.created_at }}</span></div>
                </div>
              </div>
            </div>
            <n-pagination
              class="list-pagination"
              v-model:page="commissionPagination.page"
              v-model:page-size="commissionPagination.pageSize"
              :item-count="commissionPagination.itemCount"
              :page-sizes="commissionPagination.pageSizes"
              show-size-picker
              @update:page="(p: number) => { commissionPagination.page = p; loadCommissionLogs() }"
              @update:page-size="(ps: number) => { commissionPagination.pageSize = ps; commissionPagination.page = 1; loadCommissionLogs() }"
            />
          </template>
        </n-tab-pane>

        <n-tab-pane name="system" tab="系统日志">
          <!-- 全选 / 批量操作：公共组件（桌面在表格上方，手机固定在底部标签栏上方）。
               只读日志没有行内操作，按契约第 2.1 节第 3 条提供「复制所选」。 -->
          <BatchSelectBar
            :total="selections.system.total.value"
            :selected-count="selections.system.count.value"
            :all-selected="selections.system.allSelected.value"
            :indeterminate="selections.system.indeterminate.value"
            label="条记录"
            @toggle-all="selections.system.toggleAll"
            @clear="selections.system.clear"
          >
            <n-button size="small" type="primary" :disabled="!selections.system.count.value" @click="copyTab('system')">复制所选</n-button>
          </BatchSelectBar>
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
              :row-key="(row: any) => row.id"
              v-model:checked-row-keys="systemCheckedKeys"
              @update:sorter="handleSystemSorterChange"
            />
          </template>
          <template v-else>
            <div class="mobile-card-list">
              <div
                v-for="item in systemData"
                :key="item.id"
                class="mobile-card is-selectable"
                :class="{ 'is-selected': selections.system.isSelected(item) }"
                @click="selections.system.toggle(item)"
              >
                <div class="card-check" @click.stop>
                  <n-checkbox :checked="selections.system.isSelected(item)" @update:checked="() => selections.system.toggle(item)" />
                </div>
                <div class="card-header">
                  <n-tag :type="item.level === 'error' ? 'error' : item.level === 'warn' ? 'warning' : 'info'" size="small">{{ item.level }}</n-tag>
                  <span style="font-size: 12px; color: var(--text-color-secondary)">{{ item.created_at }}</span>
                </div>
                <div class="card-body">
                  <div class="card-row"><span class="card-label">模块:</span><span>{{ item.module }}</span></div>
                  <div class="card-row"><span class="card-label">消息:</span><span>{{ item.message }}</span></div>
                  <div v-if="item.detail" class="card-row"><span class="card-label">详情:</span><span>{{ item.detail }}</span></div>
                </div>
              </div>
            </div>
            <n-pagination
              class="list-pagination"
              v-model:page="systemPagination.page"
              v-model:page-size="systemPagination.pageSize"
              :item-count="systemPagination.itemCount"
              :page-sizes="systemPagination.pageSizes"
              show-size-picker
              @update:page="(p: number) => { systemPagination.page = p; loadSystemLogs() }"
              @update:page-size="(ps: number) => { systemPagination.pageSize = ps; systemPagination.page = 1; loadSystemLogs() }"
            />
          </template>
        </n-tab-pane>
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup lang="tsx">
import { ref, reactive, h, computed, onMounted } from 'vue'
import { TrashOutline } from '@vicons/ionicons5'
import { clearLogs } from '@/api/admin'
import { NCard, NTabs, NTabPane, NDataTable, NTag, NPagination, NSpace, NSelect, useMessage, type DataTableColumns } from 'naive-ui'
import { getAuditLogs, getLoginLogs, getRegistrationLogs, getSubscriptionLogs, getBalanceLogs, getCommissionLogs, getSystemLogs } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { useBatchSelection, type BatchSelection } from '@/composables/useBatchSelection'
import BatchSelectBar from '@/components/BatchSelectBar.vue'
import { copyToClipboard as clipboardCopy } from '@/utils/clipboard'
import { translateLoginStatus, translateBalanceChangeType, translateCommissionType, parseDeviceInfo, formatLocation } from '@/utils/i18n'

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
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 70, resizable: true, sorter: 'default' },
  { title: '管理员ID', key: 'user_id', width: 90, resizable: true },
  { title: '操作类型', key: 'action_type', width: 170, resizable: true },
  { title: '目标类型', key: 'resource_type', width: 110, resizable: true },
  { title: '目标ID', key: 'resource_id', width: 80, resizable: true },
  { title: '描述', key: 'action_description', width: 240, resizable: true, ellipsis: { tooltip: true } },
  { title: 'IP', key: 'ip_address', width: 130, resizable: true },
  { title: '时间', key: 'created_at', width: 170, resizable: true, sorter: (a: any, b: any) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime() },
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
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 80, resizable: true, sorter: 'default' },
  { title: '用户ID', key: 'user_id', width: 100, resizable: true },
  { title: 'IP地址', key: 'ip_address', width: 140, resizable: true },
  {
    title: '位置',
    key: 'location',
    width: 180,
    resizable: true,
    ellipsis: { tooltip: true },
    render: (row: any) => formatLocation(row.location)
  },
  {
    title: '设备',
    key: 'user_agent',
    width: 200,
    resizable: true,
    ellipsis: { tooltip: true },
    render: (row: any) => parseDeviceInfo(row.user_agent)
  },
  {
    title: '状态',
    key: 'login_status',
    width: 100,
    resizable: true,
    render: (row: any) =>
      h(NTag, { type: row.login_status === 'success' ? 'success' : 'error', size: 'small' }, { default: () => translateLoginStatus(row.login_status) }),
  },
  { title: '登录时间', key: 'login_time', width: 180, resizable: true, sorter: (a: any, b: any) => new Date(a.login_time).getTime() - new Date(b.login_time).getTime() },
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
  { type: 'selection' },
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
  { type: 'selection' },
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
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 80, resizable: true, sorter: 'default' },
  { title: '用户ID', key: 'user_id', width: 100, resizable: true },
  {
    title: '类型',
    key: 'change_type',
    width: 120,
    resizable: true,
    render: (row: any) => translateBalanceChangeType(row.change_type)
  },
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
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 80, resizable: true, sorter: 'default' },
  { title: '用户ID', key: 'inviter_id', width: 100, resizable: true },
  { title: '来源用户ID', key: 'invitee_id', width: 120, resizable: true },
  { title: '金额', key: 'amount', width: 120, resizable: true },
  {
    title: '类型',
    key: 'commission_type',
    width: 120,
    resizable: true,
    render: (row: any) => translateCommissionType(row.commission_type)
  },
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
  { label: '订阅', value: 'subscription' },
  { label: '节点', value: 'node' },
  { label: '订单', value: 'order' },
  { label: '余额', value: 'balance' },
  { label: '认证', value: 'auth' },
  { label: '安全', value: 'security' },
  { label: '安全告警', value: 'security_alert' },
  { label: '备份', value: 'backup' },
  { label: '恢复', value: 'restore' },
  { label: '审计', value: 'audit' },
  { label: '管理操作', value: 'admin' },
  { label: '系统', value: 'system' },
]
const systemPagination = reactive({
  page: 1,
  pageSize: 10,
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
  { type: 'selection' },
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

/**
 * 全选 / 批量操作：7 个 tab 各一份独立选择状态（用公共 useBatchSelection）。
 * 这些 tab 全是只读日志、行内没有任何操作，按契约第 2.1 节第 3 条：
 * 提供「复制所选」作为批量动作，不允许空有勾选框却没有动作。
 */
const selections: Record<string, BatchSelection<any>> = {
  audit: useBatchSelection(() => auditData.value),
  login: useBatchSelection(() => loginData.value),
  registration: useBatchSelection(() => registrationData.value),
  subscription: useBatchSelection(() => subscriptionData.value),
  balance: useBatchSelection(() => balanceData.value),
  commission: useBatchSelection(() => commissionData.value),
  system: useBatchSelection(() => systemData.value),
}

// 桌面表格要的是数组，这里给每个 tab 做一层桥接，保证表头全选与批量栏状态一致。
// 必须是「顶层 ref」才能被模板自动解包（v-model 需要可写引用），不能挂在对象属性上。
const makeCheckedBridge = (sel: BatchSelection<any>) => computed<Array<string | number>>({
  get: () => [...sel.selectedKeys.value],
  set: (keys) => { sel.selectedKeys.value = new Set(keys) },
})
const auditCheckedKeys = makeCheckedBridge(selections.audit)
const loginCheckedKeys = makeCheckedBridge(selections.login)
const registrationCheckedKeys = makeCheckedBridge(selections.registration)
const subscriptionCheckedKeys = makeCheckedBridge(selections.subscription)
const balanceCheckedKeys = makeCheckedBridge(selections.balance)
const commissionCheckedKeys = makeCheckedBridge(selections.commission)
const systemCheckedKeys = makeCheckedBridge(selections.system)

// 每个 tab 一行一条的文本格式（复制所选时用）
const lineFormatters: Record<string, (row: any) => string> = {
  audit: (r) => `[${r.id}] 管理员#${r.user_id} ${r.action_type || '-'} ${r.resource_type || '-'}#${r.resource_id ?? '-'} IP:${r.ip_address || '-'} ${r.created_at || '-'} ${r.action_description || ''}`.trim(),
  login: (r) => `[${r.id}] 用户#${r.user_id} ${translateLoginStatus(r.login_status)} IP:${r.ip_address || '-'} 位置:${formatLocation(r.location) || '-'} 设备:${parseDeviceInfo(r.user_agent) || '-'} ${r.login_time || '-'}`,
  registration: (r) => `[${r.id}] 用户#${r.user_id} IP:${r.ip_address || '-'} 邀请码:${r.invite_code || '-'} ${r.created_at || '-'}`,
  subscription: (r) => `[${r.id}] 用户#${r.user_id} ${r.action_type || '-'} ${r.description || ''} ${r.created_at || '-'}`.trim(),
  balance: (r) => `[${r.id}] 用户#${r.user_id} ${translateBalanceChangeType(r.change_type)} 金额:${r.amount ?? '-'} 余额:${r.balance_after ?? '-'} ${r.description || ''} ${r.created_at || '-'}`.trim(),
  commission: (r) => `[${r.id}] 邀请人#${r.inviter_id} 被邀请人#${r.invitee_id} 金额:${r.amount ?? '-'} ${translateCommissionType(r.commission_type)} ${r.created_at || '-'}`,
  system: (r) => `[${r.id}] [${r.level || '-'}] ${r.module || '-'} ${r.message || ''} ${r.detail || ''} ${r.created_at || '-'}`.trim(),
}

// 只读列表的批量动作：把所选行按一行一条拼成文本复制
const copyTab = async (key: string) => {
  const sel = selections[key]
  const rows = sel?.selectedRows.value || []
  if (!rows.length) return
  const text = rows.map(lineFormatters[key]).join('\n')
  const ok = await clipboardCopy(text)
  if (ok) message.success(`已复制 ${rows.length} 条记录`)
  else message.error('复制失败，请手动选择文本')
}

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

// 清空当前类型日志
const clearing = ref(false)
const currentTabText = computed(() => {
  const map: Record<string, string> = { audit: '审计', login: '登录', registration: '注册', subscription: '订阅', balance: '余额', commission: '佣金', system: '系统' }
  return (map[currentTab.value] || '') + '日志'
})
const handleClearCurrent = async () => {
  clearing.value = true
  try {
    const res: any = await clearLogs(currentTab.value)
    message.success(`已清空 ${res.data?.deleted || 0} 条记录`)
    // 刷新当前 tab
    const loaders: Record<string, () => Promise<void>> = {
      audit: loadAuditLogs, login: loadLoginLogs, registration: loadRegistrationLogs,
      subscription: loadSubscriptionLogs, balance: loadBalanceLogs, commission: loadCommissionLogs, system: loadSystemLogs,
    }
    await (loaders[currentTab.value] || loadAuditLogs)()
  } catch (e: any) {
    message.error(e.message || '清空失败')
  } finally { clearing.value = false }
}

onMounted(() => {
  loadAuditLogs()
})
</script>

<style scoped>
.logs-container {
  padding: 20px;
}

/* 分页：桌面靠右，手机居中 */
.list-pagination { margin-top: 16px; justify-content: flex-end; }

/* 日志卡片样式（.mobile-card / .card-header / .card-row…）全部交给全局
   mobile-cards.css + admin-mobile.css，页面不再覆盖，手机端才是 App 列表样式。 */

/* 可选中卡片：左侧给复选框留位（兜底，防止被其他层叠规则盖掉） */
.mobile-card.is-selectable { padding-left: 44px !important; }
@media (max-width: 767px) {
  /* 左右留白由全局统一给（mobile-app-ui.css + admin-mobile.css），页面不再自带内边距 */
  .list-pagination { justify-content: center; }
}
</style>
