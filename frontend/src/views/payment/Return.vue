<template>
  <div class="payment-return">
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
                <span style="color: #18a058; font-weight: 600;">¥{{ orderInfo.final_amount }}</span>
              </n-descriptions-item>
              <n-descriptions-item label="支付时间">{{ formatDateTime(orderInfo.paid_at) }}</n-descriptions-item>
            </n-descriptions>
            <n-space justify="center">
              <n-button @click="$router.push('/orders')">返回订单列表</n-button>
              <n-button type="primary" @click="$router.push('/subscription')">查看订阅</n-button>
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
</style>
