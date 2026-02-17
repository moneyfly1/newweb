<template>
  <div class="invite-page">
    <n-space vertical :size="20">
      <!-- 奖励信息横幅 -->
      <n-alert v-if="rewardInfo" type="info" :bordered="false">
        <template #icon>
          <n-icon size="20">
            <svg viewBox="0 0 24 24"><path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>
          </n-icon>
        </template>
        邀请好友注册，您将获得 <strong>¥{{ rewardInfo.inviter_reward }}</strong> 奖励，好友将获得 <strong>¥{{ rewardInfo.invitee_reward }}</strong> 奖励
      </n-alert>

      <!-- 统计卡片 - 4个 -->
      <n-grid cols="2 m:4" :x-gap="16" :y-gap="16">
        <n-grid-item>
          <n-card :bordered="false" class="stat-card stat-card-1">
            <n-statistic label="累计邀请人数" :value="stats.total_invites || 0">
              <template #prefix>
                <n-icon size="24" color="#18a058">
                  <svg viewBox="0 0 24 24"><path fill="currentColor" d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/></svg>
                </n-icon>
              </template>
              <template #suffix>人</template>
            </n-statistic>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card :bordered="false" class="stat-card stat-card-2">
            <n-statistic label="已注册人数" :value="stats.registered_invites || 0">
              <template #prefix>
                <n-icon size="24" color="#2080f0">
                  <svg viewBox="0 0 24 24"><path fill="currentColor" d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/></svg>
                </n-icon>
              </template>
              <template #suffix>人</template>
            </n-statistic>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card :bordered="false" class="stat-card stat-card-3">
            <n-statistic label="已购买人数" :value="stats.purchased_invites || 0">
              <template #prefix>
                <n-icon size="24" color="#f0a020">
                  <svg viewBox="0 0 24 24"><path fill="currentColor" d="M7 18c-1.1 0-1.99.9-1.99 2S5.9 22 7 22s2-.9 2-2-.9-2-2-2zM1 2v2h2l3.6 7.59-1.35 2.45c-.16.28-.25.61-.25.96 0 1.1.9 2 2 2h12v-2H7.42c-.14 0-.25-.11-.25-.25l.03-.12.9-1.63h7.45c.75 0 1.41-.41 1.75-1.03l3.58-6.49c.08-.14.12-.31.12-.48 0-.55-.45-1-1-1H5.21l-.94-2H1zm16 16c-1.1 0-1.99.9-1.99 2s.89 2 1.99 2 2-.9 2-2-.9-2-2-2z"/></svg>
                </n-icon>
              </template>
              <template #suffix>人</template>
            </n-statistic>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card :bordered="false" class="stat-card stat-card-4">
            <n-statistic label="累计获得奖励" :value="stats.total_reward || 0">
              <template #prefix>
                <n-icon size="24" color="#d03050">
                  <svg viewBox="0 0 24 24"><path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1.41 16.09V20h-2.67v-1.93c-1.71-.36-3.16-1.46-3.27-3.4h1.96c.1 1.05.82 1.87 2.65 1.87 1.96 0 2.4-.98 2.4-1.59 0-.83-.44-1.61-2.67-2.14-2.48-.6-4.18-1.62-4.18-3.67 0-1.72 1.39-2.84 3.11-3.21V4h2.67v1.95c1.86.45 2.79 1.86 2.85 3.39H14.3c-.05-1.11-.64-1.87-2.22-1.87-1.5 0-2.4.68-2.4 1.64 0 .84.65 1.39 2.67 1.91s4.18 1.39 4.18 3.91c-.01 1.83-1.38 2.83-3.12 3.16z"/></svg>
                </n-icon>
              </template>
              <template #suffix>元</template>
            </n-statistic>
          </n-card>
        </n-grid-item>
      </n-grid>

      <!-- 邀请码列表 -->
      <n-card title="我的邀请码" :bordered="false">
        <template #header-extra>
          <n-button type="primary" @click="showCreateModal = true">
            <template #icon>
              <n-icon><svg viewBox="0 0 24 24"><path fill="currentColor" d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/></svg></n-icon>
            </template>
            生成邀请码
          </n-button>
        </template>

        <n-spin :show="loading">
          <n-empty v-if="!loading && inviteCodes.length === 0" description="暂无邀请码">
            <template #extra>
              <n-button type="primary" @click="showCreateModal = true">立即生成</n-button>
            </template>
          </n-empty>

          <template v-else>
            <n-data-table v-if="!appStore.isMobile" :columns="columns" :data="inviteCodes" :bordered="false" :single-line="false" />
            <div v-else class="mobile-card-list">
              <div v-for="code in inviteCodes" :key="code.id" class="mobile-card">
                <div class="card-row">
                  <span class="label">邀请码</span>
                  <span class="value" style="font-family: monospace; font-weight: 600;">{{ code.code }}</span>
                </div>
                <div class="card-row">
                  <span class="label">使用</span>
                  <span class="value">{{ code.used_count }} / {{ code.max_uses }}</span>
                </div>
                <div class="card-row">
                  <span class="label">奖励</span>
                  <span class="value">邀请人 ¥{{ code.inviter_reward }} / 受邀人 ¥{{ code.invitee_reward }}</span>
                </div>
                <div class="card-row">
                  <span class="label">状态</span>
                  <span class="value">
                    <n-tag :type="code.status === 'active' ? 'success' : code.status === 'expired' ? 'warning' : 'default'" size="small" :bordered="false">
                      {{ code.status === 'active' ? '有效' : code.status === 'expired' ? '已过期' : '已用完' }}
                    </n-tag>
                  </span>
                </div>
                <div class="card-actions">
                  <n-button size="small" type="primary" @click="copyToClipboard(getInviteLink(code.code))">复制链接</n-button>
                  <n-button size="small" @click="copyToClipboard(code.code)">复制码</n-button>
                  <n-button size="small" type="error" @click="handleDelete(code)">删除</n-button>
                </div>
              </div>
            </div>
          </template>
        </n-spin>
      </n-card>

      <!-- 最近邀请记录 -->
      <n-card title="最近邀请记录" :bordered="false">
        <n-spin :show="loadingRecent">
          <n-empty v-if="!loadingRecent && recentInvites.length === 0" description="暂无邀请记录" />

          <template v-else>
            <n-data-table v-if="!appStore.isMobile" :columns="recentColumns" :data="recentInvites" :bordered="false" :single-line="false" />
            <div v-else class="mobile-card-list">
              <div v-for="invite in recentInvites" :key="invite.id" class="mobile-card">
                <div class="card-row">
                  <span class="label">用户</span>
                  <span class="value" style="font-weight: 500;">{{ invite.invitee_username }}</span>
                </div>
                <div class="card-row">
                  <span class="label">注册时间</span>
                  <span class="value">{{ new Date(invite.registered_at).toLocaleString('zh-CN') }}</span>
                </div>
                <div class="card-row">
                  <span class="label">购买</span>
                  <span class="value">
                    <n-tag :type="invite.has_purchased ? 'success' : 'default'" size="small" :bordered="false">
                      {{ invite.has_purchased ? '已购买' : '未购买' }}
                    </n-tag>
                  </span>
                </div>
                <div class="card-row">
                  <span class="label">奖励</span>
                  <span class="value" style="color: #f0a020; font-weight: 500;">¥{{ invite.reward_amount.toFixed(2) }}</span>
                </div>
              </div>
            </div>
          </template>
        </n-spin>
      </n-card>
    </n-space>

    <!-- 创建邀请码弹窗 -->
    <n-modal v-model:show="showCreateModal" preset="card" title="生成邀请码" style="width: 500px">
      <n-form ref="formRef" :model="formData" :rules="rules" label-placement="left" label-width="120">
        <n-form-item label="最大使用次数" path="max_uses">
          <n-input-number
            v-model:value="formData.max_uses"
            :min="1"
            :max="100"
            placeholder="请输入最大使用次数"
            style="width: 100%"
          />
        </n-form-item>

        <n-form-item label="有效期（天）" path="expires_in_days">
          <n-input-number
            v-model:value="formData.expires_in_days"
            :min="1"
            :max="365"
            placeholder="请输入有效天数"
            style="width: 100%"
          />
        </n-form-item>

        <n-form-item label="邀请人奖励" path="inviter_reward">
          <n-input-number
            v-model:value="formData.inviter_reward"
            :min="0"
            :precision="2"
            placeholder="邀请人获得的奖励金额"
            style="width: 100%"
          >
            <template #suffix>元</template>
          </n-input-number>
        </n-form-item>

        <n-form-item label="受邀人奖励" path="invitee_reward">
          <n-input-number
            v-model:value="formData.invitee_reward"
            :min="0"
            :precision="2"
            placeholder="受邀人获得的奖励金额"
            style="width: 100%"
          >
            <template #suffix>元</template>
          </n-input-number>
        </n-form-item>
      </n-form>

      <template #footer>
        <n-space justify="end">
          <n-button @click="showCreateModal = false">取消</n-button>
          <n-button type="primary" @click="handleCreate" :loading="creating">生成</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="tsx">
