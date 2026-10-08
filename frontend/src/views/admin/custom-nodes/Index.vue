<template>
  <div class="custom-nodes-container admin-page-shell">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">专线节点管理</h2>
        <p class="page-subtitle">管理独享专线节点，支持按用户分配、批量导入、状态监控及过期自动管理</p>
      </div>
      <div class="header-right">
        <n-space>
          <n-input
            v-model:value="searchKeyword"
            clearable
            placeholder="搜索邮箱/域名/节点名称"
            style="width: 200px"
            @keyup.enter="handleSearch"
          >
            <template #prefix><n-icon :component="SearchOutline" /></template>
          </n-input>
          <n-button type="primary" @click="handleSearch">
            <template #icon><n-icon><SearchOutline /></n-icon></template>
            搜索
          </n-button>
          <n-button type="primary" @click="handleCreate">
            <template #icon><n-icon><AddOutline /></n-icon></template>
            新建节点
          </n-button>
          <n-button type="primary" @click="showImportDrawer = true">
            <template #icon><n-icon><CloudUploadOutline /></n-icon></template>
            导入链接
          </n-button>
          <n-button @click="handleResetSearch" secondary>
            <template #icon><n-icon><RefreshOutline /></n-icon></template>
            刷新
          </n-button>
        </n-space>
      </div>
    </div>

    <!-- 订阅来源：专线节点是从哪个订阅链接导入的，可换链接/立即更新/删除，后端按间隔自动同步 -->
    <n-card :bordered="false" class="admin-main-card source-card">
      <div class="source-header">
        <div class="source-title">
          <span>订阅来源</span>
          <n-tag size="small" :bordered="false">{{ sources.length }} 个</n-tag>
        </div>
        <n-space>
          <n-button size="small" type="primary" @click="openSourceForm()">新增订阅来源</n-button>
          <n-button size="small" :loading="syncingAll" :disabled="!sources.length" @click="handleSyncAllSources">全部更新</n-button>
          <n-button size="small" secondary :loading="loadingSources" @click="loadSources">刷新</n-button>
        </n-space>
      </div>
      <n-alert v-if="!sources.length" type="info" :bordered="false" style="margin-top: 10px">
        还没有订阅来源。点「新增订阅来源」填入你的订阅链接，之后节点会<b>按间隔自动更新</b>，
        订阅内容一变节点就跟着变；也可以随时换链接或删除整条来源。
      </n-alert>
      <!-- 窄屏（手机）放不下 7 列：让表格在卡片内横向滚动，避免整页被撑出横向滚动条 -->
      <div v-else class="source-table-wrap">
      <n-table :bordered="false" size="small">
        <thead>
          <tr>
            <th>名称</th>
            <th>订阅链接</th>
            <th>节点</th>
            <th>自动同步</th>
            <th>上次同步</th>
            <th>结果</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="src in sources" :key="src.id">
            <td>{{ src.name || '-' }}</td>
            <td class="source-url" :title="src.url">{{ maskSourceUrl(src.url) }}</td>
            <td>
              <n-tag size="tiny" type="success" :bordered="false">{{ src.active_nodes }} 启用</n-tag>
              <n-tag v-if="src.inactive_nodes" size="tiny" :bordered="false" style="margin-left: 4px">{{ src.inactive_nodes }} 停用</n-tag>
              <div v-if="src.assigned_users" class="source-sub">已分配 {{ src.assigned_users }} 次</div>
            </td>
            <td>
              <n-switch :value="src.enabled" size="small" @update:value="(v) => handleToggleSource(src, v)" />
              <div class="source-sub">每 {{ src.interval_hours || 0 }} 小时</div>
            </td>
            <td>{{ src.last_sync_at ? formatDateTime(src.last_sync_at) : '从未' }}</td>
            <td>
              <n-tag size="tiny" :type="src.last_status === 'ok' ? 'success' : (src.last_status ? 'error' : 'default')" :bordered="false">
                {{ src.last_status === 'ok' ? '成功' : (src.last_status === 'error' ? '失败' : '未同步') }}
              </n-tag>
              <div class="source-sub">{{ src.last_message || '-' }}</div>
            </td>
            <td>
              <n-space :size="4">
                <n-button size="tiny" type="primary" :loading="syncingId === src.id" @click="handleSyncSource(src)">立即更新</n-button>
                <n-button size="tiny" secondary @click="openSourceForm(src)">改链接</n-button>
                <n-button size="tiny" type="error" secondary @click="handleDeleteSource(src)">删除</n-button>
              </n-space>
            </td>
          </tr>
        </tbody>
      </n-table>
      </div>
    </n-card>

    <!-- 新增/修改订阅来源 -->
    <common-drawer
      v-model:show="showSourceDrawer"
      :title="sourceForm.id ? '修改订阅来源' : '新增订阅来源'"
      :width="560"
      show-footer
      :loading="savingSource"
      @confirm="handleSourceSubmit"
      @cancel="showSourceDrawer = false"
    >
      <n-form label-placement="top">
        <n-form-item label="订阅链接">
          <n-input v-model:value="sourceForm.url" placeholder="https://example.com/sub?token=xxx" />
        </n-form-item>
        <n-form-item label="名称（可选，便于识别）">
          <n-input v-model:value="sourceForm.name" placeholder="例如：某机场专线" />
        </n-form-item>
        <n-form-item label="自动同步间隔（小时，0 = 只手动更新）">
          <n-input-number v-model:value="sourceForm.interval_hours" :min="0" :max="168" style="width: 100%" />
        </n-form-item>
        <n-alert type="info" :bordered="false">
          保存后会立即同步一次：按节点名称更新配置（<b>已分配给用户的节点分配关系不变</b>），
          新节点自动加入，订阅里已消失的节点会<b>停用</b>（不删除，也不再下发给用户，可在节点列表里手动删掉）。
          换链接后旧链接导入的节点会先迁移到新链接再同步，新订阅里没有的同样转为停用。
        </n-alert>
      </n-form>
    </common-drawer>

    <n-card :bordered="false" class="admin-main-card">
      <div v-if="appStore.isMobile" class="mobile-toolbar">
        <div class="mobile-toolbar-search">
          <n-input
            v-model:value="searchKeyword"
            clearable
            placeholder="搜索节点名称/域名"
            @keyup.enter="handleSearch"
          >
            <template #prefix><n-icon :component="SearchOutline" /></template>
          </n-input>
          <n-button type="info" @click="handleSearch">搜索</n-button>
        </div>
        <div class="mobile-toolbar-row">
          <n-button size="small" type="primary" @click="handleCreate">新建</n-button>
          <n-button size="small" type="primary" @click="showImportDrawer = true">导入</n-button>
          <n-button size="small" secondary @click="handleResetSearch">刷新</n-button>
        </div>
      </div>

      <!-- 全选 / 批量操作：公共组件（桌面在表格上方，手机固定在底部标签栏上方） -->
      <BatchSelectBar
        :total="selection.total.value"
        :selected-count="selection.count.value"
        :all-selected="selection.allSelected.value"
        :indeterminate="selection.indeterminate.value"
        label="个节点"
        @toggle-all="selection.toggleAll"
        @clear="selection.clear"
      >
        <n-button size="small" type="info" :disabled="!selection.count.value" @click="handleBatchAssign">
          <template #icon><n-icon><PeopleOutline /></n-icon></template>
          批量分配
        </n-button>
        <n-button size="small" type="error" :disabled="!selection.count.value" @click="handleBatchDelete">
          <template #icon><n-icon><TrashOutline /></n-icon></template>
          批量删除
        </n-button>
      </BatchSelectBar>

      <template v-if="!appStore.isMobile">
        <n-data-table
          class="unified-admin-table"
          remote
          :columns="columns"
          :data="tableData"
          :loading="loading"
          :pagination="pagination"
          :bordered="false"
          :row-key="(row) => row.id"
          v-model:checked-row-keys="checkedRowKeys"
          @update:sorter="handleSorterChange"
          @update:page="(p) => { pagination.page = p; fetchData() }"
          @update:page-size="(ps) => { pagination.pageSize = ps; pagination.page = 1; fetchData() }"
        />
      </template>

      <template v-else>
        <div class="mobile-card-list">
          <div
            v-for="row in tableData"
            :key="row.id"
            class="mobile-card is-selectable"
            :class="{ 'is-selected': selection.isSelected(row) }"
            @click="selection.toggle(row)"
          >
            <div class="card-check" @click.stop>
              <n-checkbox :checked="selection.isSelected(row)" @update:checked="() => selection.toggle(row)" />
            </div>
            <div class="card-header">
              <span class="card-title">{{ row.display_name }}</span>
              <n-tag :type="protocolColorMap[row.protocol] || 'default'" size="small">
                {{ row.protocol.toUpperCase() }}
              </n-tag>
            </div>
            <div class="card-body">
              <div class="card-row">
                <span class="card-label">节点名称</span>
                <span>{{ row.name }}</span>
              </div>
              <div class="card-row">
                <span class="card-label">服务器</span>
                <span>{{ row.domain }}:{{ row.port }}</span>
              </div>
              <div class="card-row">
                <span class="card-label">状态</span>
                <n-switch :value="row.is_active" @update:value="(value) => handleToggleActive(row, value)" />
              </div>
              <div class="card-row">
                <span class="card-label">过期时间</span>
                <span>{{ row.expire_time ? formatFullDateTime(row.expire_time) : '-' }}</span>
              </div>
            </div>
            <div class="card-actions" @click.stop>
              <n-button size="small" type="primary" @click="handleEdit(row)">
                <template #icon><n-icon><CreateOutline /></n-icon></template>
                编辑
              </n-button>
              <n-button size="small" type="info" @click="handleAssign(row)">
                <template #icon><n-icon><PeopleOutline /></n-icon></template>
                分配
              </n-button>
              <n-button size="small" @click="handleViewLink(row)">
                <template #icon><n-icon><LinkOutline /></n-icon></template>
                链接
              </n-button>
              <n-button size="small" type="error" @click="handleDelete(row)">
                <template #icon><n-icon><TrashOutline /></n-icon></template>
                删除
              </n-button>
            </div>
          </div>
        </div>

        <n-pagination
          v-model:page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :item-count="pagination.itemCount"
          :page-sizes="pagination.pageSizes"
          show-size-picker
          style="margin-top: 16px; justify-content: flex-end"
          @update:page="fetchData"
          @update:page-size="(ps) => { pagination.pageSize = ps; pagination.page = 1; fetchData() }"
        />
      </template>
    </n-card>

    <!-- Create/Edit Drawer -->
    <common-drawer
      v-model:show="showEditDrawer"
      :title="editId ? '编辑专线节点' : '创建专线节点'"
      :width="700"
      show-footer
      :loading="submitting"
      @confirm="handleSubmit"
      @cancel="showEditDrawer = false"
    >
      <n-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-placement="left"
        label-width="120"
      >
        <n-form-item label="节点名称" path="name">
          <n-input v-model:value="formData.name" placeholder="请输入节点名称（内部标识）" />
        </n-form-item>

        <n-form-item label="显示名称" path="display_name">
          <n-input v-model:value="formData.display_name" placeholder="请输入显示名称（用户可见）" />
        </n-form-item>

        <n-form-item label="协议" path="protocol">
          <n-select
            v-model:value="formData.protocol"
            placeholder="请选择协议"
            :options="protocolOptions"
          />
        </n-form-item>

        <n-form-item label="域名/IP" path="domain">
          <n-input v-model:value="formData.domain" placeholder="请输入域名或IP地址" />
        </n-form-item>

        <n-form-item label="端口" path="port">
          <n-input-number v-model:value="formData.port" :min="1" :max="65535" style="width: 100%" placeholder="请输入端口号" />
        </n-form-item>

        <n-form-item label="配置信息" path="config">
          <n-input
            v-model:value="formData.config"
            type="textarea"
            placeholder="请输入节点配置信息（JSON格式）"
            :rows="6"
          />
        </n-form-item>

        <n-form-item label="启用状态" path="is_active">
          <n-switch v-model:value="formData.is_active" />
        </n-form-item>

        <n-form-item label="过期时间" path="expire_time">
          <n-date-picker
            v-model:value="formData.expire_time"
            type="datetime"
            clearable
            style="width: 100%"
            placeholder="选择过期时间（可选）"
          />
        </n-form-item>

        <n-form-item label="跟随用户过期" path="follow_user_expire">
          <n-switch v-model:value="formData.follow_user_expire" />
          <n-text depth="3" style="margin-left: 12px; font-size: 12px">
            启用后，节点将在用户订阅过期时自动失效
          </n-text>
        </n-form-item>
      </n-form>
    </common-drawer>

    <!-- Assign Drawer -->
    <common-drawer
      v-model:show="showAssignDrawer"
      title="分配节点给用户"
      :width="600"
      show-footer
      :loading="assigning"
      @confirm="handleAssignSubmit"
      @cancel="showAssignDrawer = false"
    >
      <n-form label-placement="top">
        <n-form-item label="选择用户">
          <n-select
            v-model:value="assignUserIds"
            class="assign-user-select"
            multiple
            remote
            filterable
            clearable
            size="medium"
            :max-tag-count="2"
            placeholder="输入邮箱/用户名搜索并选择用户"
            :options="userOptions"
            :loading="loadingUsers"
            :show-arrow="true"
            @search="handleUserSearch"
          />
        </n-form-item>
        <n-form-item label="专线独立到期时间（可选）">
          <n-date-picker
            v-model:value="assignExpiresAt"
            type="datetime"
            clearable
            style="width: 100%"
            placeholder="不设置则跟随订阅到期时间"
          />
        </n-form-item>
        <n-form-item label="显示模式">
          <n-switch v-model:value="assignDedicatedOnly">
            <template #checked>
              只显示专线节点
            </template>
            <template #unchecked>
              显示全部节点
            </template>
          </n-switch>
        </n-form-item>
        <n-form-item label="限制设备数量">
          <n-switch v-model:value="assignLimitDevices">
            <template #checked>
              跟随系统限制
            </template>
            <template #unchecked>
              不限制设备数量
            </template>
          </n-switch>
        </n-form-item>
        <n-alert type="info" style="margin-top: 12px">
          <template v-if="assignDedicatedOnly && !assignLimitDevices">
            用户订阅<b>只显示专线节点</b>，且<b>不限制设备数量</b>。适合独享专线 VIP 用户。
          </template>
          <template v-else-if="assignDedicatedOnly && assignLimitDevices">
            用户订阅<b>只显示专线节点</b>，设备数量<b>跟随系统限制</b>。
          </template>
          <template v-else-if="!assignDedicatedOnly && !assignLimitDevices">
            专线节点<b>附加到公共节点列表</b>中，且<b>不限制设备数量</b>。
          </template>
          <template v-else>
            专线节点<b>附加到公共节点列表</b>中，设备数量<b>跟随系统限制</b>。
          </template>
        </n-alert>
      </n-form>
    </common-drawer>

    <!-- Import Links Drawer -->
    <common-drawer
      v-model:show="showImportDrawer"
      title="导入专线节点"
      :width="600"
      show-footer
      :loading="importing"
      @confirm="handleImportSubmit"
      @cancel="showImportDrawer = false"
    >
      <n-form label-placement="top">
        <n-form-item label="导入方式">
          <n-radio-group v-model:value="importType" @update:value="importTypeChanged">
            <n-radio-button value="subscription">订阅地址</n-radio-button>
            <n-radio-button value="links">节点链接</n-radio-button>
          </n-radio-group>
        </n-form-item>

        <template v-if="importType === 'subscription'">
          <n-alert type="info" :bordered="false" style="margin-bottom: 16px;">
            订阅导入采用<b>同步更新</b>模式：同一订阅地址重新导入时，
            按节点名称更新配置（<b>已分配给用户的节点保持分配不变</b>）；
            新节点自动加入；原订阅中已消失的节点将停用（不删除）。
          </n-alert>
          <n-form-item label="订阅地址">
            <n-input
              v-model:value="importUrl"
              placeholder="https://example.com/sub?token=xxx"
              @keyup.enter="handleImportSubmit"
            />
          </n-form-item>
        </template>

        <template v-else>
          <n-form-item label="节点链接">
            <n-input
              v-model:value="importLinks"
              type="textarea"
              placeholder="每行一个节点链接，支持 vmess:// vless:// trojan:// ss://"
              :rows="8"
            />
          </n-form-item>
        </template>
      </n-form>
    </common-drawer>

    <!-- View Link Modal -->
    <n-modal
      v-model:show="showLinkModal"
      title="节点链接"
      preset="card"
      :style="appStore.isMobile ? 'width: 95vw; max-width: 600px' : 'width: 600px'"
    >
      <n-form label-placement="top">
        <n-form-item :label="linkData.name">
          <n-input
            :value="linkData.link"
            type="textarea"
            readonly
            :rows="4"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 12px">
          <n-button @click="handleCopyLink">复制链接</n-button>
          <n-button @click="showLinkModal = false">关闭</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup>
