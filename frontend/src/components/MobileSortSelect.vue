<!--
  MobileSortSelect —— 手机端的「排序」入口（公共组件）

  桌面端表格可以点表头排序，但手机端的卡片列表没有表头 —— 之前手机端根本没法排序
  （用户反馈：用户列表的「注册时间排序」有问题，其实就是手机端没有排序入口）。

  这个组件在手机端渲染一行紧凑的排序控件：
    [排序]  注册时间 ↑↓   ← 点字段名切换正序/倒序，点「排序」换字段

  用法：
    const sortOptions = [
      { label: '注册时间', value: 'created_at' },
      { label: '最后登录', value: 'last_login' },
      { label: 'ID', value: 'id' },
    ]
    <MobileSortSelect
      v-if="appStore.isMobile"
      :sort="sortState.sort"
      :order="sortState.order"
      :options="sortOptions"
      @change="handleMobileSortChange"
    />

    function handleMobileSortChange({ sort, order }) {
      sortState.value = { sort, order }   // useTable 暴露的 sortState
      pagination.page = 1
      loadData()
    }
-->
<template>
  <div class="mobile-sort-row">
    <n-dropdown trigger="click" :options="dropdownOptions" @select="handleSelect">
      <n-button size="small" secondary>
        <template #icon><n-icon :component="SwapVerticalOutline" /></template>
        {{ currentLabel || '排序' }}
      </n-button>
    </n-dropdown>
    <n-button size="small" quaternary class="mobile-sort-direction" @click="toggleOrder">
      <template #icon><n-icon :component="arrowIcon" /></template>
      {{ order === 'asc' ? '正序' : '倒序' }}
    </n-button>
  </div>
</template>

<script setup lang="ts">
import { computed, h } from 'vue'
import { NButton, NIcon } from 'naive-ui'
import { SwapVerticalOutline, ArrowUpOutline, ArrowDownOutline } from '@vicons/ionicons5'

interface SortOption {
  label: string
  value: string
}

const props = withDefaults(
  defineProps<{
    /** 当前排序字段（对应后端 sort 参数，如 created_at / last_login / id） */
    sort: string
    /** 当前排序方向 */
    order: 'asc' | 'desc'
    /** 可排序字段 */
    options: SortOption[]
  }>(),
  {},
)

const emit = defineEmits<{
  (e: 'change', payload: { sort: string; order: 'asc' | 'desc' }): void
}>()

const currentLabel = computed(() => props.options.find(o => o.value === props.sort)?.label || '')
const arrowIcon = computed(() => (props.order === 'asc' ? ArrowUpOutline : ArrowDownOutline))

const dropdownOptions = computed(() =>
  props.options.map(o => ({
    key: o.value,
    label: () => h('span', { style: 'display:inline-flex;align-items:center;gap:6px' }, [
      o.label,
      o.value === props.sort ? h('span', { style: 'font-size:12px;opacity:.6' }, props.order === 'asc' ? '（正序）' : '（倒序）') : null,
    ]),
  })),
)

function handleSelect(key: string) {
  // 换字段时保留当前方向，点击字段本身不做切换（切换方向有专门的按钮）
  emit('change', { sort: key, order: props.order })
}

function toggleOrder() {
  emit('change', { sort: props.sort, order: props.order === 'asc' ? 'desc' : 'asc' })
}
</script>

<style scoped>
.mobile-sort-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
