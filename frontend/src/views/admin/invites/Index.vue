<template>
  <div class="admin-invites-page">
    <n-card :title="appStore.isMobile ? undefined : '邀请码管理'" :bordered="false" class="page-card">
      <n-space vertical :size="16">
        <!-- Stats -->
        <div class="stats-row">
          <div class="stat-item">
            <span class="stat-val">{{ stats.total_codes || 0 }}</span>
            <span class="stat-lbl">总邀请码</span>
          </div>
          <div class="stat-item">
            <span class="stat-val" style="color: #18a058">{{ stats.active_codes || 0 }}</span>
            <span class="stat-lbl">有效</span>
          </div>
          <div class="stat-item">
            <span class="stat-val">{{ stats.total_invites || 0 }}</span>
            <span class="stat-lbl">邀请人数</span>
          </div>
          <div class="stat-item">
            <span class="stat-val" style="color: #f0a020">¥{{ (stats.total_inviter_reward || 0).toFixed(2) }}</span>
            <span class="stat-lbl">邀请人奖励</span>
          </div>
          <div class="stat-item">
            <span class="stat-val" style="color: #2080f0">¥{{ (stats.total_invitee_reward || 0).toFixed(2) }}</span>
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

        <!-- Mobile toolbar -->
        <div v-if="appStore.isMobile" class="mobile-toolbar">
          <div class="mobile-toolbar-title">邀请码管理</div>
          <div class="mobile-toolbar-controls">
            <n-input v-model:value="search" placeholder="搜索邀请码或用户" clearable size="small" @keyup.enter="fetchCodes">
              <template #prefix><n-icon :component="SearchOutline" /></template>
            </n-input>
            <div class="mobile-toolbar-row">
              <n-button size="small" @click="fetchCodes">搜索</n-button>
              <n-button size="small" @click="fetchCodes">
                <template #icon><n-icon :component="RefreshOutline" /></template>
                刷新
              </n-button>
            </div>
          </div>
        </div>
<!-- PLACEHOLDER_TABS -->
        <!-- Tabs -->
        <n-tabs type="line" animated>
          <n-tab-pane name="codes" tab="邀请码列表">
            <template v-if="!appStore.isMobile">
              <n-data-table :columns="codeColumns" :data="codes" :loading="loadingCodes" :pagination="false" :bordered="false" :single-line="false" />
            </template>
            <template v-else>
              <n-spin :show="loadingCodes">
                <div v-if="codes.length === 0" style="text-align:center;padding:40px;color:#999">暂无数据</div>
                <div v-else class="mobile-card-list">
                  <div v-for="code in codes" :key="code.id" class="mobile-card">
                    <div class="card-header">
                      <span class="card-title" style="font-family:monospace">{{ code.code }}</span>
                      <n-tag :type="statusType(code.status)" size="small">{{ statusText(code.status) }}</n-tag>
                    </div>
                    <div class="card-body">
                      <div class="card-row"><span class="card-label">创建者</span><span>{{ code.username }}</span></div>
                      <div class="card-row"><span class="card-label">使用</span><span>{{ code.used_count }} / {{ code.max_uses || '∞' }}</span></div>
                      <div class="card-row"><span class="card-label">邀请人奖励</span><span>¥{{ code.inviter_reward }}</span></div>
                      <div class="card-row"><span class="card-label">受邀人奖励</span><span>¥{{ code.invitee_reward }}</span></div>
                    </div>
                    <div class="card-actions">
                      <n-button size="small" @click="handleToggle(code)">{{ code.is_active ? '禁用' : '启用' }}</n-button>
                      <n-button size="small" type="error" @click="handleDelete(code)">删除</n-button>
                    </div>
                  </div>
                </div>
              </n-spin>
            </template>
            <n-pagination v-model:page="codePage" :page-count="codeTotalPages" style="margin-top:16px" @update:page="fetchCodes" />
          </n-tab-pane>

          <n-tab-pane name="relations" tab="邀请记录">
            <template v-if="!appStore.isMobile">
              <n-data-table :columns="relColumns" :data="relations" :loading="loadingRels" :pagination="false" :bordered="false" :single-line="false" />
            </template>
            <template v-else>
              <n-spin :show="loadingRels">
                <div v-if="relations.length === 0" style="text-align:center;padding:40px;color:#999">暂无数据</div>
                <div v-else class="mobile-card-list">
                  <div v-for="rel in relations" :key="rel.id" class="mobile-card">
                    <div class="card-body">
                      <div class="card-row"><span class="card-label">邀请人</span><span>{{ rel.inviter_username }}</span></div>
                      <div class="card-row"><span class="card-label">受邀人</span><span>{{ rel.invitee_username }}</span></div>
                      <div class="card-row"><span class="card-label">邀请码</span><span style="font-family:monospace">{{ rel.invite_code }}</span></div>
                      <div class="card-row"><span class="card-label">邀请人奖励</span><span>¥{{ rel.inviter_reward_amount }} {{ rel.inviter_reward_given ? '✓' : '✗' }}</span></div>
                      <div class="card-row"><span class="card-label">受邀人奖励</span><span>¥{{ rel.invitee_reward_amount }} {{ rel.invitee_reward_given ? '✓' : '✗' }}</span></div>
                      <div class="card-row"><span class="card-label">时间</span><span>{{ fmtDate(rel.created_at) }}</span></div>
                    </div>
                  </div>
                </div>
              </n-spin>
            </template>
            <n-pagination v-model:page="relPage" :page-count="relTotalPages" style="margin-top:16px" @update:page="fetchRelations" />
          </n-tab-pane>
        </n-tabs>
      </n-space>
    </n-card>
  </div>
