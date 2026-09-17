// 统一日期时间格式化工具（收敛 18 个视图的本地实现）
// 空值统一返回 '-'

export function formatDate(d?: string | number | Date | null): string {
  if (!d) return '-'
  const date = new Date(d)
  if (isNaN(date.getTime())) return '-'
  return date.toLocaleDateString('zh-CN')
}

export function formatDateTime(d?: string | number | Date | null): string {
  if (!d) return '-'
  const date = new Date(d)
  if (isNaN(date.getTime())) return '-'
  return date.toLocaleDateString('zh-CN') + ' ' + date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

export function formatFullDateTime(d?: string | number | Date | null): string {
  if (!d) return '-'
  const date = new Date(d)
  if (isNaN(date.getTime())) return '-'
  return date.toLocaleString('zh-CN')
}

// 紧凑型（省略年份）：用于卡片、动态列表等窄栏位
export function formatCompactDateTime(d?: string | number | Date | null): string {
  if (!d) return '-'
  const date = new Date(d)
  if (isNaN(date.getTime())) return '-'
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}
