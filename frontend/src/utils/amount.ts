export function formatAmount(value: number | string | null | undefined): string {
  const amount = Number(value ?? 0)
  if (!Number.isFinite(amount)) return '0'
  return Number.isInteger(amount) ? String(amount) : amount.toFixed(2).replace(/\.?0+$/, '')
}

export function formatCurrency(value: number | string | null | undefined, prefix = '¥'): string {
  return `${prefix}${formatAmount(value)}`
}
