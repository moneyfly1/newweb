<template>
  <div class="stats-container">
    <n-spin :show="loading">
      <n-space vertical :size="20">
        <n-card title="收入统计" :bordered="false">
          <n-grid :cols="4" :x-gap="16">
            <n-gi>
              <n-statistic label="总收入" :value="revenueStats.total_revenue">
                <template #prefix>¥</template>
              </n-statistic>
            </n-gi>
            <n-gi>
              <n-statistic label="今日收入" :value="revenueStats.today_revenue">
                <template #prefix>¥</template>
              </n-statistic>
            </n-gi>
            <n-gi>
              <n-statistic label="本月收入" :value="revenueStats.monthly_revenue">
                <template #prefix>¥</template>
              </n-statistic>
            </n-gi>
            <n-gi>
              <n-statistic label="已支付订单" :value="revenueStats.paid_orders_count" />
            </n-gi>
          </n-grid>
        </n-card>

        <n-card title="用户统计" :bordered="false">
          <n-grid :cols="4" :x-gap="16">
            <n-gi>
              <n-statistic label="总用户数" :value="userStats.total_users" />
            </n-gi>
            <n-gi>
              <n-statistic label="活跃用户" :value="userStats.active_users" />
            </n-gi>
            <n-gi>
              <n-statistic label="今日新增" :value="userStats.today_new_users" />
            </n-gi>
            <n-gi>
              <n-statistic label="付费用户" :value="userStats.paid_users" />
            </n-gi>
          </n-grid>
        </n-card>

        <n-card title="用户地区分布" :bordered="false">
          <n-spin :show="regionLoading">
            <div v-if="regionStats.length > 0">
              <div v-for="(item, index) in regionStats" :key="index" class="region-item">
                <div class="region-info">
                  <span class="region-rank">{{ index + 1 }}</span>
                  <span class="region-name">{{ item.location || '未知' }}</span>
                  <span class="region-count">{{ item.count }} 人</span>
                </div>
                <n-progress
                  type="line"
                  :percentage="Math.round((item.count / maxRegionCount) * 100)"
                  :show-indicator="false"
                  :height="8"
                  :border-radius="4"
                  :color="getRegionColor(index)"
                />
              </div>
            </div>
            <n-empty v-else description="暂无地区数据，用户登录后将自动记录" style="padding: 40px 0" />
          </n-spin>
        </n-card>
      </n-space>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { getRevenueStats, getUserStats, getRegionStats } from '@/api/admin'

const message = useMessage()
const loading = ref(false)
const regionLoading = ref(false)

const revenueStats = ref({
  total_revenue: 0,
  today_revenue: 0,
  monthly_revenue: 0,
  paid_orders_count: 0,
})

const userStats = ref({
  total_users: 0,
  active_users: 0,
  today_new_users: 0,
  paid_users: 0,
})

const regionStats = ref<Array<{ location: string; count: number }>>([])

const maxRegionCount = computed(() => {
  if (regionStats.value.length === 0) return 1
  return regionStats.value[0]?.count || 1
})

const regionColors = ['#18a058', '#2080f0', '#f0a020', '#d03050', '#8a2be2', '#36ad6a', '#4098fc', '#f2c97d', '#e88080', '#a78bfa']
const getRegionColor = (index: number) => regionColors[index % regionColors.length]

const loadRevenueStats = async () => {
  try {
    const res = await getRevenueStats()
    revenueStats.value = res.data || revenueStats.value
  } catch (error: any) {
    message.error(error.message || '加载收入统计失败')
  }
}

const loadUserStats = async () => {
  try {
    const res = await getUserStats()
    userStats.value = res.data || userStats.value
  } catch (error: any) {
    message.error(error.message || '加载用户统计失败')
  }
}

const loadRegionStats = async () => {
  regionLoading.value = true
  try {
    const res = await getRegionStats()
    regionStats.value = res.data || []
  } catch (error: any) {
    message.error(error.message || '加载地区统计失败')
  } finally {
    regionLoading.value = false
  }
}

const loadAllStats = async () => {
  loading.value = true
  try {
    await Promise.all([loadRevenueStats(), loadUserStats(), loadRegionStats()])
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadAllStats()
})
</script>

<style scoped>
.stats-container {
  padding: 20px;
}

.region-item {
  margin-bottom: 12px;
}

.region-info {
  display: flex;
  align-items: center;
  margin-bottom: 4px;
  font-size: 14px;
}

.region-rank {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #f0f0f0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  margin-right: 8px;
  color: #666;
}

.region-name {
  flex: 1;
}

.region-count {
  color: #999;
  font-size: 13px;
}

@media (max-width: 767px) {
  .stats-container { padding: 8px; }
}
</style>
