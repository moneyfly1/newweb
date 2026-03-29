// 虚拟滚动配置
export const VIRTUAL_SCROLL_CONFIG = {
  itemHeight: 50, // 每行高度
  buffer: 5, // 缓冲区行数
}

// 计算可见范围
export function calculateVisibleRange(
  scrollTop: number,
  containerHeight: number,
  itemHeight: number,
  totalItems: number,
  buffer: number = 5
) {
  const start = Math.max(0, Math.floor(scrollTop / itemHeight) - buffer)
  const visibleCount = Math.ceil(containerHeight / itemHeight)
  const end = Math.min(totalItems, start + visibleCount + buffer * 2)

  return { start, end }
}
