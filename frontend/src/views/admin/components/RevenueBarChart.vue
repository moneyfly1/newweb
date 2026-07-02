<template>
  <v-chart :option="option" autoresize style="height: 320px;" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent, VisualMapComponent } from 'echarts/components'
import VChart from 'vue-echarts'

const props = defineProps<{
  data: Array<{ date: string; value: number }>
}>()

use([CanvasRenderer, BarChart, GridComponent, TooltipComponent, LegendComponent, VisualMapComponent])

const option = computed(() => ({
  tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
  grid: { left: '4%', right: '4%', top: '10%', bottom: '10%', containLabel: true },
  xAxis: {
    type: 'category',
    data: props.data.map(d => d.date.slice(5)),
    axisLine: { lineStyle: { color: '#eee' } },
    axisLabel: { color: '#999' },
  },
  yAxis: {
    type: 'value',
    splitLine: { lineStyle: { type: 'dashed', color: '#f5f5f5' } },
    axisLabel: { color: '#999' },
  },
  series: [{
    name: '收入',
    type: 'bar',
    data: props.data.map(d => d.value),
    itemStyle: {
      color: {
        type: 'linear',
        x: 0,
        y: 0,
        x2: 0,
        y2: 1,
        colorStops: [{ offset: 0, color: '#3b82f6' }, { offset: 1, color: '#60a5fa' }],
      },
      borderRadius: [6, 6, 0, 0],
    },
    barMaxWidth: 16,
  }],
}))
</script>