import { ref, reactive, computed, h, onActivated, onMounted } from 'vue'
import { usePageLoading } from '@/composables/usePageLoading'
import { NButton, NTag, NSpace, NIcon, NSwitch, NRadioGroup, NRadioButton, useMessage, useDialog } from 'naive-ui'
import {
  CreateOutline,
  AddOutline,
  TrashOutline,
  PeopleOutline,
  CloudUploadOutline,
  LinkOutline,
  SearchOutline,
  RefreshOutline
} from '@vicons/ionicons5'
import {
  listCustomNodes,
  createCustomNode,
  updateCustomNode,
  deleteCustomNode,
  assignCustomNode,
  batchAssignCustomNodes,
  listUsers,
  importCustomNodeLinks,
  importCustomNodes,
  batchDeleteCustomNodes,
  getCustomNodeLink,
  listCustomNodeSources,
  createCustomNodeSource,
  updateCustomNodeSource,
  deleteCustomNodeSource,
  syncCustomNodeSource,
  syncAllCustomNodeSources
} from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { copyToClipboard as clipboardCopy } from '@/utils/clipboard'
import { formatFullDateTime, formatDateTime } from '@/utils/date'
import CommonDrawer from '@/components/CommonDrawer.vue'
import BatchSelectBar from '@/components/BatchSelectBar.vue'
import { useBatchSelection } from '@/composables/useBatchSelection'

