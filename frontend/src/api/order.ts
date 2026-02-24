import request from '@/utils/request'

export const listOrders = (params?: any) => request.get('/orders', { params })
export const createOrder = (data: { package_id: number; coupon_code?: string }) =>
  request.post('/orders', data)
export const payOrder = (orderNo: string, data: { payment_method: string }) =>
  request.post(`/orders/${orderNo}/pay`, data)
export const cancelOrder = (orderNo: string) => request.post(`/orders/${orderNo}/cancel`)
export const getOrderStatus = (orderNo: string) => request.get(`/orders/${orderNo}/status`)
export const createPayment = (data: { order_id: number; payment_method_id: number }) =>
  request.post('/payment', data)
export const createCustomOrder = (data: { devices: number; months: number; coupon_code?: string }) =>
  request.post('/orders/custom', data)

/** 计算「增加设备 + 可选续期」应付金额 */
export const calcUpgradePrice = (data: { add_devices: number; extend_months?: number }) =>
  request.post<{
    price_per_device_year: number
    current_device_limit: number
    remaining_days: number
    add_devices: number
    extend_months: number
    fee_extend: number
    fee_new_devices: number
    total: number
  }>('/orders/upgrade/calc', data)

/** 创建「增加设备 + 可选续期」订单 */
export const createUpgradeOrder = (data: { add_devices: number; extend_months?: number; coupon_code?: string }) =>
  request.post('/orders/upgrade', data)