import { ref, h, onMounted, computed } from 'vue'
import { NButton, NSpace, NTag, NTime, useMessage, useDialog } from 'naive-ui'
import { listInviteCodes, createInviteCode, getInviteStats, deleteInviteCode, getPublicConfig } from '@/api/common'
import { useAppStore } from '@/stores/app'
import { copyToClipboard as clipboardCopy } from '@/utils/clipboard'

interface InviteCode {
  id: number
  code: string
  max_uses: number
  used_count: number
  expires_at: string
  status: string
  inviter_reward: number
  invitee_reward: number
  created_at: string
}

interface RecentInvite {
  id: number
  invitee_username: string
  invitee_email: string
  registered_at: string
  has_purchased: boolean
  consumption_amount: number
  reward_status: string
  reward_amount: number
}

interface Stats {
  total_invites: number
  registered_invites: number
  purchased_invites: number
  total_reward: number
  recent_invites: RecentInvite[]
}

const message = useMessage()
const dialog = useDialog()
const appStore = useAppStore()
const loading = ref(false)
const loadingRecent = ref(false)
const creating = ref(false)
const showCreateModal = ref(false)
const inviteCodes = ref<InviteCode[]>([])
const recentInvites = ref<RecentInvite[]>([])
const stats = ref<Stats>({
  total_invites: 0,
  registered_invites: 0,
  purchased_invites: 0,
  total_reward: 0,
  recent_invites: []
})
const siteUrl = ref('')

