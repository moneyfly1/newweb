import { ref } from 'vue'

/**
 * usePullRefresh —— 轻量下拉刷新（App 原生感）。
 * 在页面根容器绑定 @touchstart/@touchmove/@touchend；当滚动容器处于顶部且向下拉
 * 超过阈值时触发 onRefresh。不阻止原生滚动（避免影响滚动性能）。
 */
export function usePullRefresh(onRefresh: () => Promise<void>) {
  const distance = ref(0)
  const refreshing = ref(false)
  const startY = ref(0)
  let scrollEl: HTMLElement | null = null

  function findScrollParent(el: HTMLElement | null): HTMLElement | null {
    let cur: HTMLElement | null = el
    while (cur && cur !== document.body) {
      const style = window.getComputedStyle(cur)
      if (/(auto|scroll)/.test(style.overflowY)) return cur
      cur = cur.parentElement
    }
    return null
  }

  function atTop(): boolean {
    if (scrollEl) return scrollEl.scrollTop <= 0
    return window.scrollY <= 0
  }

  function onTouchStart(e: TouchEvent) {
    const target = e.currentTarget as HTMLElement
    if (!scrollEl) scrollEl = findScrollParent(target)
    if (atTop()) startY.value = e.touches[0].clientY
  }

  function onTouchMove(e: TouchEvent) {
    if (!startY.value || !atTop()) return
    const diff = e.touches[0].clientY - startY.value
    if (diff > 0) {
      distance.value = Math.min(diff * 0.4, 70)
    } else {
      distance.value = 0
    }
  }

  async function onTouchEnd() {
    const d = distance.value
    distance.value = 0
    startY.value = 0
    if (d >= 55 && !refreshing.value) {
      refreshing.value = true
      try {
        await onRefresh()
      } finally {
        refreshing.value = false
      }
    }
  }

  return { distance, refreshing, onTouchStart, onTouchMove, onTouchEnd }
}
