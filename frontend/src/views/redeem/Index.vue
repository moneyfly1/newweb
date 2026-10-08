<template>
  <div class="redeem-page">
    <!-- 手机端：兑换码输入做成吸顶工具条（App 顶栏常驻，不跟着列表滚走） -->
    <div v-if="appStore.isMobile" class="app-sticky-toolbar redeem-toolbar">
      <n-input-group>
        <n-input v-model:value="code" placeholder="请输入兑换码" clearable size="large" />
        <n-button type="primary" size="large" :loading="submitting" @click="handleRedeem" :disabled="!code.trim()">兑换</n-button>
      </n-input-group>
      <n-alert v-if="result" :type="result.type" :title="result.title" class="redeem-result">{{ result.message }}</n-alert>
    </div>
    <n-card v-else title="卡密兑换">
      <n-space vertical :size="16">
        <n-input-group>
          <n-input v-model:value="code" placeholder="请输入兑换码" clearable size="large" />
          <n-button type="primary" size="large" :loading="submitting" @click="handleRedeem" :disabled="!code.trim()">兑换</n-button>
        </n-input-group>
        <n-alert v-if="result" :type="result.type" :title="result.title">{{ result.message }}</n-alert>
      </n-space>
    </n-card>
    <n-card title="兑换记录" class="redeem-history-card">
      <template v-if="!appStore.isMobile">
        <n-data-table :columns="columns" :data="history" :loading="loadingHistory" :bordered="false" />
      </template>
      <template v-else>
        <div v-if="loadingHistory" class="redeem-state"><n-spin size="medium" /></div>
        <div v-else-if="history.length === 0" class="mobile-empty">暂无兑换记录</div>
        <div v-else class="mobile-card-list">
          <div v-for="item in history" :key="item.id" class="mobile-card">
            <div class="card-header redeem-card-head">
              <span class="card-title">{{ item.code }}</span>
              <n-tag :type="item.type === 'balance' ? 'success' : 'info'" size="small" :bordered="false">{{ item.type === 'balance' ? '余额' : '套餐' }}</n-tag>
            </div>
            <div class="card-row"><span class="card-label">兑换值</span><span>{{ item.value }}</span></div>
            <div class="card-row"><span class="card-label">时间</span><span>{{ formatDateTime(item.created_at) }}</span></div>
          </div>
        </div>
      </template>
      <n-pagination
        v-if="totalHistory > pageSize"
        v-model:page="currentPage"
        v-model:page-size="pageSize"
        :item-count="totalHistory"
        :page-sizes="[10, 20, 50]"
        show-size-picker
        class="redeem-pagination"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { NTag } from 'naive-ui'
import { useAppStore } from '@/stores/app'
import { redeemCode, getRedeemHistory } from '@/api/common'
import { useMessage } from 'naive-ui'
import { formatDateTime } from '@/utils/date'

const appStore = useAppStore()
const message = useMessage()

const code = ref('')
const submitting = ref(false)
const result = ref<{ type: 'success' | 'error' | 'warning' | 'info'; title: string; message: string } | null>(null)
const history = ref<any[]>([])
const loadingHistory = ref(false)
const currentPage = ref(1)
const pageSize = ref(10)
const totalHistory = ref(0)

const columns = [
  { title: '兑换码', key: 'code' },
  { title: '类型', key: 'type', render: (row: any) => h(NTag, { type: row.type === 'balance' ? 'success' : 'info', size: 'small' }, { default: () => row.type === 'balance' ? '余额' : '套餐' }) },
  { title: '兑换值', key: 'value' },
  { title: '兑换时间', key: 'created_at', render: (row: any) => formatDateTime(row.created_at) },
]

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
    const res: any = await getRedeemHistory({ page: currentPage.value, page_size: pageSize.value })
    const data = res.data
    if (Array.isArray(data)) {
      history.value = data
      totalHistory.value = data.length
    } else {
      history.value = data?.items || []
      totalHistory.value = data?.total || 0
    }
  } catch (e: any) {
    message.error(e.message || '加载兑换记录失败')
  } finally {
    loadingHistory.value = false
  }
}

const handlePageChange = (page: number) => {
  currentPage.value = page
  loadHistory()
}

const handlePageSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
  loadHistory()
}

onMounted(() => {
  loadHistory()
})
</script>

<style scoped>
.redeem-page { padding: 24px; }
.redeem-history-card { margin-top: 16px; }
.redeem-result { margin-top: 10px; }
.redeem-state { text-align: center; padding: 32px 0; }
.redeem-pagination { margin-top: 16px; justify-content: flex-end; flex-wrap: wrap; row-gap: 8px; }

@media (max-width: 767px) {
  /* 手机端根容器不再自带左右内边距（左右留白由全局统一给 10px） */
  .redeem-page { padding: 10px 0 0; }
  .redeem-history-card { margin-top: 12px; }

  /* 吸顶工具条：全局 .app-sticky-toolbar 带 -12px 负外边距（给有内边距的容器用），
     本页根容器左右内边距已是 0，这里把负边距收回，避免撑出横向溢出 */
  .redeem-toolbar { margin-left: 0; margin-right: 0; }

  /* 卡片头部行：全局 .card-header 只有 flex 布局、没给内边距（手机端卡片 padding 为 0），这里只补 padding，不覆盖全局任何属性 */
  .mobile-card .redeem-card-head { padding: 14px 12px 0; }

  /* 分页在手机上居中并允许换行，避免页码 + 每页条数挤出一行 */
  .redeem-pagination { margin-top: 14px; justify-content: center; }
}
</style>