const message = useMessage()
const dialog = useDialog()
const appStore = useAppStore()

const { loading, beginLoad, endLoad } = usePageLoading()
const submitting = ref(false)
const assigning = ref(false)
const loadingUsers = ref(false)
const showEditDrawer = ref(false)
const showAssignDrawer = ref(false)
const tableData = ref([])
const formRef = ref(null)
const editId = ref(null)
const assignNodeId = ref(null)
const assignNodeIds = ref([])
const assignUserIds = ref([])
const assignExpiresAt = ref(null)
const assignDedicatedOnly = ref(false)
const assignLimitDevices = ref(false)
const userOptions = ref([])
const showImportDrawer = ref(false)
const showLinkModal = ref(false)
const importing = ref(false)
const importType = ref('subscription')
const importUrl = ref('')
const importLinks = ref('')
// 全选 / 多选：全站统一实现（桌面表格与手机卡片共用同一份选择状态）
const selection = useBatchSelection(() => tableData.value)
// Naive 的表格要的是数组，这里做一层桥接，保证两边状态一致
const checkedRowKeys = computed({
  get: () => [...selection.selectedKeys.value],
  set: (keys) => { selection.selectedKeys.value = new Set(keys) }
})
const linkData = reactive({ link: '', name: '', protocol: '' })
const sortState = ref({ sort: 'id', order: 'desc' })
const searchKeyword = ref('')

