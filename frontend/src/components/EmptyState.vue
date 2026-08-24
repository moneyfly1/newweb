<template>
  <div class="empty-state">
    <div class="empty-state-icon" :class="{ 'is-custom': !!icon }">
      <n-icon v-if="icon" :component="icon" :size="34" />
      <n-icon v-else :component="defaultIcon" :size="34" />
    </div>
    <p class="empty-state-description">{{ description }}</p>
    <n-button
      v-if="actionText"
      type="primary"
      size="medium"
      class="empty-state-action"
      @click="emit('action')"
    >
      {{ actionText }}
    </n-button>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import { FileTrayFullOutline } from '@vicons/ionicons5'

defineOptions({ name: 'EmptyState' })

withDefaults(
  defineProps<{
    /** 空态文案 */
    description: string
    /** 可选图标（n-icon 组件）；缺省使用内置「空托盘」图标 */
    icon?: Component
    /** 可选按钮文字；提供时渲染一个主按钮 */
    actionText?: string
  }>(),
  {
    icon: undefined,
    actionText: undefined,
  }
)

const emit = defineEmits<{
  (e: 'action'): void
}>()

const defaultIcon = FileTrayFullOutline
</script>

<style scoped>
/* 统一空态：居中图标（CSS 圆底）+ 文案 + 可选按钮，明暗主题自适应 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  padding: 56px 20px;
  text-align: center;
}

/* CSS 绘制的圆底 */
.empty-state-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 76px;
  height: 76px;
  border-radius: 50%;
  color: var(--primary-color, #667eea);
  background: var(--primary-color-soft, rgba(102, 126, 234, 0.08));
  border: 1px solid var(--primary-color-hover, rgba(102, 126, 234, 0.14));
  box-shadow: inset 0 0 0 6px var(--primary-color-soft, rgba(102, 126, 234, 0.05));
  flex-shrink: 0;
}

.empty-state-icon.is-custom {
  box-shadow: none;
}

.empty-state-description {
  margin: 0;
  max-width: 320px;
  font-size: 14px;
  line-height: 1.7;
  color: var(--text-color-secondary, #94a3b8);
}

.empty-state-action {
  margin-top: 2px;
}

/* 移动端适配 */
@media (max-width: 767px) {
  .empty-state {
    padding: 40px 16px;
    gap: 12px;
  }
  .empty-state-icon {
    width: 66px;
    height: 66px;
  }
  .empty-state-description {
    font-size: 13px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .empty-state-icon {
    transition: none;
  }
}
</style>
