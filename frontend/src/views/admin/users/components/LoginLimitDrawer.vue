<template>
  <CommonDrawer
    :show="show"
    title="登录限制 / 解封"
    :width="appStore.isMobile ? '100%' : 720"
    @update:show="emit('update:show', $event)"
  >
    <div class="unlock-panel">
      <n-alert :type="status?.lockout_enabled ? 'info' : 'warning'" :show-icon="true" style="margin-bottom: 14px">
        {{ status?.lockout_note || '加载中…' }}
        <div v-if="status && !status.redis_enabled" style="margin-top: 4px; font-size: 12px">
          提示：当前未启用 Redis，IP 限流计数保存在进程内存中（每分钟自动过期），本页仍可强制清零。
        </div>
      </n-alert>

      <n-space align="center" style="margin-bottom: 14px">
        <n-input
          v-model:value="identifier"
          placeholder="邮箱或用户名（如 user@example.com）"
          style="width: 260px"
          clearable
        />
        <n-input
          v-model:value="ipAddress"
          placeholder="IP（如 1.2.3.4 / 240e:...）"
          style="width: 220px"
          clearable
        />
        <n-button type="primary" :loading="unlocking" @click="handleUnlock()">立即解封</n-button>
        <n-button secondary :loading="loading" @click="load">刷新</n-button>
      </n-space>

      <n-divider style="margin: 6px 0 12px">被锁定的账号（同一账号 + 同一 IP 失败次数超限）</n-divider>
      <n-empty v-if="!status?.locked_accounts?.length" description="当前没有被锁定的账号" size="small" />
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
          <tr v-for="item in status.locked_accounts" :key="item.username + item.ip_address">
            <td>{{ item.username }}</td>
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
            <td>{{ item.username }}</td>
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
import { ref, watch } from 'vue'
import { NAlert, NButton, NDivider, NEmpty, NInput, NSpace, NTable, NTag, useMessage } from 'naive-ui'
import { listLoginLimits, unlockLoginLimit, clearAllRateLimits } from '@/api/admin'
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
const identifier = ref('')
const ipAddress = ref('')

const formatRemaining = (seconds?: number) => {
  const s = Number(seconds || 0)
  if (!s || s <= 0) return '已过期'
  if (s < 60) return `${s} 秒`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m} 分钟`
  return `${Math.floor(m / 60)} 小时 ${m % 60} 分钟`
}

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

const handleUnlock = async (id?: string, ip?: string) => {
  const useIdentifier = id !== undefined ? id : identifier.value.trim()
  const useIp = ip !== undefined ? ip : ipAddress.value.trim()
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
    if (d.cleared_redis_keys) parts.push(`清除 Redis 限流 ${d.cleared_redis_keys} 个`)
    if (d.cleared_memory_entries) parts.push(`清除内存限流 ${d.cleared_memory_entries} 条`)
    message.success(parts.length ? `已解封：${parts.join('，')}` : '已解封（该账号当前没有被限制的记录）')
    identifier.value = ''
    ipAddress.value = ''
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
</style>
