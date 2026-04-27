export function getErrorMessage(error: unknown, fallback = '操作失败'): string {
  if (error instanceof Error && error.message) {
    return error.message
  }

  if (typeof error === 'string' && error.trim()) {
    return error
  }

  if (error && typeof error === 'object') {
    const maybeMessage = (error as any).message
    if (typeof maybeMessage === 'string' && maybeMessage.trim()) {
      return maybeMessage
    }
  }

  return fallback
}

/**
 * Silently catch errors but log them in development mode
 * Use this for optional data loading that shouldn't block the UI
 */
export function silentCatch(error: unknown, context?: string): void {
  if (import.meta.env.DEV) {
    const prefix = context ? `[${context}]` : '[Silent Error]'
    console.warn(prefix, error)
  }
}
