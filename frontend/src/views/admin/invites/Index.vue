<template>
  <div class="admin-invites-page admin-page-shell">
    <n-card :title="appStore.isMobile ? undefined : '邀请码管理'" :bordered="false" class="page-card admin-main-card">
      <n-space vertical :size="16">
        <!-- Stats -->
        <div class="stats-row">
          <div class="stat-item">
            <span class="stat-val">{{ stats.total_codes || 0 }}</span>
            <span class="stat-lbl">总邀请码</span>
          </div>
          <div class="stat-item">
            <span class="stat-val" style="color: var(--success-color)">{{ stats.active_codes || 0 }}</span>
            <span class="stat-lbl">有效</span>
          </div>
          <div class="stat-item">
            <span class="stat-val">{{ stats.total_invites || 0 }}</span>
            <span class="stat-lbl">邀请人数</span>
          </div>
          <div class="stat-item">
            <span class="stat-val" style="color: var(--warning-color)">{{ formatCurrency(stats.total_inviter_reward) }}</span>
            <span class="stat-lbl">邀请人奖励</span>
          </div>
          <div class="stat-item">
            <span class="stat-val" style="color: #2080f0">{{ formatCurrency(stats.total_invitee_reward) }}</span>
            <span class="stat-lbl">受邀人奖励</span>
          </div>
        </div>

        <!-- Desktop Search -->
        <n-space v-if="!appStore.isMobile" align="center">
          <n-input v-model:value="search" placeholder="搜索邀请码或用户" clearable style="width: 240px" @keyup.enter="fetchCodes">
            <template #prefix><n-icon :component="SearchOutline" /></template>
          </n-input>
          <n-button @click="fetchCodes">搜索</n-button>
          <n-button @click="fetchCodes">
            <template #icon><n-icon :component="RefreshOutline" /></template>
            刷新
          </n-button>
        </n-space>

        <!-- Mobile toolbar：吸顶工具条（滚动时搜索框不跟着滚走）。
             注意不要用全局的 .mobile-toolbar-row：它把行内按钮按网格拉成整行宽，
             两个按钮加起来就超出屏幕（体检脚本报的 button 宽 373 跑到 764 就是这个原因）。 -->
        <div v-if="appStore.isMobile" class="app-sticky-toolbar invites-toolbar">
          <div class="invites-toolbar__title">邀请码管理</div>
          <n-input v-model:value="search" placeholder="搜索邀请码或用户" clearable size="small" @keyup.enter="fetchCodes">
            <template #prefix><n-icon :component="SearchOutline" /></template>
          </n-input>
          <div class="invites-toolbar__actions">
            <n-button size="small" @click="fetchCodes">搜索</n-button>
            <n-button size="small" @click="fetchCodes">
              <template #icon><n-icon :component="RefreshOutline" /></template>
              刷新
            </n-button>
          </div>
        </div>

        <!-- Tabs -->
        <n-tabs type="line" animated>
          <n-tab-pane name="codes" tab="邀请码列表">
            <!-- 全选 / 批量操作：公共组件（桌面在列表上方，手机固定在底部标签栏上方） -->
            <BatchSelectBar
              :total="selection.total.value"
              :selected-count="selection.count.value"
              :all-selected="selection.allSelected.value"
              :indeterminate="selection.indeterminate.value"
              label="个邀请码"
              @toggle-all="selection.toggleAll"
              @clear="selection.clear"
            >
              <n-button size="small" type="error" :disabled="!selection.count.value" @click="handleBatchDelete">
                批量删除
              </n-button>
            </BatchSelectBar>

            <template v-if="!appStore.isMobile">
              <n-data-table class="unified-admin-table" :columns="codeColumns" :data="codes" :loading="loadingCodes" :pagination="false" :bordered="false" :single-line="false" :row-key="(row) => row.id" v-model:checked-row-keys="checkedRowKeys" />
            </template>
            <template v-else>
              <n-spin :show="loadingCodes">
                <div v-if="codes.length === 0" class="mobile-empty">暂无数据</div>
                <div v-else class="mobile-card-list">
                  <div
                    v-for="code in codes"
                    :key="code.id"
                    class="mobile-card is-selectable"
                    :class="{ 'is-selected': selection.isSelected(code) }"
                    @click="selection.toggle(code)"
                  >
                    <div class="card-check" @click.stop>
                      <n-checkbox :checked="selection.isSelected(code)" @update:checked="() => selection.toggle(code)" />
                    </div>
                    <div class="card-header">
                      <span class="card-title" style="font-family:monospace">{{ code.code }}</span>
                      <n-tag :type="statusType(code.status)" size="small">{{ statusText(code.status) }}</n-tag>
                    </div>
                    <div class="card-body">
                      <div class="card-row"><span class="card-label">创建者</span><span>{{ code.username }}</span></div>
                      <div class="card-row"><span class="card-label">使用</span><span>{{ code.used_count }} / {{ code.max_uses || '∞' }}</span></div>
                      <div class="card-row"><span class="card-label">邀请人奖励</span><span>{{ formatCurrency(code.inviter_reward) }}</span></div>
                      <div class="card-row"><span class="card-label">受邀人奖励</span><span>{{ formatCurrency(code.invitee_reward) }}</span></div>
                    </div>
                    <div class="card-actions" @click.stop>
                      <n-button size="small" @click="handleToggle(code)">{{ code.is_active ? '禁用' : '启用' }}</n-button>
                      <n-button size="small" type="error" @click="handleDelete(code)">删除</n-button>
                    </div>
                  </div>
                </div>
              </n-spin>
            </template>
            <n-pagination class="list-pagination" v-model:page="codePage" :page-count="codeTotalPages" @update:page="fetchCodes" />
          </n-tab-pane>

          <n-tab-pane name="relations" tab="邀请记录">
            <template v-if="!appStore.isMobile">
              <n-data-table class="unified-admin-table" :columns="relColumns" :data="relations" :loading="loadingRels" :pagination="false" :bordered="false" :single-line="false" />
            </template>
            <template v-else>
              <n-spin :show="loadingRels">
                <div v-if="relations.length === 0" class="mobile-empty">暂无数据</div>
                <div v-else class="mobile-card-list">
                  <div v-for="rel in relations" :key="rel.id" class="mobile-card">
                    <div class="card-body">
                      <div class="card-row"><span class="card-label">邀请人</span><span>{{ rel.inviter_username }}</span></div>
                      <div class="card-row"><span class="card-label">受邀人</span><span>{{ rel.invitee_username }}</span></div>
                      <div class="card-row"><span class="card-label">邀请码</span><span style="font-family:monospace">{{ rel.invite_code }}</span></div>
                      <div class="card-row"><span class="card-label">邀请人奖励</span><span>{{ formatCurrency(rel.inviter_reward_amount) }} {{ rel.inviter_reward_given ? '✓' : '✗' }}</span></div>
                      <div class="card-row"><span class="card-label">受邀人奖励</span><span>{{ formatCurrency(rel.invitee_reward_amount) }} {{ rel.invitee_reward_given ? '✓' : '✗' }}</span></div>
                      <div class="card-row"><span class="card-label">时间</span><span>{{ formatFullDateTime(rel.created_at) }}</span></div>
                    </div>
                  </div>
                </div>
              </n-spin>
            </template>
            <n-pagination class="list-pagination" v-model:page="relPage" :page-count="relTotalPages" @update:page="fetchRelations" />
          </n-tab-pane>
        </n-tabs>
      </n-space>
    </n-card>
  </div>