const formRef = ref()
const formData = ref({
  max_uses: 10,
  expires_in_days: 30,
  inviter_reward: 5,
  invitee_reward: 5
})

const rules = {
  max_uses: { required: true, type: 'number', message: '请输入最大使用次数', trigger: 'blur' },
  expires_in_days: { required: true, type: 'number', message: '请输入有效天数', trigger: 'blur' },
  inviter_reward: { required: true, type: 'number', message: '请输入邀请人奖励', trigger: 'blur' },
  invitee_reward: { required: true, type: 'number', message: '请输入受邀人奖励', trigger: 'blur' }
}

// 计算奖励信息
const rewardInfo = computed(() => {
  if (inviteCodes.value.length > 0) {
    const firstCode = inviteCodes.value[0]
    return {
      inviter_reward: firstCode.inviter_reward,
      invitee_reward: firstCode.invitee_reward
    }
  }
  return null
})

const copyToClipboard = async (text: string) => {
  const ok = await clipboardCopy(text)
  ok ? message.success('复制成功') : message.error('复制失败，请手动复制')
}

const getInviteLink = (code: string) => {
  const baseUrl = siteUrl.value || window.location.origin
  return `${baseUrl}/register?code=${code}`
}

const handleDelete = (row: InviteCode) => {
  dialog.warning({
    title: '确认删除',
    content: `确定要删除邀请码 ${row.code} 吗？此操作不可恢复。`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteInviteCode(row.id)
        message.success('删除成功')
        await fetchInviteCodes()
        await fetchStats()
      } catch (error: any) {
        message.error(error.message || '删除失败')
      }
    }
  })
}

