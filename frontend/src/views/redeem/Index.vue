<template>
  <div class="redeem-page">
    <n-card title="卡密兑换">
      <n-space vertical :size="16">
        <n-input-group>
          <n-input v-model:value="code" placeholder="请输入兑换码" clearable size="large" />
          <n-button type="primary" size="large" :loading="submitting" @click="handleRedeem" :disabled="!code.trim()">兑换</n-button>
        </n-input-group>
        <n-alert v-if="result" :type="result.type" :title="result.title">{{ result.message }}</n-alert>
      </n-space>
    </n-card>
    <n-card title="兑换记录" style="margin-top:16px">
      <template v-if="!appStore.isMobile">
        <n-data-table :columns="columns" :data="history" :loading="loadingHistory" :bordered="false" />
      </template>
      <template v-else>
        <div v-if="loadingHistory" style="text-align:center;padding:40px"><n-spin size="medium" /></div>
        <div v-else-if="history.length === 0" style="text-align:center;padding:40px;color:#999">暂无兑换记录</div>
        <div v-else class="mobile-card-list">
          <div v-for="item in history" :key="item.id" class="mobile-card">
            <div class="card-header">
              <span class="card-title">{{ item.code }}</span>
              <n-tag :type="item.type === 'balance' ? 'success' : 'info'" size="small">{{ item.type === 'balance' ? '余额' : '套餐' }}</n-tag>
            </div>
            <div class="card-body">
              <div class="card-row"><span class="card-label">兑换值</span><span>{{ item.value }}</span></div>
              <div class="card-row"><span class="card-label">时间</span><span>{{ formatDate(item.created_at) }}</span></div>
            </div>
          </div>
        </div>
      </template>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { NTag } from 'naive-ui'
import { useAppStore } from '@/stores/app'
import { redeemCode, getRedeemHistory } from '@/api/common'
import { useMessage } from 'naive-ui'

const appStore = useAppStore()
const message = useMessage()

const code = ref('')
const submitting = ref(false)
const result = ref<{ type: 'success' | 'error' | 'warning' | 'info'; title: string; message: string } | null>(null)
const history = ref<any[]>([])
const loadingHistory = ref(false)

const columns = [
  { title: '兑换码', key: 'code' },
  { title: '类型', key: 'type', render: (row: any) => h(NTag, { type: row.type === 'balance' ? 'success' : 'info', size: 'small' }, { default: () => row.type === 'balance' ? '余额' : '套餐' }) },
  { title: '兑换值', key: 'value' },
  { title: '兑换时间', key: 'created_at', render: (row: any) => formatDate(row.created_at) },
]

const formatDate = (dateStr: string) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

const handleRedeem = async () => {
  if (!code.value.trim()) return

  submitting.value = true
  result.value = null

  try {
    const res: any = await redeemCode({ code: code.value.trim() })
    result.value = {
      type: 'success',
      title: '兑换成功',
      message: res.message || '卡密兑换成功'
    }
    message.success('兑换成功')
    code.value = ''
    loadHistory()
  } catch (error: any) {
    result.value = {
      type: 'error',
      title: '兑换失败',
      message: error.response?.data?.message || error.message || '兑换失败，请检查兑换码是否正确'
    }
    message.error(result.value.message)
  } finally {
    submitting.value = false
  }
}

const loadHistory = async () => {
  loadingHistory.value = true
  try {
    const res: any = await getRedeemHistory()
    history.value = res.data?.items || res.data || []
  } catch (e: any) {
    message.error(e.message || '加载兑换记录失败')
  } finally {
    loadingHistory.value = false
  }
}

onMounted(() => {
  loadHistory()
})
</script>

<style scoped>
.redeem-page { padding: 24px; }
.mobile-card-list { display: flex; flex-direction: column; gap: 12px; }
.mobile-card { background: #fff; border-radius: 10px; box-shadow: 0 1px 4px rgba(0,0,0,0.08); overflow: hidden; }
.card-header { display: flex; align-items: center; justify-content: space-between; padding: 12px 14px; border-bottom: 1px solid #f0f0f0; }
.card-title { font-weight: 600; font-size: 14px; }
.card-body { padding: 10px 14px; }
.card-row { display: flex; justify-content: space-between; padding: 4px 0; font-size: 13px; }
.card-label { color: #999; }
@media (max-width: 767px) { .redeem-page { padding: 0 12px; } }
</style>
