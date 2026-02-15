<template>
  <div class="mystery-box-admin">
    <n-card title="盲盒管理">
      <template #header-extra>
        <n-button type="primary" @click="handleAddPool">创建奖池</n-button>
      </template>

      <!-- 统计概览 -->
      <n-grid :cols="appStore.isMobile ? 2 : 4" :x-gap="12" :y-gap="12" style="margin-bottom:20px">
        <n-gi><n-statistic label="总开启次数" :value="stats.total_opens" /></n-gi>
        <n-gi><n-statistic label="总收入"><template #prefix>¥</template>{{ stats.total_revenue?.toFixed(2) || '0.00' }}</n-statistic></n-gi>
        <n-gi><n-statistic label="总奖品价值"><template #prefix>¥</template>{{ stats.total_prize_value?.toFixed(2) || '0.00' }}</n-statistic></n-gi>
        <n-gi><n-statistic label="奖池数量" :value="pools.length" /></n-gi>
      </n-grid>

      <!-- 奖池列表 -->
      <n-collapse>
        <n-collapse-item v-for="pool in pools" :key="pool.id" :title="pool.name" :name="pool.id">
          <template #header-extra>
            <n-space :size="8" @click.stop>
              <n-tag :type="pool.is_active ? 'success' : 'default'" size="small">{{ pool.is_active ? '启用' : '停用' }}</n-tag>
              <n-tag size="small">{{ pool.price }} 元</n-tag>
              <n-button size="tiny" @click.stop="handleEditPool(pool)">编辑</n-button>
              <n-button size="tiny" type="error" @click.stop="handleDeletePool(pool.id)">删除</n-button>
            </n-space>
          </template>

          <div style="margin-bottom:12px">
            <n-space :size="8">
              <n-tag v-if="pool.min_level" size="tiny">等级≥{{ pool.min_level }}</n-tag>
              <n-tag v-if="pool.min_balance" size="tiny">余额≥{{ pool.min_balance }}</n-tag>
              <n-tag v-if="pool.max_opens_per_day" size="tiny">每日限{{ pool.max_opens_per_day }}次</n-tag>
              <n-tag v-if="pool.max_opens_total" size="tiny">总限{{ pool.max_opens_total }}次</n-tag>
            </n-space>
          </div>

          <n-space justify="space-between" align="center" style="margin-bottom:8px">
            <n-text strong>奖品列表</n-text>
            <n-button size="small" type="primary" @click="handleAddPrize(pool.id)">添加奖品</n-button>
          </n-space>
          <n-data-table :columns="getPrizeColumns(pool)" :data="pool.prizes || []" :bordered="false" size="small" />
        </n-collapse-item>
      </n-collapse>
      <div v-if="pools.length === 0 && !loading" style="text-align:center;padding:40px 0;color:#999">暂无奖池</div>
    </n-card>

    <!-- 奖池表单弹窗 -->
    <n-modal v-model:show="showPoolModal" preset="card" :title="editingPool ? '编辑奖池' : '创建奖池'" :style="appStore.isMobile ? 'width:95vw' : 'width:600px'" :segmented="{ content: 'soft' }">
      <n-form :model="poolForm" label-placement="left" label-width="100">
        <n-form-item label="名称"><n-input v-model:value="poolForm.name" placeholder="奖池名称" /></n-form-item>
        <n-form-item label="描述"><n-input v-model:value="poolForm.description" type="textarea" placeholder="奖池描述（可选）" /></n-form-item>
        <n-form-item label="价格"><n-input-number v-model:value="poolForm.price" :min="0" :precision="2" style="width:100%" placeholder="开启价格" /></n-form-item>
        <n-form-item label="启用"><n-switch v-model:value="poolForm.is_active" /></n-form-item>
        <n-form-item label="排序"><n-input-number v-model:value="poolForm.sort_order" :min="0" style="width:100%" /></n-form-item>
        <n-form-item label="最低等级"><n-input-number v-model:value="poolForm.min_level" :min="0" style="width:100%" placeholder="不限制留空" clearable /></n-form-item>
        <n-form-item label="最低余额"><n-input-number v-model:value="poolForm.min_balance" :min="0" :precision="2" style="width:100%" placeholder="不限制留空" clearable /></n-form-item>
        <n-form-item label="每日限次"><n-input-number v-model:value="poolForm.max_opens_per_day" :min="1" style="width:100%" placeholder="不限制留空" clearable /></n-form-item>
        <n-form-item label="总限次"><n-input-number v-model:value="poolForm.max_opens_total" :min="1" style="width:100%" placeholder="不限制留空" clearable /></n-form-item>
        <n-form-item label="开始时间"><n-date-picker v-model:value="poolForm.start_time_ts" type="datetime" clearable style="width:100%" /></n-form-item>
        <n-form-item label="结束时间"><n-date-picker v-model:value="poolForm.end_time_ts" type="datetime" clearable style="width:100%" /></n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showPoolModal = false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="handleSubmitPool">确定</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 奖品表单弹窗 -->
    <n-modal v-model:show="showPrizeModal" preset="card" :title="editingPrize ? '编辑奖品' : '添加奖品'" :style="appStore.isMobile ? 'width:95vw' : 'width:500px'" :segmented="{ content: 'soft' }">
      <n-form :model="prizeForm" label-placement="left" label-width="80">
        <n-form-item label="名称"><n-input v-model:value="prizeForm.name" placeholder="奖品名称" /></n-form-item>
        <n-form-item label="类型">
          <n-select v-model:value="prizeForm.type" :options="prizeTypeOptions" />
        </n-form-item>
        <n-form-item label="数值">
          <n-input-number v-model:value="prizeForm.value" :min="0" :precision="2" style="width:100%" :placeholder="prizeValuePlaceholder" />
        </n-form-item>
        <n-form-item label="权重"><n-input-number v-model:value="prizeForm.weight" :min="1" style="width:100%" placeholder="越大概率越高" /></n-form-item>
        <n-form-item label="库存"><n-input-number v-model:value="prizeForm.stock" :min="0" style="width:100%" placeholder="留空为无限" clearable /></n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showPrizeModal = false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="handleSubmitPrize">确定</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, h, onMounted } from 'vue'
