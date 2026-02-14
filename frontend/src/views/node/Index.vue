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
          <span class="stat-value">{{ stats.online }}</span>
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
        <n-select
          v-model:value="filterCountry"
          :options="countryOptions"
          placeholder="筛选国家/地区"
          clearable
          style="width: 150px;"
          @update:value="handleFilter"
        />
        <n-select
          v-model:value="filterProtocol"
          :options="protocolOptions"
          placeholder="筛选协议"
          clearable
          style="width: 140px;"
          @update:value="handleFilter"
        />
      </n-space>
      <span class="filter-result-count">
        共 {{ filteredNodes.length }} 个节点
      </span>
    </div>

    <!-- Node List grouped by country -->
    <n-spin :show="loading">
      <n-empty v-if="!loading && filteredNodes.length === 0" description="暂无节点数据" />
      <template v-else>
        <div v-for="group in groupedNodes" :key="group.country" class="country-group">
          <div class="country-group-header">
            <span class="country-flag">{{ getCountryFlag(group.country) }}</span>
            <span class="country-name">{{ group.country }}</span>
            <span class="country-count">{{ group.nodes.length }} 个节点</span>
          </div>
          <div class="node-grid">
            <div v-for="node in group.nodes" :key="node.id" class="node-card">
              <div class="node-card-header">
                <div class="node-name-row">
                  <span
                    class="status-dot"
                    :class="node.status === 'online' ? 'online' : 'offline'"
                  ></span>
                  <span class="node-name">{{ node.name }}</span>
                </div>
                <n-tag
                  :bordered="false"
                  size="small"
                  :type="node.status === 'online' ? 'success' : 'error'"
                  round
                >
                  {{ node.status === 'online' ? '在线' : '离线' }}
                </n-tag>
              </div>
              <div class="node-info-grid">
                <div class="info-item">
                  <span class="info-label">协议</span>
                  <span class="info-value protocol">{{ node.protocol }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">倍率</span>
                  <span class="info-value rate">{{ node.rate }}x</span>
                </div>
                <div class="info-item">
                  <span class="info-label">延迟</span>
                  <span class="info-value latency-val">{{ formatLatency(node) }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">地区</span>
                  <span class="info-value">{{ node.country }}</span>
                </div>
              </div>
              <div class="node-card-actions">
                <n-button size="tiny" :loading="testingNodes[node.id]" @click="handleTestNode(node)" :disabled="node.status !== 'online'">
                  {{ testResults[node.id] || '测试延迟' }}
                </n-button>
              </div>
            </div>
          </div>
        </div>
      </template>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NIcon } from 'naive-ui'
import { listNodes, testNode } from '@/api/node'
import { useMessage } from 'naive-ui'
import {
  ServerOutline,
  CheckmarkCircleOutline,
  SpeedometerOutline,
  GlobeOutline
} from '@vicons/ionicons5'

interface Node {
  id: number
  name: string
  country: string
  protocol: string
  status: string
  rate: number
  latency?: number
  online_users?: number
}

const message = useMessage()
const loading = ref(false)
const nodes = ref<Node[]>([])
const filterCountry = ref<string | null>(null)
const filterProtocol = ref<string | null>(null)
const testingNodes = ref<Record<number, boolean>>({})
const testResults = ref<Record<number, string>>({})

const stats = computed(() => {
  const all = nodes.value
  const onlineNodes = all.filter(n => n.status === 'online')
  const latencies = all.filter(n => n.latency && n.latency > 0).map(n => n.latency!)
  const avgLat = latencies.length > 0
    ? Math.round(latencies.reduce((a, b) => a + b, 0) / latencies.length)
    : null
  const regions = new Set(all.map(n => n.country)).size
  return {
    total: all.length,
    online: onlineNodes.length,
    avgLatency: avgLat !== null ? `${avgLat}ms` : '-',
    regions
  }
})

const countryOptions = computed(() => {
  const countries = [...new Set(nodes.value.map(n => n.country))]
  return countries.map(c => ({ label: `${getCountryFlag(c)} ${c}`, value: c }))
})
const protocolOptions = computed(() => {
  const protocols = [...new Set(nodes.value.map(n => n.protocol))]
  return protocols.map(p => ({ label: p, value: p }))
})

const filteredNodes = computed(() => {
  return nodes.value.filter(node => {
    if (filterCountry.value && node.country !== filterCountry.value) return false
    if (filterProtocol.value && node.protocol !== filterProtocol.value) return false
    return true
  })
})

const groupedNodes = computed(() => {
  const map = new Map<string, Node[]>()
  for (const node of filteredNodes.value) {
    if (!map.has(node.country)) map.set(node.country, [])
    map.get(node.country)!.push(node)
  }
  return Array.from(map.entries())
    .sort((a, b) => b[1].length - a[1].length)
    .map(([country, nodes]) => ({
      country,
      nodes: nodes.sort((a, b) => (a.status === 'online' ? 0 : 1) - (b.status === 'online' ? 0 : 1))
    }))
})

const formatLatency = (node: Node): string => {
  if (node.latency && node.latency > 0) return `${node.latency}ms`
  return node.status === 'online' ? '-' : '--'
}

const getCountryFlag = (country: string): string => {
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
  return flagMap[country] || '\u{1F310}'
}

const handleFilter = () => {}

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
    const res = await listNodes()
    const items = res.data?.items || res.data || []
    nodes.value = items.map((n: any) => ({ ...n, protocol: n.type || n.protocol }))
  } catch (error: any) {
    message.error(error.message || '获取节点列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchNodes()
})
</script>

