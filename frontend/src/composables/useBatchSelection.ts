/**
 * useBatchSelection —— 列表「全选 / 多选 / 批量操作」的唯一实现
 *
 * 为什么要有这个：此前每个列表页各写一套 checkedRowKeys + 全选按钮，
 * 行为不一致（有的只能单选、有的没有全选、有的全选只选当前页却不提示），
 * 手机端更是大多没有全选入口。这里统一成一份逻辑，配合公共组件
 * <BatchSelectBar /> 使用：
 *
 *   const selection = useBatchSelection(() => tableData.value)
 *   // 卡片/表格行：:checked="selection.isSelected(row)" @update:checked="() => selection.toggle(row)"
 *   // 批量操作：selection.selectedRows.value
 *   <BatchSelectBar
 *     :total="selection.total.value"
 *     :selected-count="selection.count.value"
 *     :all-selected="selection.allSelected.value"
 *     :indeterminate="selection.indeterminate.value"
 *     @toggle-all="selection.toggleAll"
 *     @clear="selection.clear"
 *   >
 *     <n-button :disabled="!selection.count.value" @click="batchDelete">批量删除</n-button>
 *   </BatchSelectBar>
 *
 * 关键点：
 *  · 默认全选「当前列表里的全部行」（含后端返回的整页数据），不是只选可见部分
 *  · 用 id 记录选择，翻页/刷新数据后仍然稳定；可选 keepOnRefresh 控制是否清理失效项
 *  · isSelectable 可排除不可选行（例如已停用/已过期的记录）
 */
import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'

export interface BatchSelectionOptions<T> {
  /** 取唯一标识，默认取 row.id */
  getId?: (row: T) => string | number
  /** 该行是否可被选中，默认全部可选 */
  isSelectable?: (row: T) => boolean
  /** 数据刷新后是否保留仍然存在的选择项，默认 true（翻页/搜索后自动丢弃已消失的） */
  pruneOnChange?: boolean
}

export interface BatchSelection<T> {
  /** 已选 id 集合（响应式，供需要自己判断的场景） */
  selectedKeys: Ref<Set<string | number>>
  /** 当前列表全部行 */
  rows: ComputedRef<T[]>
  /** 可被选中的行数（总数，用于「全选 N 项」） */
  total: ComputedRef<number>
  /** 已选数量 */
  count: ComputedRef<number>
  /** 已选行对象（批量操作直接用） */
  selectedRows: ComputedRef<T[]>
  /** 是否全部选中 */
  allSelected: ComputedRef<boolean>
  /** 是否半选（部分选中） */
  indeterminate: ComputedRef<boolean>
  isSelected: (row: T) => boolean
  toggle: (row: T) => void
  /** 单行勾选状态直接赋值（配合 n-checkbox 的 update:checked） */
  setSelected: (row: T, checked: boolean) => void
  /** 全选 / 取消全选 */
  toggleAll: () => void
  /** 只选中当前页（数据是分页时给「再选下一页」留的口子） */
  selectAll: () => void
  clear: () => void
  /** 勾选状态变化（用于批量栏的显示/隐藏动画） */
  hasSelection: ComputedRef<boolean>
}

export function useBatchSelection<T extends Record<string, any>>(
  source: Ref<T[]> | ComputedRef<T[]> | (() => T[]),
  options: BatchSelectionOptions<T> = {},
): BatchSelection<T> {
  const getId = options.getId || ((row: T) => row?.id)
  const isSelectable = options.isSelectable || (() => true)
  const pruneOnChange = options.pruneOnChange !== false

  const selectedKeys = ref<Set<string | number>>(new Set())

  const rows = computed<T[]>(() => {
    const raw = typeof source === 'function' ? source() : source.value
    return Array.isArray(raw) ? raw : []
  })

  const selectableRows = computed(() => rows.value.filter(isSelectable))
  const total = computed(() => selectableRows.value.length)
  const count = computed(() => selectedKeys.value.size)
  const hasSelection = computed(() => count.value > 0)

  const selectedRows = computed(() => rows.value.filter(r => selectedKeys.value.has(getId(r))))
  const allSelected = computed(() => total.value > 0 && selectableRows.value.every(r => selectedKeys.value.has(getId(r))))
  const indeterminate = computed(() => count.value > 0 && !allSelected.value)

  const isSelected = (row: T) => selectedKeys.value.has(getId(row))

  const setSelected = (row: T, checked: boolean) => {
    const id = getId(row)
    const next = new Set(selectedKeys.value)
    if (checked) next.add(id)
    else next.delete(id)
    selectedKeys.value = next
  }

  const toggle = (row: T) => setSelected(row, !isSelected(row))

  const selectAll = () => {
    const next = new Set(selectedKeys.value)
    for (const row of selectableRows.value) next.add(getId(row))
    selectedKeys.value = next
  }

  const clear = () => {
    selectedKeys.value = new Set()
  }

  const toggleAll = () => {
    if (allSelected.value) clear()
    else selectAll()
  }

  // 列表数据变化（翻页、搜索、删除后刷新）时，丢掉已经不存在的选择，
  // 否则「已选 3 项」里可能有 2 项已经不在列表里，批量删除会误伤。
  if (pruneOnChange) {
    watch(rows, (list) => {
      if (!selectedKeys.value.size) return
      const alive = new Set(list.map(getId))
      let changed = false
      const next = new Set<string | number>()
      for (const key of selectedKeys.value) {
        if (alive.has(key)) next.add(key)
        else changed = true
      }
      if (changed) selectedKeys.value = next
    })
  }

  return {
    selectedKeys,
    rows,
    total,
    count,
    selectedRows,
    allSelected,
    indeterminate,
    isSelected,
    toggle,
    setSelected,
    toggleAll,
    selectAll,
    clear,
    hasSelection,
  }
}

export default useBatchSelection
