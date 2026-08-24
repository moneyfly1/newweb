<template>
  <v-chart :option="option" autoresize style="height: 240px;" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { PieChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent } from 'echarts/components'
import VChart from 'vue-echarts'

const props = defineProps<{
  data: Array<{ name: string; value: number }>
}>()

use([CanvasRenderer, PieChart, TooltipComponent, LegendComponent])

const option = computed(() => ({
  tooltip: { trigger: 'item', formatter: '{b}: {c} 元 ({d}%)' },
  legend: { bottom: 0, type: 'scroll', textStyle: { fontSize: 11, color: '#888' } },
  series: [{
    name: '支付方式',
    type: 'pie',
    radius: ['45%', '70%'],
    center: ['50%', '42%'],
    avoidLabelOverlap: true,
    itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
    label: { show: false },
    emphasis: { label: { show: true, fontSize: 13, fontWeight: 'bold' } },
    data: props.data,
  }],
}))
</script>