</template>

<script setup>
import { ref, computed, h, onActivated, onMounted } from 'vue'
import { NButton, NTag, NIcon, useMessage, useDialog } from 'naive-ui'
import { SearchOutline, RefreshOutline } from '@vicons/ionicons5'
import { listAdminInviteCodes, getAdminInviteStats, listAdminInviteRelations, deleteAdminInviteCode, toggleAdminInviteCode } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { useBatchSelection } from '@/composables/useBatchSelection'
import BatchSelectBar from '@/components/BatchSelectBar.vue'
import { formatCurrency } from '@/utils/amount'
import { formatFullDateTime } from '@/utils/date'

const message = useMessage()
const dialog = useDialog()
const appStore = useAppStore()

const search = ref('')
const stats = ref({})
const codes = ref([])
const relations = ref([])
const loadingCodes = ref(false)
const loadingRels = ref(false)
const codePage = ref(1)
const relPage = ref(1)
const codeTotalPages = ref(0)
const relTotalPages = ref(0)
const pageSize = 20

// 全选 / 多选：全站统一实现（桌面表格与手机卡片共用同一份选择状态）
const selection = useBatchSelection(() => codes.value)
// Naive 的表格要的是数组，这里做一层桥接，保证两边状态一致
const checkedRowKeys = computed({
  get: () => [...selection.selectedKeys.value],
  set: (keys) => { selection.selectedKeys.value = new Set(keys) }
})

