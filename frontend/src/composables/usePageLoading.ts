import { ref } from 'vue'

/**
 * usePageLoading —— 统一列表加载状态（公共解决翻页卡顿）：
 * - 首次加载/筛选变化：显示全表遮罩（loading=true）
 * - 翻页/刷新：保留旧数据、不触发遮罩（loading 保持 false），
 *   新数据到达后平滑替换，消除"整表消失→遮罩→新表"的闪烁卡顿
 *
 * 用法:
 *   const { loading, setLoading, isFirstLoad } = usePageLoading()
 *   async function fetchData() {
 *     const isFirst = isFirstLoad()
 *     if (isFirst) setLoading(true)
 *     try { ... } finally { setLoading(false) }
 *   }
 */
export function usePageLoading() {
  const loading = ref(false)

  /** 判断是否为首次加载（tableData 尚未有数据）——需在 fetch 前由调用方传入当前列表长度 */
  let hadData = false

  function beginLoad(hasData: boolean) {
    if (hasData) {
      // 翻页/刷新：已有数据，不遮罩
      hadData = true
      loading.value = false
    } else {
      // 首次加载：遮罩
      hadData = false
      loading.value = true
    }
    return loading.value
  }

  function endLoad() {
    loading.value = false
  }

  function isFirstLoad() {
    return !hadData
  }

  return { loading, beginLoad, endLoad, isFirstLoad }
}
