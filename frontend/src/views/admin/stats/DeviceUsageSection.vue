<template>
  <div class="device-usage-section">

    <!-- 概览统计（对齐 myweb：设备 / 用户 / 官方 / 第三方） -->
    <n-grid :cols="appStore.isMobile ? 2 : 4" :x-gap="12" :y-gap="12" class="summary-grid">
      <n-gi>
        <n-card :bordered="false" class="summary-card">
          <div class="summary-label">设备</div>
          <div class="summary-value">{{ summary.devices || 0 }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card :bordered="false" class="summary-card">
          <div class="summary-label">用户</div>
          <div class="summary-value">{{ summary.users || 0 }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card :bordered="false" class="summary-card clickable" :class="{ active: filters.official === 'official' }" @click="switchOfficial('official')">
          <div class="summary-label">官方客户端</div>
          <div class="summary-value official">{{ summary.official_devices || 0 }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card :bordered="false" class="summary-card clickable" :class="{ active: filters.official === 'third_party' }" @click="switchOfficial('third_party')">
          <div class="summary-label">第三方客户端（点击只看第三方）</div>
          <div class="summary-value third">{{ summary.third_party_devices || 0 }}</div>
        </n-card>
      </n-gi>
    </n-grid>

    <n-card :bordered="false" class="page-card">
      <n-space vertical :size="14">
        <n-tabs v-model:value="tab" type="line" @update:value="onTabChange">
          <n-tab-pane name="user" tab="按用户">
            <n-space :wrap="true" :size="12" align="center" class="toolbar">
              <n-input
                v-model:value="filters.q"
                placeholder="搜索型号 / 系统 / 客户端 / 版本 / 用户"
                clearable
                style="width: 280px"
                @keyup.enter="loadUsers(1)"
              />
              <n-radio-group v-model:value="filters.official" size="small" @update:value="loadUsers(1)">
                <n-radio-button value="all">全部</n-radio-button>
                <n-radio-button value="official">官方</n-radio-button>
                <n-radio-button value="third_party">第三方</n-radio-button>
              </n-radio-group>
              <n-select v-model:value="filters.days" :options="daysOptions" style="width: 130px" @update:value="loadUsers(1)" />
              <n-button type="primary" @click="loadUsers(1)">搜索</n-button>
              <n-button type="primary" plain :disabled="!checkedKeys.length" @click="openNotify(checkedKeys)">
                提醒所选用户<template v-if="checkedKeys.length">（{{ checkedKeys.length }}）</template>
              </n-button>
            </n-space>

            <n-data-table
              :columns="userColumns"
              :data="users"
              :loading="userLoading"
              :bordered="false"
              :scroll-x="1220"
              :row-key="(r) => r.user_id"
              v-model:checked-row-keys="checkedKeys"
              size="small"
            />
            <div class="pager">
              <n-pagination
                v-model:page="userPage"
                :page-size="pageSize"
                :item-count="userTotal"
                :page-sizes="[20, 50, 100, 200]"
                show-size-picker
                @update:page="loadUsers"
                @update:page-size="onPageSizeChange"
              />
            </div>
          </n-tab-pane>

          <n-tab-pane name="group" tab="按分组">
            <n-space align="center" :size="12" style="margin-bottom: 12px">
              <span class="muted">分组维度</span>
              <n-select v-model:value="dim" :options="dimOptions" style="width: 180px" @update:value="loadGroups(1)" />
              <span class="muted" v-if="groupSummary.devices">共 {{ groupSummary.devices }} 台设备 / {{ groupSummary.users }} 个用户</span>
            </n-space>
            <n-data-table
              :columns="groupColumns"
              :data="groups"
              :loading="groupLoading"
              :bordered="false"
              size="small"
              :row-key="(r) => r.key"
            />
            <div class="pager">
              <n-pagination
                v-model:page="groupPage"
                :page-size="pageSize"
                :item-count="groupTotal"
                :page-sizes="[20, 50, 100, 200]"
                show-size-picker
                @update:page="loadGroups"
                @update:page-size="onPageSizeChange"
              />
            </div>
          </n-tab-pane>
        </n-tabs>
      </n-space>
    </n-card>

    <!-- 升级提醒：先预览（dry_run），确认后再入队 -->
    <n-modal
      v-model:show="notifyVisible"
      preset="card"
      title="发送升级提醒"
      style="width: 720px; max-width: 94vw"
      :bordered="false"
    >
      <n-spin :show="notifyLoading">
        <n-space vertical :size="12">
          <n-alert v-if="notifyResult && notifyResult.targets !== undefined && !notifyResult.smtp_configured" type="warning" :bordered="false">
            邮件服务（SMTP）未配置，入队后可能发不出去，请先在「系统设置 → 邮件」配置。
          </n-alert>
          <div v-if="notifyResult && notifyResult.targets !== undefined" class="notify-stats">
            <span>目标用户 <b>{{ notifyResult.targets }}</b> 人</span>
            <span v-if="notifyResult.skipped_no_email">无邮箱 {{ notifyResult.skipped_no_email }}</span>
            <span v-if="notifyResult.skipped_inactive">已禁用 {{ notifyResult.skipped_inactive }}</span>
            <span v-if="notifyResult.skipped_unsubscribed">已退订 {{ notifyResult.skipped_unsubscribed }}</span>
            <span v-if="notifyResult.truncated">超出批次截断 {{ notifyResult.truncated }}</span>
          </div>
          <div v-if="notifyResult && notifyResult.queued">已入队 {{ notifyResult.queued }} 封，失败 {{ notifyResult.failed }} 封</div>
          <n-input v-model:value="notifySubject" placeholder="邮件主题（留空用默认）" />
          <div v-if="notifyResult && notifyResult.preview" class="preview-box">
            <div class="preview-subject">{{ notifyResult.preview.subject }}</div>
            <div class="preview-content" v-html="notifyResult.preview.content"></div>
          </div>
          <n-empty v-else-if="!notifyLoading" description="暂无可提醒的用户" />
        </n-space>
      </n-spin>
      <template #footer>
        <n-space justify="end">
          <n-button @click="notifyVisible = false">取消</n-button>
          <n-button type="primary" :loading="notifyLoading" :disabled="!notifyResult || !notifyResult.targets" @click="confirmNotify">
            确认发送
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup>
import { computed, h, onMounted, ref } from 'vue'
import { NButton, NTag, useMessage } from 'naive-ui'
import { getDeviceUsageGroups, getDeviceUsageUsers, notifyClientUpgrade } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import '@/styles/admin-common.css'

const appStore = useAppStore()
const message = useMessage()

const filters = ref({ q: '', official: 'all', days: '0' })
const tab = ref('user')
const pageSize = ref(20)

const daysOptions = [
  { label: '全部时间', value: '0' },
  { label: '近 7 天', value: '7' },
  { label: '近 30 天', value: '30' },
  { label: '近 90 天', value: '90' },
]
const dimOptions = [
  { label: '客户端软件', value: 'client' },
  { label: '客户端版本', value: 'client_version' },
  { label: '平台（系统）', value: 'platform' },
  { label: '系统版本', value: 'os_version' },
  { label: 'iOS 设备型号', value: 'ios_model' },
  { label: '设备类型', value: 'device_type' },
  { label: '设备品牌', value: 'brand' },
]

const summary = ref({})
const checkedKeys = ref([])

// 官方客户端是 Mclash（大小写/版本后缀都算）
const isOfficialClient = (name) => String(name || '').trim().toLowerCase().startsWith('mclash')

// 官方客户端绿色、第三方客户端黄色（对齐 myweb 的彩色标签）
const clientTagType = (label) => (isOfficialClient(String(label).split(' ')[0]) ? 'success' : 'warning')

const userColumns = computed(() => [
  { type: 'selection', width: 44 },
  {
    title: '用户', key: 'user', width: 190,
    render: (r) => h('div', [
      h('div', { style: 'font-weight:500' }, r.username || r.email || `用户 #${r.user_id}`),
      h('div', { style: 'color:var(--text-color-3);font-size:12px' }, r.email || ''),
    ]),
  },
  {
    title: '使用的客户端（软件 + 版本）', key: 'clients', minWidth: 320,
    render: (r) => {
      const list = r.clients || []
      if (!list.length) return h('span', { class: 'muted' }, 'Unknown')
      return h('div', { class: 'client-tags' }, list.map((c) => h(
        NTag,
        { size: 'small', bordered: true, type: clientTagType(c) },
        { default: () => c },
      )))
    },
  },
  { title: '设备', key: 'devices', width: 90, render: (r) => `${r.online_count || 0}/${r.devices || 0}` },
  {
    title: '机型 / 系统', key: 'brands', width: 220,
    render: (r) => (r.brands && r.brands.length ? r.brands.join('、') : '-'),
  },
  { title: '最近活跃', key: 'last_access', width: 150, render: (r) => r.last_access || '-' },
  { title: '订阅到期', key: 'sub_expire', width: 110, render: (r) => r.sub_expire || '-' },
  {
    title: '操作', key: 'actions', width: 120, fixed: 'right',
    render: (r) => h(NButton, {
      size: 'small', type: 'primary', text: true,
      onClick: () => openNotify([r.user_id]),
    }, { default: () => '发送升级通知' }),
  },
])

const users = ref([])
const userLoading = ref(false)
const userPage = ref(1)
const userTotal = ref(0)

const loadUsers = async (page = userPage.value) => {
  userLoading.value = true
  try {
    userPage.value = page
    const res = await getDeviceUsageUsers({
      q: filters.value.q || undefined,
      official: filters.value.official,
      days: filters.value.days,
      page: userPage.value,
      page_size: pageSize.value,
    })
    const data = res.data || {}
    users.value = data.items || []
    userTotal.value = data.total || 0
  } finally {
    userLoading.value = false
  }
}

const groups = ref([])
const groupLoading = ref(false)
const groupPage = ref(1)
const groupTotal = ref(0)
const groupSummary = ref({})
const dim = ref('client')

const loadGroups = async (page = groupPage.value) => {
  groupLoading.value = true
  try {
    groupPage.value = page
    const res = await getDeviceUsageGroups({
      dim: dim.value,
      q: filters.value.q || undefined,
      official: filters.value.official,
      days: filters.value.days,
      page: groupPage.value,
      page_size: pageSize.value,
    })
    const data = res.data || {}
    groups.value = data.groups || []
    groupTotal.value = data.group_total || 0
    groupSummary.value = data.summary || {}
  } finally {
    groupLoading.value = false
  }
}

const loadSummary = async () => {
  const res = await getDeviceUsageGroups({ dim: 'client', q: filters.value.q || undefined, official: 'all', days: filters.value.days, page: 1, page_size: 1 })
  const data = res.data || {}
  summary.value = data.summary || {}
}

const groupColumns = computed(() => [
  {
    title: '分组', key: 'label',
    render: (r) => h('div', { class: 'group-label' }, [
      h('span', r.official ? { style: 'color:var(--success-color);font-weight:500' } : {}, r.label || '未知'),
      r.official ? h(NTag, { size: 'tiny', type: 'success', bordered: false, style: 'margin-left:6px' }, { default: () => '官方' }) : null,
    ]),
  },
  { title: '设备数', key: 'devices', width: 100 },
  { title: '用户数', key: 'users', width: 100 },
  { title: '占比', key: 'share', width: 100, render: (r) => `${r.share || 0}%` },
  { title: '最近活跃', key: 'last_access', width: 160, render: (r) => r.last_access || '-' },
  {
    title: '操作', key: 'actions', width: 120,
    render: (r) => h(NButton, {
      size: 'small', type: 'primary', text: true,
      onClick: () => drillDown(r),
    }, { default: () => '查看用户' }),
  },
])

// 点分组 → 切到按用户并只看该分组（下钻）
const drillDown = (row) => {
  filters.value.q = row.key || ''
  tab.value = 'user'
  loadUsers(1)
}

const notifyVisible = ref(false)
const notifyLoading = ref(false)
const notifyResult = ref(null)
const notifySubject = ref('')
const notifyUserIDs = ref([])

const openNotify = async (ids) => {
  notifyUserIDs.value = [...ids]
  notifyResult.value = null
  notifySubject.value = ''
  notifyVisible.value = true
  notifyLoading.value = true
  try {
    const res = await notifyClientUpgrade({ user_ids: notifyUserIDs.value, dry_run: true })
    notifyResult.value = res.data || null
  } catch (e) {
    message.error('预览失败')
    notifyVisible.value = false
  } finally {
    notifyLoading.value = false
  }
}

const confirmNotify = async () => {
  notifyLoading.value = true
  try {
    const res = await notifyClientUpgrade({
      user_ids: notifyUserIDs.value,
      subject: notifySubject.value || undefined,
      dry_run: false,
    })
    notifyResult.value = res.data || null
    message.success('已加入邮件队列')
  } catch (e) {
    message.error('发送失败')
  } finally {
    notifyLoading.value = false
  }
}

const switchOfficial = (value) => {
  filters.value.official = filters.value.official === value ? 'all' : value
  loadSummary()
  if (tab.value === 'user') loadUsers(1)
  else loadGroups(1)
}

const onPageSizeChange = (size) => {
  pageSize.value = size
  if (tab.value === 'user') loadUsers(1)
  else loadGroups(1)
}

const onTabChange = (name) => {
  if (name === 'group' && groups.value.length === 0) loadGroups(1)
  if (name === 'user' && users.value.length === 0) loadUsers(1)
}

const reloadAll = () => {
  loadSummary()
  if (tab.value === 'user') loadUsers(1)
  else loadGroups(1)
}

onMounted(() => {
  loadSummary()
  loadUsers(1)
})
</script>

<style scoped>
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 12px; }
.page-title { margin: 0; font-size: 20px; }
.page-subtitle { margin: 4px 0 0; color: var(--text-color-3); font-size: 13px; }
.summary-grid { margin-bottom: 12px; }
.summary-card { padding: 4px 2px; }
.summary-card.clickable { cursor: pointer; }
.summary-card.active { box-shadow: 0 0 0 2px var(--primary-color); }
.summary-label { color: var(--text-color-3); font-size: 12px; }
.summary-value { font-size: 22px; font-weight: 700; margin-top: 2px; }
.summary-value.official { color: var(--success-color); }
.summary-value.third { color: var(--warning-color, #f0a020); }
.toolbar { margin-bottom: 12px; }
.client-tags { display: flex; flex-wrap: wrap; gap: 4px; }
.muted { color: var(--text-color-3); font-size: 12px; }
.pager { display: flex; justify-content: flex-end; margin-top: 12px; }
.group-label { display: flex; align-items: center; }
.notify-stats { display: flex; flex-wrap: wrap; gap: 12px; color: var(--text-color-2); font-size: 13px; }
.preview-box { border: 1px solid var(--border-color); border-radius: 6px; padding: 12px; background: var(--body-color); }
.preview-subject { font-weight: 600; margin-bottom: 8px; }
.preview-content { font-size: 13px; line-height: 1.7; }
</style>