import { NButton, NTag, NSpace, useMessage, useDialog } from 'naive-ui'
import { useAppStore } from '@/stores/app'
import {
  listAdminMysteryBoxPools, createMysteryBoxPool, updateMysteryBoxPool, deleteMysteryBoxPool,
  addMysteryBoxPrize, updateMysteryBoxPrize, deleteMysteryBoxPrize, getMysteryBoxStats,
} from '@/api/admin'

const appStore = useAppStore()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const submitting = ref(false)
const pools = ref<any[]>([])
const stats = ref<any>({})

// Pool form
const showPoolModal = ref(false)
const editingPool = ref<any>(null)
const poolForm = reactive({
  name: '', description: '' as string | null, price: 0, is_active: true, sort_order: 0,
  min_level: null as number | null, min_balance: null as number | null,
  max_opens_per_day: null as number | null, max_opens_total: null as number | null,
  start_time_ts: null as number | null, end_time_ts: null as number | null,
})

// Prize form
const showPrizeModal = ref(false)
const editingPrize = ref<any>(null)
const currentPoolId = ref<number>(0)
const prizeForm = reactive({ name: '', type: 'balance', value: 0, weight: 1, stock: null as number | null })

const prizeTypeOptions = [
  { label: '余额', value: 'balance' },
  { label: '优惠券', value: 'coupon' },
  { label: '订阅天数', value: 'subscription_days' },
  { label: '谢谢参与', value: 'nothing' },
]

const prizeValuePlaceholder = computed(() => {
  const map: Record<string, string> = { balance: '余额金额', coupon: '优惠券面值', subscription_days: '天数', nothing: '0' }
  return map[prizeForm.type] || '数值'
})

const prizeTagType = (type: string) => {
  const map: Record<string, any> = { balance: 'success', coupon: 'info', subscription_days: 'warning', nothing: 'default' }
  return map[type] || 'default'
}
const prizeTypeLabel = (type: string) => {
  const map: Record<string, string> = { balance: '余额', coupon: '优惠券', subscription_days: '订阅天数', nothing: '谢谢参与' }
  return map[type] || type
}

const getProbability = (pool: any, prize: any) => {
  if (!pool?.prizes?.length) return '0%'
  const totalWeight = pool.prizes.reduce((sum: number, p: any) => sum + (p.weight || 0), 0)
  if (totalWeight <= 0) return '0%'
  return ((prize.weight / totalWeight) * 100).toFixed(1) + '%'
}

const getPrizeColumns = (pool: any) => [
  { title: '名称', key: 'name' },
  { title: '类型', key: 'type', width: 100, render: (row: any) => h(NTag, { type: prizeTagType(row.type), size: 'small' }, { default: () => prizeTypeLabel(row.type) }) },
  { title: '数值', key: 'value', width: 80 },
  { title: '权重', key: 'weight', width: 80 },
  { title: '概率', key: 'probability', width: 80, render: (row: any) => getProbability(pool, row) },
  { title: '库存', key: 'stock', width: 80, render: (row: any) => row.stock === null || row.stock === undefined ? '无限' : row.stock },
  {
    title: '操作', key: 'actions', width: 140,
    render: (row: any) => h(NSpace, { size: 4 }, {
      default: () => [
        h(NButton, { size: 'tiny', onClick: () => handleEditPrize(row) }, { default: () => '编辑' }),
        h(NButton, { size: 'tiny', type: 'error', onClick: () => handleDeletePrize(row.id) }, { default: () => '删除' }),
      ]
    })
  },
]

