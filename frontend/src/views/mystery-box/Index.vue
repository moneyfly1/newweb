<template>
  <div class="mystery-box-page">
    <n-tabs v-model:value="activeTab" type="line">
      <n-tab-pane name="pools" tab="盲盒奖池">
        <!-- 玩法说明 -->
        <n-alert type="info" :bordered="false" style="margin-bottom:16px" closable>
          <template #header>盲盒玩法说明</template>
          <div class="rules-content">
            <p>1. 选择一个奖池，点击「开启盲盒」按钮，系统将从您的账户余额中扣除对应费用。</p>
            <p>2. 系统会根据奖品概率随机抽取一个奖品发放给您，每个奖品旁标注了中奖概率。</p>
            <p>3. 奖品类型说明：</p>
            <ul>
              <li><b>余额奖励</b> — 直接充入您的账户余额，可用于购买套餐或继续开盲盒。</li>
              <li><b>优惠券</b> — 获得一张优惠券码，下单时输入券码即可抵扣。请妥善保存券码。</li>
              <li><b>订阅天数</b> — 自动延长您当前订阅的到期时间。若无订阅则自动创建。</li>
              <li><b>谢谢参与</b> — 未中奖，费用不退还。</li>
            </ul>
            <p>4. 部分奖池可能有开启次数限制、等级要求或最低余额要求，请留意标签提示。</p>
            <p>5. 开启记录可在「开启记录」标签页中查看。</p>
          </div>
        </n-alert>

        <n-spin :show="loadingPools">
          <div v-if="pools.length === 0 && !loadingPools" class="mobile-empty">暂无可用奖池</div>
          <!-- 手机端：奖池走 App 卡片列表（桌面端保持三列网格） -->
          <div v-else-if="appStore.isMobile" class="mobile-card-list">
            <div v-for="pool in pools" :key="pool.id" class="mobile-card">
              <div class="card-header mb-card-head">
                <span class="card-title">{{ pool.name }}</span>
                <n-tag type="warning" size="small" :bordered="false">{{ formatAmount(pool.price) }} 元/次</n-tag>
              </div>
              <p v-if="pool.description" class="mb-pool-desc">{{ pool.description }}</p>
              <div v-if="pool.max_opens_per_day || pool.max_opens_total || pool.min_level || pool.min_balance" class="mb-pool-tags">
                <n-tag v-if="pool.max_opens_per_day" size="small" :bordered="false">每日限{{ pool.max_opens_per_day }}次</n-tag>
                <n-tag v-if="pool.max_opens_total" size="small" :bordered="false">总限{{ pool.max_opens_total }}次</n-tag>
                <n-tag v-if="pool.min_level" size="small" :bordered="false">等级≥{{ pool.min_level }}</n-tag>
                <n-tag v-if="pool.min_balance" size="small" :bordered="false">余额≥{{ pool.min_balance }}</n-tag>
              </div>
              <div v-if="pool.prizes && pool.prizes.length" class="mb-prizes">
                <div class="mb-prizes-label">奖品与概率</div>
                <div class="mb-prize-list">
                  <div v-for="prize in pool.prizes" :key="prize.id" class="mb-prize">
                    <n-tag :type="prizeTagType(prize.type)" size="small" :bordered="false">{{ prize.name }}</n-tag>
                    <span class="mb-prize-prob">{{ getPrizeProbability(pool, prize) }}</span>
                    <span class="mb-prize-meta">
                      {{ prizeTypeLabel(prize.type) }}：{{ prize.value }}{{ prize.type === 'subscription_days' ? ' 天' : ' 元' }}
                      <template v-if="prize.stock !== null && prize.stock !== undefined"> · 剩余 {{ prize.stock }} 份</template>
                    </span>
                  </div>
                </div>
              </div>
              <div class="mb-pool-actions">
                <n-button type="primary" block :loading="openingPoolId === pool.id" @click="handleOpen(pool)">
                  开启盲盒（{{ formatAmount(pool.price) }} 元）
                </n-button>
              </div>
            </div>
          </div>
          <n-grid v-else :cols="3" :x-gap="16" :y-gap="16">
            <n-gi v-for="pool in pools" :key="pool.id">
              <n-card hoverable>
                <template #header>
                  <div style="display:flex;align-items:center;justify-content:space-between">
                    <span>{{ pool.name }}</span>
                    <n-tag type="warning" size="small">{{ formatAmount(pool.price) }} 元/次</n-tag>
                  </div>
                </template>
                <p v-if="pool.description" style="color:#666;font-size:13px;margin:0 0 12px">{{ pool.description }}</p>
                <n-space :size="4" style="margin-bottom:12px" wrap>
                  <n-tag v-if="pool.max_opens_per_day" size="tiny" :bordered="false">每日限{{ pool.max_opens_per_day }}次</n-tag>
                  <n-tag v-if="pool.max_opens_total" size="tiny" :bordered="false">总限{{ pool.max_opens_total }}次</n-tag>
                  <n-tag v-if="pool.min_level" size="tiny" :bordered="false">等级≥{{ pool.min_level }}</n-tag>
                  <n-tag v-if="pool.min_balance" size="tiny" :bordered="false">余额≥{{ pool.min_balance }}</n-tag>
                </n-space>
                <div v-if="pool.prizes && pool.prizes.length" style="margin-bottom:12px">
                  <n-text depth="3" style="font-size:12px">奖品列表（点击查看详情）：</n-text>
                  <n-space :size="4" style="margin-top:4px" wrap>
                    <n-tooltip v-for="prize in pool.prizes" :key="prize.id" trigger="hover">
                      <template #trigger>
                        <n-tag :type="prizeTagType(prize.type)" size="small">
                          {{ prize.name }} ({{ getPrizeProbability(pool, prize) }})
                        </n-tag>
                      </template>
                      {{ prizeTypeLabel(prize.type) }}：{{ prize.value }}{{ prize.type === 'subscription_days' ? ' 天' : ' 元' }}
                      <span v-if="prize.stock !== null && prize.stock !== undefined"> | 剩余 {{ prize.stock }} 份</span>
                    </n-tooltip>
                  </n-space>
                </div>
                <n-button type="primary" block :loading="openingPoolId === pool.id" @click="handleOpen(pool)">
                  开启盲盒（{{ formatAmount(pool.price) }} 元）
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
          <div v-if="historyData.length === 0 && !loadingHistory" class="mobile-empty">暂无记录</div>
          <div v-else class="mobile-card-list">
            <div v-for="item in historyData" :key="item.id" class="mobile-card">
              <div class="card-header mb-card-head">
                <span class="card-title">{{ item.prize_name }}</span>
                <n-tag :type="prizeTagType(item.prize_type)" size="small" :bordered="false">{{ prizeTypeLabel(item.prize_type) }}</n-tag>
              </div>
              <div class="card-row"><span class="card-label">奖品价值</span><span>{{ item.prize_value }}</span></div>
              <div class="card-row"><span class="card-label">消费</span><span>{{ item.cost }} 元</span></div>
              <div class="card-row"><span class="card-label">时间</span><span>{{ formatDateTime(item.created_at) }}</span></div>
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
          <div v-if="formatCouponCode(prizeResult)" style="margin-top:16px;padding:12px;background:#f6ffed;border-radius:8px;border:1px solid #b7eb8f">
            <n-text depth="3" style="font-size:13px;display:block;margin-bottom:4px">优惠券码（下单时使用）</n-text>
            <n-text strong style="font-size:18px;letter-spacing:2px;font-family:monospace">{{ formatCouponCode(prizeResult) }}</n-text>
          </div>
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
import { NTag, useMessage, NTooltip } from 'naive-ui'
import { useAppStore } from '@/stores/app'
import { getMysteryBoxPools, openMysteryBox, getMysteryBoxHistory } from '@/api/common'
import { formatDateTime } from '@/utils/date'
import { formatAmount } from '@/utils/amount'

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
  page: 1, pageSize: 10, itemCount: 0, showSizePicker: true, pageSizes: [10, 20, 50, 100],
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

