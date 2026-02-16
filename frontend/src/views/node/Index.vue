<template>
  <div class="node-page">
    <!-- Stats Bar -->
    <div class="stats-bar">
      <div class="stat-card">
        <div class="stat-icon total">
          <n-icon :size="22"><ServerOutline /></n-icon>
        </div>
        <div class="stat-info">
          <span class="stat-value">{{ stats.total }}</span>
          <span class="stat-label">总节点</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon online">
          <n-icon :size="22"><CheckmarkCircleOutline /></n-icon>
        </div>
        <div class="stat-info">
          <span class="stat-value online-val">{{ stats.online }}</span>
          <span class="stat-label">在线节点</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon latency">
          <n-icon :size="22"><SpeedometerOutline /></n-icon>
        </div>
        <div class="stat-info">
          <span class="stat-value">{{ stats.avgLatency }}</span>
          <span class="stat-label">平均延迟</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon regions">
          <n-icon :size="22"><GlobeOutline /></n-icon>
        </div>
        <div class="stat-info">
          <span class="stat-value">{{ stats.regions }}</span>
          <span class="stat-label">覆盖地区</span>
        </div>
      </div>
    </div>

    <!-- Filters -->
    <div class="filter-bar">
      <n-space :wrap="true" :size="10" align="center">
        <n-select v-model:value="filterRegion" :options="regionOptions" placeholder="筛选地区" clearable style="width: 160px;" />
        <n-select v-model:value="filterProtocol" :options="protocolOptions" placeholder="筛选协议" clearable style="width: 140px;" />
        <n-button size="small" :loading="loading" @click="fetchNodes">刷新</n-button>
      </n-space>
      <span class="filter-result-count">共 {{ filteredNodes.length }} 个节点</span>
    </div>
    <!-- Desktop Table -->
    <div class="desktop-table">
      <n-spin :show="loading">
        <n-empty v-if="!loading && filteredNodes.length === 0" description="暂无节点数据" />
        <template v-else>
          <div v-for="group in groupedNodes" :key="group.region" class="region-group">
            <div class="region-group-header">
              <span class="region-flag">{{ getRegionFlag(group.region) }}</span>
              <span class="region-name">{{ group.region || '未知地区' }}</span>
              <n-tag size="small" :bordered="false" round>{{ group.nodes.length }} 个节点</n-tag>
            </div>
            <div class="table-wrap">
              <table class="node-table">
                <thead>
                  <tr>
                    <th class="col-status">状态</th>
                    <th class="col-name">节点名称</th>
                    <th class="col-protocol">协议</th>
                    <th class="col-latency">延迟</th>
                    <th class="col-action">操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="node in group.nodes" :key="node.id" :class="{ 'row-offline': node.status !== 'online' }">
                    <td class="col-status">
                      <span class="status-badge" :class="node.status === 'online' ? 'online' : 'offline'">
                        {{ node.status === 'online' ? '在线' : '离线' }}
                      </span>
                    </td>
                    <td class="col-name">
                      <span class="status-dot" :class="node.status === 'online' ? 'dot-online' : 'dot-offline'"></span>
                      {{ node.name }}
                    </td>
                    <td class="col-protocol"><span class="protocol-tag">{{ node.protocol }}</span></td>
                    <td class="col-latency"><span :class="getLatencyClass(node)">{{ formatLatency(node) }}</span></td>
                    <td class="col-action">
                      <n-button size="tiny" :loading="testingNodes[node.id]" @click="handleTestNode(node)" :disabled="node.status !== 'online'" quaternary type="primary">
                        {{ testResults[node.id] || '测试' }}
                      </n-button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </template>
      </n-spin>
    </div>
    <!-- Mobile Cards -->
    <div class="mobile-cards">
      <n-spin :show="loading">
        <n-empty v-if="!loading && filteredNodes.length === 0" description="暂无节点数据" />
        <template v-else>
          <div v-for="group in groupedNodes" :key="group.region" class="region-group">
            <div class="region-group-header">
              <span class="region-flag">{{ getRegionFlag(group.region) }}</span>
              <span class="region-name">{{ group.region || '未知地区' }}</span>
              <n-tag size="small" :bordered="false" round>{{ group.nodes.length }}</n-tag>
            </div>
            <div class="mobile-node-list">
              <div v-for="node in group.nodes" :key="node.id" class="mobile-node-card" :class="{ 'card-offline': node.status !== 'online' }">
                <div class="mobile-card-top">
                  <div class="mobile-node-name">
                    <span class="status-dot" :class="node.status === 'online' ? 'dot-online' : 'dot-offline'"></span>
                    <span>{{ node.name }}</span>
                  </div>
                  <span class="status-badge" :class="node.status === 'online' ? 'online' : 'offline'">
                    {{ node.status === 'online' ? '在线' : '离线' }}
                  </span>
                </div>
                <div class="mobile-card-info">
                  <div class="mobile-info-item">
                    <span class="mobile-info-label">协议</span>
                    <span class="protocol-tag">{{ node.protocol }}</span>
                  </div>
                  <div class="mobile-info-item">
                    <span class="mobile-info-label">延迟</span>
                    <span :class="getLatencyClass(node)">{{ formatLatency(node) }}</span>
                  </div>
                  <div class="mobile-info-item mobile-info-action">
                    <n-button size="tiny" :loading="testingNodes[node.id]" @click="handleTestNode(node)" :disabled="node.status !== 'online'" quaternary type="primary">
                      {{ testResults[node.id] || '测试延迟' }}
                    </n-button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </template>
      </n-spin>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NIcon } from 'naive-ui'