// ===== 订阅来源（专线节点的导入来源，可换链接/立即更新/删除）=====
const sources = ref([])
const loadingSources = ref(false)
const syncingId = ref(null)
const syncingAll = ref(false)
const showSourceDrawer = ref(false)
const savingSource = ref(false)
const sourceForm = reactive({ id: null, name: '', url: '', interval_hours: 6 })

const formData = reactive({
  name: '',
  display_name: '',
  protocol: 'vmess',
  domain: '',
  port: 443,
  config: '',
  is_active: true,
  expire_time: null,
  follow_user_expire: false
})

const rules = {
  name: { required: true, message: '请输入节点名称', trigger: 'blur' },
  display_name: { required: true, message: '请输入显示名称', trigger: 'blur' },
  protocol: { required: true, message: '请选择协议', trigger: 'change' },
  domain: { required: true, message: '请输入域名或IP', trigger: 'blur' },
  port: { required: true, type: 'number', message: '请输入端口号', trigger: 'blur' }
}

const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100]
})

const protocolOptions = [
  { label: 'VMess', value: 'vmess' },
  { label: 'VLESS', value: 'vless' },
  { label: 'Trojan', value: 'trojan' },
  { label: 'Shadowsocks', value: 'ss' },
  { label: 'Hysteria2', value: 'hysteria2' }
]

