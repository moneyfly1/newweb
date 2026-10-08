<template>
  <div class="my-coupons-page">
    <div class="page-head">
      <h2 class="page-title">我的优惠券</h2>
      <p class="page-subtitle">已使用过的优惠券记录</p>
    </div>

    <n-spin :show="loading">
      <div v-if="coupons.length === 0" class="mobile-empty">暂无优惠券使用记录</div>
      <div v-else class="mobile-card-list">
        <div v-for="c in coupons" :key="c.id" class="mobile-card">
          <div class="card-header">
            <span class="card-title coupon-amount">
              <span class="amount-symbol">¥</span><span class="amount-value">{{ formatAmount(c.discount_amount) }}</span>
            </span>
            <n-tag size="small" :type="c.coupon_status === 'active' ? 'success' : 'default'" :bordered="false">
              {{ c.coupon_status === 'active' ? '有效' : '已失效' }}
            </n-tag>
          </div>
          <div class="card-row">
            <span class="label">券名称</span>
            <span class="value">{{ c.coupon_name || c.code || '优惠券' }}</span>
          </div>
          <div class="card-row" v-if="c.code">
            <span class="label">券码</span>
            <span class="value coupon-code">{{ c.code }}</span>
          </div>
          <div class="card-row" v-if="c.order_no">
            <span class="label">关联订单</span>
            <span class="value coupon-code">{{ c.order_no }}</span>
          </div>
          <div class="card-row">
            <span class="label">使用时间</span>
            <span class="value">{{ formatFullDateTime(c.used_at) }}</span>
          </div>
        </div>
      </div>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getMyCoupons } from '@/api/common'
import { formatAmount } from '@/utils/amount'
import { formatFullDateTime } from '@/utils/date'

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

onMounted(loadCoupons)
</script>

<style scoped>
/* 手机端横向留白由全局统一（10px）：页面根容器不再叠加左右 padding */
.my-coupons-page { padding: 8px 0 12px; }
.page-head { margin-bottom: 12px; }
.page-title { font-size: 20px; font-weight: 700; color: var(--text-color); margin: 0; }
.page-subtitle { font-size: 13px; color: var(--text-color-secondary, #666); margin: 4px 0 0; }

/* 卡片外观走全局 .mobile-card（16px 圆角/轻阴影/纯色底），
   这里只用页面专属类名补金额排版 */
.coupon-amount { display: flex; align-items: baseline; gap: 1px; color: var(--danger-color, #dc2626); }
.amount-symbol { font-size: 14px; font-weight: 600; }
.amount-value { font-size: 22px; font-weight: 700; line-height: 1.1; }
.coupon-code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; word-break: break-all; }

@media (max-width: 767px) {
  /* 卡片自身 padding 为 0（全局列表行自带内边距），只给标题行补内边距避免金额贴边 */
  .my-coupons-page :deep(.mobile-card .card-header) { padding: 12px 12px 0; margin-bottom: 8px; }
}
</style>
