import request from '@/utils/request'

// Dashboard
export const getAdminDashboard = () => request.get('/admin/dashboard')
export const getAdminStats = () => request.get('/admin/stats')

// Users
export const listUsers = (params?: any) => request.get('/admin/users', { params })
export const getUser = (id: number) => request.get(`/admin/users/${id}`)
export const updateUser = (id: number, data: any) => request.put(`/admin/users/${id}`, data)
export const deleteUser = (id: number) => request.delete(`/admin/users/${id}`)
export const toggleUserActive = (id: number) => request.post(`/admin/users/${id}/toggle-active`)
export const createUser = (data: any) => request.post('/admin/users', data)
export const resetUserPassword = (id: number, data: { password: string }) => request.post(`/admin/users/${id}/reset-password`, data)
export const getAbnormalUsers = (params?: any) => request.get('/admin/users/abnormal', { params })
export const loginAsUser = (id: number) => request.post(`/admin/users/${id}/login-as`)
export const deleteUserDevice = (userId: number, deviceId: number) => request.delete(`/admin/users/${userId}/devices/${deviceId}`)
export const batchUserAction = (data: { user_ids: number[], action: string, data?: any }) => request.post('/admin/users/batch-action', data)
export const exportUsersCSV = (params?: any) => request.get('/admin/users/export', { params, responseType: 'blob', timeout: 60000 } as any)
export const importUsersCSV = (data: FormData) => request.post('/admin/users/import', data, { headers: { 'Content-Type': 'multipart/form-data' } })
export const updateUserNotes = (userId: number, notes: string) => request.put(`/admin/users/${userId}/notes`, { notes })

// Orders
export const listAdminOrders = (params?: any) => request.get('/admin/orders', { params })
export const getAdminOrder = (id: number) => request.get(`/admin/orders/${id}`)
export const refundOrder = (id: number) => request.post(`/admin/orders/${id}/refund`)

// Packages
export const listAdminPackages = (params?: any) => request.get('/admin/packages', { params })
export const createPackage = (data: any) => request.post('/admin/packages', data)
export const updatePackage = (id: number, data: any) => request.put(`/admin/packages/${id}`, data)
export const deletePackage = (id: number) => request.delete(`/admin/packages/${id}`)

// Nodes
export const listAdminNodes = (params?: any) => request.get('/admin/nodes', { params })
export const createNode = (data: any) => request.post('/admin/nodes', data)
export const updateNode = (id: number, data: any) => request.put(`/admin/nodes/${id}`, data)
export const deleteNode = (id: number) => request.delete(`/admin/nodes/${id}`)

// Subscriptions
export const listAdminSubscriptions = (params?: any) => request.get('/admin/subscriptions', { params })
export const getAdminSubscription = (id: number) => request.get(`/admin/subscriptions/${id}`)
export const resetAdminSubscription = (id: number) => request.post(`/admin/subscriptions/${id}/reset`)
export const extendSubscription = (id: number, data: { days: number }) =>
  request.post(`/admin/subscriptions/${id}/extend`, data)
export const updateSubscriptionDeviceLimit = (id: number, data: any) =>
  request.put(`/admin/subscriptions/${id}`, data)
export const sendSubscriptionEmail = (id: number) =>
  request.post(`/admin/subscriptions/${id}/send-email`)
export const setSubscriptionExpireTime = (id: number, data: { expire_time: string }) =>
  request.post(`/admin/subscriptions/${id}/set-expire`, data)
export const deleteUserFull = (id: number) => request.delete(`/admin/users/${id}/full`)

// Coupons
export const listAdminCoupons = (params?: any) => request.get('/admin/coupons', { params })
export const createCoupon = (data: any) => request.post('/admin/coupons', data)
export const updateCoupon = (id: number, data: any) => request.put(`/admin/coupons/${id}`, data)
export const deleteCoupon = (id: number) => request.delete(`/admin/coupons/${id}`)

// Tickets
export const listAdminTickets = (params?: any) => request.get('/admin/tickets', { params })
export const getAdminTicket = (id: number) => request.get(`/admin/tickets/${id}`)
export const updateTicket = (id: number, data: any) => request.put(`/admin/tickets/${id}`, data)
export const replyAdminTicket = (id: number, data: { content: string }) =>
  request.post(`/admin/tickets/${id}/reply`, data)

// Settings
export const getSettings = () => request.get('/admin/settings')
export const updateSettings = (data: any) => request.put('/admin/settings', data)
export const sendTestEmail = (data: { email: string }) => request.post('/admin/settings/test-email', data)
export const testTelegram = () => request.post('/admin/settings/test-telegram')

// Announcements
export const listAnnouncements = (params?: any) => request.get('/admin/announcements', { params })
export const createAnnouncement = (data: any) => request.post('/admin/announcements', data)
export const updateAnnouncement = (id: number, data: any) => request.put(`/admin/announcements/${id}`, data)
export const deleteAnnouncement = (id: number) => request.delete(`/admin/announcements/${id}`)

// Stats
export const getRevenueStats = () => request.get('/admin/stats/revenue')
export const getUserStats = () => request.get('/admin/stats/users')
export const getRegionStats = () => request.get('/admin/stats/regions')
export const getFinancialReport = (params?: any) => request.get('/admin/stats/financial', { params })
export const exportFinancialReport = (params?: any) => request.get('/admin/stats/financial/export', { params, responseType: 'blob' })