const formatCouponCode = (result: any) => {
  if (result?.prize_type === 'coupon' && result?.coupon_code) {
    return result.coupon_code
  }
  return ''
}

const getPrizeProbability = (pool: any, prize: any) => {
  if (!pool?.prizes?.length) return '0%'
  const totalWeight = pool.prizes.reduce((sum: number, p: any) => sum + (p.weight || 0), 0)
  if (totalWeight <= 0) return '0%'
  return ((prize.weight / totalWeight) * 100).toFixed(1) + '%'
}

const historyColumns = [
  { title: '奖品', key: 'prize_name' },
  { title: '类型', key: 'prize_type', width: 100, render: (row: any) => h(NTag, { type: prizeTagType(row.prize_type), size: 'small' }, { default: () => prizeTypeLabel(row.prize_type) }) },
  { title: '价值', key: 'prize_value', width: 100 },
  { title: '消费', key: 'cost', width: 100, render: (row: any) => `${row.cost} 元` },
  { title: '时间', key: 'created_at', width: 160, render: (row: any) => formatDateTime(row.created_at) },
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
  } catch {
    // silently ignore
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
.mystery-box-page { padding: 24px; }
.prize-animation { opacity: 0; transform: scale(0.5); transition: all 0.5s ease; }
.prize-animation.revealed { opacity: 1; transform: scale(1); }
.prize-icon { font-size: 64px; line-height: 1; }
.rules-content p { margin: 4px 0; font-size: 13px; line-height: 1.6; color: #555; }
.rules-content ul { margin: 4px 0 4px 18px; padding: 0; }
.rules-content li { font-size: 13px; line-height: 1.6; color: #555; margin: 2px 0; }
.rules-content b { color: #333; }

@media (max-width: 767px) {
  /* 手机端根容器：左右内边距必须为 0（左右留白由全局统一给 10px），只保留纵向间距。
     加 !important 是防旧层 / 后台层用高特异性规则再塞回左右内边距。 */
  .mystery-box-page { padding: 8px 0 12px !important; }

  /* naive 的 .n-spin-container 会被过宽的容器选择器（[class$="-container"]）塞进
     12px 内边距，奖池卡片每侧少 12px（实测 349px，应为 373px）。这里按页面根类名收掉。 */
  .mystery-box-page :deep(.n-spin-container) { padding: 0 !important; }

  /* 旧层 user-mobile.css 的 `.user-mobile-content .mobile-card/.n-card { !important }`
     会把契约的 16px 圆角压回 8px，这里用「页面根类名 + :deep()」拉回契约观感。 */
  .mystery-box-page :deep(.mobile-card),
  .mystery-box-page :deep(.n-card) {
    border-radius: 16px !important;
    box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04), 0 6px 16px rgba(15, 23, 42, 0.04) !important;
  }
  .mystery-box-page :deep(.mb-pool-actions .n-button) { border-radius: 10px !important; }

  /* 奖池/记录两个 Tab 是页面唯一的「工具栏」，吸顶不跟着内容滚走。
     顶部栏自己也吸顶（--mobile-header-h），工具条贴它下面 ——
     参数与全局 .app-sticky-toolbar 保持一致 */
  .mystery-box-page :deep(.n-tabs-nav) {
    position: sticky;
    top: var(--mobile-header-h, 52px);
    z-index: 9;
    padding: 2px 2px 0;
    background: color-mix(in srgb, var(--bg-color, #fff) 88%, transparent);
    backdrop-filter: saturate(180%) blur(14px);
    -webkit-backdrop-filter: saturate(180%) blur(14px);
  }

  /* 奖池卡片：只用页面专属类名补内边距/排版（只做加法，不覆盖全局属性），
     不改全局 .mobile-card / .card-header / .card-row 的 App 风格 */
  .mobile-card .mb-card-head { padding: 14px 12px 0; }
  .mb-pool-desc { margin: 0 12px 10px; font-size: 13px; line-height: 1.6; color: var(--text-color-secondary); }
  .mb-pool-tags { display: flex; flex-wrap: wrap; gap: 6px; padding: 0 12px 6px; }
  .mb-prizes { padding: 0 12px 4px; }
  .mb-prizes-label { font-size: 13px; font-weight: 600; color: var(--text-color); margin-bottom: 6px; }
  .mb-prize-list { display: flex; flex-direction: column; gap: 8px; }
  .mb-prize { display: flex; align-items: center; flex-wrap: wrap; gap: 6px; min-width: 0; }
  .mb-prize-prob { font-size: 13px; font-weight: 600; color: var(--primary-color, #4f46e5); }
  .mb-prize-meta { font-size: 12px; line-height: 1.5; color: var(--text-color-secondary); max-width: 100%; overflow-wrap: anywhere; }
  .mb-pool-actions { padding: 8px 12px 12px; }
}
</style>