const protocolColorMap = {
  vmess: 'info',
  vless: 'success',
  trojan: 'warning',
  ss: 'default',
  hysteria2: 'error'
}

const columns = [
  { type: 'selection' },
  { title: 'ID', key: 'id', width: 80, resizable: true, sorter: 'default' },
  { title: '节点名称', key: 'name', ellipsis: { tooltip: true }, minWidth: 150 },
  { title: '显示名称', key: 'display_name', ellipsis: { tooltip: true }, minWidth: 150 },
  {
    title: '协议',
    key: 'protocol',
    width: 120,
    resizable: true,
    render: (row) => {
      const type = protocolColorMap[row.protocol] || 'default'
      return h(NTag, { type }, { default: () => row.protocol.toUpperCase() })
    }
  },
  { title: '域名', key: 'domain', ellipsis: { tooltip: true }, minWidth: 180 },
  { title: '端口', key: 'port', width: 100, resizable: true },
  {
    title: '状态',
    key: 'is_active',
    width: 100,
    resizable: true,
    render: (row) => {
      return h(NSwitch, {
        value: row.is_active,
        onUpdateValue: (value) => handleToggleActive(row, value)
      })
    }
  },
  {
    title: '过期时间',
    key: 'expire_time',
    width: 160,
    resizable: true,
    render: (row) => row.expire_time ? formatFullDateTime(row.expire_time) : '-'
  },
  {
    title: '操作',
    key: 'actions',
    width: 280,
    fixed: 'right',
    render: (row) => {
      return h(NSpace, {}, {
        default: () => [
          h(NButton, {
            size: 'small',
            type: 'primary',
            text: true,
            onClick: () => handleEdit(row)
          }, { default: () => '编辑', icon: () => h(NIcon, {}, { default: () => h(CreateOutline) }) }),
          h(NButton, {
            size: 'small',
            type: 'info',
            text: true,
            onClick: () => handleAssign(row)
          }, { default: () => '分配', icon: () => h(NIcon, {}, { default: () => h(PeopleOutline) }) }),
          h(NButton, {
            size: 'small',
            text: true,
            onClick: () => handleViewLink(row)
          }, { default: () => '链接', icon: () => h(NIcon, {}, { default: () => h(LinkOutline) }) }),
          h(NButton, {
            size: 'small',
            type: 'error',
            text: true,
            onClick: () => handleDelete(row)
          }, { default: () => '删除', icon: () => h(NIcon, {}, { default: () => h(TrashOutline) }) })
        ]
      })
    }
  }
]

// ===== 订阅来源管理 =====
const loadSources = async () => {
  loadingSources.value = true
  try {
    const res = await listCustomNodeSources()
    sources.value = res.data.list || []
  } catch (error) {
    message.error(error.message || '获取订阅来源失败')
  } finally {
    loadingSources.value = false
  }
}

// 链接过长时中间省略，鼠标悬停看完整链接（title 属性）
const maskSourceUrl = (url) => {
  const u = String(url || '')
  if (u.length <= 52) return u
  return u.slice(0, 30) + ' ... ' + u.slice(-18)
}

const openSourceForm = (src = null) => {
  sourceForm.id = src ? src.id : null
  sourceForm.name = src ? src.name || '' : ''
  sourceForm.url = src ? src.url || '' : ''
  sourceForm.interval_hours = src ? (src.interval_hours ?? 6) : 6
  showSourceDrawer.value = true
}