import { listNodes, testNode } from '@/api/node'
import { useMessage } from 'naive-ui'
import { ServerOutline, CheckmarkCircleOutline, SpeedometerOutline, GlobeOutline } from '@vicons/ionicons5'

interface Node {
  id: number
  name: string
  region: string
  protocol: string
  status: string
  latency?: number
}

const message = useMessage()
const loading = ref(false)
const nodes = ref<Node[]>([])
const filterRegion = ref<string | null>(null)
const filterProtocol = ref<string | null>(null)
const testingNodes = ref<Record<number, boolean>>({})
const testResults = ref<Record<number, string>>({})

const stats = computed(() => {
  const all = nodes.value
  const onlineNodes = all.filter(n => n.status === 'online')
  const latencies = all.filter(n => n.latency && n.latency > 0).map(n => n.latency!)
  const avgLat = latencies.length > 0
    ? Math.round(latencies.reduce((a, b) => a + b, 0) / latencies.length) : null
  const regions = new Set(all.map(n => n.region).filter(Boolean)).size
  return { total: all.length, online: onlineNodes.length, avgLatency: avgLat !== null ? `${avgLat}ms` : '-', regions }
})

const regionOptions = computed(() => {
  const regions = [...new Set(nodes.value.map(n => n.region).filter(Boolean))]
  return regions.sort().map(r => ({ label: `${getRegionFlag(r)} ${r}`, value: r }))
})
const protocolOptions = computed(() => {
  const protocols = [...new Set(nodes.value.map(n => n.protocol).filter(Boolean))]
  return protocols.sort().map(p => ({ label: p.toUpperCase(), value: p }))
})
const filteredNodes = computed(() => {
  return nodes.value.filter(node => {
    if (filterRegion.value && node.region !== filterRegion.value) return false
    if (filterProtocol.value && node.protocol !== filterProtocol.value) return false
    return true
  })
})

const groupedNodes = computed(() => {
  const map = new Map<string, Node[]>()
  for (const node of filteredNodes.value) {
    const key = node.region || '未知地区'
    if (!map.has(key)) map.set(key, [])
    map.get(key)!.push(node)
  }
  return Array.from(map.entries())
    .sort((a, b) => b[1].length - a[1].length)
    .map(([region, nodes]) => ({
      region,
      nodes: nodes.sort((a, b) => (a.status === 'online' ? 0 : 1) - (b.status === 'online' ? 0 : 1))
    }))
})

const formatLatency = (node: Node): string => {
  if (node.latency && node.latency > 0) return `${node.latency}ms`
  return node.status === 'online' ? '-' : '--'
}
const getLatencyClass = (node: Node): string => {
  if (!node.latency || node.latency <= 0) return 'latency-none'
  if (node.latency < 100) return 'latency-good'
  if (node.latency < 200) return 'latency-medium'
  return 'latency-poor'
}