const statusType = (s) => ({ active: 'success', expired: 'warning', exhausted: 'default', disabled: 'error' }[s] || 'default')
const statusText = (s) => ({ active: '有效', expired: '已过期', exhausted: '已用完', disabled: '已禁用' }[s] || s)

const fetchStats = async () => {
  try {
    const res = await getAdminInviteStats()
    stats.value = res.data || {}
  } catch {}
}

const fetchCodes = async () => {
  loadingCodes.value = true
  try {
    const res = await listAdminInviteCodes({ page: codePage.value, page_size: pageSize, search: search.value || undefined })
    codes.value = res.data?.items || []
    codeTotalPages.value = Math.ceil((res.data?.total || 0) / pageSize)
  } catch (e) {
    message.error(e.message || '获取邀请码失败')
  } finally { loadingCodes.value = false }
}

const fetchRelations = async () => {
  loadingRels.value = true
  try {
    const res = await listAdminInviteRelations({ page: relPage.value, page_size: pageSize })
    relations.value = res.data?.items || []
    relTotalPages.value = Math.ceil((res.data?.total || 0) / pageSize)
  } catch (e) {
    message.error(e.message || '获取邀请记录失败')
  } finally { loadingRels.value = false }
}

const handleToggle = async (code) => {
  try {
    await toggleAdminInviteCode(code.id)
    message.success(code.is_active ? '已禁用' : '已启用')
    fetchCodes()
    fetchStats()
  } catch (e) { message.error(e.message || '操作失败') }
}