const handleSourceSubmit = async () => {
  const url = String(sourceForm.url || '').trim()
  if (!/^https?:\/\//i.test(url)) {
    message.warning('请填写以 http:// 或 https:// 开头的订阅链接')
    return
  }
  savingSource.value = true
  try {
    const payload = {
      name: String(sourceForm.name || '').trim(),
      url,
      interval_hours: Number(sourceForm.interval_hours) || 0
    }
    const res = sourceForm.id
      ? await updateCustomNodeSource(sourceForm.id, payload)
      : await createCustomNodeSource(payload)
    // 首次同步失败时后端仍返回 200，但带 sync_error，要如实提示而不是假装成功
    if (res.data && res.data.sync_error) {
      message.warning(res.data.message || ('订阅已保存，但同步失败：' + res.data.sync_error), { duration: 6000 })
    } else {
      message.success((res.data && res.data.message) || '保存成功')
    }
    showSourceDrawer.value = false
    await Promise.all([loadSources(), fetchData()])
  } catch (error) {
    message.error(error.message || '保存订阅来源失败')
  } finally {
    savingSource.value = false
  }
}

const handleSyncSource = async (src) => {
  syncingId.value = src.id
  try {
    const res = await syncCustomNodeSource(src.id)
    message.success((res.data && res.data.message) || '同步完成')
    await Promise.all([loadSources(), fetchData()])
  } catch (error) {
    message.error(error.message || '同步失败')
    await loadSources()
  } finally {
    syncingId.value = null
  }
}

const handleSyncAllSources = async () => {
  syncingAll.value = true
  try {
    const res = await syncAllCustomNodeSources()
    const d = (res.data || {})
    // 后端返回：synced / skipped / inserted / updated / failed / messages
    let text = `全部更新完成：已更新 ${d.synced || 0} 个来源，新增 ${d.inserted || 0}，更新 ${d.updated || 0}`
    if (d.skipped) text += `（已跳过 ${d.skipped} 个关闭自动同步的来源）`
    if (d.failed) {
      text += `，失败 ${d.failed} 个`
      const detail = (d.messages || []).join('；')
      message.warning(detail ? `${text}（${detail}）` : text, { duration: 8000 })
    } else {
      message.success(text)
    }
    await Promise.all([loadSources(), fetchData()])
  } catch (error) {
    message.error(error.message || '全部更新失败')
  } finally {
    syncingAll.value = false
  }
}

// 开关自动同步（关闭后仍可点「立即更新」手动同步）
const handleToggleSource = async (src, value) => {
  try {
    await updateCustomNodeSource(src.id, { enabled: value })
    src.enabled = value
    message.success(value ? '已开启自动同步' : '已关闭自动同步（仍可手动更新）')
  } catch (error) {
    message.error(error.message || '修改失败')
    await loadSources()
  }
}

const handleDeleteSource = (src) => {
  dialog.warning({
    title: '删除订阅来源',
    content: () =>
      h('div', { style: 'line-height:1.9' }, [
        h('div', `来源：${src.name || src.url}`),
        h('div', `该来源下现有 ${src.active_nodes || 0} 个启用节点，已被分配 ${src.assigned_users || 0} 次。`),
        h('div', { style: 'color:#d03050;margin-top:6px' }, '点「一起删除」会同时删掉该来源导入的节点和用户分配；点「保留节点」只删来源，节点留在列表里但不再自动更新。')
      ]),
    positiveText: '一起删除',
    negativeText: '保留节点',
    onPositiveClick: () => doDeleteSource(src, true),
    onNegativeClick: () => doDeleteSource(src, false)
  })
}

const doDeleteSource = async (src, deleteNodes) => {
  try {
    const res = await deleteCustomNodeSource(src.id, deleteNodes)
    message.success((res.data && res.data.message) || '已删除订阅来源')
    await Promise.all([loadSources(), fetchData()])
  } catch (error) {
    message.error(error.message || '删除失败')
  }
}

const fetchData = async () => {
  beginLoad(tableData.value.length > 0)
  try {
    const res = await listCustomNodes({
      page: pagination.page,
      page_size: pagination.pageSize,
      sort: sortState.value.sort,
      order: sortState.value.order,
      search: searchKeyword.value.trim()
    })
    tableData.value = res.data.items || []
    pagination.itemCount = res.data.total || 0
  } catch (error) {
    message.error(error.message || '获取专线节点列表失败')
  } finally {
    endLoad()
  }
}

const handleSorterChange = (sorter) => {
  if (sorter && sorter.columnKey && sorter.order) {
    sortState.value.sort = sorter.columnKey
    sortState.value.order = sorter.order === 'ascend' ? 'asc' : 'desc'
  } else {
    sortState.value.sort = 'id'
    sortState.value.order = 'desc'
  }
  pagination.page = 1
  fetchData()
}

