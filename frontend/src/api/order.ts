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
