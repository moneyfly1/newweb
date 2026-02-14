<template>
  <div class="node-page">
    <n-card title="节点列表" :bordered="false">
      <template #header-extra>
        <n-space :wrap="true" :size="8">
          <n-select
            v-model:value="filterCountry"
            :options="countryOptions"
            placeholder="筛选国家/地区"
            clearable
            style="width: 130px; min-width: 100px;"
            @update:value="handleFilter"
          />
          <n-select
            v-model:value="filterProtocol"
            :options="protocolOptions"
            placeholder="筛选协议"
            clearable
            style="width: 120px; min-width: 100px;"
            @update:value="handleFilter"
          />
        </n-space>
      </template>

      <n-spin :show="loading">
        <n-empty v-if="!loading && filteredNodes.length === 0" description="暂无节点数据" />
        <n-grid v-else cols="1 s:2 l:3" :x-gap="16" :y-gap="16">
          <n-grid-item v-for="node in filteredNodes" :key="node.id">
            <n-card
              :bordered="true"
              class="node-card"
              hoverable
              :segmented="{
                content: true,
                footer: 'soft'
              }"
            >
              <template #header>
                <n-space align="center" :wrap="false">
                  <span class="node-flag">{{ getCountryFlag(node.country) }}</span>
                  <n-ellipsis style="max-width: 200px">
                    {{ node.name }}
                  </n-ellipsis>
                </n-space>
              </template>

              <n-space vertical :size="12">
                <n-space align="center" justify="space-between">
                  <span class="label">协议类型</span>
                  <n-tag :bordered="false" size="small" type="info">
                    {{ node.protocol }}
                  </n-tag>
                </n-space>

                <n-space align="center" justify="space-between">
                  <span class="label">节点状态</span>
                  <n-space align="center" :size="6">
                    <span
                      class="status-dot"
                      :class="node.status === 'online' ? 'online' : 'offline'"
                    ></span>
                    <span :style="{ color: node.status === 'online' ? '#18a058' : '#d03050' }">
                      {{ node.status === 'online' ? '在线' : '离线' }}
                    </span>
                  </n-space>
                </n-space>

                <n-space align="center" justify="space-between">
                  <span class="label">倍率</span>
                  <span class="rate">{{ node.rate }}x</span>
                </n-space>

                <n-space align="center" justify="space-between">
                  <span class="label">地区</span>
                  <span>{{ node.country }}</span>
                </n-space>
              </n-space>
            </n-card>
          </n-grid-item>
        </n-grid>
      </n-spin>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { listNodes } from '@/api/node'
import { useMessage } from 'naive-ui'

interface Node {
  id: number
  name: string
  country: string
  protocol: string
  status: string
  rate: number
}

const message = useMessage()
const loading = ref(false)
const nodes = ref<Node[]>([])
const filterCountry = ref<string | null>(null)
const filterProtocol = ref<string | null>(null)

const countryOptions = computed(() => {
  const countries = [...new Set(nodes.value.map(n => n.country))]
  return countries.map(c => ({ label: c, value: c }))
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

const getCountryFlag = (country: string): string => {
  const flagMap: Record<string, string> = {
    '中国': '🇨🇳',
    '香港': '🇭🇰',
    '台湾': '🇹🇼',
    '日本': '🇯🇵',
    '韩国': '🇰🇷',
    '新加坡': '🇸🇬',
    '美国': '🇺🇸',
    '英国': '🇬🇧',
    '德国': '🇩🇪',
    '法国': '🇫🇷',
    '加拿大': '🇨🇦',
    '澳大利亚': '🇦🇺',
    '俄罗斯': '🇷🇺',
    '印度': '🇮🇳',
    '巴西': '🇧🇷',
    '荷兰': '🇳🇱',
    '瑞士': '🇨🇭',
    '意大利': '🇮🇹',
    '西班牙': '🇪🇸',
    '泰国': '🇹🇭',
    '越南': '🇻🇳',
    '马来西亚': '🇲🇾',
    '菲律宾': '🇵🇭',
    '印度尼西亚': '🇮🇩',
    '土耳其': '🇹🇷',
    '阿根廷': '🇦🇷',
    '墨西哥': '🇲🇽',
    '南非': '🇿🇦',
    '埃及': '🇪🇬',
    '以色列': '🇮🇱',
    '阿联酋': '🇦🇪',
    '沙特阿拉伯': '🇸🇦'
  }
  return flagMap[country] || '🌐'
}

const handleFilter = () => {
  // Filter is reactive, no action needed
}

const fetchNodes = async () => {
  loading.value = true
  try {
    const res = await listNodes()
    nodes.value = res.data || []
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
}

.node-card {
  height: 100%;
  border: 1px solid #e0e0e6;
  transition: all 0.3s ease;
}

.node-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  transform: translateY(-2px);
}

.node-flag {
  font-size: 24px;
  line-height: 1;
}

.label {
  color: #666;
  font-size: 14px;
}

.rate {
  font-weight: 600;
  color: #18a058;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}

.status-dot.online {
  background-color: #18a058;
  box-shadow: 0 0 4px rgba(24, 160, 88, 0.5);
}

.status-dot.offline {
  background-color: #d03050;
  box-shadow: 0 0 4px rgba(208, 48, 80, 0.5);
}

/* Mobile Responsive */
@media (max-width: 767px) {
  .node-page { padding: 0; }
  .node-card:hover { transform: none; }
  .node-flag { font-size: 20px; }
}
</style>