const getRegionFlag = (region: string): string => {
  const flagMap: Record<string, string> = {
    '中国': '\u{1F1E8}\u{1F1F3}', '香港': '\u{1F1ED}\u{1F1F0}',
    '台湾': '\u{1F1F9}\u{1F1FC}', '日本': '\u{1F1EF}\u{1F1F5}',
    '韩国': '\u{1F1F0}\u{1F1F7}', '新加坡': '\u{1F1F8}\u{1F1EC}',
    '美国': '\u{1F1FA}\u{1F1F8}', '英国': '\u{1F1EC}\u{1F1E7}',
    '德国': '\u{1F1E9}\u{1F1EA}', '法国': '\u{1F1EB}\u{1F1F7}',
    '加拿大': '\u{1F1E8}\u{1F1E6}', '澳大利亚': '\u{1F1E6}\u{1F1FA}',
    '俄罗斯': '\u{1F1F7}\u{1F1FA}', '印度': '\u{1F1EE}\u{1F1F3}',
    '巴西': '\u{1F1E7}\u{1F1F7}', '荷兰': '\u{1F1F3}\u{1F1F1}',
    '瑞士': '\u{1F1E8}\u{1F1ED}', '意大利': '\u{1F1EE}\u{1F1F9}',
    '西班牙': '\u{1F1EA}\u{1F1F8}', '泰国': '\u{1F1F9}\u{1F1ED}',
    '越南': '\u{1F1FB}\u{1F1F3}', '马来西亚': '\u{1F1F2}\u{1F1FE}',
    '菲律宾': '\u{1F1F5}\u{1F1ED}', '印度尼西亚': '\u{1F1EE}\u{1F1E9}',
    '土耳其': '\u{1F1F9}\u{1F1F7}', '阿根廷': '\u{1F1E6}\u{1F1F7}',
    '墨西哥': '\u{1F1F2}\u{1F1FD}', '南非': '\u{1F1FF}\u{1F1E6}',
    '埃及': '\u{1F1EA}\u{1F1EC}', '以色列': '\u{1F1EE}\u{1F1F1}',
    '阿联酋': '\u{1F1E6}\u{1F1EA}', '沙特阿拉伯': '\u{1F1F8}\u{1F1E6}'
  }
  return flagMap[region] || '\u{1F310}'
}
const handleTestNode = async (node: Node) => {
  testingNodes.value[node.id] = true
  testResults.value[node.id] = ''
  try {
    const res = await testNode(node.id)
    const latency = res.data?.latency
    if (latency && latency > 0) {
      testResults.value[node.id] = `${latency}ms`
      node.latency = latency
      message.success(`${node.name}: ${latency}ms`)
    } else {
      testResults.value[node.id] = '超时'
      message.warning(`${node.name}: 测试超时`)
    }
  } catch (e: any) {
    testResults.value[node.id] = '失败'
    message.error(e.message || '测试失败')
  } finally {
    testingNodes.value[node.id] = false
  }
}

