import { useNotification } from 'naive-ui'

let notification: any = null

export function initNotification() {
  notification = useNotification()
}

export function showNotification(title: string, content: string, type: 'success' | 'info' | 'warning' | 'error' = 'info') {
  if (!notification) return

  notification[type]({
    title,
    content,
    duration: 5000,
    keepAliveOnHover: true,
  })
}

// 新订单通知
export function notifyNewOrder(orderNo: string) {
  showNotification('新订单', `订单号：${orderNo}`, 'success')
}

// 新工单通知
export function notifyNewTicket(ticketId: number) {
  showNotification('新工单', `工单 #${ticketId} 需要处理`, 'warning')
}

// 系统通知
export function notifySystem(message: string) {
  showNotification('系统通知', message, 'info')
}
