<template>
  <CommonDrawer
    :show="show"
    title="登录限制 / 解封"
    :width="appStore.isMobile ? '100%' : 760"
    @update:show="emit('update:show', $event)"
  >
    <div class="unlock-panel">
      <n-alert :type="status?.lockout_enabled ? 'info' : 'warning'" :show-icon="true" style="margin-bottom: 14px">
        {{ status?.lockout_note || '加载中…' }}
        <div v-if="status && !status.redis_enabled" style="margin-top: 4px; font-size: 12px">
          提示：当前未启用 Redis，IP 限流计数保存在进程内存中（每分钟自动过期），本页仍可强制清零。
        </div>
      </n-alert>

      <!-- 查询客户：客服通常只拿到一个邮箱（或一个 IP），先查清状态再解封 -->
      <n-divider style="margin: 0 0 12px">查询客户（按邮箱 / 用户名 / IP）</n-divider>
      <n-space align="center" style="margin-bottom: 12px">
        <n-input
          v-model:value="queryText"
          placeholder="客户邮箱 / 用户名，或 IP（如 user@example.com / 1.2.3.4）"
          style="width: 340px"
          clearable
          @keyup.enter="handleLookup"
        />
        <n-button type="primary" :loading="looking" @click="handleLookup">查询</n-button>
      </n-space>

      <n-card v-if="lookup" size="small" :bordered="true" class="lookup-card">
        <div class="lookup-line">
          <n-tag v-if="!lookup.found" type="warning" size="small">未找到该用户</n-tag>
          <n-tag v-else-if="lookup.limited" type="error" size="small">已锁定</n-tag>
          <n-tag v-else type="success" size="small">未锁定</n-tag>
          <n-tag v-if="lookup.user && lookup.user.is_active === false" type="error" size="small">账号已禁用</n-tag>
          <span v-if="lookup.user" class="lookup-title">{{ lookup.user.username }}（{{ lookup.user.email }}）</span>
          <span v-if="lookup.limited" class="lookup-meta">
            失败 {{ lookup.fail_count }} 次 · 剩余 {{ formatRemaining(lookup.remaining_seconds) }}
          </span>
          <span v-else-if="lookup.fail_count" class="lookup-meta">窗口内失败 {{ lookup.fail_count }} 次（未达阈值）</span>
        </div>

        <div v-if="lookup.unlock_at" class="lookup-meta">自动解锁时间：{{ formatFullDateTime(lookup.unlock_at) }}</div>
        <div v-if="lookup.last_success_at" class="lookup-meta">最近一次成功登录：{{ formatFullDateTime(lookup.last_success_at) }}</div>
        <div v-if="lookup.verify_fail_count" class="lookup-meta">验证码校验失败：{{ lookup.verify_fail_count }} 次</div>

        <div v-if="lookup.source_ips && lookup.source_ips.length" class="lookup-ips">
          <span class="lookup-meta">来源 IP：</span>
          <n-tag
            v-for="ip in lookup.source_ips"
            :key="ip.ip_address"
            size="tiny"
            :type="ip.rate_limited ? 'error' : 'default'"
            style="margin-right: 6px"
          >
            {{ ip.ip_address }}（失败 {{ ip.fail_count }} 次{{ ip.rate_limited ? ' · 正被限流 ' + (ip.rate_limit_ttl || 0) + 's' : '' }}）
          </n-tag>
        </div>

        <div v-for="(note, i) in lookup.notes || []" :key="i" class="lookup-note">· {{ note }}</div>

        <n-space style="margin-top: 10px">
          <n-button
            size="small"
            type="primary"
            :disabled="!lookup.found && !(lookup.source_ips || []).length"
            :loading="unlocking"
            @click="handleUnlock(lookup.user?.email || queryText, undefined)"
          >
            立即解封{{ lookup.user ? '（' + lookup.user.email + '）' : '' }}
          </n-button>
          <n-button
            v-for="ip in (lookup.source_ips || []).filter((x: any) => x.rate_limited)"
            :key="'u' + ip.ip_address"
            size="small"
            secondary
            @click="handleUnlock('', ip.ip_address)"
          >
            解除 IP 限流 {{ ip.ip_address }}
          </n-button>
        </n-space>
      </n-card>

      <n-divider style="margin: 18px 0 12px">
        被锁定的账号（同一账号 + 同一 IP 失败次数超限）
        <n-input
          v-model:value="lockFilter"
          size="tiny"
          placeholder="按邮箱/IP 过滤"
          style="width: 180px; margin-left: 8px"
          clearable
        />
      </n-divider>
      <n-empty v-if="!filteredLocks.length" description="当前没有被锁定的账号" size="small" />
      <n-table v-else :bordered="false" size="small" class="unlock-table">
        <thead>
          <tr>
            <th>账号</th>
            <th>来源 IP</th>
            <th>失败次数</th>
            <th>剩余时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in filteredLocks" :key="item.username + item.ip_address">
            <td>
              <button type="button" class="link-btn" @click="lookupBy(item.username)">{{ item.username }}</button>
            </td>
            <td>{{ item.ip_address || '-' }}</td>
            <td>{{ item.fail_count }}</td>
            <td>{{ formatRemaining(item.remaining_seconds) }}</td>
            <td>
              <n-button size="tiny" type="primary" @click="handleUnlock(item.username, item.ip_address)">解封</n-button>
            </td>
          </tr>
        </tbody>
      </n-table>

      <n-divider style="margin: 18px 0 12px">被限流的 IP（某 IP 对某接口请求超上限）</n-divider>
      <n-empty v-if="!status?.rate_limited?.length" description="当前没有被限流的 IP" size="small" />
      <n-table v-else :bordered="false" size="small" class="unlock-table">
        <thead>
          <tr>
            <th>IP</th>
            <th>接口</th>
            <th>当前计数</th>
            <th>剩余时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in status.rate_limited" :key="item.scope + item.path + item.ip">
            <td>{{ item.ip }}</td>
            <td>
              <n-tag size="tiny" :type="item.scope === 'redis' ? 'success' : 'default'">{{ item.scope }}</n-tag>
              {{ item.path || '-' }}
            </td>
            <td>{{ item.count }}<span v-if="item.limit"> / {{ item.limit }}</span></td>
            <td>{{ formatRemaining(item.ttl_seconds) }}</td>
            <td>
              <n-button size="tiny" type="primary" @click="handleUnlock('', item.ip)">解封</n-button>
            </td>
          </tr>
        </tbody>
      </n-table>

      <n-divider style="margin: 18px 0 12px">
        近 24 小时登录失败（客户说「登不上」时先看这里）
        <n-button size="tiny" style="margin-left: 8px" :loading="clearing" @click="handleClearAllRateLimits">
          清空全部 IP 限流
        </n-button>
      </n-divider>
      <n-empty v-if="!status?.recent_failures?.length" description="近 24 小时没有登录失败记录" size="small" />
      <n-table v-else :bordered="false" size="small" class="unlock-table">
        <thead>
          <tr>
            <th>账号</th>
            <th>来源 IP</th>
            <th>失败次数</th>
            <th>最近一次</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in status.recent_failures" :key="'f' + item.username + item.ip_address">
            <td>
              <button type="button" class="link-btn" @click="lookupBy(item.username)">{{ item.username }}</button>
            </td>
            <td>{{ item.ip_address || '-' }}</td>
            <td>{{ item.fail_count }}</td>
            <td>{{ formatFullDateTime(item.last_attempt) }}</td>
            <td>
              <n-button size="tiny" @click="handleUnlock(item.username, item.ip_address)">解封</n-button>
            </td>
          </tr>
        </tbody>
      </n-table>
    </div>

    <template #footer>
      <n-space justify="end">
        <n-button @click="emit('update:show', false)">关闭</n-button>
      </n-space>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  NAlert, NButton, NCard, NDivider, NEmpty, NInput, NSpace, NTable, NTag, useMessage,
} from 'naive-ui'
import { listLoginLimits, lookupLoginLimit, unlockLoginLimit, clearAllRateLimits } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { formatFullDateTime } from '@/utils/date'
import CommonDrawer from '@/components/CommonDrawer.vue'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ (e: 'update:show', v: boolean): void }>()

