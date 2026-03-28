import { ref } from 'vue'
import { listAdminOrders } from '@/api/admin'
import { notifyNewOrder, notifyNewTicket } from './notification'

const lastOrderId = ref(0)
const lastTicketId = ref(0)
let pollingTimer: any = null

// 检查新订单
async function checkNewOrders() {
  try {
    const res = await listAdminOrders({ page: 1, page_size: 1, sort: 'id', order: 'desc' })
    if (res.code === 0 && res.data && res.data.length > 0) {
      const latestOrder = res.data[0]
      if (lastOrderId.value > 0 && latestOrder.id > lastOrderId.value) {
        notifyNewOrder(latestOrder.order_no)
      }
      lastOrderId.value = latestOrder.id
    }
  } catch {}
}

// 启动轮询
export function startNotificationPolling() {
  if (pollingTimer) return

  // 立即检查一次，初始化 lastOrderId
  checkNewOrders()

  // 每30秒检查一次
  pollingTimer = setInterval(() => {
    checkNewOrders()
  }, 30000)
}

// 停止轮询
export function stopNotificationPolling() {
  if (pollingTimer) {
    clearInterval(pollingTimer)
    pollingTimer = null
  }
}
