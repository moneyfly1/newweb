import request from '@/utils/request'

export const getCurrentUser = () => request.get('/users/me')
export const updateProfile = (data: any) => request.put('/users/me', data)
export const changePassword = (data: { old_password: string; new_password: string }) =>
  request.post('/users/change-password', data)
export const getDashboardInfo = () => request.get('/users/dashboard-info')
export const getLoginHistory = (params?: any) => request.get('/users/login-history', { params })
export const getNotificationSettings = () => request.get('/users/notification-settings')
export const updateNotificationSettings = (data: any) => request.put('/users/notification-settings', data)
export const getPrivacySettings = () => request.get('/users/privacy-settings')
export const updatePrivacySettings = (data: any) => request.put('/users/privacy-settings', data)
export const getMyLevel = () => request.get('/users/my-level')
export const getActivities = (params?: any) => request.get('/users/activities', { params })
export const getUserDevices = () => request.get('/users/devices')
export const getSubscriptionResets = (params?: any) => request.get('/users/subscription-resets', { params })
