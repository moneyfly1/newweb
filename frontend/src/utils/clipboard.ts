/**
 * Mobile-friendly clipboard utility.
 * Uses Clipboard API with fallback for older browsers / iOS.
 */

const isMobile = /iPhone|iPad|iPod|Android/i.test(navigator.userAgent)
const isIOS = /iPhone|iPad|iPod/i.test(navigator.userAgent)

export async function copyToClipboard(text: string): Promise<boolean> {
  // Try modern Clipboard API first
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // Fall through to fallback
    }
  }

  // Fallback: textarea method (works on iOS and older Android)
  return fallbackCopy(text)
}

function fallbackCopy(text: string): boolean {
  const ta = document.createElement('textarea')
  ta.value = text
  ta.style.cssText = 'position:fixed;top:-9999px;left:-9999px;opacity:0;font-size:16px'
  document.body.appendChild(ta)

  if (isMobile) {
    ta.contentEditable = 'true'
    ta.readOnly = false
  }

  ta.focus()
  ta.select()

  if (isIOS) {
    ta.setSelectionRange(0, text.length)
  }

  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch {
    ok = false
  }

  document.body.removeChild(ta)
  return ok
}
