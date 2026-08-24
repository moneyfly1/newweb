<template>
  <div class="payment-return" :class="{ 'is-success': status === 'success' }">
    <!-- 成功庆祝：轻量 CSS confetti（纯装饰，pointer-events: none） -->
    <div v-if="status === 'success' && !loading" class="confetti" aria-hidden="true">
      <i v-for="piece in confettiPieces" :key="piece.id" class="confetti-piece" :style="piece.style" />
    </div>
    <n-card :bordered="false" style="max-width: 600px; margin: 0 auto;">
      <n-spin :show="loading">
        <n-result
          v-if="!loading"
          :status="resultStatus"
          :title="resultTitle"
          :description="resultDesc"
        >
          <template #footer>
            <n-alert v-if="status === 'success' && shouldAutoRedirect" type="success" :bordered="false" style="margin-bottom: 16px; text-align: left;">
              已成功购买 <strong>{{ orderInfo?.package_name || '套餐' }}</strong>，支付金额 <strong>¥{{ orderInfo?.final_amount }}</strong>。页面将在 {{ countdown }} 秒后自动跳转到仪表盘。
            </n-alert>
            <n-descriptions v-if="orderInfo" :column="1" bordered style="margin-bottom: 24px;">
              <n-descriptions-item label="订单号">{{ orderInfo.order_no }}</n-descriptions-item>
              <n-descriptions-item label="套餐名称">{{ orderInfo.package_name }}</n-descriptions-item>
              <n-descriptions-item label="支付金额">
                <span style="color: var(--success-color); font-weight: 600;">¥{{ orderInfo.final_amount }}</span>
              </n-descriptions-item>
              <n-descriptions-item label="支付时间">{{ formatDateTime(orderInfo.paid_at) }}</n-descriptions-item>
            </n-descriptions>
            <n-space justify="center">
              <n-button @click="$router.push('/orders')">返回订单列表</n-button>
              <n-button v-if="status === 'success'" type="primary" @click="$router.push('/subscription')">查看订阅</n-button>
              <n-button v-if="status === 'fail'" type="primary" @click="$router.push('/orders')">重新支付</n-button>
            </n-space>
          </template>
        </n-result>
        <div v-else style="min-height: 300px;" />
      </n-spin>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { getOrderStatus } from '@/api/order'

const route = useRoute()
const router = useRouter()
const message = useMessage()

const loading = ref(true)
const orderInfo = ref<any>(null)
const status = ref<'success' | 'fail' | 'pending'>('pending')
const countdown = ref(2)
let pollTimer: ReturnType<typeof setInterval> | null = null
let redirectTimer: ReturnType<typeof setInterval> | null = null
let pollCount = 0

// 成功庆祝：CSS confetti 粒子（一次性生成，纯装饰）
const confettiColors = ['#667eea', '#764ba2', '#f093fb', '#4facfe', '#43e97b', '#fa709a', '#ffd166', '#06d6a0']
const confettiPieces = Array.from({ length: 16 }, (_, i) => {
  const size = 6 + Math.random() * 5
  return {
    id: i,
    style: {
      left: `${(i * 6.25 + Math.random() * 3) % 100}%`,
      width: `${size}px`,
      height: `${size * 1.4}px`,
      background: confettiColors[i % confettiColors.length],
      animationDelay: `${(Math.random() * 0.7).toFixed(2)}s`,
      animationDuration: `${(2.4 + Math.random() * 1.2).toFixed(2)}s`,
      '--sway': `${(Math.random() * 80 - 40).toFixed(0)}px`,
    },
  }
})

const source = computed(() => route.query.source || 'purchase')
const shouldAutoRedirect = computed(() => route.query.redirect === 'dashboard')
const redirectTarget = computed(() => ({ name: 'Dashboard' as const }))

const resultStatus = computed(() => {
  if (status.value === 'success') return 'success'
  if (status.value === 'fail') return 'error'
  return 'info'
})