</template>

<script setup>
import { ref, h, onMounted } from 'vue'
import { NButton, NTag, NIcon, useMessage, useDialog } from 'naive-ui'
import { SearchOutline, RefreshOutline } from '@vicons/ionicons5'
import { listAdminInviteCodes, getAdminInviteStats, listAdminInviteRelations, deleteAdminInviteCode, toggleAdminInviteCode } from '@/api/admin'
import { useAppStore } from '@/stores/app'

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

const fmtDate = (d) => d ? new Date(d).toLocaleString('zh-CN') : '-'
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

const codeColumns = [
  { title: '邀请码', key: 'code', width: 120, render: (r) => h('span', { style: 'font-family:monospace;font-weight:600' }, r.code) },
  { title: '创建者', key: 'username', width: 120 },
  { title: '使用/上限', key: 'usage', width: 100, render: (r) => `${r.used_count} / ${r.max_uses || '∞'}` },
  { title: '邀请人奖励', key: 'inviter_reward', width: 100, render: (r) => `¥${r.inviter_reward}` },
  { title: '受邀人奖励', key: 'invitee_reward', width: 100, render: (r) => `¥${r.invitee_reward}` },
  { title: '状态', key: 'status', width: 80, render: (r) => h(NTag, { type: statusType(r.status), size: 'small' }, { default: () => statusText(r.status) }) },
  { title: '过期时间', key: 'expires_at', width: 160, render: (r) => fmtDate(r.expires_at) },
  { title: '创建时间', key: 'created_at', width: 160, render: (r) => fmtDate(r.created_at) },
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
  { title: '邀请人奖励', key: 'inviter_reward_amount', width: 110, render: (r) => h('span', null, [`¥${r.inviter_reward_amount} `, h(NTag, { type: r.inviter_reward_given ? 'success' : 'default', size: 'tiny', bordered: false }, { default: () => r.inviter_reward_given ? '已发' : '未发' })]) },
  { title: '受邀人奖励', key: 'invitee_reward_amount', width: 110, render: (r) => h('span', null, [`¥${r.invitee_reward_amount} `, h(NTag, { type: r.invitee_reward_given ? 'success' : 'default', size: 'tiny', bordered: false }, { default: () => r.invitee_reward_given ? '已发' : '未发' })]) },
  { title: '时间', key: 'created_at', width: 160, render: (r) => fmtDate(r.created_at) },
]

onMounted(() => { fetchStats(); fetchCodes(); fetchRelations() })
</script>

<style scoped>
.admin-invites-page { padding: 20px; }
.page-card { border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.08); }
.stats-row { display: flex; gap: 24px; flex-wrap: wrap; padding: 12px 0; }
.stat-item { display: flex; flex-direction: column; align-items: center; min-width: 80px; }
.stat-val { font-size: 22px; font-weight: 700; color: var(--text-color, #333); }
.stat-lbl { font-size: 12px; color: var(--text-color-secondary, #999); margin-top: 2px; }
.mobile-card-list { display: flex; flex-direction: column; gap: 12px; }
.mobile-card { background: var(--bg-color, #fff); border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.08); overflow: hidden; }
.card-header { display: flex; align-items: center; justify-content: space-between; padding: 12px 14px; border-bottom: 1px solid var(--border-color, #f0f0f0); }
.card-title { font-weight: 600; font-size: 14px; color: var(--text-color, #333); }
.card-body { padding: 10px 14px; }
.card-row { display: flex; justify-content: space-between; padding: 4px 0; font-size: 13px; }
.card-row > span:last-child { color: var(--text-color, #333); }
.card-label { color: var(--text-color-secondary, #999); flex-shrink: 0; }
.card-actions { display: flex; gap: 8px; padding: 10px 14px; border-top: 1px solid var(--border-color, #f0f0f0); }
@media (max-width: 767px) {
  .admin-invites-page { padding: 8px; }
  .stats-row { gap: 12px; justify-content: space-around; }
  .stat-val { font-size: 18px; }
}
.mobile-toolbar { margin-bottom: 12px; }
.mobile-toolbar-title { font-size: 17px; font-weight: 600; margin-bottom: 10px; color: var(--text-color, #333); }
.mobile-toolbar-controls { display: flex; flex-direction: column; gap: 8px; }
.mobile-toolbar-row { display: flex; gap: 8px; align-items: center; }
</style>