const fetchNodes = async () => {
  loading.value = true
  try {
    const res = await listNodes({ page_size: 500 })
    const items = res.data?.items || res.data || []
    nodes.value = items.map((n: any) => ({
      id: n.id,
      name: n.name,
      region: n.region || '',
      protocol: n.protocol || n.type || '',
      status: n.status || 'offline',
      latency: n.latency || 0,
    }))
  } catch (error: any) {
    message.error(error.message || '获取节点列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => { fetchNodes() })
</script>

<style scoped>
.node-page { padding: 20px; max-width: 1400px; margin: 0 auto; }
.stats-bar { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 20px; }
.stat-card {
  display: flex; align-items: center; gap: 14px; padding: 18px 20px;
  background: var(--bg-color, #fff); border-radius: 12px; border: 1px solid var(--border-color, #eef0f3); transition: box-shadow 0.2s;
}
.stat-card:hover { box-shadow: 0 2px 12px rgba(0,0,0,0.05); }
.stat-icon { width: 44px; height: 44px; border-radius: 10px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.stat-icon.total { background: rgba(99,125,255,0.12); color: #637dff; }
.stat-icon.online { background: rgba(24,160,88,0.12); color: #18a058; }
.stat-icon.latency { background: rgba(245,166,35,0.12); color: #f5a623; }
.stat-icon.regions { background: rgba(114,46,209,0.12); color: #722ed1; }
.stat-info { display: flex; flex-direction: column; gap: 2px; }
.stat-value { font-size: 22px; font-weight: 700; line-height: 1.2; color: var(--text-color, #1a1a1a); }
.stat-label { font-size: 13px; color: var(--text-color-secondary, #999); }
.filter-bar {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 20px; padding: 12px 16px; border-radius: 10px; border: 1px solid var(--border-color, #eef0f3);
}
.filter-result-count { font-size: 13px; color: var(--text-color-secondary, #999); white-space: nowrap; }
.region-group { margin-bottom: 24px; }
.region-group-header { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; padding-bottom: 8px; border-bottom: 2px solid var(--border-color, #f0f2f5); }
.region-flag { font-size: 22px; line-height: 1; }
.region-name { font-size: 16px; font-weight: 600; color: var(--text-color, #1a1a1a); }
.desktop-table { display: block; }
.mobile-cards { display: none; }
.table-wrap { overflow-x: auto; }
.node-table { width: 100%; border-collapse: collapse; border-radius: 8px; overflow: hidden; border: 1px solid var(--border-color, #eef0f3); }
.node-table thead th { padding: 12px 16px; text-align: left; font-size: 13px; font-weight: 600; color: var(--text-color-secondary, #666); background: rgba(0,0,0,0.03); border-bottom: 1px solid var(--border-color, #eef0f3); white-space: nowrap; }
.node-table tbody td { padding: 12px 16px; font-size: 14px; color: var(--text-color, #333); border-bottom: 1px solid var(--border-color, #f5f5f5); vertical-align: middle; }
.node-table tbody tr:last-child td { border-bottom: none; }
.node-table tbody tr:hover { background: rgba(0,0,0,0.02); }
.row-offline td { opacity: 0.5; }
.col-status { width: 70px; }
.col-name { min-width: 200px; }
.col-protocol { width: 100px; }
.col-latency { width: 90px; }
.col-action { width: 80px; text-align: center; }
.status-badge { display: inline-block; padding: 2px 10px; border-radius: 10px; font-size: 12px; font-weight: 500; }
.status-badge.online { background: rgba(24,160,88,0.12); color: #18a058; }
.status-badge.offline { background: rgba(208,48,80,0.12); color: #d03050; }
.status-dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 8px; vertical-align: middle; }
.dot-online { background: #18a058; box-shadow: 0 0 0 0 rgba(24,160,88,0.4); animation: pulse-green 2s infinite; }
.dot-offline { background: #d03050; }
@keyframes pulse-green {
  0% { box-shadow: 0 0 0 0 rgba(24,160,88,0.4); }
  70% { box-shadow: 0 0 0 6px rgba(24,160,88,0); }
  100% { box-shadow: 0 0 0 0 rgba(24,160,88,0); }
}
.protocol-tag { display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: 12px; font-weight: 500; background: rgba(99,125,255,0.12); color: #637dff; }
.latency-good { color: #18a058; font-weight: 600; }
.latency-medium { color: #f5a623; font-weight: 600; }
.latency-poor { color: #d03050; font-weight: 600; }
.latency-none { color: var(--text-color-secondary, #ccc); }
.mobile-node-list { display: flex; flex-direction: column; gap: 10px; }
.mobile-node-card { border-radius: 10px; padding: 14px 16px; border: 1px solid var(--border-color, #eef0f3); border-left: 4px solid #18a058; }
.mobile-node-card.card-offline { border-left-color: #d03050; opacity: 0.7; }
.mobile-card-top { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; padding-bottom: 10px; border-bottom: 1px solid var(--border-color, #f5f5f5); }
.mobile-node-name { display: flex; align-items: center; font-size: 14px; font-weight: 600; color: var(--text-color, #1a1a1a); min-width: 0; }
.mobile-node-name span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mobile-card-info { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
.mobile-info-item { display: flex; align-items: center; justify-content: space-between; padding: 4px 0; }
.mobile-info-action { grid-column: 1 / -1; justify-content: flex-end; }
.mobile-info-label { font-size: 12px; color: var(--text-color-secondary, #999); }

@media (max-width: 767px) {
  .node-page { padding: 12px; }
  .stats-bar { grid-template-columns: repeat(2, 1fr); gap: 10px; }
  .stat-card { padding: 14px; }
  .stat-value { font-size: 18px; }
  .stat-icon { width: 38px; height: 38px; }
  .filter-bar { flex-direction: column; gap: 8px; align-items: flex-start; }
  .desktop-table { display: none; }
  .mobile-cards { display: block; }
}
@media (max-width: 480px) {
  .stats-bar { gap: 8px; }
  .stat-card { padding: 12px; gap: 10px; }
  .stat-value { font-size: 16px; }
  .stat-icon { width: 34px; height: 34px; }
  .stat-icon :deep(svg) { width: 18px; height: 18px; }
}
</style>