// 加载用户选项（支持远程搜索：有关键词调后端 search，无关键词加载默认前 50 个）
const fetchUsers = async (keyword = '') => {
  loadingUsers.value = true
  try {
    const params = { page: 1, page_size: 100 }
    if (String(keyword).trim()) params.search = String(keyword).trim()
    const res = await listUsers(params)
    const newOptions = (res.data.items || []).map(user => ({
      // 管理员账号同样可被搜索/分配，用后缀标识出来便于辨认
      label: `${user.email}${user.username ? ' · ' + user.username : ''} (ID: ${user.id})${user.is_admin ? ' [管理员]' : ''}`,
      value: user.id
    }))
    // 保留已选中的用户选项（远程搜索切换时已选的不消失）
    const selected = new Set(assignUserIds.value)
    const merged = [...userOptions.value.filter(o => selected.has(o.value)), ...newOptions]
    const seen = new Set()
    userOptions.value = merged.filter(o => (seen.has(o.value) ? false : seen.add(o.value)))
  } catch (error) {
    message.error(error.message || '获取用户列表失败')
  } finally {
    loadingUsers.value = false
  }
}

// 远程搜索用户（输入关键词触发，300ms 防抖）
let userSearchTimer = null
const handleUserSearch = (query) => {
  if (userSearchTimer) clearTimeout(userSearchTimer)
  userSearchTimer = setTimeout(() => fetchUsers(query), 300)
}

const handlePageChange = (page) => {
  pagination.page = page
  fetchData()
}

const handleSearch = () => {
  pagination.page = 1
  fetchData()
}

const handleResetSearch = () => {
  searchKeyword.value = ''
  pagination.page = 1
  fetchData()
}

const resetForm = () => {
  Object.assign(formData, {
    name: '',
    display_name: '',
    protocol: 'vmess',
    domain: '',
    port: 443,
    config: '',
    is_active: true,
    expire_time: null,
    follow_user_expire: false
  })
  formRef.value?.restoreValidation()
}

const handleCreate = () => {
  editId.value = null
  resetForm()
  showEditDrawer.value = true
}

const handleEdit = (row) => {
  editId.value = row.id
  Object.assign(formData, {
    name: row.name,
    display_name: row.display_name,
    protocol: row.protocol,
    domain: row.domain,
    port: row.port,
    config: row.config || '',
    is_active: row.is_active,
    expire_time: row.expire_time ? new Date(row.expire_time).getTime() : null,
    follow_user_expire: row.follow_user_expire || false
  })
  showEditDrawer.value = true
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    await formRef.value?.validate()

    const data = {
      ...formData,
      expire_time: formData.expire_time ? new Date(formData.expire_time).toISOString() : null
    }

    if (editId.value) {
      await updateCustomNode(editId.value, data)
      message.success('更新专线节点成功')
    } else {
      await createCustomNode(data)
      message.success('创建专线节点成功')
    }

    showEditDrawer.value = false
    fetchData()
  } catch (error) {
    if (error.message) {
      message.error(error.message || '操作失败')
    }
  } finally {
    submitting.value = false
  }
}

const handleToggleActive = async (row, value) => {
  try {
    await updateCustomNode(row.id, { is_active: value })
    row.is_active = value // 本地更新，避免整表刷新卡顿
    message.success('更新状态成功')
  } catch (error) {
    message.error(error.message || '更新状态失败')
  }
}

const handleDelete = async (row) => {
  try {
    await deleteCustomNode(row.id)
    message.success('删除专线节点成功')
    fetchData()
  } catch (error) {
    message.error(error.message || '删除专线节点失败')
  }
}

const handleAssign = (row) => {
  assignNodeId.value = row.id
  assignNodeIds.value = [row.id]
  assignUserIds.value = []
  assignExpiresAt.value = null
  assignDedicatedOnly.value = false
  assignLimitDevices.value = false
  showAssignDrawer.value = true
  // 每次打开刷新默认用户列表（前 50 个），支持输入关键词远程搜索更多
  fetchUsers()
}

const handleBatchAssign = () => {
  if (!selection.count.value) return
  assignNodeId.value = null
  assignNodeIds.value = selection.selectedRows.value.map(r => r.id)
  assignUserIds.value = []
  assignExpiresAt.value = null
  assignDedicatedOnly.value = false
  assignLimitDevices.value = false
  showAssignDrawer.value = true
  // 每次打开刷新默认用户列表（前 50 个），支持输入关键词远程搜索更多
  fetchUsers()
}

const handleAssignSubmit = async () => {
  if (assignUserIds.value.length === 0) {
    message.warning('请至少选择一个用户')
    return
  }

  if (assignNodeIds.value.length === 0) {
    message.warning('请选择要分配的专线节点')
    return
  }

  const expiresAt = assignExpiresAt.value ? new Date(assignExpiresAt.value).toISOString() : null

  assigning.value = true
  try {
    if (assignNodeIds.value.length === 1) {
      await assignCustomNode(assignNodeIds.value[0], {
        user_ids: assignUserIds.value,
        expires_at: expiresAt,
        dedicated_only: assignDedicatedOnly.value,
        limit_devices: assignLimitDevices.value
      })
      message.success('分配节点成功')
    } else {
      const res = await batchAssignCustomNodes({
        ids: assignNodeIds.value,
        user_ids: assignUserIds.value,
        expires_at: expiresAt,
        dedicated_only: assignDedicatedOnly.value,
        limit_devices: assignLimitDevices.value
      })
      const successCount = res.data?.success || 0
      const totalCount = res.data?.total || assignNodeIds.value.length
      if (successCount !== totalCount) {
        message.warning(`部分分配成功：成功 ${successCount} 个，失败 ${totalCount - successCount} 个`)
      } else {
        message.success(`批量分配成功，共 ${successCount} 个节点`)
      }
    }

    showAssignDrawer.value = false
    selection.clear()
    assignNodeId.value = null
    assignNodeIds.value = []
    assignExpiresAt.value = null
    fetchData()
  } catch (error) {
    message.error(error.message || '分配节点失败')
  } finally {
    assigning.value = false
  }
}

