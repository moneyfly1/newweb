<template>
  <div class="mystery-box-page">
    <n-tabs v-model:value="activeTab" type="line">
      <n-tab-pane name="pools" tab="盲盒奖池">
        <n-spin :show="loadingPools">
          <div v-if="pools.length === 0 && !loadingPools" style="text-align:center;padding:40px 0;color:#999">
            暂无可用奖池
          </div>
          <n-grid :cols="appStore.isMobile ? 1 : 3" :x-gap="16" :y-gap="16" v-else>
            <n-gi v-for="pool in pools" :key="pool.id">
              <n-card hoverable>
                <template #header>
                  <div style="display:flex;align-items:center;justify-content:space-between">
                    <span>{{ pool.name }}</span>
                    <n-tag type="warning" size="small">{{ pool.price }} 元</n-tag>
                  </div>
                </template>
                <p v-if="pool.description" style="color:#666;font-size:13px;margin:0 0 12px">{{ pool.description }}</p>
                <n-space :size="4" style="margin-bottom:12px">
                  <n-tag v-if="pool.max_opens_per_day" size="tiny" :bordered="false">每日限{{ pool.max_opens_per_day }}次</n-tag>
                  <n-tag v-if="pool.max_opens_total" size="tiny" :bordered="false">总限{{ pool.max_opens_total }}次</n-tag>
                  <n-tag v-if="pool.min_level" size="tiny" :bordered="false">等级≥{{ pool.min_level }}</n-tag>
                  <n-tag v-if="pool.min_balance" size="tiny" :bordered="false">余额≥{{ pool.min_balance }}</n-tag>
                </n-space>
                <div v-if="pool.prizes && pool.prizes.length" style="margin-bottom:12px">
                  <n-text depth="3" style="font-size:12px">可能获得：</n-text>
                  <n-space :size="4" style="margin-top:4px">
                    <n-tag v-for="prize in pool.prizes" :key="prize.id" :type="prizeTagType(prize.type)" size="small">
                      {{ prize.name }}
                    </n-tag>
                  </n-space>
                </div>
                <n-button type="primary" block :loading="openingPoolId === pool.id" @click="handleOpen(pool)">
                  开启盲盒
                </n-button>
              </n-card>
            </n-gi>
          </n-grid>
        </n-spin>
      </n-tab-pane>
      <n-tab-pane name="history" tab="开启记录">
        <template v-if="!appStore.isMobile">
          <n-data-table remote :columns="historyColumns" :data="historyData" :loading="loadingHistory"
            :pagination="historyPagination" :bordered="false"
            @update:page="(p: number) => { historyPagination.page = p; loadHistory() }"
            @update:page-size="(ps: number) => { historyPagination.pageSize = ps; historyPagination.page = 1; loadHistory() }"
          />
        </template>
        <template v-else>
          <div v-if="historyData.length === 0 && !loadingHistory" style="text-align:center;padding:40px 0;color:#999">暂无记录</div>
          <div v-else class="mobile-card-list">
            <div v-for="item in historyData" :key="item.id" class="mobile-card">
              <div class="card-header">
                <span class="card-title">{{ item.prize_name }}</span>
                <n-tag :type="prizeTagType(item.prize_type)" size="small">{{ prizeTypeLabel(item.prize_type) }}</n-tag>
              </div>
              <div class="card-body">
                <div class="card-row"><span class="card-label">奖品价值</span><span>{{ item.prize_value }}</span></div>
                <div class="card-row"><span class="card-label">消费</span><span>{{ item.cost }} 元</span></div>
                <div class="card-row"><span class="card-label">时间</span><span>{{ formatDate(item.created_at) }}</span></div>
              </div>
            </div>
          </div>
        </template>
      </n-tab-pane>
    </n-tabs>

    <!-- 开启结果弹窗 -->
    <n-modal v-model:show="showResult" preset="card" title="开启结果" :style="appStore.isMobile ? 'width:90vw' : 'width:400px'" :segmented="{ content: 'soft' }">
      <div v-if="prizeResult" style="text-align:center;padding:20px 0">
        <div class="prize-animation" :class="{ revealed: prizeRevealed }">
          <div class="prize-icon">{{ prizeEmoji(prizeResult.prize_type) }}</div>
          <n-h3 style="margin:12px 0 4px">{{ prizeResult.prize_name }}</n-h3>
          <n-tag :type="prizeTagType(prizeResult.prize_type)" size="large">
            {{ prizeLabel(prizeResult) }}
          </n-tag>
          <p style="color:#999;margin-top:12px;font-size:13px">消费 {{ prizeResult.cost }} 元</p>
        </div>
      </div>
      <template #footer>
        <n-space justify="center">
          <n-button @click="showResult = false">关闭</n-button>
          <n-button type="primary" @click="showResult = false; handleOpen(lastPool!)">再来一次</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, h, onMounted } from 'vue'