const resultTitle = computed(() => {
  if (status.value === 'success') return '套餐购买成功'
  if (status.value === 'fail') return '支付确认失败'
  return '系统正在确认支付结果'
})

const resultDesc = computed(() => {
  if (status.value === 'success') {
    const pkgName = orderInfo.value?.package_name || '套餐'
    return `您已成功购买 ${pkgName}，系统正在为您同步最新订阅状态${shouldAutoRedirect.value ? `，${countdown.value} 秒后将跳转到仪表盘` : ''}`
  }
  if (status.value === 'fail') return '支付未完成、已取消，或系统确认超时，请稍后重试或联系客服'
  return '已收到支付结果，正在等待系统最终确认，请稍候...'
})

const formatDateTime = (dateStr: string) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit',
  })
}

const startRedirectCountdown = () => {
  if (!shouldAutoRedirect.value) return
  if (redirectTimer) clearInterval(redirectTimer)
  redirectTimer = setInterval(() => {
    countdown.value -= 1
    if (countdown.value <= 0) {
      stopRedirectCountdown()
      router.push(redirectTarget.value)
    }
  }, 1000)
}

const stopRedirectCountdown = () => {
  if (redirectTimer) {
    clearInterval(redirectTimer)
    redirectTimer = null
  }
}

const checkOrderStatus = async () => {
  const orderNo = route.query.order_no as string
  if (!orderNo) {
    status.value = 'fail'
    loading.value = false
    return
  }

  try {
    const res = await getOrderStatus(orderNo)
    const data = res.data
    if (data?.status === 'paid') {
      status.value = 'success'
      orderInfo.value = data
      loading.value = false
      stopPolling()
      startRedirectCountdown()
    } else if (data?.status === 'cancelled' || data?.status === 'expired') {
      status.value = 'fail'
      loading.value = false
      stopPolling()
    } else {
      pollCount++
      if (pollCount >= 10) {
        status.value = 'fail'
        loading.value = false
        stopPolling()
        message.warning('系统确认超时，请稍后到订单页手动刷新查看支付结果')
      }
    }
  } catch (error: any) {
    status.value = 'fail'
    loading.value = false
    stopPolling()
    message.error(error.message || '查询订单状态失败')
  }
}

const stopPolling = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

onMounted(() => {
  checkOrderStatus()
  pollTimer = setInterval(checkOrderStatus, 3000)
})

onUnmounted(() => {
  stopPolling()
  stopRedirectCountdown()
})
</script>

<style scoped>
.payment-return {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  min-height: 60vh;
  padding-top: 60px;
}

@media (max-width: 767px) {
  .payment-return { padding: 16px; padding-top: 30px; }
}

/* ---------- 成功庆祝动效（轻量，仅 transform/opacity） ---------- */
.confetti {
  position: fixed;
  inset: 0;
  pointer-events: none;
  overflow: hidden;
  z-index: 100;
}
.confetti-piece {
  position: absolute;
  top: -20px;
  border-radius: 2px;
  opacity: 0;
  animation-name: confetti-fall;
  animation-timing-function: linear;
  animation-iteration-count: 1;
  animation-fill-mode: forwards;
}
@keyframes confetti-fall {
  0% {
    transform: translate3d(0, 0, 0) rotate(0deg);
    opacity: 1;
  }
  100% {
    transform: translate3d(var(--sway, 20px), 108vh, 0) rotate(560deg);
    opacity: 0.45;
  }
}

/* 成功图标放大淡入 */
.payment-return.is-success :deep(.n-result__icon) {
  animation: success-pop 0.5s cubic-bezier(0.22, 1, 0.36, 1) both;
}
@keyframes success-pop {
  0% {
    transform: scale(0.4);
    opacity: 0;
  }
  70% {
    transform: scale(1.08);
    opacity: 1;
  }
  100% {
    transform: scale(1);
    opacity: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .confetti {
    display: none;
  }
  .payment-return.is-success :deep(.n-result__icon) {
    animation: none;
  }
}
</style>
