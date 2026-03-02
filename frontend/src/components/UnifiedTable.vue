<!-- 统一的数据表格组件（桌面端+移动端自适应） -->
<template>
  <div class="unified-table-wrapper">
    <!-- 桌面端表格 -->
    <n-data-table
      v-if="!appStore.isMobile"
      :columns="columns"
      :data="data"
      :loading="loading"
      :pagination="paginationConfig"
      :bordered="bordered"
      :single-line="singleLine"
      :scroll-x="scrollX"
      :row-key="rowKey"
      :checked-row-keys="checkedRowKeys"
      @update:checked-row-keys="handleCheck"
      @update:sorter="handleSorterChange"
    />

    <!-- 移动端卡片列表 -->
    <template v-else>
      <unified-card-list
        :data="data"
        :fields="mobileFields"
        :actions="mobileActions"
        :title="mobileTitle"
        :key-field="rowKey"
        :loading="loading"
        :empty-text="emptyText"
        :clickable="clickable"
        @click="handleRowClick"
      >
        <template v-if="$slots.mobileHeader" #header="{ item, index }">
          <slot name="mobileHeader" :item="item" :index="index" />
        </template>
        <template v-if="$slots.mobileContent" #content="{ item, index }">
          <slot name="mobileContent" :item="item" :index="index" />
        </template>
        <template v-if="$slots.mobileActions" #actions="{ item, index }">
          <slot name="mobileActions" :item="item" :index="index" />
        </template>
      </unified-card-list>

      <!-- 移动端分页 -->
      <n-pagination
        v-if="pagination && data.length > 0"
        v-model:page="currentPage"
        v-model:page-size="currentPageSize"
        :item-count="totalCount"
        :page-sizes="pageSizes"
        show-size-picker
        style="margin-top: 16px; justify-content: center"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useAppStore } from '@/stores/app'
import UnifiedCardList from './UnifiedCardList.vue'
import type { DataTableColumns } from 'naive-ui'

interface Props {
  columns: DataTableColumns
  data: any[]
  loading?: boolean
  pagination?: boolean | object
  bordered?: boolean
  singleLine?: boolean
  scrollX?: number
  rowKey?: string
  checkedRowKeys?: Array<string | number>
  mobileFields?: Array<{ key: string; label: string; format?: (value: any, item: any) => string }>
  mobileActions?: Array<any>
  mobileTitle?: string | ((item: any) => string)
  emptyText?: string
  clickable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  pagination: false,
  bordered: false,
  singleLine: false,
  rowKey: 'id',
  emptyText: '暂无数据',
  clickable: false
})

const emit = defineEmits<{
  'update:checkedRowKeys': [keys: Array<string | number>]
  'update:sorter': [sorter: any]
  'update:page': [page: number]
  'update:pageSize': [pageSize: number]
  'rowClick': [row: any]
}>()

const appStore = useAppStore()

const currentPage = ref(1)
const currentPageSize = ref(20)
const totalCount = ref(0)
const pageSizes = [10, 20, 50, 100]

// 分页配置
const paginationConfig = computed(() => {
  if (!props.pagination) return false
  if (typeof props.pagination === 'object') {
    return props.pagination
  }
  return {
    page: currentPage.value,
    pageSize: currentPageSize.value,
    showSizePicker: true,
    pageSizes,
    onChange: handlePageChange,
    onUpdatePageSize: handlePageSizeChange
  }
})

// 监听外部分页变化
watch(() => props.pagination, (val: any) => {
  if (typeof val === 'object' && val) {
    if ('page' in val) currentPage.value = val.page as number
    if ('pageSize' in val) currentPageSize.value = val.pageSize as number
    if ('itemCount' in val) totalCount.value = val.itemCount as number
  }
}, { deep: true, immediate: true })

const handleCheck = (keys: Array<string | number>) => {
  emit('update:checkedRowKeys', keys)
}

const handleSorterChange = (sorter: any) => {
  emit('update:sorter', sorter)
}

const handlePageChange = (page: number) => {
  currentPage.value = page
  emit('update:page', page)
}

const handlePageSizeChange = (pageSize: number) => {
  currentPageSize.value = pageSize
  currentPage.value = 1
  emit('update:pageSize', pageSize)
}

const handleRowClick = (row: any) => {
  if (props.clickable) {
    emit('rowClick', row)
  }
}
</script>

<style scoped>
.unified-table-wrapper {
  width: 100%;
}

:deep(.n-data-table) {
  font-size: 14px;
}

:deep(.n-data-table .n-data-table-th) {
  font-weight: 600;
  background: var(--n-th-color);
}

:deep(.n-data-table .n-data-table-td) {
  padding: 12px 16px;
}

@media (max-width: 767px) {
  .unified-table-wrapper {
    padding: 0;
  }
}
</style>
