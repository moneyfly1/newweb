import { reactive, ref } from 'vue'
import { useMessage } from 'naive-ui'

export interface TableFetcherParams {
  page: number
  page_size: number
  sort: string
  order: string
  [key: string]: any
}

export interface TablePageResult {
  data?: {
    items?: any[]
    total?: number
  }
}

export interface UseTableOptions {
  defaultPageSize?: number
  pageSizes?: number[]
  /** 默认排序（如节点列表按 order_index 升序） */
  defaultSort?: { sort: string; order: 'asc' | 'desc' }
  /** 额外筛选/搜索参数（响应式），每次 loadData 时合并进请求参数 */
  getParams?: () => Record<string, any>
  /** 数据加载完成后对当前页 items 的补充处理（如派生字段） */
  afterLoad?: (items: any[]) => void
}

/**
 * useTable —— 统一的后端分页表格状态管理（loading / data / 分页 / 排序 / 批量选择）。
 * 取代各 admin 页面重复手写的「tableData + pagination reactive + checkedRowKeys + loadData + fetchData + handleSorterChange + onMounted」模式。
 *
 * 用法：
 *   const { loading, tableData, checkedRowKeys, pagination, loadData, handleSorterChange } =
 *     useTable(listAnnouncements, { getParams: () => ({ status: statusFilter.value }) })
 *   onMounted(() => loadData())
 */
export function useTable(
  fetcher: (params: TableFetcherParams) => Promise<TablePageResult>,
  options: UseTableOptions = {},
) {
  const message = useMessage()
  const loading = ref(false)
  const initialLoading = ref(false) // 首载/筛选用；翻页不触发全表遮罩，避免闪烁卡顿
  const tableData = ref<any[]>([])
  const checkedRowKeys = ref<any[]>([])
  const sortState = ref(options.defaultSort || { sort: 'id', order: 'desc' })

  const pagination = reactive({
    page: 1,
    pageSize: options.defaultPageSize || 10,
    itemCount: 0,
    showSizePicker: true,
    pageSizes: options.pageSizes || [10, 20, 50, 100],
    onChange: (page: number) => {
      pagination.page = page
      loadData()
    },
    onUpdatePageSize: (pageSize: number) => {
      pagination.pageSize = pageSize
      pagination.page = 1
      loadData()
    },
  })

  // 请求序号守卫：快速切换筛选/翻页时，丢弃过期响应，防止旧数据覆盖新数据（竞态）
  let reqSeq = 0

  async function loadData() {
    const seq = ++reqSeq
    // 首次加载（tableData 为空）显示全表遮罩；翻页/刷新时保留旧数据，
    // 不触发遮罩（避免整表消失→遮罩→新表的闪烁卡顿）
    const isFirst = tableData.value.length === 0
    if (isFirst) { loading.value = true; initialLoading.value = true }
    try {
      const res = await fetcher({
        page: pagination.page,
        page_size: pagination.pageSize,
        sort: sortState.value.sort,
        order: sortState.value.order,
        ...(options.getParams ? options.getParams() : {}),
      })
      if (seq !== reqSeq) return // 已有更新的请求发出，丢弃本次结果
      tableData.value = res?.data?.items || []
      pagination.itemCount = res?.data?.total || 0
      // 数据刷新后清空勾选：翻页/筛选/搜索后旧勾选指向的行已不在当前视图，
      // 若不清空，批量操作会误伤上一页/旧筛选条件下的行
      checkedRowKeys.value = []
      if (options.afterLoad) options.afterLoad(tableData.value)
    } catch (error: any) {
      if (seq !== reqSeq) return
      message.error(error.message || '加载失败')
    } finally {
      if (seq === reqSeq) { loading.value = false; initialLoading.value = false }
    }
  }

  /** 重置到第一页并重新加载（搜索/筛选变化时调用） */
  function reload() {
    pagination.page = 1
    loadData()
  }

  function handleSorterChange(sorter: { columnKey?: string; order?: 'ascend' | 'descend' } | null) {
    if (sorter && sorter.columnKey && sorter.order) {
      sortState.value.sort = sorter.columnKey
      sortState.value.order = sorter.order === 'ascend' ? 'asc' : 'desc'
    } else {
      const d = options.defaultSort || { sort: 'id', order: 'desc' as const }
      sortState.value.sort = d.sort
      sortState.value.order = d.order
    }
    pagination.page = 1
    loadData()
  }

  function resetSelection() {
    checkedRowKeys.value = []
  }

  return { loading, initialLoading, tableData, checkedRowKeys, sortState, pagination, loadData, reload, handleSorterChange, resetSelection }
}