const handleImportSubmit = async () => {
  if (importType.value === 'subscription') {
    if (!importUrl.value.trim()) {
      message.warning('请输入订阅地址')
      return
    }
  } else if (!importLinks.value.trim()) {
    message.warning('请输入节点链接')
    return
  }
  importing.value = true
  try {
    const payload = importType.value === 'subscription'
      ? { type: 'subscription', url: importUrl.value.trim() }
      : { type: 'links', links: importLinks.value }
    const res = await importCustomNodes(payload)
    const d = res.data
    if (importType.value === 'subscription') {
      message.success(`同步完成: 新增 ${d.inserted || 0}, 更新 ${d.updated || 0}, 停用 ${d.deactivated || 0}`)
    } else {
      message.success(`导入完成: 成功 ${d.success || 0}/${d.total || 0} 个`)
    }
    showImportDrawer.value = false
    importUrl.value = ''
    importLinks.value = ''
    fetchData()
  } catch (error) {
    message.error(error.message || '导入失败')
  } finally {
    importing.value = false
  }
}

const importTypeChanged = () => { /* 切换方式时无需清理 */ }

const handleBatchDelete = async () => {
  if (!selection.count.value) return
  try {
    await batchDeleteCustomNodes({ ids: selection.selectedRows.value.map(r => r.id) })
    message.success('批量删除成功')
    selection.clear()
    fetchData()
  } catch (error) {
    message.error(error.message || '批量删除失败')
  }
}

const handleViewLink = async (row) => {
  try {
    const res = await getCustomNodeLink(row.id)
    Object.assign(linkData, res.data)
    showLinkModal.value = true
  } catch (error) {
    message.error(error.message || '获取链接失败')
  }
}

const handleCopyLink = async () => {
  if (linkData.link) {
    const ok = await clipboardCopy(linkData.link)
    ok ? message.success('链接已复制') : message.error('复制失败')
  }
}

onMounted(() => {
  fetchData()
  loadSources()
})

// KeepAlive 缓存激活时静默刷新数据（不清 loading 遮罩、不重置分页）
onActivated(() => {
  fetchData()
  loadSources()
})
</script>

<style scoped>
/* ===== 订阅来源卡片 ===== */
.source-card {
  margin-bottom: 16px;
}

.source-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.source-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
}

.source-table-wrap {
  margin-top: 10px;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

/* 列太挤会把链接折成一列一个字符，给表格一个最小宽度并允许横向滚动 */
.source-table-wrap :deep(table) {
  min-width: 760px;
}

.source-url {
  max-width: 320px;
  word-break: break-all;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
}

.source-sub {
  margin-top: 2px;
  font-size: 12px;
  color: var(--n-text-color-3, #999);
  word-break: break-all;
}

/* 多选下拉的输入框宽度由内部 mirror 撑开：空输入时只有几像素，
   视觉上像「大框里套了个极小的输入框」，也让人不易察觉这里可以打字搜索。
   必须用 :deep()：scoped 会把 [data-v-x] 加到最后一个选择器上，而 naive-ui 内部元素没有该属性。 */
.assign-user-select :deep(.n-base-selection-input-tag),
.assign-user-select :deep(.n-base-selection-input-tag__input),
.assign-user-select :deep(.n-base-selection-input-tag__mirror) {
  min-width: 100px;
}

.desktop-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.desktop-toolbar-search {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.mobile-card-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.mobile-card {
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.08);
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-bottom: 1px solid #f0f0f0;
}

.card-title {
  font-weight: 600;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  margin-right: 8px;
}

.card-body {
  padding: 10px 14px;
}

.card-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
  font-size: 13px;
}

.card-label {
  color: #999;
}

.card-actions {
  display: flex;
  gap: 8px;
  padding: 10px 14px;
  border-top: 1px solid #f0f0f0;
  flex-wrap: wrap;
}

@media (max-width: 767px) {
  .custom-nodes-container { padding: 8px; }
}
.mobile-toolbar { margin-bottom: 12px; }
.mobile-toolbar-title { font-size: 17px; font-weight: 600; margin-bottom: 10px; color: var(--text-color, #333); }
.mobile-toolbar-search { display: flex; flex-direction: column; gap: 8px; margin-bottom: 10px; }
.mobile-toolbar-search-actions { display: flex; gap: 8px; }
.mobile-toolbar-row { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
</style>
