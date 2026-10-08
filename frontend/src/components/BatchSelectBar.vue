<!--
  BatchSelectBar —— 全站统一的「全选 + 批量操作」栏（公共组件，别在页面里另写一套）

  桌面端：渲染成列表上方的一条工具栏（全选复选框 + 已选数量 + 清空 + 操作按钮）
  手机端：同一份 DOM 通过 CSS 变成**固定在底部**的操作栏（贴着底部标签栏，
          带毛玻璃与安全区处理），符合 App 里「选中内容 → 底部浮出操作条」的习惯。

  用法（配合 @/composables/useBatchSelection）：
    const selection = useBatchSelection(() => tableData.value)

    <BatchSelectBar
      :total="selection.total.value"
      :selected-count="selection.count.value"
      :all-selected="selection.allSelected.value"
      :indeterminate="selection.indeterminate.value"
      label="节点"
      @toggle-all="selection.toggleAll"
      @clear="selection.clear"
    >
      <n-button size="small" type="error" :disabled="!selection.count.value" @click="handleBatchDelete">批量删除</n-button>
    </BatchSelectBar>

  说明：
  · total 为 0 时整条隐藏，避免空列表上挂一条没用的栏
  · 未选中时操作按钮由页面自己 disabled；组件只负责展示与「全选」交互
  · 手机端默认「始终显示」，这样用户不点任何东西也能看到有全选（用户反馈的核心问题）
-->
<template>
  <div v-if="total > 0" class="batch-select-bar" :class="{ 'batch-select-bar--empty': selectedCount === 0 }">
    <n-checkbox
      :checked="allSelected"
      :indeterminate="indeterminate"
      :disabled="disabled"
      @update:checked="emit('toggle-all')"
    >
      <span class="batch-select-bar__all">全选</span>
    </n-checkbox>

    <span class="batch-select-bar__info">
      <template v-if="selectedCount > 0">已选 {{ selectedCount }}/{{ total }}{{ label }}</template>
      <template v-else>共 {{ total }}{{ label }}</template>
    </span>

    <n-button v-if="selectedCount > 0" size="tiny" quaternary @click="emit('clear')">清空</n-button>

    <div class="batch-select-bar__actions">
      <slot :selected-count="selectedCount" :all-selected="allSelected" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { NButton, NCheckbox } from 'naive-ui'

withDefaults(
  defineProps<{
    /** 可被选中的行数 */
    total: number
    /** 已选数量 */
    selectedCount: number
    /** 是否全部选中 */
    allSelected: boolean
    /** 是否半选 */
    indeterminate?: boolean
    /** 数量单位，例如「个节点」「条订单」 */
    label?: string
    /** 整条禁用（例如加载中） */
    disabled?: boolean
  }>(),
  { indeterminate: false, label: '', disabled: false },
)

const emit = defineEmits<{
  (e: 'toggle-all'): void
  (e: 'clear'): void
}>()
</script>

<style scoped>
.batch-select-bar__all {
  font-size: 14px;
  font-weight: 500;
}
</style>
