/**
 * 支付相关公共工具函数
 * 统一 isQrCodeUrl / isCodepayPayType / isCodepayPageUrl，消除 4 个页面的重复定义。
 */

/** 判断支付 URL 是否应作为二维码展示 */
export function isQrCodeUrl(url: string): boolean {
  if (!url) return false
  // 支付宝二维码
  if (url.includes('qr.alipay.com')) return true
  // 通用二维码链接（短链接）
  if (url.startsWith('https://qr.') && url.length < 200) return true
  // 码支付二维码（通常是短链接或包含特定关键词）
  if (url.includes('qrcode') || url.includes('qr_code')) return true
  // 微信支付二维码
  if (url.includes('wxpay') && url.startsWith('weixin://')) return true
  // 其他常见二维码模式：短链接（长度小于100）且以 http 开头
  if ((url.startsWith('http://') || url.startsWith('https://')) && url.length < 100) return true
  return false
}

/** 判断 pay_type 是否为码支付网关 */
export function isCodepayPayType(payType?: string): boolean {
  return !!payType && (payType === 'codepay' || payType.startsWith('codepay_'))
}

/** 判断支付 URL 是否为码支付的网页收银台页面 */
export function isCodepayPageUrl(url: string): boolean {
  if (!url) return false
  return url.includes('/submit.php') || url.includes('/xpay/epay/submit.php')
}
