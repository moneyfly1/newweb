<!-- 统一的列表卡片组件（移动端） -->
<template>
  <div class="unified-card-list">
    <div v-for="(item, index) in data" :key="getKey(item, index)" class="unified-card" @click="handleClick(item)">
      <!-- 卡片头部 -->
      <div v-if="$slots.header || title" class="card-header">
        <slot name="header" :item="item" :index="index">
          <span class="card-title">{{ getTitle(item) }}</span>
        </slot>
        <slot name="headerExtra" :item="item" :index="index" />
      </div>

      <!-- 卡片主体 -->
      <div class="card-body">
        <slot name="content" :item="item" :index="index">
          <div v-for="field in fields" :key="field.key" class="card-row">
            <span class="card-label">{{ field.label }}:</span>
            <span class="card-value">{{ formatValue(item, field) }}</span>
          </div>
        </slot>
      </div>

      <!-- 卡片底部操作 -->
      <div v-if="$slots.actions || actions.length" class="card-actions">
        <slot name="actions" :item="item" :index="index">
          <n-button
            v-for="action in actions"
            :key="action.key"
            :size="action.size || 'small'"
            :type="action.type"
            :secondary="action.secondary"
            :quaternary="action.quaternary"
            :disabled="action.disabled?.(item)"
            @click.stop="action.onClick(item)"
          >
            <template v-if="action.icon" #icon>
              <n-icon :component="action.icon" />
            </template>
            {{ action.label }}
          </n-button>
        </slot>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="!loading && data.length === 0" class="card-empty">
      <slot name="empty">
        <n-empty :description="emptyText" />
      </slot>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="card-loading">
      <n-spin size="medium" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'

interface Field {
  key: string
  label: string
  format?: (value: any, item: any) => string
}

interface Action {
  key: string
  label: string
  type?: 'default' | 'primary' | 'info' | 'success' | 'warning' | 'error'
  size?: 'tiny' | 'small' | 'medium' | 'large'
  secondary?: boolean
  quaternary?: boolean
  icon?: Component
  disabled?: (item: any) => boolean
  onClick: (item: any) => void
}

interface Props {
  data: any[]
  fields?: Field[]
  actions?: Action[]
  title?: string | ((item: any) => string)
  keyField?: string
  loading?: boolean
  emptyText?: string
  clickable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  fields: () => [],
  actions: () => [],
  keyField: 'id',
  loading: false,
  emptyText: '暂无数据',
  clickable: false
})

const emit = defineEmits<{
  'click': [item: any]
}>()

const getKey = (item: any, index: number) => {
  return item[props.keyField] ?? index
}

const getTitle = (item: any) => {
  if (typeof props.title === 'function') {
    return props.title(item)
  }
  return props.title || `项目 ${getKey(item, 0)}`
}

const formatValue = (item: any, field: Field) => {
  const value = item[field.key]
  if (field.format) {
    return field.format(value, item)
  }
  return value ?? '-'
}

const handleClick = (item: any) => {
  if (props.clickable) {
    emit('click', item)
  }
}
</script>

<style scoped>
.unified-card-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.unified-card {
  background: var(--n-color);
  border-radius: 12px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
  overflow: hidden;
  transition: all 0.2s ease;
}

.unified-card:active {
  transform: scale(0.98);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-bottom: 1px solid var(--n-border-color);
  background: var(--n-color-target);
}

.card-title {
  font-weight: 600;
  font-size: 15px;
  color: var(--n-text-color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  margin-right: 8px;
}

.card-body {
  padding: 12px 16px;
}

.card-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 6px 0;
  font-size: 14px;
  gap: 12px;
}

.card-label {
  color: var(--n-text-color-3);
  flex-shrink: 0;
  min-width: 70px;
}

.card-value {
  text-align: right;
  word-break: break-word;
  color: var(--n-text-color);
  flex: 1;
}

.card-actions {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--n-border-color);
  flex-wrap: wrap;
  background: var(--n-color-target);
}

.card-empty {
  padding: 40px 20px;
  text-align: center;
}

.card-loading {
  padding: 40px 20px;
  display: flex;
  justify-content: center;
  align-items: center;
}

@media (max-width: 767px) {
  .unified-card {
    border-radius: 10px;
  }

  .card-header {
    padding: 12px 14px;
  }

  .card-title {
    font-size: 14px;
  }

  .card-body {
    padding: 10px 14px;
  }

  .card-row {
    font-size: 13px;
    padding: 4px 0;
  }

  .card-label {
    min-width: 60px;
  }

  .card-actions {
    padding: 10px 14px;
  }
}
</style>