const handleDelete = (code) => {
  dialog.warning({
    title: '确认删除',
    content: `确定要删除邀请码 ${code.code} 吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteAdminInviteCode(code.id)
        message.success('已删除')
        fetchCodes()
        fetchStats()
      } catch (e) { message.error(e.message || '删除失败') }
    }
  })
}

const handleBatchDelete = () => {
  // 后端没有批量删除接口，这里按 id 逐个调用（与单条删除同一接口）
  const ids = selection.selectedRows.value.map(r => r.id)
  if (!ids.length) return
  dialog.warning({
    title: '批量删除',
    content: `确定要删除选中的 ${ids.length} 个邀请码吗？此操作不可恢复。`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await Promise.all(ids.map(id => deleteAdminInviteCode(id)))
        message.success('批量删除成功')
        selection.clear()
        fetchCodes()
        fetchStats()
      } catch { message.error('批量删除失败') }
    }
  })
}

const codeColumns = [
  { type: 'selection' },
  { title: '邀请码', key: 'code', width: 120, render: (r) => h('span', { style: 'font-family:monospace;font-weight:600' }, r.code) },
  { title: '创建者', key: 'username', width: 120 },
  { title: '使用/上限', key: 'usage', width: 100, render: (r) => `${r.used_count} / ${r.max_uses || '∞'}` },
  { title: '邀请人奖励', key: 'inviter_reward', width: 100, render: (r) => formatCurrency(r.inviter_reward) },
  { title: '受邀人奖励', key: 'invitee_reward', width: 100, render: (r) => formatCurrency(r.invitee_reward) },
  { title: '状态', key: 'status', width: 80, render: (r) => h(NTag, { type: statusType(r.status), size: 'small' }, { default: () => statusText(r.status) }) },
  { title: '过期时间', key: 'expires_at', width: 160, render: (r) => formatFullDateTime(r.expires_at) },
  { title: '创建时间', key: 'created_at', width: 160, render: (r) => formatFullDateTime(r.created_at) },
  {
    title: '操作', key: 'actions', width: 140,
    render: (r) => h(NButton.Group, null, {
      default: () => [
        h(NButton, { size: 'small', onClick: () => handleToggle(r) }, { default: () => r.is_active ? '禁用' : '启用' }),
        h(NButton, { size: 'small', type: 'error', onClick: () => handleDelete(r) }, { default: () => '删除' }),
      ]
    })
  }
]

const relColumns = [
  { title: '邀请人', key: 'inviter_username', width: 120 },
  { title: '受邀人', key: 'invitee_username', width: 120 },
  { title: '邀请码', key: 'invite_code', width: 100, render: (r) => h('span', { style: 'font-family:monospace' }, r.invite_code) },
  { title: '邀请人奖励', key: 'inviter_reward_amount', width: 110, render: (r) => h('span', null, [formatCurrency(r.inviter_reward_amount) + ' ', h(NTag, { type: r.inviter_reward_given ? 'success' : 'default', size: 'tiny', bordered: false }, { default: () => r.inviter_reward_given ? '已发' : '未发' })]) },
  { title: '受邀人奖励', key: 'invitee_reward_amount', width: 110, render: (r) => h('span', null, [formatCurrency(r.invitee_reward_amount) + ' ', h(NTag, { type: r.invitee_reward_given ? 'success' : 'default', size: 'tiny', bordered: false }, { default: () => r.invitee_reward_given ? '已发' : '未发' })]) },
  { title: '时间', key: 'created_at', width: 160, render: (r) => formatFullDateTime(r.created_at) },
]

onMounted(() => { fetchStats(); fetchCodes(); fetchRelations() })

// KeepAlive 缓存激活时静默刷新数据（不清 loading 遮罩、不重置分页）
onActivated(() => { fetchStats(); fetchCodes(); fetchRelations() })
</script>

<style scoped>
.stats-row { display: flex; gap: 24px; flex-wrap: wrap; padding: 12px 0; }
.stat-item { display: flex; flex-direction: column; align-items: center; min-width: 80px; }
.stat-val { font-size: 22px; font-weight: 700; color: var(--text-color, #333); }
.stat-lbl { font-size: 12px; color: var(--text-color-secondary, #999); margin-top: 2px; }

/* 分页：桌面靠右，手机居中（App 里分页居中更自然） */
.list-pagination { margin-top: 16px; justify-content: flex-end; }

/* 手机端工具条：标题 + 搜索框各占一行，按钮自带一行并允许换行（不会横向溢出） */
/* 卡片内容区在手机端无左右内边距，工具条不再用全局的 -12px 出血，避免越过卡片边界 */
.invites-toolbar { display: flex; flex-direction: column; gap: 8px; margin-left: 0; margin-right: 0; }
.invites-toolbar__title { font-size: 16px; font-weight: 650; color: var(--text-color, #333); }
.invites-toolbar__actions { display: flex; flex-wrap: wrap; gap: 8px; }
.invites-toolbar__actions .n-button { flex: 1 1 0; min-width: 0; }
.invites-toolbar :deep(.n-input) { width: 100%; min-width: 0; }

@media (max-width: 767px) {
  /* 手机端左右留白与底部批量栏空间都由全局统一给（见 mobile-app-ui.css /
     admin-mobile.css），页面不再自己加内边距 */
  .stats-row { gap: 12px; justify-content: space-around; padding: 4px 0; }
  .stat-val { font-size: 18px; }
  .stat-item { min-width: 60px; }
  .list-pagination { justify-content: center; }
  /* 全局 .admin-page-shell .mobile-card 用了 padding:0 !important，
     这里补回左侧复选框的位置，并让选中态可见 */
  .admin-invites-page :deep(.mobile-card.is-selectable) { padding-left: 44px !important; }
  .admin-invites-page :deep(.mobile-card.is-selected) {
    border-color: var(--primary-color, #4f46e5) !important;
    background: color-mix(in srgb, var(--primary-color-soft, rgba(102, 126, 234, 0.08)) 70%, var(--bg-color, #fff)) !important;
  }
}
</style>
