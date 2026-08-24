<template>
  <div class="sf-bar">
    <n-input
      v-if="searchPlaceholder"
      v-model:value="values.search"
      :placeholder="searchPlaceholder"
      clearable
      class="sf-search"
      @keyup.enter="onSearch"
      @clear="onSearch"
    >
      <template #prefix><n-icon><search-outline /></n-icon></template>
    </n-input>

    <n-button
      v-if="searchPlaceholder"
      type="primary"
      class="sf-search-btn"
      @click="onSearch"
    >
      <template #icon><n-icon><search-outline /></n-icon></template>
      搜索
    </n-button>

    <n-select
      v-for="f in filters"
      :key="f.key"
      v-model:value="values[f.key]"
      :placeholder="f.placeholder"
      :options="f.options"
      clearable
      class="sf-filter"
      @update:value="onSearch"
    />

    <slot name="extra" />
  </div>
</template>

<script setup lang="ts">
import { SearchOutline } from '@vicons/ionicons5'

export interface SearchFilterConfig {
  key: string
  placeholder?: string
  options: any[]
}

// 统一搜索筛选工具栏：桌面端强制单行（不换行），移动端响应式两列
const props = defineProps<{
  searchPlaceholder?: string
  filters?: SearchFilterConfig[]
  /** 搜索/筛选值对象（{ search, [filterKey]: value }） */
  values: Record<string, any>
}>()

const emit = defineEmits<{ search: [] }>()

function onSearch() {
  emit('search')
}
</script>

<style scoped>
/* 桌面端：单行不换行，控件按比例分配宽度 */
.sf-bar {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
  width: 100%;
}
.sf-search {
  flex: 1.6 1 220px;
  min-width: 160px;
  max-width: 320px;
}
.sf-filter {
  flex: 1 1 120px;
  min-width: 100px;
  max-width: 180px;
}
.sf-search-btn {
  flex: 0 0 auto;
}

/* 移动端：搜索框+按钮同行，筛选器换行两列，不溢出屏幕 */
@media (max-width: 767px) {
  .sf-bar {
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 10px;
  }
  .sf-search {
    flex: 1 1 60%;
    max-width: none;
    min-width: 0;
  }
  .sf-search-btn {
    flex: 0 0 auto;
  }
  .sf-filter {
    flex: 1 1 45%;
    max-width: none;
    min-width: 0;
  }
  .sf-bar :deep(.n-input),
  .sf-bar :deep(.n-select) {
    width: 100%;
  }
}
</style>
