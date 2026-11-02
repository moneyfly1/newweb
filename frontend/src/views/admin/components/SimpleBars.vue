<template>
  <div class="simple-bars-wrap">
    <div class="simple-bars" :style="{ height: height + 'px' }">
      <div
        v-for="(it, i) in items"
        :key="i"
        class="bar-col"
        :title="`${it.label}：${it.display ?? it.value}`"
      >
        <div class="bar-fill" :style="{ height: pct(it.value) + '%' }"></div>
      </div>
    </div>
    <div class="simple-bars-axis">
      <span v-for="(it, i) in items" :key="i" class="axis-label">
        {{ i % step === 0 ? it.label : '' }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

// 轻量柱状图：数据量小（≤31 个点）时用纯 CSS 渲染，
// 替代 echarts（543KB）——后台首页/统计概览不需要那么重的图表库。
const props = withDefaults(defineProps<{
  items: Array<{ label: string; value: number; display?: string }>
  height?: number
  step?: number
}>(), {
  height: 280,
  step: 5,
})

const maxValue = computed(() => Math.max(1, ...props.items.map(i => Number(i.value) || 0)))
const step = computed(() => props.step)

function pct(v: number) {
  return Math.max(1, Math.round(((Number(v) || 0) / maxValue.value) * 100))
}
</script>

<style scoped>
.simple-bars-wrap { width: 100%; }
.simple-bars {
  display: flex;
  align-items: flex-end;
  gap: 3px;
  padding: 8px 0 0;
  border-bottom: 1px solid var(--border-color, #e5e7eb);
}
.bar-col {
  flex: 1 1 0;
  min-width: 2px;
  height: 100%;
  display: flex;
  align-items: flex-end;
  cursor: default;
}
.bar-fill {
  width: 100%;
  border-radius: 4px 4px 0 0;
  background: linear-gradient(180deg, #60a5fa 0%, #3b82f6 100%);
  transition: opacity 0.15s ease;
}
.bar-col:hover .bar-fill { opacity: 0.75; }
.simple-bars-axis {
  display: flex;
  gap: 3px;
  margin-top: 6px;
}
.axis-label {
  flex: 1 1 0;
  min-width: 0;
  font-size: 11px;
  color: var(--text-color-secondary, #94a3b8);
  text-align: center;
  white-space: nowrap;
  overflow: hidden;
}
</style>