const appStore = useAppStore()
const message = useMessage()

const status = ref<any>(null)
const loading = ref(false)
const unlocking = ref(false)
const clearing = ref(false)
const looking = ref(false)
const queryText = ref('')
const lookup = ref<any>(null)
const lockFilter = ref('')

const formatRemaining = (seconds?: number) => {
  const s = Number(seconds || 0)
  if (!s || s <= 0) return '已过期'
  if (s < 60) return `${s} 秒`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m} 分钟`
  return `${Math.floor(m / 60)} 小时 ${m % 60} 分钟`
}

const filteredLocks = computed(() => {
  const list = status.value?.locked_accounts || []
  const q = lockFilter.value.trim().toLowerCase()
  if (!q) return list
  return list.filter((x: any) =>
    String(x.username || '').toLowerCase().includes(q) || String(x.ip_address || '').toLowerCase().includes(q))
})

const load = async () => {
  loading.value = true
  try {
    const res: any = await listLoginLimits()
    status.value = res.data || null
  } catch (e: any) {
    message.error(e?.message || '加载登录限制失败')
  } finally {
    loading.value = false
  }
}

// 查客户：邮箱/用户名/IP 三选一，先看清状态再决定解封
const handleLookup = async () => {
  const text = queryText.value.trim()
  if (!text) {
    message.warning('请输入客户邮箱、用户名或 IP')
    return
  }
  looking.value = true
  try {
    // 纯 IP 形态（含 IPv6）走 IP 查询，其余按邮箱/用户名查
    const isIP = /^[0-9a-fA-F:.]+$/.test(text) && /[.:]/.test(text)
    const res: any = await lookupLoginLimit(isIP ? { ip: text } : { identifier: text })
    lookup.value = res.data || null
  } catch (e: any) {
    message.error(e?.message || '查询失败')
  } finally {
    looking.value = false
  }
}

const lookupBy = (text: string) => {
  queryText.value = text
  handleLookup()
}

const handleUnlock = async (id?: string, ip?: string) => {
  const useIdentifier = (id || '').trim()
  const useIp = (ip || '').trim()
  if (!useIdentifier && !useIp) {
    message.warning('请填写邮箱/用户名或 IP')
    return
  }
  unlocking.value = true
  try {
    const res: any = await unlockLoginLimit({
      identifier: useIdentifier || undefined,
      ip_address: useIp || undefined,
    })
    const d = res.data || {}
    const parts: string[] = []
    if (d.deleted_login_attempts) parts.push(`清除失败记录 ${d.deleted_login_attempts} 条`)
    if (d.deleted_verifications) parts.push(`清除验证码失败 ${d.deleted_verifications} 条`)
    if (d.cleared_redis_keys) parts.push(`清除 Redis 限流 ${d.cleared_redis_keys} 个`)
    if (d.cleared_memory_entries) parts.push(`清除内存限流 ${d.cleared_memory_entries} 条`)
    message.success(parts.length ? `已解封：${parts.join('，')}` : '已解封（该账号当前没有被限制的记录）')
    if (lookup.value) await handleLookup()
    await load()
  } catch (e: any) {
    message.error(e?.message || '解封失败')
  } finally {
    unlocking.value = false
  }
}

const handleClearAllRateLimits = async () => {
  clearing.value = true
  try {
    const res: any = await clearAllRateLimits()
    message.success(`已清空 IP 限流：Redis ${res.data?.cleared_redis_keys || 0} 个，内存 ${res.data?.cleared_memory_entries || 0} 条`)
    await load()
  } catch (e: any) {
    message.error(e?.message || '清空失败')
  } finally {
    clearing.value = false
  }
}

watch(() => props.show, (v) => { if (v) load() })
</script>

<style scoped>
.unlock-panel {
  padding-bottom: 8px;
}
.unlock-table {
  font-size: 13px;
}
.unlock-table :deep(th),
.unlock-table :deep(td) {
  padding: 6px 8px;
}
.lookup-card {
  background: var(--primary-color-soft, rgba(79, 70, 229, 0.06));
}
.lookup-line {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.lookup-title {
  font-weight: 600;
}
.lookup-meta {
  font-size: 12px;
  color: var(--text-color-secondary, #64748b);
  margin-top: 4px;
}
.lookup-ips {
  margin-top: 6px;
}
.lookup-note {
  font-size: 12px;
  color: var(--text-color-secondary, #64748b);
  margin-top: 4px;
  line-height: 1.6;
}
.link-btn {
  background: none;
  border: none;
  padding: 0;
  color: var(--primary-color, #4f46e5);
  cursor: pointer;
  font-size: 13px;
  text-align: left;
}
.link-btn:hover {
  text-decoration: underline;
}
</style>