<style scoped>
.node-page {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

/* Stats Bar */
.stats-bar {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 18px 20px;
  background: var(--n-color, #fff);
  border-radius: 12px;
  border: 1px solid var(--n-border-color, #e8e8ec);
  transition: box-shadow 0.2s;
}

.stat-card:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
}

.stat-icon {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-icon.total { background: rgba(99, 125, 255, 0.1); color: #637dff; }
.stat-icon.online { background: rgba(24, 160, 88, 0.1); color: #18a058; }
.stat-icon.latency { background: rgba(245, 166, 35, 0.1); color: #f5a623; }
.stat-icon.regions { background: rgba(114, 46, 209, 0.1); color: #722ed1; }

.stat-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.stat-value {
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
  color: var(--n-text-color, #333);
}

.stat-label {
  font-size: 13px;
  color: var(--n-text-color-3, #999);
}
/* Filter Bar */
.filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
  padding: 12px 16px;
  background: var(--n-color, #fff);
  border-radius: 10px;
  border: 1px solid var(--n-border-color, #e8e8ec);
}

.filter-result-count {
  font-size: 13px;
  color: var(--n-text-color-3, #999);
  white-space: nowrap;
}

/* Country Group */
.country-group {
  margin-bottom: 24px;
}

.country-group-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--n-border-color, #eee);
}

.country-flag {
  font-size: 22px;
  line-height: 1;
}

.country-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--n-text-color, #333);
}

.country-count {
  font-size: 12px;
  color: var(--n-text-color-3, #999);
  margin-left: 4px;
}
/* Node Grid */
.node-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 14px;
}

.node-card {
  padding: 16px;
  background: var(--n-color, #fff);
  border-radius: 10px;
  border: 1px solid var(--n-border-color, #e8e8ec);
  transition: all 0.25s ease;
}

.node-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.07);
  transform: translateY(-1px);
}

.node-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.node-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.node-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--n-text-color, #333);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Status Dot with pulse animation */
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
  flex-shrink: 0;
}

.status-dot.online {
  background-color: #18a058;
  box-shadow: 0 0 0 0 rgba(24, 160, 88, 0.5);
  animation: pulse-green 2s infinite;
}

.status-dot.offline {
  background-color: #d03050;
}

@keyframes pulse-green {
  0% { box-shadow: 0 0 0 0 rgba(24, 160, 88, 0.5); }
  70% { box-shadow: 0 0 0 6px rgba(24, 160, 88, 0); }
  100% { box-shadow: 0 0 0 0 rgba(24, 160, 88, 0); }
}
/* Info Grid */
.node-info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.info-label {
  font-size: 11px;
  color: var(--n-text-color-3, #aaa);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.info-value {
  font-size: 13px;
  font-weight: 500;
  color: var(--n-text-color, #555);
}

.info-value.protocol {
  color: #637dff;
}

.info-value.rate {
  color: #18a058;
  font-weight: 600;
}

.info-value.latency-val {
  color: #f5a623;
}

.node-card-actions {
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px solid var(--n-border-color, #eee);
  display: flex;
  justify-content: flex-end;
}

/* Mobile Responsive */
@media (max-width: 767px) {
  .node-page { padding: 12px; }
  .stats-bar { grid-template-columns: repeat(2, 1fr); gap: 10px; }
  .stat-card { padding: 14px; }
  .stat-value { font-size: 18px; }
  .stat-icon { width: 38px; height: 38px; }
  .filter-bar { flex-direction: column; gap: 8px; align-items: flex-start; }
  .node-grid { grid-template-columns: 1fr; }
  .node-card:hover { transform: none; }
}

@media (max-width: 480px) {
  .stats-bar { grid-template-columns: repeat(2, 1fr); gap: 8px; }
  .stat-card { padding: 12px; gap: 10px; }
  .stat-value { font-size: 16px; }
  .stat-icon { width: 34px; height: 34px; }
  .stat-icon :deep(svg) { width: 18px; height: 18px; }
}
</style>
