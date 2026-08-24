<template>
  <div class="my-coupons-page">
    <div class="page-head">
      <h2 class="page-title">我的优惠券</h2>
      <p class="page-subtitle">已使用过的优惠券记录</p>
    </div>

    <n-spin :show="loading">
      <div v-if="coupons.length === 0" class="mobile-empty">暂无优惠券使用记录</div>
      <div v-else class="coupon-list">
        <div v-for="c in coupons" :key="c.id" class="coupon-card">
          <div class="coupon-amount">
            <span class="amount-symbol">¥</span>
            <span class="amount-value">{{ formatAmount(c.discount_amount) }}</span>
          </div>
          <div class="coupon-info">
            <div class="coupon-name">{{ c.coupon_name || c.code || '优惠券' }}</div>
            <div class="coupon-code" v-if="c.code">券码 {{ c.code }}</div>
            <div class="coupon-meta">
              <span v-if="c.order_no">订单 {{ c.order_no }}</span>
              <span>{{ formatTime(c.used_at) }}</span>
            </div>
          </div>
          <n-tag size="small" :type="c.coupon_status === 'active' ? 'success' : 'default'" :bordered="false">
            {{ c.coupon_status === 'active' ? '有效' : '已失效' }}
          </n-tag>
        </div>
      </div>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getMyCoupons } from '@/api/common'
import { formatAmount } from '@/utils/amount'

const loading = ref(false)
const coupons = ref<any[]>([])

const loadCoupons = async () => {
  loading.value = true
  try {
    const res: any = await getMyCoupons()
    coupons.value = res.data || []
  } catch (e: any) {
    // 静默失败（展示空态）
  } finally {
    loading.value = false
  }
}

const formatTime = (t: string) => {
  if (!t) return ''
  return new Date(t).toLocaleString('zh-CN', { hour12: false })
}

onMounted(loadCoupons)
</script>

<style scoped>
.my-coupons-page { padding: 12px; }
.page-head { margin-bottom: 14px; }
.page-title { font-size: 20px; font-weight: 700; color: var(--text-color); margin: 0; }
.page-subtitle { font-size: 13px; color: var(--text-color-secondary, #666); margin: 4px 0 0; }

.coupon-list { display: flex; flex-direction: column; gap: 10px; }
.coupon-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px;
  border-radius: 14px;
  background: var(--bg-color, #fff);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}
.coupon-amount {
  display: flex;
  align-items: baseline;
  color: var(--danger-color, #dc2626);
  flex-shrink: 0;
  min-width: 76px;
  justify-content: center;
}
.amount-symbol { font-size: 14px; font-weight: 600; }
.amount-value { font-size: 24px; font-weight: 700; }
.coupon-info { flex: 1; min-width: 0; }
.coupon-name { font-size: 14px; font-weight: 600; color: var(--text-color); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.coupon-code { font-size: 12px; color: var(--text-color-secondary, #888); margin-top: 2px; }
.coupon-meta { display: flex; gap: 10px; font-size: 11px; color: var(--text-color-secondary, #999); margin-top: 6px; }
</style>
