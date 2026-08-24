import { onUnmounted, ref, watch } from 'vue'

export interface UseCountUpOptions {
  /** 动画时长（ms），默认 600 */
  duration?: number
  /** 显示小数位；缺省时按目标值自动判断（整数 → 0 位，否则 2 位） */
  decimals?: number
}

const easeOutCubic = (t: number) => 1 - Math.pow(1 - t, 3)

function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined') return false
  return window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false
}

/**
 * useCountUp —— 轻量数字滚动（原生 requestAnimationFrame，零依赖）。
 * 监听 target 变化：首次从 0 滚动到目标值，之后从当前显示值滚动到新目标值。
 * 遵守 prefers-reduced-motion：用户要求减少动效时直接跳到目标值。
 */
export function useCountUp(target: () => number, options: UseCountUpOptions = {}) {
  const duration = options.duration ?? 600
  const decimals = options.decimals

  const value = ref(0)
  let rafId = 0
  let from = 0
  let to = 0
  let startTime = 0

  const resolveDecimals = (n: number) =>
    decimals !== undefined ? decimals : Number.isInteger(n) ? 0 : 2

  const round = (n: number) => {
    const factor = Math.pow(10, resolveDecimals(to))
    return Math.round(n * factor) / factor
  }

  const animate = (now: number) => {
    const progress = Math.min((now - startTime) / duration, 1)
    const eased = easeOutCubic(progress)
    value.value = round(from + (to - from) * eased)
    if (progress < 1) {
      rafId = requestAnimationFrame(animate)
    } else {
      value.value = to
      rafId = 0
    }
  }

  const start = () => {
    cancelAnimationFrame(rafId)
    to = Number(target()) || 0
    from = value.value
    if (from === to || prefersReducedMotion()) {
      value.value = to
      return
    }
    startTime = performance.now()
    rafId = requestAnimationFrame(animate)
  }

  watch(target, start, { immediate: true })

  onUnmounted(() => cancelAnimationFrame(rafId))

  return { value, start }
}
