<template>
  <v-chart :option="option" autoresize style="height: 300px;" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import VChart from 'vue-echarts'

const props = defineProps<{
  data: Array<{ date: string; revenue: number; recharge: number; orders?: number }>
}>()

use([CanvasRenderer, BarChart, GridComponent, TooltipComponent, LegendComponent])

const option = computed(() => ({
  tooltip: {
    trigger: 'axis',
    axisPointer: { type: 'shadow' },
    formatter: (params: any[]) => {
      const first = params[0]
      if (!first) return ''
      const item = props.data[first.dataIndex] || {}
      let html = `${item.date}<br/>`
      params.forEach(p => { html += `${p.marker}${p.seriesName}: ¥${Number(p.value || 0).toFixed(2)}<br/>` })
      if (item.orders != null) html += `订单: ${item.orders} 单`
      return html
    },
  },
  legend: { bottom: 0, textStyle: { fontSize: 12, color: '#888' } },
  grid: { left: '4%', right: '4%', top: '8%', bottom: '14%', containLabel: true },
  xAxis: {
    type: 'category',
    data: props.data.map(d => d.date.slice(5)),
    axisLine: { lineStyle: { color: '#e5e7eb' } },
    axisLabel: { color: '#999', fontSize: 11 },
  },
  yAxis: {
    type: 'value',
    splitLine: { lineStyle: { type: 'dashed', color: '#f0f0f0' } },
    axisLabel: { color: '#999', fontSize: 11 },
  },
  series: [
    {
      name: '收入',
      type: 'bar',
      data: props.data.map(d => d.revenue || 0),
      itemStyle: { color: '#3b82f6', borderRadius: [5, 5, 0, 0] },
      barMaxWidth: 14,
    },
    {
      name: '充值',
      type: 'bar',
      data: props.data.map(d => d.recharge || 0),
      itemStyle: { color: '#10b981', borderRadius: [5, 5, 0, 0] },
      barMaxWidth: 14,
    },
  ],
}))
</script>
