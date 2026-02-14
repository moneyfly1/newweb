import request from '@/utils/request'

export const getSubscription = () => request.get('/subscriptions/user-subscription')
export const getSubscriptionDevices = () => request.get('/subscriptions/devices')
export const deleteDevice = (id: number) => request.delete(`/subscriptions/devices/${id}`)
export const resetSubscription = () => request.post('/subscriptions/reset-subscription')
export const convertToBalance = () => request.post('/subscriptions/convert-to-balance')
export const sendSubscriptionEmail = () => request.post('/subscriptions/send-subscription-email')
