/**
 * Validate and perform a safe redirect to a payment URL.
 * Only allows http/https URLs to prevent javascript: or data: protocol attacks.
 */
export function safeRedirect(url: string): boolean {
  try {
    const parsed = new URL(url, window.location.origin)
    if (parsed.protocol === 'https:' || parsed.protocol === 'http:') {
      window.location.href = url
      return true
    }
  } catch {
    // invalid URL
  }
  return false
}