// Logs
export const getAuditLogs = (params?: any) => request.get('/admin/logs/audit', { params })
export const getLoginLogs = (params?: any) => request.get('/admin/logs/login', { params })
export const getRegistrationLogs = (params?: any) => request.get('/admin/logs/registration', { params })
export const getSubscriptionLogs = (params?: any) => request.get('/admin/logs/subscription', { params })
export const getBalanceLogs = (params?: any) => request.get('/admin/logs/balance', { params })
export const getCommissionLogs = (params?: any) => request.get('/admin/logs/commission', { params })
export const getSystemLogs = (params?: any) => request.get('/admin/logs/system', { params })

// Monitoring
export const getMonitoring = () => request.get('/admin/monitoring')

// User Levels
export const listUserLevels = (params?: any) => request.get('/admin/user-levels', { params })
export const createUserLevel = (data: any) => request.post('/admin/user-levels', data)
export const updateUserLevel = (id: number, data: any) => request.put(`/admin/user-levels/${id}`, data)
export const deleteUserLevel = (id: number) => request.delete(`/admin/user-levels/${id}`)

// Redeem Codes
export const listRedeemCodes = (params?: any) => request.get('/admin/redeem-codes', { params })
export const createRedeemCodes = (data: any) => request.post('/admin/redeem-codes', data)
export const deleteRedeemCode = (id: number) => request.delete(`/admin/redeem-codes/${id}`)

// Email Queue
export const listEmailQueue = (params?: any) => request.get('/admin/email-queue', { params })
export const retryEmail = (id: number) => request.post(`/admin/email-queue/${id}/retry`)
export const deleteEmail = (id: number) => request.delete(`/admin/email-queue/${id}`)

// Custom Nodes
export const listCustomNodes = (params?: any) => request.get('/admin/custom-nodes', { params })
export const createCustomNode = (data: any) => request.post('/admin/custom-nodes', data)
export const updateCustomNode = (id: number, data: any) => request.put(`/admin/custom-nodes/${id}`, data)
export const deleteCustomNode = (id: number) => request.delete(`/admin/custom-nodes/${id}`)
export const assignCustomNode = (id: number, data: any) => request.post(`/admin/custom-nodes/${id}/assign`, data)
export const importCustomNodeLinks = (data: { links: string }) => request.post('/admin/custom-nodes/import-links', data)
export const batchDeleteCustomNodes = (data: { ids: number[] }) => request.post('/admin/custom-nodes/batch-delete', data)
export const getCustomNodeLink = (id: number) => request.get(`/admin/custom-nodes/${id}/link`)
export const getCustomNodeUsers = (id: number) => request.get(`/admin/custom-nodes/${id}/users`)

// Config Update
export const getConfigUpdateStatus = () => request.get('/admin/config-update/status')
export const getConfigUpdateConfig = () => request.get('/admin/config-update/config')
export const saveConfigUpdateConfig = (data: any) => request.put('/admin/config-update/config', data)
export const startConfigUpdate = () => request.post('/admin/config-update/start')
export const stopConfigUpdate = () => request.post('/admin/config-update/stop')
export const getConfigUpdateLogs = () => request.get('/admin/config-update/logs')
export const clearConfigUpdateLogs = () => request.post('/admin/config-update/logs/clear')

// Backup
export const createBackup = () => request.post('/admin/backup')
export const listBackups = () => request.get('/admin/backup')
export const getUploadStatus = (taskId: string) => request.get(`/admin/backup/upload-status/${taskId}`)
export const testGitHubConnection = (data?: any) => request.post('/admin/backup/test-github', data)

// Node Import & Test
export const importNodes = (data: any) => request.post('/admin/nodes/import', data)
export const testNode = (id: number) => request.post(`/admin/nodes/${id}/test`)

// 盲盒管理
export const listAdminMysteryBoxPools = () => request.get('/admin/mystery-box/pools')
export const createMysteryBoxPool = (data: any) => request.post('/admin/mystery-box/pools', data)
export const updateMysteryBoxPool = (id: number, data: any) => request.put(`/admin/mystery-box/pools/${id}`, data)
export const deleteMysteryBoxPool = (id: number) => request.delete(`/admin/mystery-box/pools/${id}`)
export const addMysteryBoxPrize = (poolId: number, data: any) => request.post(`/admin/mystery-box/pools/${poolId}/prizes`, data)
export const updateMysteryBoxPrize = (id: number, data: any) => request.put(`/admin/mystery-box/prizes/${id}`, data)
export const deleteMysteryBoxPrize = (id: number) => request.delete(`/admin/mystery-box/prizes/${id}`)
export const getMysteryBoxStats = () => request.get('/admin/mystery-box/stats')

// Check-in
export const getCheckInStats = () => request.get('/admin/checkin/stats')

// Invite Management
export const listAdminInviteCodes = (params?: any) => request.get('/admin/invites', { params })
export const getAdminInviteStats = () => request.get('/admin/invites/stats')
export const listAdminInviteRelations = (params?: any) => request.get('/admin/invites/relations', { params })
export const deleteAdminInviteCode = (id: number) => request.delete(`/admin/invites/${id}`)
export const toggleAdminInviteCode = (id: number) => request.post(`/admin/invites/${id}/toggle`)