const columns = [
  {
    title: '邀请链接',
    key: 'link',
    width: 300,
    resizable: true,
    render: (row: InviteCode) => {
      const link = getInviteLink(row.code)
      return h(
        NSpace,
        { vertical: true, size: 4 },
        {
          default: () => [
            h('span', { style: 'font-size: 12px; color: #999' }, '邀请码:'),
            h(
              NSpace,
              { align: 'center', size: 8 },
              {
                default: () => [
                  h('span', { style: 'font-family: monospace; font-weight: 600' }, row.code),
                  h(
                    NButton,
                    {
                      size: 'tiny',
                      secondary: true,
                      onClick: () => copyToClipboard(row.code)
                    },
                    { default: () => '复制码' }
                  )
                ]
              }
            ),
            h('span', { style: 'font-size: 12px; color: #999; margin-top: 4px' }, '完整链接:'),
            h(
              NSpace,
              { align: 'center', size: 8 },
              {
                default: () => [
                  h('span', {
                    style: 'font-size: 12px; color: #2080f0; max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap'
                  }, link),
                  h(
                    NButton,
                    {
                      size: 'tiny',
                      type: 'primary',
                      secondary: true,
                      onClick: () => copyToClipboard(link)
                    },
                    { default: () => '复制链接' }
                  )
                ]
              }
            )
          ]
        }
      )
    }
  },
  {
    title: '使用情况',
    key: 'usage',
    width: 120,
    resizable: true,
    render: (row: InviteCode) => {
      return h('span', `${row.used_count} / ${row.max_uses}`)
    }
  },
  {
    title: '奖励',
    key: 'reward',
    width: 150,
    resizable: true,
    render: (row: InviteCode) => {
      return h(
        NSpace,
        { size: 4, vertical: true },
        {
          default: () => [
            h('span', { style: 'font-size: 12px' }, `邀请人: ¥${row.inviter_reward}`),
            h('span', { style: 'font-size: 12px' }, `受邀人: ¥${row.invitee_reward}`)
          ]
        }
      )
    }
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    resizable: true,
    render: (row: InviteCode) => {
      const statusMap: Record<string, { type: any; text: string }> = {
        active: { type: 'success', text: '有效' },
        expired: { type: 'warning', text: '已过期' },
        exhausted: { type: 'default', text: '已用完' }
      }
      const status = statusMap[row.status] || { type: 'default', text: row.status }
      return h(NTag, { type: status.type, size: 'small', bordered: false }, { default: () => status.text })
    }
  },
  {
    title: '过期时间',
    key: 'expires_at',
    width: 180,
    resizable: true,
    render: (row: InviteCode) => {
      return h(NTime, { time: new Date(row.expires_at), format: 'yyyy-MM-dd HH:mm' })
    }
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 180,
    resizable: true,
    render: (row: InviteCode) => {
      return h(NTime, { time: new Date(row.created_at), format: 'yyyy-MM-dd HH:mm' })
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 100,
    render: (row: InviteCode) => {
      return h(
        NButton,
        {
          size: 'small',
          type: 'error',
          secondary: true,
          onClick: () => handleDelete(row)
        },
        { default: () => '删除' }
      )
    }
  }
]

const recentColumns = [
  {
    title: '受邀用户',
    key: 'invitee_username',
    width: 150,
    resizable: true,
    render: (row: RecentInvite) => {
      return h('span', { style: 'font-weight: 500' }, row.invitee_username)
    }
  },
  {
    title: '邮箱',
    key: 'invitee_email',
    width: 200,
    resizable: true
  },
  {
    title: '注册时间',
    key: 'registered_at',
    width: 180,
    resizable: true,
    render: (row: RecentInvite) => {
      return h(NTime, { time: new Date(row.registered_at), format: 'yyyy-MM-dd HH:mm' })
    }
  },
  {
    title: '购买状态',
    key: 'has_purchased',
    width: 100,
    resizable: true,
    render: (row: RecentInvite) => {
      return h(
        NTag,
        {
          type: row.has_purchased ? 'success' : 'default',
          size: 'small',
          bordered: false
        },
        { default: () => row.has_purchased ? '已购买' : '未购买' }
      )
    }
  },
  {
    title: '消费金额',
    key: 'consumption_amount',
    width: 120,
    resizable: true,
    render: (row: RecentInvite) => {
      return h('span', `¥${(row.consumption_amount || 0).toFixed(2)}`)
    }
  },
  {
    title: '奖励状态',
    key: 'reward_status',
    width: 120,
    resizable: true,
    render: (row: RecentInvite) => {
      const statusMap: Record<string, { type: any; text: string }> = {
        pending: { type: 'warning', text: '待发放' },
        paid: { type: 'success', text: '已发放' },
        cancelled: { type: 'error', text: '已取消' }
      }
      const status = statusMap[row.reward_status] || { type: 'default', text: row.reward_status }
      return h(NTag, { type: status.type, size: 'small', bordered: false }, { default: () => status.text })
    }
  },
  {
    title: '奖励金额',
    key: 'reward_amount',
    width: 120,
    resizable: true,
    render: (row: RecentInvite) => {
      return h('span', { style: 'color: #f0a020; font-weight: 500' }, `¥${(row.reward_amount || 0).toFixed(2)}`)
    }
  }
]

const fetchInviteCodes = async () => {
  loading.value = true
  try {
    const res = await listInviteCodes()
    inviteCodes.value = res.data || []
  } catch (error: any) {
    message.error(error.message || '获取邀请码列表失败')
  } finally {
    loading.value = false
  }
}

const fetchStats = async () => {
  loadingRecent.value = true
  try {
    const res = await getInviteStats()
    if (res.data) {
      stats.value = {
        total_invites: res.data.total_invites || 0,
        registered_invites: res.data.registered_invites || 0,
        purchased_invites: res.data.purchased_invites || 0,
        total_reward: res.data.total_reward || 0,
        recent_invites: res.data.recent_invites || []
      }
      recentInvites.value = res.data.recent_invites || []
    }
  } catch {
    // silently ignore
  } finally {
    loadingRecent.value = false
  }
}

const fetchSiteConfig = async () => {
  try {
    const res = await getPublicConfig()
    if (res.data && res.data.site_url) {
      siteUrl.value = res.data.site_url
    }
  } catch {
    // silently ignore
  }
}

const handleCreate = async () => {
  try {
    await formRef.value?.validate()
    creating.value = true
    await createInviteCode(formData.value)
    message.success('邀请码生成成功')
    showCreateModal.value = false
    await fetchInviteCodes()
    await fetchStats()
  } catch (error: any) {
    if (error.message) {
      message.error(error.message || '生成邀请码失败')
    }
  } finally {
    creating.value = false
  }
}

onMounted(() => {
  fetchSiteConfig()
  fetchInviteCodes()
  fetchStats()
})
</script>

<style scoped>
.invite-page {
  padding: 24px;
}

.stat-card {
  color: white;
}

.stat-card-1 {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.stat-card-2 {
  background: linear-gradient(135deg, #2193b0 0%, #6dd5ed 100%);
}

.stat-card-3 {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.stat-card-4 {
  background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);
}

.stat-card :deep(.n-statistic__label) {
  color: rgba(255, 255, 255, 0.9);
}

.stat-card :deep(.n-statistic__value) {
  color: white;
}

.mobile-card-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.mobile-card {
  border: 1px solid var(--border-color, #eef0f3);
  border-radius: 10px;
  padding: 14px 16px;
}

.card-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 0;
}

.card-row .label {
  font-size: 13px;
  color: var(--text-color-secondary, #999);
  flex-shrink: 0;
}

.card-row .value {
  font-size: 13px;
  color: var(--text-color, #333);
  text-align: right;
  word-break: break-all;
}

.card-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--border-color, #f0f0f0);
}

@media (max-width: 767px) {
  .invite-page { padding: 0; }
  .stat-card :deep(.n-statistic__label) { font-size: 12px; }
  .stat-card :deep(.n-statistic-value__content) { font-size: 20px; }
}
</style>