import { NTag, useMessage } from 'naive-ui'
import { useAppStore } from '@/stores/app'
import { getMysteryBoxPools, openMysteryBox, getMysteryBoxHistory } from '@/api/common'

const appStore = useAppStore()
const message = useMessage()

const activeTab = ref('pools')
const loadingPools = ref(false)
const pools = ref<any[]>([])
const openingPoolId = ref<number | null>(null)
const showResult = ref(false)
const prizeResult = ref<any>(null)
const prizeRevealed = ref(false)
const lastPool = ref<any>(null)

// History
const loadingHistory = ref(false)
const historyData = ref<any[]>([])
const historyPagination = reactive({
  page: 1, pageSize: 20, itemCount: 0, showSizePicker: true, pageSizes: [20, 50, 100],
})

const prizeTagType = (type: string) => {
  const map: Record<string, any> = { balance: 'success', coupon: 'info', subscription_days: 'warning', nothing: 'default' }
  return map[type] || 'default'
}
const prizeTypeLabel = (type: string) => {
  const map: Record<string, string> = { balance: '余额', coupon: '优惠券', subscription_days: '订阅天数', nothing: '谢谢参与' }
  return map[type] || type
}
const prizeEmoji = (type: string) => {
  const map: Record<string, string> = { balance: '💰', coupon: '🎫', subscription_days: '📅', nothing: '🎭' }
  return map[type] || '🎁'
}
const prizeLabel = (result: any) => {
  if (result.prize_type === 'balance') return `+${result.prize_value} 元余额`
  if (result.prize_type === 'coupon') return `${result.prize_value} 元优惠券`
  if (result.prize_type === 'subscription_days') return `+${result.prize_value} 天订阅`
  return '谢谢参与'
}

const formatDate = (dateStr: string) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

const historyColumns = [
  { title: '奖品', key: 'prize_name' },
  { title: '类型', key: 'prize_type', width: 100, render: (row: any) => h(NTag, { type: prizeTagType(row.prize_type), size: 'small' }, { default: () => prizeTypeLabel(row.prize_type) }) },
  { title: '价值', key: 'prize_value', width: 100 },
  { title: '消费', key: 'cost', width: 100, render: (row: any) => `${row.cost} 元` },
  { title: '时间', key: 'created_at', width: 160, render: (row: any) => formatDate(row.created_at) },
]

const loadPools = async () => {
  loadingPools.value = true
  try {
    const res: any = await getMysteryBoxPools()
    pools.value = res.data || []
  } catch (e: any) {
    message.error(e.message || '加载奖池失败')
  } finally {
    loadingPools.value = false
  }
}

const loadHistory = async () => {
  loadingHistory.value = true
  try {
    const res: any = await getMysteryBoxHistory({ page: historyPagination.page, page_size: historyPagination.pageSize })
    historyData.value = res.data?.items || []
    historyPagination.itemCount = res.data?.total || 0
  } catch (e: any) {
    console.error('Failed to load history:', e)
  } finally {
    loadingHistory.value = false
  }
}

const handleOpen = async (pool: any) => {
  lastPool.value = pool
  openingPoolId.value = pool.id
  prizeRevealed.value = false
  try {
    const res: any = await openMysteryBox({ pool_id: pool.id })
    prizeResult.value = res.data
    showResult.value = true
    setTimeout(() => { prizeRevealed.value = true }, 300)
    loadPools()
    loadHistory()
  } catch (e: any) {
    message.error(e.message || '开启失败')
  } finally {
    openingPoolId.value = null
  }
}

onMounted(() => {
  loadPools()
  loadHistory()
})
</script>

<style scoped>
.mystery-box-page { padding: 20px; }
.prize-animation { opacity: 0; transform: scale(0.5); transition: all 0.5s ease; }
.prize-animation.revealed { opacity: 1; transform: scale(1); }
.prize-icon { font-size: 64px; line-height: 1; }
.mobile-card-list { display: flex; flex-direction: column; gap: 12px; }
.mobile-card { background: #fff; border-radius: 10px; box-shadow: 0 1px 4px rgba(0,0,0,0.08); overflow: hidden; }
.card-header { display: flex; align-items: center; justify-content: space-between; padding: 12px 14px; border-bottom: 1px solid #f0f0f0; }
.card-title { font-weight: 600; font-size: 14px; }
.card-body { padding: 10px 14px; }
.card-row { display: flex; justify-content: space-between; padding: 4px 0; font-size: 13px; }
.card-label { color: #999; }
@media (max-width: 767px) { .mystery-box-page { padding: 8px; } }
</style>
