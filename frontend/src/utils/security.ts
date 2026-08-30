/**
 * Validate and perform a safe redirect to a payment URL.
 * Only allows http/https URLs and restricts to same-origin or trusted payment domains.
 * When the target is a non-trusted (e.g. self-hosted aggregator) domain, fallback
 * (if provided) is invoked so the caller can open a new tab and inform the user —
 * never silently fail the payment flow.
 */
export function safeRedirect(url: string, fallback?: () => void): boolean {
  try {
    const parsed = new URL(url, window.location.origin)
    if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') {
      fallback?.()
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

    // 非信任域名（如自建易支付/码支付聚合网关）：交给 fallback 新窗口打开 + 提示，
    // 不静默失败。若调用方未提供 fallback，则直接新窗口打开（保持支付可用）。
    console.warn('[security] 非白名单支付域名，使用新窗口打开:', parsed.hostname)
    if (fallback) {
      fallback()
    } else {
      window.open(url, '_blank', 'noopener')
    }
    return false
  } catch {
    // invalid URL
    fallback?.()
  }
  return false
}
