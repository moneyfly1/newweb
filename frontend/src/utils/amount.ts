export function formatAmount(value: number | string | null | undefined): string {
  const amount = Number(value ?? 0)
  if (!Number.isFinite(amount)) return '0'
  return Number.isInteger(amount) ? String(amount) : amount.toFixed(2).replace(/\.?0+$/, '')
}

export function formatCurrency(value: number | string | null | undefined, prefix = '¥'): string {
  return `${prefix}${formatAmount(value)}`
}

/** 后台配置的「分」→ 元显示。
 *
 * 为何单独一个函数：签到奖励等设置存的是「分」，管理面板此前在模板里手写 `/100`，
 * 而后端入账漏了同样的换算（把分当元入账，1 天签到能领到 100 倍奖励）。
 * 展示与入账现在都走同一口径，模板里不再出现裸 /100。
 */
export function formatCents(cents: number | string | null | undefined, prefix = '¥'): string {
  const value = Number(cents ?? 0)
  if (!Number.isFinite(value)) return `${prefix}0`
  return formatCurrency(value / 100, prefix)
}
