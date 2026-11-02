import request from '@/utils/request'

const PUBLIC_CONFIG_KEY = 'public_config_cache'
const PUBLIC_CONFIG_TTL = 5 * 60 * 1000

// 站点配置变更频率极低，落盘缓存 5 分钟：
// 冷启动时省掉一次「浏览器 → CF → 源站」的往返（约 120ms），过期后后台静默刷新。
export const getPublicConfig = async () => {
  try {
    const raw = localStorage.getItem(PUBLIC_CONFIG_KEY)
    if (raw) {
      const cached = JSON.parse(raw)
      if (cached && Date.now() - cached.at < PUBLIC_CONFIG_TTL) {
        return cached.res
      }
    }
  } catch {}
  const res = await request.get('/config')
  try {
    localStorage.setItem(PUBLIC_CONFIG_KEY, JSON.stringify({ at: Date.now(), res }))
  } catch {}
  return res
}
export const listPackages = (params?: any) => request.get('/packages', { params })
export const listNotifications = (params?: any) => request.get('/notifications', { params })
export const getUnreadCount = () => request.get('/notifications/unread-count')
export const markNotificationRead = (id: number) => request.put(`/notifications/${id}/read`)
export const markAllRead = () => request.put('/notifications/read-all')
export const deleteNotification = (id: number) => request.delete(`/notifications/${id}`)
export const verifyCoupon = (data: { code: string; package_id: number }) =>
  request.post('/coupons/verify', data)
export const getMyCoupons = () => request.get('/coupons/my')
export const listInviteCodes = (params?: any) => request.get('/invites', { params })
export const createInviteCode = (data: any) => request.post('/invites', data)
export const getInviteStats = (params?: any) => request.get('/invites/stats', { params })
export const validateInviteCode = (code: string) => request.get(`/invites/validate/${code}`)
export const redeemCode = (data: { code: string }) => request.post('/redeem', data)
export const getRedeemHistory = (params?: any) => request.get('/redeem/history', { params })
export const listRechargeRecords = (params?: any) => request.get('/recharge', { params })
export const createRecharge = (data: any) => request.post('/recharge', data)
export const getRechargeStatus = (id: number) => request.get(`/recharge/${id}/status`)
export const createRechargePayment = (id: number, data: { recharge_id: number; payment_method_id: number; is_mobile?: boolean }) =>
  request.post<{
    record?: any
    transaction_id: string
    payment_url?: string
    payment_mode?: 'qrcode' | 'page' | 'redirect'
    pay_type?: string
    crypto_info?: {
      wallet_address: string
      network: string
      currency: string
      amount_usdt: string
    }
  }>(`/recharge/${id}/pay`, data)
export const cancelRecharge = (id: number) => request.post(`/recharge/${id}/cancel`)
export const getPaymentMethods = () => request.get('/payment/methods')
export const listPublicAnnouncements = () => request.get('/announcements')
export const deleteInviteCode = (id: number) => request.delete(`/invites/${id}`)

// 盲盒
export const getMysteryBoxPools = () => request.get('/mystery-box/pools')
export const openMysteryBox = (data: { pool_id: number }) => request.post('/mystery-box/open', data)
export const getMysteryBoxHistory = (params?: any) => request.get('/mystery-box/history', { params })
