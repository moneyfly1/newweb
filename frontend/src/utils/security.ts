/**
 * Validate and perform a safe redirect to a payment URL.
 * Only allows http/https URLs and restricts to same-origin or trusted payment domains.
 */
export function safeRedirect(url: string): boolean {
  try {
    const parsed = new URL(url, window.location.origin)
    if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') {
      return false
    }

    // 允许同源重定向
    if (parsed.origin === window.location.origin) {
      window.location.href = url
      return true
    }

    // 允许已知支付网关域名
    const trustedDomains = [
      'alipay.com', 'alipaydev.com',
      'wx.tenpay.com', 'pay.weixin.qq.com',
      'paypal.com', 'sandbox.paypal.com',
      'checkout.stripe.com', 'stripe.com',
      'qr.alipay.com',
    ]
    const hostname = parsed.hostname.toLowerCase()
    const isTrusted = trustedDomains.some(d =>
      hostname === d || hostname.endsWith('.' + d)
    )

    if (isTrusted) {
      window.location.href = url
      return true
    }

    // 非信任域名，阻止重定向
    console.warn('[security] 阻止重定向到不受信任的域名:', parsed.hostname)
    return false
  } catch {
    // invalid URL
  }
  return false
}