const loadPools = async () => {
  loading.value = true
  try {
    const res: any = await listAdminMysteryBoxPools()
    pools.value = res.data || []
  } catch (e: any) {
    message.error(e.message || '加载奖池失败')
  } finally {
    loading.value = false
  }
}

const loadStats = async () => {
  try {
    const res: any = await getMysteryBoxStats()
    stats.value = res.data || {}
  } catch {}
}

const resetPoolForm = () => {
  poolForm.name = ''; poolForm.description = null; poolForm.price = 0; poolForm.is_active = true
  poolForm.sort_order = 0; poolForm.min_level = null; poolForm.min_balance = null
  poolForm.max_opens_per_day = null; poolForm.max_opens_total = null
  poolForm.start_time_ts = null; poolForm.end_time_ts = null
}

const handleAddPool = () => {
  editingPool.value = null
  resetPoolForm()
  showPoolModal.value = true
}

const handleEditPool = (pool: any) => {
  editingPool.value = pool
  poolForm.name = pool.name; poolForm.description = pool.description; poolForm.price = pool.price
  poolForm.is_active = pool.is_active; poolForm.sort_order = pool.sort_order
  poolForm.min_level = pool.min_level; poolForm.min_balance = pool.min_balance
  poolForm.max_opens_per_day = pool.max_opens_per_day; poolForm.max_opens_total = pool.max_opens_total
  poolForm.start_time_ts = pool.start_time ? new Date(pool.start_time).getTime() : null
  poolForm.end_time_ts = pool.end_time ? new Date(pool.end_time).getTime() : null
  showPoolModal.value = true
}

const handleSubmitPool = async () => {
  if (!poolForm.name) { message.warning('请输入奖池名称'); return }
  submitting.value = true
  const data: any = {
    name: poolForm.name, description: poolForm.description || null, price: poolForm.price,
    is_active: poolForm.is_active, sort_order: poolForm.sort_order,
    min_level: poolForm.min_level || null, min_balance: poolForm.min_balance || null,
    max_opens_per_day: poolForm.max_opens_per_day || null, max_opens_total: poolForm.max_opens_total || null,
    start_time: poolForm.start_time_ts ? new Date(poolForm.start_time_ts).toISOString() : null,
    end_time: poolForm.end_time_ts ? new Date(poolForm.end_time_ts).toISOString() : null,
  }
  try {
    if (editingPool.value) {
      await updateMysteryBoxPool(editingPool.value.id, data)
      message.success('更新成功')
    } else {
      await createMysteryBoxPool(data)
      message.success('创建成功')
    }
    showPoolModal.value = false
    loadPools()
  } catch (e: any) {
    message.error(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleDeletePool = (id: number) => {
  dialog.warning({
    title: '确认删除', content: '删除奖池将同时删除其所有奖品，确定继续？',
    positiveText: '确定', negativeText: '取消',
    onPositiveClick: async () => {
      try { await deleteMysteryBoxPool(id); message.success('删除成功'); loadPools() }
      catch (e: any) { message.error(e.message || '删除失败') }
    }
  })
}

const handleAddPrize = (poolId: number) => {
  editingPrize.value = null
  currentPoolId.value = poolId
  prizeForm.name = ''; prizeForm.type = 'balance'; prizeForm.value = 0; prizeForm.weight = 1; prizeForm.stock = null
  showPrizeModal.value = true
}

const handleEditPrize = (prize: any) => {
  editingPrize.value = prize
  currentPoolId.value = prize.pool_id
  prizeForm.name = prize.name; prizeForm.type = prize.type; prizeForm.value = prize.value
  prizeForm.weight = prize.weight; prizeForm.stock = prize.stock
  showPrizeModal.value = true
}

const handleSubmitPrize = async () => {
  if (!prizeForm.name) { message.warning('请输入奖品名称'); return }
  submitting.value = true
  const data = { name: prizeForm.name, type: prizeForm.type, value: prizeForm.value, weight: prizeForm.weight, stock: prizeForm.stock }
  try {
    if (editingPrize.value) {
      await updateMysteryBoxPrize(editingPrize.value.id, data)
      message.success('更新成功')
    } else {
      await addMysteryBoxPrize(currentPoolId.value, data)
      message.success('添加成功')
    }
    showPrizeModal.value = false
    loadPools()
  } catch (e: any) {
    message.error(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleDeletePrize = (id: number) => {
  dialog.warning({
    title: '确认删除', content: '确定要删除这个奖品吗？',
    positiveText: '确定', negativeText: '取消',
    onPositiveClick: async () => {
      try { await deleteMysteryBoxPrize(id); message.success('删除成功'); loadPools() }
      catch (e: any) { message.error(e.message || '删除失败') }
    }
  })
}

onMounted(() => { loadPools(); loadStats() })
</script>

<style scoped>
.mystery-box-admin { padding: 20px; }
@media (max-width: 767px) { .mystery-box-admin { padding: 8px; } }
</style>
